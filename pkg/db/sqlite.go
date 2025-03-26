package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var _conn *sql.DB

func InitSqlite() error {
	conn, err := sql.Open("sqlite3", "file:database.sqlite?cache=shared")
	if err != nil {
		return err
	}
	_conn = conn
	return nil
}
func GetConn() *sql.DB {
	return _conn
}
