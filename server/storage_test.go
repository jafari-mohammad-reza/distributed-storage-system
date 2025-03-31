package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestRegisterSystem(t *testing.T) {

	storages = make(map[string]pkg.Storage)

	redisClient := db.NewRedisClient()
	assert.Len(t, storages, 0)

	initRegisterSystem("server1", redisClient)
	time.Sleep(1 * time.Second)

	for i := 0; i < 5; i++ {
		db.Produce(context.Background(), redisClient, "storage-stream", map[string]interface{}{
			"ID":   fmt.Sprintf("storage%d", i),
			"Port": i,
		})
		time.Sleep(time.Second)
	}

	assert.Equal(t, 5, len(storages), "Expected 5 storages to be registered")

	fmt.Println("Storage count after addition:", len(storages))

	for i := 0; i < 5; i++ {
		db.Produce(context.Background(), redisClient, "disconnect-stream", map[string]interface{}{
			"ID":   fmt.Sprintf("storage%d", i),
			"Port": i,
		})
		time.Sleep(1 * time.Second)
	}

	assert.Equal(t, 0, len(storages), "Expected all storages to be removed")

	fmt.Println("Storage count after removal:", len(storages))
}
