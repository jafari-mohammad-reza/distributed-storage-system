package pkg

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInitTcpListener(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		err := InitTcpListener(8080, func(tr *TransferPacket, packetBytes []byte) error {
			return nil
		})
		assert.Nil(t, err)
		time.Sleep(1 * time.Second)
	}()

	packet, err := CompressFile("p2p.go", SenderMeta{
		Email: "test@gmail.com",
		Agent: "test-agent",
	})
	if err != nil {
		assert.Nil(t, err)
	}

	serialized, err := SerializePacket(packet)
	if err != nil {
		assert.Nil(t, err)
	}
	err = SendDataOverTcp(8080, int64(len(serialized)), serialized)
	assert.Nil(t, err)

	wg.Wait()
}
