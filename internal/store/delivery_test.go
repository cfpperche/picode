package store

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func deliveryFixture(t *testing.T, s *Store) Delivery {
	t.Helper()
	d, err := s.ApplyDelivery("repo", "agent", DeliveryMutation{Action: "register", RequestID: "create", Title: "Fix", Branch: "feat/fix", Revision: strings.Repeat("a", 40), Target: "main"})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDeliveryMutationDecisionTable(t *testing.T) {
	s := openTest(t)
	d := deliveryFixture(t, s)
	rows := []struct {
		name            string
		m               DeliveryMutation
		repo, principal string
		want            error
	}{
		{"review", DeliveryMutation{Action: "request-review", ID: d.ID, ExpectedVersion: 1, RequestID: "review"}, "repo", "agent", nil},
		{"stale write", DeliveryMutation{Action: "withdraw-review", ID: d.ID, ExpectedVersion: 1, RequestID: "stale"}, "repo", "agent", ErrDeliveryConflict},
		{"other actor", DeliveryMutation{Action: "withdraw-review", ID: d.ID, ExpectedVersion: 2, RequestID: "foreign"}, "repo", "other", ErrNotFound},
		{"other repo", DeliveryMutation{Action: "withdraw-review", ID: d.ID, ExpectedVersion: 2, RequestID: "foreign"}, "other", "agent", ErrNotFound},
		{"withdraw", DeliveryMutation{Action: "withdraw-review", ID: d.ID, ExpectedVersion: 2, RequestID: "withdraw"}, "repo", "agent", nil},
		{"review again", DeliveryMutation{Action: "request-review", ID: d.ID, ExpectedVersion: 3, RequestID: "review-again"}, "repo", "agent", nil},
		{"update resets review", DeliveryMutation{Action: "update", ID: d.ID, ExpectedVersion: 4, RequestID: "update", Title: "New title", Branch: "feat/fix", Revision: strings.Repeat("b", 40), Target: "main"}, "repo", "agent", nil},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			got, err := s.ApplyDelivery(row.repo, row.principal, row.m)
			if !errors.Is(err, row.want) {
				t.Fatalf("%+v %v", got, err)
			}
		})
	}
	current, err := s.GetDelivery("repo", d.ID)
	if err != nil || current.Version != 5 || current.Review != "not-requested" || current.ReviewRequestedAt != "" {
		t.Fatalf("%+v %v", current, err)
	}
	// Replaying the old review returns that receipt, without reverting the update.
	old, err := s.ApplyDelivery("repo", "agent", rows[0].m)
	if err != nil || old.Version != 2 {
		t.Fatalf("old receipt %+v %v", old, err)
	}
	current, _ = s.GetDelivery("repo", d.ID)
	if current.Version != 5 {
		t.Fatal("retry changed current state")
	}
	bad := rows[0].m
	bad.ExpectedVersion = 5
	if _, err = s.ApplyDelivery("repo", "agent", bad); !errors.Is(err, ErrDeliveryConflict) {
		t.Fatal(err)
	}
	events, _ := s.EventsOfType("delivery.changed", 100)
	if len(events) != 5 {
		t.Fatalf("events %d", len(events))
	}
}

func TestDeliveryConcurrencyAndRollback(t *testing.T) {
	s := openTest(t)
	d := deliveryFixture(t, s)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := s.ApplyDelivery("repo", "agent", DeliveryMutation{Action: "request-review", ID: d.ID, ExpectedVersion: 1, RequestID: fmt.Sprint(i)})
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	success, conflict := 0, 0
	for err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, ErrDeliveryConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("%d %d", success, conflict)
	}
	_, err := s.db.Exec(`CREATE TRIGGER delivery_fail_event BEFORE INSERT ON events WHEN NEW.type='delivery.changed' BEGIN SELECT RAISE(ABORT,'fixture failure'); END`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.ApplyDelivery("repo", "agent", DeliveryMutation{Action: "withdraw-review", ID: d.ID, ExpectedVersion: 2, RequestID: "rollback"})
	if err == nil {
		t.Fatal("expected event failure")
	}
	got, _ := s.GetDelivery("repo", d.ID)
	if got.Version != 2 {
		t.Fatal("row escaped rollback")
	}
	var n int
	_ = s.db.QueryRow(`SELECT count(*) FROM delivery_requests WHERE request_id='rollback'`).Scan(&n)
	if n != 0 {
		t.Fatal("receipt escaped rollback")
	}
}

