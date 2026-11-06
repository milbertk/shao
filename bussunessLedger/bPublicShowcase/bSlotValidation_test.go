package blpublicshowcase

import (
	"sync"
	"testing"
	"time"
)

func TestHHMM(t *testing.T) {
	for in, want := range map[string]string{"9:00": "09:00", "09:30:00": "09:30", "": "", "14:05": "14:05"} {
		if got := hhmm(in); got != want {
			t.Fatalf("%q -> %q", in, got)
		}
	}
}

func TestTokens(t *testing.T) {
	if !validToken("a_0123456789abcdef0123456789abcdef0123456789abcdef") || validToken("x_0123456789abcdef0123") || validToken("a_../../etc") {
		t.Fatal("token validation")
	}
}

func TestLockSerializes(t *testing.T) {
	var wg sync.WaitGroup
	inside, max := 0, 0
	var mu sync.Mutex
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := lockSlot("c", "s", "2026-10-30")
			mu.Lock()
			inside++
			if inside > max {
				max = inside
			}
			mu.Unlock()
			time.Sleep(2 * time.Millisecond)
			mu.Lock()
			inside--
			mu.Unlock()
			unlock()
		}()
	}
	wg.Wait()
	if max != 1 || len(slotLocks) != 0 {
		t.Fatal("lock", max, len(slotLocks))
	}
}
