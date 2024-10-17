package main

import (
	"database/sql"
	"os"
	"path/filepath"
)

type TaskStore struct {
	db *sql.DB
}

func NewTaskStore(db *sql.DB) TaskStore {
	return TaskStore{db: db}
}

func checkDB() error {
	envPath := os.Getenv("TODO_DBFILE")
	if len(envPath) > 0 {
		dbPath = envPath
	} else {
		appPath, err := os.Executable()
		if err != nil {
			return err
		}
		dbPath = filepath.Join(filepath.Dir(appPath), dbName)
	}

	exist, err := dbExist()
	if err != nil {
		return err
	}

	if !exist {
		if err := createDB(); err != nil {
			return err
		}
	}
	return nil
}

func dbExist() (bool, error) {

	_, err := os.Stat(dbPath)

	var install bool
	if err != nil {
		install = true
	}

	return !install, nil
}

func createDB() error {

	file, err := os.Create(dbPath)
	if err != nil {
		return err
	}
	file.Close()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	schema := `CREATE TABLE IF NOT EXISTS scheduler (
		id		INTEGER PRIMARY KEY AUTOINCREMENT,
		date	VARCHAR(8),
		title	VARCHAR(128),
		comment	VARCHAR(128),
		repeat	VARCHAR(128)
	);
	`
	query, err := db.Prepare(schema)
	if err != nil {
		return err
	}
	_, err = query.Exec()
	if err != nil {
		return err
	}

	return nil
}