func TestDeliveryRegistrationRetryAndPagination(t *testing.T) {
	s := openTest(t)
	deliveryFixture(t, s)
	again := deliveryFixture(t, s)
	for i := 0; i < 101; i++ {
		_, err := s.ApplyDelivery("repo", "agent", DeliveryMutation{Action: "register", RequestID: fmt.Sprint(i), Title: "More", Branch: "fix", Revision: strings.Repeat("a", 40), Target: "main"})
		if err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.ListDeliveries("repo", 0)
	if err != nil || len(page) != 101 {
		t.Fatalf("%d %v", len(page), err)
	}
	next, err := s.ListDeliveries("repo", page[99].Sequence)
	if err != nil || len(next) != 2 || next[1].ID != again.ID {
		t.Fatalf("%+v %v", next, err)
	}
	if _, err = s.GetDelivery("other", again.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestDeliveryInputValidation(t *testing.T) {
	valid := DeliveryMutation{Action: "register", RequestID: "r", Title: "Fix", Branch: "fix", Revision: strings.Repeat("a", 40), Target: "main"}
	for _, edit := range []func(*DeliveryMutation){func(m *DeliveryMutation) { m.RequestID = "" }, func(m *DeliveryMutation) { m.Revision = "short" }, func(m *DeliveryMutation) { m.Title = "" }, func(m *DeliveryMutation) { m.Title = "bad\nline" }, func(m *DeliveryMutation) { m.Action = "merge" }, func(m *DeliveryMutation) { m.ExpectedVersion = 2 }, func(m *DeliveryMutation) { m.ID = "invented" }} {
		m := valid
		edit(&m)
		if ValidateDeliveryMutation(m) == nil {
			t.Fatalf("accepted %+v", m)
		}
	}
}

func TestDeliveryCapacity(t *testing.T) {
	for _, table := range []string{"declarations", "receipts"} {
		t.Run(table, func(t *testing.T) {
			s := openTest(t)
			d := deliveryFixture(t, s)
			var err error
			if table == "declarations" {
				_, err = s.db.Exec(`WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<999) INSERT INTO delivery_intents(id,repo,principal,body) SELECT 'f-'||x,'repo','agent','{}' FROM n`)
			} else {
				_, err = s.db.Exec(`WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<9999) INSERT INTO delivery_requests(repo,principal,request_id,payload,result) SELECT 'repo','agent','f-'||x,'{}','{}' FROM n`)
			}
			if err != nil {
				t.Fatal(err)
			}
			m := DeliveryMutation{Action: "register", RequestID: "next", Title: "Fix", Branch: d.Branch, Revision: d.Revision, Target: d.Target}
			if _, err = s.ApplyDelivery("repo", "agent", m); !errors.Is(err, ErrDeliveryCapacity) {
				t.Fatal(err)
			}
			m.RequestID = "create"
			if got, err := s.ApplyDelivery("repo", "agent", m); err != nil || got.ID != d.ID {
				t.Fatalf("retry %+v %v", got, err)
			}
		})
	}
}

func TestDeliveryRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	d := deliveryFixture(t, s)
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.GetDelivery("repo", d.ID)
	if err != nil || got != d {
		t.Fatalf("%+v %v", got, err)
	}
	m := DeliveryMutation{Action: "register", RequestID: "create", Title: "Fix", Branch: d.Branch, Revision: d.Revision, Target: d.Target}
	got, err = s.ApplyDelivery("repo", "agent", m)
	if err != nil || got != d {
		t.Fatalf("%+v %v", got, err)
	}
	events, err := s.EventsOfType("delivery.changed", 100)
	if err != nil || len(events) != 1 {
		t.Fatalf("%+v %v", events, err)
	}
}
