package server

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvokeAndRevoke(t *testing.T) {
	err := InitSql()
	assert.Nil(t, err)
	createUserErr := createUser("testuser@gmail.com", "test-agent", "testpassowrd")
	assert.Nil(t, createUserErr)
	foundUser, err := findUser("testuser@gmail.com")
	assert.Nil(t, err)
	assert.NotNil(t, foundUser)
	assert.Equal(t, foundUser.email, "testuser@gmail.com")
	createUser2Err := createUser("testuser@gmail.com", "test-agent", "testpassowrd")
	assert.NotNil(t, createUser2Err)
	agentExist := agentExist(foundUser.id, "test-agent")
	assert.True(t, agentExist)
	delteAgentErr := deleteAgent(foundUser.email, foundUser.agents[0])
	assert.Nil(t, delteAgentErr)
	token, err := generateApiKey(foundUser.email, foundUser.agents[0])
	assert.Nil(t, err)
	assert.NotNil(t, token)
	decoded, err := decodeToken(token)
	assert.Nil(t, err)
	assert.NotNil(t, decoded)
	assert.Equal(t, decoded["email"].(string), foundUser.email)
	assert.Equal(t, decoded["agent"].(string), foundUser.agents[0])
	defer cleanup()
}
func cleanup() {
	os.Remove("database.sqlite")
}
