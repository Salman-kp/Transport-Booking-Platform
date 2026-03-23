package utils

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return client, mr
}

func TestLockSeat_AcquireAndRelease(t *testing.T) {
	rdb, mr := newTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	scheduleID := "sched-001"
	seatID := "seat-001"
	userID := "user-abc"

	acquired, err := LockSeat(ctx, rdb, scheduleID, seatID, userID)
	if err != nil {
		t.Fatalf("unexpected error on first lock: %v", err)
	}
	if !acquired {
		t.Fatal("expected lock to be acquired on first attempt")
	}

	acquired, err = LockSeat(ctx, rdb, scheduleID, seatID, userID)
	if err != nil {
		t.Fatalf("unexpected error on second lock: %v", err)
	}
	if acquired {
		t.Fatal("expected lock to fail — seat should already be locked")
	}

	if err := UnlockSeat(ctx, rdb, scheduleID, seatID); err != nil {
		t.Fatalf("unexpected error on unlock: %v", err)
	}

	acquired, err = LockSeat(ctx, rdb, scheduleID, seatID, "user-xyz")
	if err != nil {
		t.Fatalf("unexpected error after unlock: %v", err)
	}
	if !acquired {
		t.Fatal("expected lock to succeed after unlock")
	}
}

func TestLockSeat_DifferentSeats(t *testing.T) {
	rdb, mr := newTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	scheduleID := "sched-001"

	acquired, err := LockSeat(ctx, rdb, scheduleID, "seat-001", "user-a")
	if err != nil || !acquired {
		t.Fatal("seat-001 lock should succeed")
	}

	acquired, err = LockSeat(ctx, rdb, scheduleID, "seat-002", "user-b")
	if err != nil || !acquired {
		t.Fatal("seat-002 lock should succeed independently")
	}
}

func TestLockSeats_AllOrNothing(t *testing.T) {
	rdb, mr := newTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	scheduleID := "sched-001"

	_, _ = LockSeat(ctx, rdb, scheduleID, "seat-002", "other-user")

	errResult, conflictSeat := LockSeats(ctx, rdb, scheduleID,
		[]string{"seat-001", "seat-002", "seat-003"}, "user-abc")

	if errResult != nil {
		t.Fatalf("unexpected Redis error: %v", errResult)
	}
	if conflictSeat != "seat-002" {
		t.Fatalf("expected conflict on seat-002, got: %s", conflictSeat)
	}

	locked, _ := IsSeatLocked(ctx, rdb, scheduleID, "seat-001")
	if locked {
		t.Fatal("seat-001 should have been released after batch failure")
	}
	locked, _ = IsSeatLocked(ctx, rdb, scheduleID, "seat-003")
	if locked {
		t.Fatal("seat-003 should have been released after batch failure")
	}
}

func TestLockSeat_Concurrency(t *testing.T) {
	rdb, mr := newTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	scheduleID := "sched-flash"
	seatID := "seat-hot"

	const goroutines = 100
	successCount := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			userID := fmt.Sprintf("user-%d", n)
			acquired, err := LockSeat(ctx, rdb, scheduleID, seatID, userID)
			if err != nil {
				t.Errorf("goroutine %d: Redis error: %v", n, err)
				return
			}
			if acquired {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 lock acquisition, got %d", successCount)
	}
}

func TestIsSeatLocked(t *testing.T) {
	rdb, mr := newTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	scheduleID := "sched-001"
	seatID := "seat-001"

	locked, err := IsSeatLocked(ctx, rdb, scheduleID, seatID)
	if err != nil || locked {
		t.Fatal("seat should not be locked before any lock call")
	}

	_, _ = LockSeat(ctx, rdb, scheduleID, seatID, "user-a")

	locked, err = IsSeatLocked(ctx, rdb, scheduleID, seatID)
	if err != nil || !locked {
		t.Fatal("seat should be locked after LockSeat call")
	}
}

func TestUnlockSeat_NonExistentKey(t *testing.T) {
	rdb, mr := newTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	err := UnlockSeat(ctx, rdb, "sched-x", "seat-x")
	if err != nil {
		t.Fatalf("expected no error unlocking non-existent key, got: %v", err)
	}
}
