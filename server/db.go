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
		Files:    []db.File{},
	}
	return db.Insert(context.Background(), redisClient, email, user)
}

func findUser(email string) (*db.User, error) {

	var users []db.User
	result, err := db.Get(context.Background(), redisClient, email)
	if err != nil {
		return nil, err
	}
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
	newAgent, _ := json.Marshal(db.Agent{Name: agent})
	return db.AppendArray(context.Background(), redisClient, email, newAgent, "$.agents")
}

func deleteAgent(email, agent string) error {
	user, err := findUser(email)
	if err != nil {
		return err
	}
	index := -1
	for i, ag := range user.Agents {
		if ag.Name == agent {
			index = i
		}
	}
	if index == -1 {
		return errors.New("invalid agent to remove")
	}
	return db.PopArray(context.Background(), redisClient, email, "$.agents[0]", agent, index)
}

func uploadFile(tr *pkg.TransferPacket, uploadHash string) error {
	userFilesPath := "$.files[*]"
	existingFiles, err := db.GetArray(context.Background(), redisClient, tr.Email, userFilesPath)
	if err != nil {
		return err
	}
	meta := tr.Meta
	for _, file := range existingFiles {
		filename := meta["FileName"]
		dir := meta["Dir"]
		if file["name"] == filename && file["path"] == dir {
			version := db.FileVersion{
				ID:        uuid.New().String(),
				Hash:      uploadHash,
				Storages:  []string{},
				CreatedAt: time.Now().Format(time.RFC3339),
			}

			fileVersionPath := fmt.Sprintf("$.files[?(@.name=='%s' && @.path=='%s')].versions", filename, dir)
			versionJson, _ := json.Marshal(version)
			return db.AppendArray(context.Background(), redisClient, tr.Email, string(versionJson), fileVersionPath)
		}
	}

	uploadedIn, _ := time.Parse("2006-01-02T15:04:05.000Zs", tr.Meta["UploadedIn"])
	upload := db.File{
		ID:         uuid.New().String(),
		Name:       meta["FileName"],
		Path:       meta["Dir"],
		UploadedAt: uploadedIn.Format(time.RFC3339),
		UploadedBy: tr.Agent,
		Versions: []db.FileVersion{
			{
				ID:        uuid.New().String(),
				Hash:      uploadHash,
				Storages:  []string{},
				CreatedAt: time.Now().Format(time.RFC3339),
			},
		},
	}
	uploadJson, _ := json.Marshal(upload)
	return db.AppendArray(context.Background(), redisClient, tr.Email, string(uploadJson), "$.files")
}
func updateFileStorages(tr *pkg.TransferPacket, uploadHash, storageId string) error {
	path := fmt.Sprintf("$.files[*].versions[?(@.hash=='%s')].storages", uploadHash)
	id, _ := json.Marshal(storageId)
	return db.AppendArray(context.Background(), redisClient, tr.Email, string(id), path)
}

func flushRedis() {
	err := db.FlushRedis(context.Background(), redisClient)
	if err != nil {
		log.Error("failed to flush redis")
	}
}
