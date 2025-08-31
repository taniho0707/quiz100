package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	*sql.DB
	path string
}

func NewDatabase(dataSourceName string) (*Database, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	database := &Database{
		DB:   db,
		path: dataSourceName,
	}

	if err := database.InitSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %v", err)
	}

	return database, nil
}

func (db *Database) InitSchema() error {
	schemaPath := filepath.Join("database", "init.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %v", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to execute schema: %v", err)
	}

	log.Println("Database schema initialized successfully")
	return nil
}

func (db *Database) Close() error {
	return db.DB.Close()
}

func (db *Database) GetPath() string {
	return db.path
}
