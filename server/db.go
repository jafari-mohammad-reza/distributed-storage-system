package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

func init() {
	redisClient = db.NewRedisClient()
}

func createUser(email, agent, password string) error {
	if user, err := findUser(email); err == nil && user != nil {
		return errors.New("email already exists")
	}
	agents := []db.Agent{{Name: agent}}
	id := uuid.New().String()
	user := db.User{
		ID:       id,
		Email:    email,
		Agents:   agents,
		Password: password,
	}
	return db.Insert(context.Background(), redisClient, email, user)
}

func findUser(email string) (*db.User, error) {

	var users []db.User
	result, err := db.Get(context.Background(), redisClient, email)
	if err != nil {
		return nil, err
	}
	fmt.Println("RESULT", result)
	json.Unmarshal([]byte(result), &users)
	if len(users) == 0 {
		return nil, nil
	}
	return &users[0], nil
}

func agentExists(email, agent string) (bool, error) {
	user, err := findUser(email)
	if err != nil {
		return false, err
	}
	for _, a := range user.Agents {
		if a.Name == agent {
			return true, nil
		}
	}
	return false, nil
}

func updateAgents(email, agent string) error {
	exists, err := agentExists(email, agent)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	user, err := findUser(email)
	if err != nil {
		return err
	}

	user.Agents = append(user.Agents, db.Agent{Name: agent})
	return db.Insert(context.Background(), redisClient, email, user)
}

func deleteAgent(email, agent string) error {
	user, err := findUser(email)
	if err != nil {
		return err
	}

	newAgents := []db.Agent{}
	for _, a := range user.Agents {
		if a.Name != agent {
			newAgents = append(newAgents, a)
		}
	}

	user.Agents = newAgents
	return db.Insert(context.Background(), redisClient, email, user)
}

func uploadFile(tr *pkg.TransferPacket, uploadHash string) error {
	_, err := findUser(tr.Email)
	if err != nil {
		return err
	}
	id := uuid.New().String()
	versions := []db.FileVersion{}
	versions = append(versions, db.FileVersion{
		ID:        uuid.New().String(),
		Hash:      uploadHash,
		CreatedAt: time.Now(),
	})
	upload := db.File{
		ID:         id,
		Name:       tr.FileName,
		Path:       tr.Dir,
		UploadedAt: tr.UploadedIn,
		UploadedBy: tr.Agent,
		Versions:   versions,
	}
	return db.Insert(context.Background(), redisClient, uploadHash, upload)
}
func updateFileStorages(uploadHash, storageId string) error {
	return db.AppendArray(context.Background(), redisClient, uploadHash, storageId, "$storages")
}
func flushRedis() {
	fmt.Println("flushing redis")
	err := db.FlushRedis(context.Background(), redisClient)
	if err != nil {
		log.Error("failed to flush redis")
	}
}
