package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

func NewDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return db, nil
}

func RunMigrations(db *sql.DB) error {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_name = 'memories'
		)`).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("database schema not found - please run docs/schema.sql in Supabase first")
	}
	return nil
}