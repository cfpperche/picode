package store

import (
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

var queueTag atomic.Int64

// queueFixture registers a fresh delivery and enqueues it. Every call uses its
// own request ids so a row that needs an entry of its own never replays the
// previous one's receipt.
func queueFixture(t *testing.T, s *Store) QueueEntry {
	t.Helper()
	tag := strconv.FormatInt(queueTag.Add(1), 10)
	d, err := s.ApplyDelivery("repo", "agent", DeliveryMutation{Action: "register", RequestID: "create-" + tag,
		Title: "Fix " + tag, Branch: "feat/fix-" + tag, Revision: strings.Repeat("a", 40), Target: "main"})
	if err != nil {
		t.Fatal(err)
	}
	e, err := s.ApplyQueueMutation("repo", "agent", QueueMutation{Action: "enqueue", RequestID: "enqueue-" + tag,
		DeliveryID: d.ID, Revision: d.Revision, Target: d.Target})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// queueAt returns a fresh entry advanced to the requested state by the owner,
// which is how the executor reaches running and the terminal states.
func queueAt(t *testing.T, s *Store, state string) QueueEntry {
	t.Helper()
	e := queueFixture(t, s)
	if state == QueueWaiting {
		return e
	}
	step := func(m QueueMutation) {
		t.Helper()
		next, err := s.ApplyQueueMutation("repo", OwnerActor, m)
		if err != nil {
			t.Fatalf("advance to %s: %v", state, err)
		}
		e = next
	}
	step(QueueMutation{Action: "authorize", ID: e.ID, ExpectedVersion: e.Version, RequestID: "auth-" + e.ID})
	if state == QueueAuthorized {
		return e
	}
	step(QueueMutation{Action: "start", ID: e.ID, ExpectedVersion: e.Version, RequestID: "start-" + e.ID})
	if state == QueueRunning {
		return e
	}
	step(QueueMutation{Action: "finish", ID: e.ID, ExpectedVersion: e.Version, RequestID: "finish-" + e.ID, Note: "integrated"})
	return e
}

// The shape check: what each action must and must not carry.
func TestQueueMutationValidation(t *testing.T) {
	const rev = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for _, tc := range []struct {
		name string
		m    QueueMutation
		want string
	}{
		{"no request id", QueueMutation{Action: "enqueue", DeliveryID: "d", Revision: rev, Target: "main"}, "requestId is required"},
		{"unknown action", QueueMutation{Action: "rebase", RequestID: "r"}, "unsupported queue action"},
		{"enqueue without a delivery", QueueMutation{Action: "enqueue", RequestID: "r", Revision: rev, Target: "main"}, "deliveryId, a full lowercase revision and target are required"},
		{"enqueue with a short revision", QueueMutation{Action: "enqueue", RequestID: "r", DeliveryID: "d", Revision: "abc", Target: "main"}, "deliveryId, a full lowercase revision and target are required"},
		{"enqueue with an id", QueueMutation{Action: "enqueue", RequestID: "r", ID: "e1", DeliveryID: "d", Revision: rev, Target: "main"}, "enqueue takes deliveryId, revision and target only"},
		{"withdraw without an id", QueueMutation{Action: "withdraw", RequestID: "r", ExpectedVersion: 1}, "id and expectedVersion are required"},
		{"withdraw with a note", QueueMutation{Action: "withdraw", RequestID: "r", ID: "e1", ExpectedVersion: 1, Note: "x"}, "this action takes only id, expectedVersion and requestId"},
		{"order without a key", QueueMutation{Action: "order", RequestID: "r", ID: "e1", ExpectedVersion: 1}, "order takes an orderKey"},
		{"order with a target", QueueMutation{Action: "order", RequestID: "r", ID: "e1", ExpectedVersion: 1, OrderKey: new(int64(3)), Target: "main"}, "order takes only id, expectedVersion, orderKey and requestId"},
		{"finish without a note", QueueMutation{Action: "finish", RequestID: "r", ID: "e1", ExpectedVersion: 1}, "a finish or fail needs a one-line note"},
		{"fail with a two-line note", QueueMutation{Action: "fail", RequestID: "r", ID: "e1", ExpectedVersion: 1, Note: "one\ntwo"}, "a finish or fail needs a one-line note"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateQueueMutation(tc.m)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

// The decision table: from this state, for this actor, this action either moves
// the entry or is refused with the reason.
func TestQueueTransitionDecisionTable(t *testing.T) {
	for _, tc := range []struct {
		name      string
		state     string
		actor     string
		m         func(e QueueEntry) QueueMutation
		wantErr   string
		wantState string
	}{
		{"the owner authorizes a waiting entry", QueueWaiting, OwnerActor, func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "authorize", ID: e.ID, ExpectedVersion: e.Version, RequestID: "a"}
		}, "", QueueAuthorized},
		{"an agent cannot authorize", QueueWaiting, "agent", func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "authorize", ID: e.ID, ExpectedVersion: e.Version, RequestID: "a"}
		}, "only the owner may authorize", ""},
		{"an agent cannot order", QueueWaiting, "agent", func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "order", ID: e.ID, ExpectedVersion: e.Version, OrderKey: new(int64(1)), RequestID: "a"}
		}, "only the owner may order", ""},
		{"the owner orders a waiting entry", QueueWaiting, OwnerActor, func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "order", ID: e.ID, ExpectedVersion: e.Version, OrderKey: new(int64(7)), RequestID: "a"}
		}, "", QueueWaiting},
		{"a running entry cannot be ordered", QueueRunning, OwnerActor, func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "order", ID: e.ID, ExpectedVersion: e.Version, OrderKey: new(int64(1)), RequestID: "a"}
		}, "cannot be ordered", ""},
		{"the declarer withdraws its own entry", QueueWaiting, "agent", func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "withdraw", ID: e.ID, ExpectedVersion: e.Version, RequestID: "a"}
		}, "", QueueWithdrawn},
		{"another agent cannot withdraw", QueueWaiting, "other", func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "withdraw", ID: e.ID, ExpectedVersion: e.Version, RequestID: "a"}
		}, "only the agent that declared this delivery, or the owner, may withdraw", ""},
		{"the owner withdraws a waiting entry", QueueWaiting, OwnerActor, func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "withdraw", ID: e.ID, ExpectedVersion: e.Version, RequestID: "a"}
		}, "", QueueWithdrawn},
		{"a finished entry cannot be withdrawn", QueueDone, OwnerActor, func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "withdraw", ID: e.ID, ExpectedVersion: e.Version, RequestID: "a"}
		}, "cannot be withdrawn", ""},
		{"start needs authorization first", QueueWaiting, OwnerActor, func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "start", ID: e.ID, ExpectedVersion: e.Version, RequestID: "a"}
		}, "must be authorized first", ""},
		{"the owner's operation starts an authorized entry", QueueAuthorized, OwnerActor, func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "start", ID: e.ID, ExpectedVersion: e.Version, RequestID: "a"}
		}, "", QueueRunning},
		{"finish needs a running entry", QueueAuthorized, OwnerActor, func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "finish", ID: e.ID, ExpectedVersion: e.Version, Note: "integrated", RequestID: "a"}
		}, "cannot finish", ""},
		{"finish closes a running entry with its note", QueueRunning, OwnerActor, func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "finish", ID: e.ID, ExpectedVersion: e.Version, Note: "integrated as bbbbbbb", RequestID: "a"}
		}, "", QueueDone},
		{"fail closes a running entry with its reason", QueueRunning, OwnerActor, func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "fail", ID: e.ID, ExpectedVersion: e.Version, Note: "the base moved while it ran", RequestID: "a"}
		}, "", QueueFailed},
		{"an agent cannot finish", QueueRunning, "agent", func(e QueueEntry) QueueMutation {
			return QueueMutation{Action: "finish", ID: e.ID, ExpectedVersion: e.Version, Note: "done", RequestID: "a"}
		}, "only the owner's operation may finish", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := openTest(t)
			e := queueAt(t, s, tc.state)
			got, err := s.ApplyQueueMutation("repo", tc.actor, tc.m(e))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.State != tc.wantState {
				t.Fatalf("state = %s, want %s", got.State, tc.wantState)
			}
			stored, err := s.GetQueueEntry("repo", got.ID)
			if err != nil || stored.State != tc.wantState || stored.Version != got.Version {
				t.Fatalf("stored = %+v (%v)", stored, err)
			}
		})
	}
}

