package server

import (
	"testing"

	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/stretchr/testify/assert"
)

func TestInvokeAndRevoke(t *testing.T) {
	createUserErr := createUser("testuser@gmail.com", "test-agent", "testpassowrd")
	assert.Nil(t, createUserErr)
	foundUser, err := findUser("testuser@gmail.com")
	assert.Nil(t, err)
	assert.NotNil(t, foundUser)
	assert.Equal(t, foundUser.Email, "testuser@gmail.com")
	createUser2Err := createUser("testuser@gmail.com", "test-agent", "testpassowrd")
	assert.NotNil(t, createUser2Err)
	agentExist, err := agentExists(foundUser.Email, "test-agent")
	assert.Nil(t, err)
	assert.True(t, agentExist)
	delteAgentErr := deleteAgent(foundUser.Email, foundUser.Agents[0].Name)
	assert.Nil(t, delteAgentErr)
	token, err := pkg.GenerateApiKey(foundUser.Email, foundUser.Agents[0].Name)
	assert.Nil(t, err)
	assert.NotNil(t, token)
	decoded, err := pkg.DecodeToken(token)
	assert.Nil(t, err)
	assert.NotNil(t, decoded)
	assert.Equal(t, decoded["email"].(string), foundUser.Email)
	assert.Equal(t, decoded["agent"].(string), foundUser.Agents[0].Name)
	defer flushRedis()
}
