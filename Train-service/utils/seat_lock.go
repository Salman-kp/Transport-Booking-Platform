package utils

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	SeatLockTTL    = 10 * time.Minute
	seatLockPrefix = "seat:lock:train"
)

func seatLockKey(scheduleID, seatID string) string {
	return fmt.Sprintf("%s:%s:%s", seatLockPrefix, scheduleID, seatID)
}

func LockSeat(ctx context.Context, rdb *goredis.Client, scheduleID, seatID, userID string) (bool, error) {
	key := seatLockKey(scheduleID, seatID)
	acquired, err := rdb.SetNX(ctx, key, userID, SeatLockTTL).Result()
	if err != nil {
		return false, fmt.Errorf("redis SetNX failed for key %s: %w", key, err)
	}
	return acquired, nil
}

func LockSeats(ctx context.Context, rdb *goredis.Client, scheduleID string, seatIDs []string, userID string) (error, string) {
	locked := make([]string, 0, len(seatIDs))

	for _, seatID := range seatIDs {
		acquired, err := LockSeat(ctx, rdb, scheduleID, seatID, userID)
		if err != nil {
			_ = releaseAll(ctx, rdb, scheduleID, locked)
			return err, ""
		}
		if !acquired {
			_ = releaseAll(ctx, rdb, scheduleID, locked)
			return nil, seatID
		}
		locked = append(locked, seatID)
	}

	return nil, ""
}

func UnlockSeat(ctx context.Context, rdb *goredis.Client, scheduleID, seatID string) error {
	key := seatLockKey(scheduleID, seatID)
	if err := rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis Del failed for key %s: %w", key, err)
	}
	return nil
}

func UnlockSeats(ctx context.Context, rdb *goredis.Client, scheduleID string, seatIDs []string) error {
	if len(seatIDs) == 0 {
		return nil
	}
	keys := make([]string, len(seatIDs))
	for i, id := range seatIDs {
		keys[i] = seatLockKey(scheduleID, id)
	}
	if err := rdb.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("redis Del (batch) failed: %w", err)
	}
	return nil
}

func IsSeatLocked(ctx context.Context, rdb *goredis.Client, scheduleID, seatID string) (bool, error) {
	key := seatLockKey(scheduleID, seatID)
	val, err := rdb.Get(ctx, key).Result()
	if err == goredis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis Get failed for key %s: %w", key, err)
	}
	return val != "", nil
}

func GetSeatLockOwner(ctx context.Context, rdb *goredis.Client, scheduleID, seatID string) (string, error) {
	key := seatLockKey(scheduleID, seatID)
	val, err := rdb.Get(ctx, key).Result()
	if err == goredis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("redis Get failed for key %s: %w", key, err)
	}
	return val, nil
}

func releaseAll(ctx context.Context, rdb *goredis.Client, scheduleID string, seatIDs []string) error {
	return UnlockSeats(ctx, rdb, scheduleID, seatIDs)
}
