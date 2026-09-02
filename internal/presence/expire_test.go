package presence

import (
	"testing"
	"time"
)

func TestExpireAnnouncesOnceAndPingRevives(t *testing.T) {
	r := New(nil)
	var seen []string
	r.OnChange = func(d Device) {
		state := "on"
		if !d.Online {
			state = "off"
		}
		seen = append(seen, d.ID+":"+state)
	}
	r.Ping("d1", "ua", "10.0.0.2:1", false, "")
	time.Sleep(10 * time.Millisecond) // OnChange for pings is async
	if len(r.Expire()) != 0 {
		t.Fatal("fresh device expired")
	}
	r.mu.Lock()
	r.items["d1"].last = time.Now().Add(-staleAfter - time.Second)
	r.mu.Unlock()
	if gone := r.Expire(); len(gone) != 1 || gone[0].ID != "d1" || gone[0].Online {
		t.Fatalf("expire = %+v", gone)
	}
	if len(r.Expire()) != 0 {
		t.Fatal("expired twice")
	}
	r.Ping("d1", "ua", "10.0.0.2:1", false, "")
	time.Sleep(10 * time.Millisecond)
	if len(seen) != 3 || seen[0] != "d1:on" || seen[1] != "d1:off" || seen[2] != "d1:on" {
		t.Fatalf("seen = %v", seen)
	}
}
