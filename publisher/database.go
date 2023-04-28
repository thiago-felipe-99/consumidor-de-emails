package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type queueData struct {
	ID        uuid.UUID `bson:"_id"`
	Name      string    `bson:"name"`
	DLX       string    `bson:"dlx"`
	CreatedAt time.Time `bson:"created_at"`
}

type database struct {
	db *mongo.Database
}

func (database *database) addQueue(names, dlx string) error {
	queue := queueData{uuid.New(), names, dlx, time.Now()}

	_, err := database.db.Collection("queues").InsertOne(context.Background(), queue)
	if err != nil {
		return fmt.Errorf("error adding queue on database: %w", err)
	}

	return nil
}

func (database *database) existQueue(name string) (bool, error) {
	filter := bson.D{
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "name", Value: name}},
			bson.D{{Key: "dlx", Value: name}},
		}},
	}

	count, err := database.db.Collection("queues").CountDocuments(context.Background(), filter)
	if err != nil {
		return false, fmt.Errorf("error verify if queue exist: %w", err)
	}

	return count >= 1, nil
}

func newDatabase() (*database, error) {
	uri := "mongodb://mongo:mongo@localhost:27017/?connectTimeoutMS=10000&timeoutMS=5000&maxIdleTimeMS=100"

	connection, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("error connecting with the database: %w", err)
	}

	err = connection.Ping(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("error ping server: %w", err)
	}

	return &database{connection.Database("email")}, nil
}
