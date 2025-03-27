package server

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
	_ "github.com/mattn/go-sqlite3"
)

func InitDb() error {
	query := `CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(150) NOT NULL,
    registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
	CREATE TABLE IF NOT EXISTS agents (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	agent VARCHAR(50) NOT NULL
	);
	CREATE TABLE IF NOT EXISTS uploads (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	agent_id INTEGER NOT NULL,
	file_name VARCHAR(50) NOT NULL,
	file_path VARCHAR(150) NOT NULL,
	uploaded_at TIMESTAMP,
	uploaded_in TEXT 
	);
	`
	conn := db.GetConn()
	_, err := conn.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func createUser(email, agent, password string) error {
	conn := db.GetConn()
	query := `INSERT INTO users (email , password) VALUES (? , ?)`
	_, err := conn.Exec(query, email, password)
	if err != nil {
		return err
	}
	_, err = conn.Exec("INSERT INTO agents (user_id, agent) VALUES ((SELECT id FROM users WHERE email = ?), ?)", email, agent)
	if err != nil {
		return err
	}
	return nil
}

type findUserResult struct {
	id     int
	email  string
	agents []string
}

func findUser(email string) (*findUserResult, error) {
	conn := db.GetConn()
	var result findUserResult
	slog.Info("searching for user", "email", email)

	query := `SELECT id, email FROM users WHERE email = ?`
	err := conn.QueryRow(query, email).Scan(&result.id, &result.email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn("user not found", "email", email)
			return nil, nil
		}
		return nil, err
	}

	query = `SELECT agent FROM agents WHERE user_id = ?`
	rows, err := conn.Query(query, result.id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var agent string
		if err := rows.Scan(&agent); err != nil {
			return nil, err
		}
		result.agents = append(result.agents, agent)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &result, nil
}
func agentExist(userId int, agent string) bool {
	conn := db.GetConn()
	findQuery := `SELECT id FROM agents WHERE agent = ? AND user_id = ?`
	var foundId int
	rows, err := conn.Query(findQuery, agent, userId)
	if err != nil {
		return false
	}
	for rows.Next() {
		if err := rows.Scan(&foundId); err != nil {
			return false
		}
	}
	if foundId != 0 {
		return true
	}
	return false
}
func updateAgents(userId int, agent string) error {
	conn := db.GetConn()
	if agentExist(userId, agent) {
		return nil
	}
	query := `INSERT INTO agents (user_id, agent) VALUES (?, ?)`
	_, err := conn.Exec(query, userId, agent)
	if err != nil {
		return err
	}
	return nil
}
func deleteAgent(email, agent string) error {
	conn := db.GetConn()
	query := `DELETE FROM agents WHERE user_id = (SELECT id FROM users WHERE email = ?) AND agent = ?`
	_, err := conn.Exec(query, email, agent)
	if err != nil {
		return err
	}
	return nil
}

func insertUpload(tr *pkg.TransferPacket) error {
	conn := db.GetConn()
	user, err := findUser(tr.Email)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	query := `INSERT INTO uploads (user_id, agent_id, file_name, file_path, uploaded_at) 
			  VALUES (?, (SELECT id FROM agents WHERE agent = ? AND user_id = ?), ?, ?, ?)`
	_, err = conn.Exec(query, user.id, tr.Agent, user.id, tr.FileName, tr.Dir,tr.UploadedIn)
	if err != nil {
		return err
	}
	return nil
}

type findUploadResult struct {
	id          int
	uploaded_in string
}

func updateUploadStorage(fileName, dir, storageId string) error {
	conn := db.GetConn()
	findQuery := "SELECT id,uploaded_in FROM uploads WHERE file_name = ? AND file_path = ? ORDER BY uploaded_at DESC LIMIT 1;"
	var result findUploadResult
	err := conn.QueryRow(findQuery, fileName, dir).Scan(&result.id, &result.uploaded_in)
	if err != nil {
		return err
	}
	query := `UPDATE uploads SET uploaded_in = ? WHERE id = ?`
	var uploadedIn string
	if result.uploaded_in == "" {
		uploadedIn = storageId
	} else {
		uploadedIn = fmt.Sprintf("%s,%s", result.uploaded_in, storageId)
	}
	_, err = conn.Exec(query, uploadedIn, result.id)
	if err != nil {
		return err
	}
	return nil
}

// in server side we should handle created hahs for each same file in same dir version
