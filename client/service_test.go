package client

import (
	"testing"

	"github.com/jafari-mohammad-reza/dotsync/server"
	"github.com/stretchr/testify/assert"
)

func TestInvokeToken(t *testing.T) {
	go server.InitServer()
	err := Auth("test@gmail.com", "testPassword")
	assert.Nil(t, err)
	token, err := loadTokenFromFile()
	assert.Nil(t, err)
	assert.NotNil(t, token)
	err = removeTokenFromFile()
	assert.Nil(t, err)
	token, err = loadTokenFromFile()
	assert.Equal(t, token, "")
	assert.Nil(t, err)
}
func TestRevokeToken(t *testing.T) {
	err := Auth("test@gmail.com", "testPassword")
	assert.Nil(t, err)
	err = RevokeToken()
	assert.Nil(t, err)
	token, err := loadTokenFromFile()
	assert.Equal(t, token, "")
	assert.Nil(t, err)
}
func TestUploadFile(t *testing.T) {
	Auth("test@gmail.com", "testPassword")
	err := UploadFile("./service_test.go")
	assert.Nil(t, err)
	err = UploadFile("./service_test_invalid.go")
	assert.NotNil(t, err)
	RevokeToken()
	err = UploadFile("./service_test.go")
	assert.NotNil(t, err)
}
