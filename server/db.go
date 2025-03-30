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
	userFilesPath := "$.files[*]"
	existingFiles, err := db.GetArray(context.Background(), redisClient, tr.Email, userFilesPath)
	if err != nil {
		return err
	}

	// Look for an existing file with the same name and path
	for _, file := range existingFiles {
		if file["name"] == tr.FileName && file["path"] == tr.Dir {
			// Append a new version to the existing file
			version := db.FileVersion{
				ID:        uuid.New().String(),
				Hash:      uploadHash,
				Storages:  []string{},
				CreatedAt: time.Now().Format(time.RFC3339),
			}

			fileVersionPath := fmt.Sprintf("$.files[?(@.name=='%s' && @.path=='%s')].versions", tr.FileName, tr.Dir)
			versionJson, _ := json.Marshal(version)
			return db.AppendArray(context.Background(), redisClient, tr.Email, string(versionJson), fileVersionPath)
		}
	}

	// If file doesn't exist, create a new entry
	upload := db.File{
		ID:         uuid.New().String(),
		Name:       tr.FileName,
		Path:       tr.Dir,
		UploadedAt: tr.UploadedIn.Format(time.RFC3339),
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
