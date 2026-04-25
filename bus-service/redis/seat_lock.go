package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	seatLockPrefix = "bus:seat:lock"
)

// seatLockKey generates a deterministic lock key for a specific seat on a specific bus schedule.
func seatLockKey(busInstanceID, seatID string) string {
	return fmt.Sprintf("%s:%s:%s", seatLockPrefix, busInstanceID, seatID)
}

// LockSeat atomically claims a single seat. Returns true if acquired.
func LockSeat(ctx context.Context, rdb *redis.Client, busInstanceID, seatID, userID string, ttl time.Duration) (bool, error) {
	key := seatLockKey(busInstanceID, seatID)
	// shadow key for notification triggers (shadow:seat_lock:<userID>:<busInstanceID>:<seatID>)
	shadowKey := fmt.Sprintf("shadow:seat_lock:%s:%s:%s", userID, busInstanceID, seatID)

	acquired, err := rdb.SetNX(ctx, key, userID, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis SetNX failed for key %s: %w", key, err)
	}

	if acquired {
		// Set shadow key with the same TTL
		rdb.Set(ctx, shadowKey, "1", ttl)
	}

	return acquired, nil
}

// LockSeats atomically claims multiple seats. If any fail, it rolls back previously successfully locked seats (All-or-Nothing).
// Returns an error if any seat fails to lock natively.
func LockSeats(ctx context.Context, rdb *redis.Client, busInstanceID string, seatIDs []string, userID string, ttl time.Duration) error {
	locked := make([]string, 0, len(seatIDs))

	for _, seatID := range seatIDs {
		acquired, err := LockSeat(ctx, rdb, busInstanceID, seatID, userID, ttl)
		if err != nil {
			_ = UnlockSeatsByOwner(ctx, rdb, busInstanceID, locked, userID)
			return err
		}
		if !acquired {
			_ = UnlockSeatsByOwner(ctx, rdb, busInstanceID, locked, userID)
			return fmt.Errorf("seat %s is already locked by another user", seatID)
		}
		locked = append(locked, seatID)
	}

	return nil
}

// UnlockSeatsByOwner safely and conditionally releases multiple seat locks ONLY if the attached user is the actual verified owner.
// Executed utilizing a Lua script to guarantee atomicity and completely guard against race conditions.
func UnlockSeatsByOwner(ctx context.Context, rdb *redis.Client, busInstanceID string, seatIDs []string, ownerID string) error {
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)

	for _, seatID := range seatIDs {
		key := seatLockKey(busInstanceID, seatID)
		_ = script.Run(ctx, rdb, []string{key}, ownerID).Err()
	}

	return nil
}

// UnlockSeat unconditionally releases an immediate raw lock. Primarily used post-confirmation or backend expiration logic cleanup.
func UnlockSeat(ctx context.Context, rdb *redis.Client, busInstanceID, seatID string) error {
	key := seatLockKey(busInstanceID, seatID)
	return rdb.Del(ctx, key).Err()
}

// IsSeatLocked validates whether a given seat is actively bound inside Redis.
func IsSeatLocked(ctx context.Context, rdb *redis.Client, busInstanceID, seatID string) (bool, error) {
	key := seatLockKey(busInstanceID, seatID)
	val, err := rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis Get failed for key %s: %w", key, err)
	}
	return val != "", nil
}

// GetSeatLockOwner returns the raw ownership UUID natively associated with a locked constraint.
func GetSeatLockOwner(ctx context.Context, rdb *redis.Client, busInstanceID, seatID string) (string, error) {
	key := seatLockKey(busInstanceID, seatID)
	val, err := rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("redis Get failed for key %s: %w", key, err)
	}
	return val, nil
}
