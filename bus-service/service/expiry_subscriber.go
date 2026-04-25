package service

import (
	"context"
	"fmt"
	"log"

	"github.com/Salman-kp/tripneo/bus-service/redis"
	"github.com/Salman-kp/tripneo/bus-service/ws"
)

func StartRedisExpirySubscriber() {
	if redis.Client == nil {
		log.Fatal("Cannot start expiry subscriber: Redis client is not initialized")
	}

	ctx := context.Background()

	err := redis.Client.ConfigSet(ctx, "notify-keyspace-events", "Ex").Err()
	if err != nil {
		log.Printf("Warning: Failed to set notify-keyspace-events config. Make sure it's enabled in Redis server: %v", err)
	}

	pubsub := redis.Client.Subscribe(ctx, "__keyevent@0__:expired")
	defer pubsub.Close()

	// wait for confirmation that subscription is created before publishing anything.
	_, err = pubsub.Receive(ctx)
	if err != nil {
		log.Fatalf("Warning: Failed to subscribe to keyspace events: %v", err)
	}

	// go channel which receives messages.
	ch := pubsub.Channel()

	log.Println("Started Redis Keyspace Expiry Subscriber (Bus Service)")

	for msg := range ch {
		// msg.Payload will contain the name of the expired key, e.g., "shadow:seat_lock:<userID>:<busInstanceID>:<seatID>"
		key := msg.Payload

		var userID, busInstanceID, seatID string
		_, err := fmt.Sscanf(key, "shadow:seat_lock:%[^:]:%[^:]:%s", &userID, &busInstanceID, &seatID)
		if err == nil && userID != "" && busInstanceID != "" && seatID != "" {
			log.Printf("EXPIRED event received for seat %s on bus %s locked by %s", seatID, busInstanceID, userID)

			// notify the user via WebSocket
			message := map[string]interface{}{
				"type":    "SESSION_EXPIRED",
				"message": "Your hold on the selected seat has expired.",
				"seat_id": seatID,
				"bus_instance_id": busInstanceID,
			}
			ws.DefaultManager.SendToUser(userID, message)
		}
	}
}
