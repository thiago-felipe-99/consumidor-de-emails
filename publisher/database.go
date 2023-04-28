package main

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type queueData struct {
	Name      string    `db:"name"`
	DLX       string    `db:"dlx"`
	CreatedAt time.Time `db:"created_at"`
}

type database struct {
	db *sqlx.DB
}

func (database *database) addQueue(names, dlx string) error {
	queue := queueData{names, dlx, time.Now()}

	_, err := database.db.NamedExec(
		"INSERT INTO queues (name, dlx, created_at) VALUES (:name, :dlx, :created_at)",
		queue,
	)
	if err != nil {
		return fmt.Errorf("error adding queue on database: %w", err)
	}

	return nil
}

func (database *database) exist(name string) bool {
	queue := queueData{}

	err := database.db.Get(&queue, "SELECT name, created_at FROM queues WHERE name=$1", name)

	return err == nil
}

func newDatabase() (*database, error) {
	dbConnection, err := sqlx.Connect(
		"postgres",
		"user=postgres password=postgres dbname=postgres sslmode=disable",
	)
	if err != nil {
		return nil, fmt.Errorf("error connecting with the database: %w", err)
	}

	return &database{dbConnection}, nil
}