// Structural guards that do not depend on a state.
func TestQueueEnqueueGuards(t *testing.T) {
	s := openTest(t)
	d := deliveryFixture(t, s)
	e, err := s.ApplyQueueMutation("repo", "agent", QueueMutation{Action: "enqueue", RequestID: "one", DeliveryID: d.ID, Revision: d.Revision, Target: d.Target})
	if err != nil {
		t.Fatal(err)
	}
	if e.State != QueueWaiting || e.Version != 1 || e.Sequence == 0 || e.OrderKey != e.Sequence {
		t.Fatalf("entry = %+v", e)
	}
	// One active entry per delivery: a second one is refused while the first waits.
	if _, err := s.ApplyQueueMutation("repo", "agent", QueueMutation{Action: "enqueue", RequestID: "two", DeliveryID: d.ID, Revision: d.Revision, Target: d.Target}); !errors.Is(err, ErrQueueConflict) {
		t.Fatalf("second enqueue = %v", err)
	}
	// Withdrawn, it may be enqueued again.
	if _, err := s.ApplyQueueMutation("repo", "agent", QueueMutation{Action: "withdraw", ID: e.ID, ExpectedVersion: 1, RequestID: "three"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ApplyQueueMutation("repo", "agent", QueueMutation{Action: "enqueue", RequestID: "four", DeliveryID: d.ID, Revision: d.Revision, Target: d.Target}); err != nil {
		t.Fatalf("re-enqueue after withdraw: %v", err)
	}
	// An entry names a declaration of this repository.
	if _, err := s.ApplyQueueMutation("repo", "agent", QueueMutation{Action: "enqueue", RequestID: "five", DeliveryID: "delivery_nope", Revision: d.Revision, Target: d.Target}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown delivery = %v", err)
	}
	if _, err := s.ApplyQueueMutation("other", "agent", QueueMutation{Action: "enqueue", RequestID: "six", DeliveryID: d.ID, Revision: d.Revision, Target: d.Target}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other repository = %v", err)
	}
}

// A retry repeats the original receipt; a different payload under the same
// request ID is a conflict, and a stale version never overwrites a writer.
func TestQueueReplayAndVersions(t *testing.T) {
	s := openTest(t)
	e := queueFixture(t, s)
	m := QueueMutation{Action: "authorize", ID: e.ID, ExpectedVersion: 1, RequestID: "auth"}
	first, err := s.ApplyQueueMutation("repo", OwnerActor, m)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.ApplyQueueMutation("repo", OwnerActor, m)
	if err != nil {
		t.Fatal(err)
	}
	if again.Version != first.Version || again.State != QueueAuthorized {
		t.Fatalf("replay = %+v, first = %+v", again, first)
	}
	changed := m
	changed.ID = "queue_other"
	if _, err := s.ApplyQueueMutation("repo", OwnerActor, changed); !errors.Is(err, ErrQueueConflict) {
		t.Fatalf("reused request id = %v", err)
	}
	if _, err := s.ApplyQueueMutation("repo", OwnerActor, QueueMutation{Action: "start", ID: e.ID, ExpectedVersion: 1, RequestID: "start"}); !errors.Is(err, ErrQueueConflict) {
		t.Fatalf("stale version = %v", err)
	}
}

// The list is what runs next: active entries by order key, then the rest.
func TestQueueListOrdersActiveFirst(t *testing.T) {
	s := openTest(t)
	first := queueFixture(t, s)
	second := queueFixture(t, s)
	if _, err := s.ApplyQueueMutation("repo", OwnerActor, QueueMutation{Action: "order", ID: second.ID, ExpectedVersion: second.Version, OrderKey: new(int64(0)), RequestID: "order"}); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListQueueEntries("repo", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != second.ID || list[1].ID != first.ID {
		t.Fatalf("list = %+v", list)
	}
}

// An entry is bound to what the delivery declares: a caller that names a
// different revision or target is asking for something nobody reviewed.
func TestQueueEntryMustMatchTheDeclaration(t *testing.T) {
	s := openTest(t)
	d, err := s.ApplyDelivery("repo", "agent", DeliveryMutation{Action: "register", RequestID: "create",
		Title: "Fix", Branch: "feat/fix", Revision: strings.Repeat("a", 40), Target: "main"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		revision string
		target   string
	}{
		{"another revision", strings.Repeat("b", 40), "main"},
		{"another target", d.Revision, "release"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.ApplyQueueMutation("repo", "agent", QueueMutation{Action: "enqueue", RequestID: "e-" + tc.name,
				DeliveryID: d.ID, Revision: tc.revision, Target: tc.target})
			if !errors.Is(err, ErrQueueConflict) {
				t.Fatalf("err = %v", err)
			}
			if entries, err := s.QueueEntriesForDelivery("repo", d.ID); err != nil || len(entries) != 0 {
				t.Fatalf("refused enqueue left %v (%v)", entries, err)
			}
		})
	}
	e := queueFixture(t, s)
	entries, err := s.QueueEntriesForDelivery("repo", e.DeliveryID)
	if err != nil || len(entries) != 1 || entries[0].ID != e.ID {
		t.Fatalf("entries = %v (%v)", entries, err)
	}
	if other, err := s.QueueEntriesForDelivery("repo", "d_missing"); err != nil || len(other) != 0 {
		t.Fatalf("unknown delivery = %v (%v)", other, err)
	}
}
