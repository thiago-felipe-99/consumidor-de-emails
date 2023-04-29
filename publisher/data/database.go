package data

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/thiago-felipe-99/mail/publisher/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewDatabase() (*mongo.Client, error) {
	uri := "mongodb://mongo:mongo@localhost:27017/?connectTimeoutMS=10000&timeoutMS=5000&maxIdleTimeMS=100"

	connection, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("error connecting with the database: %w", err)
	}

	err = connection.Ping(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("error ping server: %w", err)
	}

	return connection, nil
}

type Queue struct {
	db *mongo.Database
}

func (database *Queue) Add(queue model.Queue) error {
	_, err := database.db.Collection("queues").InsertOne(context.Background(), queue)
	if err != nil {
		return fmt.Errorf("error adding queue on database: %w", err)
	}

	return nil
}

func (database *Queue) GetAll() ([]model.Queue, error) {
	queues := []model.Queue{}

	cursor, err := database.db.Collection("queues").Find(context.Background(), bson.D{})
	if err != nil {
		return nil, fmt.Errorf("error getting queues: %w", err)
	}

	err = cursor.All(context.Background(), &queues)
	if err != nil {
		return nil, fmt.Errorf("error parsing queues: %w", err)
	}

	return queues, nil
}

func (database *Queue) Exist(name string) (bool, error) {
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

func (database *Queue) Delete(name string) error {
	filter := bson.D{{Key: "name", Value: name}}

	result := database.db.Collection("queues").FindOneAndDelete(context.Background(), filter)
	if result.Err() != nil {
		return fmt.Errorf("error deleting queue: %w", result.Err())
	}

	return nil
}

func (database *Queue) SaveEmail(email model.Email) error {
	email.ID = uuid.New()

	_, err := database.db.Collection("emails_sent").InsertOne(context.Background(), email)
	if err != nil {
		return fmt.Errorf("error adding queue on database: %w", err)
	}

	return nil
}

func NewQueueDatabase(connection *mongo.Client) *Queue {
	return &Queue{connection.Database("email")}
}

type Template struct {
	db *mongo.Database
}

func NewTemplateDatabase(connection *mongo.Client) *Template {
	return &Template{connection.Database("templates")}
}
