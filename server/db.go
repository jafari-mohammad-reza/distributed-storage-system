package server

import (
	"database/sql"
	"errors"
	"log/slog"

	_ "github.com/mattn/go-sqlite3"
)

var _conn *sql.DB

func InitSql() error {
	conn, err := sql.Open("sqlite3", "file:database.sqlite?cache=shared")
	if err != nil {
		return err
	}
	_conn = conn
	if err := initDb(); err != nil {
		return err
	}
	return nil
}
func initDb() error {
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
	`
	_, err := _conn.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func createUser(email, agent, password string) error {
	query := `INSERT INTO users (email , password) VALUES (? , ?)`
	_, err := _conn.Exec(query, email, password)
	if err != nil {
		return err
	}
	_, err = _conn.Exec("INSERT INTO agents (user_id, agent) VALUES ((SELECT id FROM users WHERE email = ?), ?)", email, agent)
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
	var result findUserResult
	slog.Info("searching for user", "email", email)

	query := `SELECT id, email FROM users WHERE email = ?`
	err := _conn.QueryRow(query, email).Scan(&result.id, &result.email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn("user not found", "email", email)
			return nil, nil
		}
		return nil, err
	}

	query = `SELECT agent FROM agents WHERE user_id = ?`
	rows, err := _conn.Query(query, result.id)
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
	findQuery := `SELECT id FROM agents WHERE agent = ? AND user_id = ?`
	var foundId int
	rows, err := _conn.Query(findQuery, agent, userId)
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
	if agentExist(userId, agent) {
		return nil
	}
	query := `INSERT INTO agents (user_id, agent) VALUES (?, ?)`
	_, err := _conn.Exec(query, userId, agent)
	if err != nil {
		return err
	}
	return nil
}
func deleteAgent(email, agent string) error {
	query := `DELETE FROM agents WHERE user_id = (SELECT id FROM users WHERE email = ?) AND agent = ?`
	_, err := _conn.Exec(query, email, agent)
	if err != nil {
		return err
	}
	return nil
}
