package server

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestRegisterSystem(t *testing.T) {
	storages = make(map[string]pkg.Storage)
	redisClient := db.NewRedisClient()
	assert.Len(t, storages, 0)
	wg := sync.WaitGroup{}
	wg.Add(5)
	initRegisterSystem("server1", redisClient, &wg)
	for i := range 5 {
		fmt.Println("producing", i)
		db.Produce(context.Background(), redisClient, "storage-stream", map[string]interface{}{"ID": fmt.Sprintf("storage%d", i)})
	}
	assert.Greater(t, len(storages), 0)
	wg.Wait()
}

// TODO: write test for healthCheckStorages
