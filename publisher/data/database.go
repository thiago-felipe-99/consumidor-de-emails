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

func NewDatabase(uri string) (*mongo.Client, error) {
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

func (database *Queue) Create(queue model.Queue) error {
	_, err := database.db.Collection("queues").InsertOne(context.Background(), queue)
	if err != nil {
		return fmt.Errorf("error creating queue in database: %w", err)
	}

	return nil
}

func (database *Queue) GetAll() ([]model.Queue, error) {
	queues := []model.Queue{}

	cursor, err := database.db.Collection("queues").Find(context.Background(), bson.D{})
	if err != nil {
		return nil, fmt.Errorf("error getting queues from database: %w", err)
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
		return false, fmt.Errorf("error verify if queue exist in database: %w", err)
	}

	return count >= 1, nil
}

func (database *Queue) Delete(name string) error {
	filter := bson.D{{Key: "name", Value: name}}

	result := database.db.Collection("queues").FindOneAndDelete(context.Background(), filter)
	if result.Err() != nil {
		return fmt.Errorf("error deleting queue from database: %w", result.Err())
	}

	return nil
}

func (database *Queue) SaveEmail(email model.Email) error {
	_, err := database.db.Collection("emails_sent").InsertOne(context.Background(), email)
	if err != nil {
		return fmt.Errorf("error saving email in database: %w", err)
	}

	return nil
}

func NewQueueDatabase(connection *mongo.Client) *Queue {
	return &Queue{connection.Database("email")}
}

type Template struct {
	db *mongo.Database
}

func (database *Template) Create(template model.Template) error {
	_, err := database.db.Collection("templates").InsertOne(context.Background(), template)
	if err != nil {
		return fmt.Errorf("error creating template in database: %w", err)
	}

	return nil
}

func (database *Template) Update(template model.Template) error {
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "template", Value: template.Template},
			{Key: "fields", Value: template.Fields},
		}},
	}

	_, err := database.db.Collection("templates").
		UpdateByID(context.Background(), template.ID, update)
	if err != nil {
		return fmt.Errorf("error updating template in database: %w", err)
	}

	return nil
}

func (database *Template) Exist(name string) (bool, error) {
	filter := bson.D{{Key: "name", Value: name}}

	count, err := database.db.Collection("templates").CountDocuments(context.Background(), filter)
	if err != nil {
		return false, fmt.Errorf("error getting template from database: %w", err)
	}

	return count >= 1, nil
}

func (database *Template) Get(name string) (*model.Template, error) {
	filter := bson.D{{Key: "name", Value: name}}

	template := &model.Template{}

	err := database.db.Collection("templates").
		FindOne(context.Background(), filter).
		Decode(template)
	if err != nil {
		return nil, fmt.Errorf("error getting template from database: %w", err)
	}

	return template, nil
}

func (database *Template) GetID(name string) (uuid.UUID, error) {
	filter := bson.D{{Key: "name", Value: name}}

	template := &model.Template{}

	err := database.db.Collection("templates").
		FindOne(context.Background(), filter).
		Decode(template)
	if err != nil {
		return uuid.Nil, fmt.Errorf("error getting template from database: %w", err)
	}

	return template.ID, nil
}

func (database *Template) GetAll() ([]model.Template, error) {
	templates := []model.Template{}

	cursor, err := database.db.Collection("templates").Find(context.Background(), bson.D{})
	if err != nil {
		return nil, fmt.Errorf("error getting all templates from database: %w", err)
	}

	err = cursor.All(context.Background(), &templates)
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %w", err)
	}

	return templates, nil
}

func (database *Template) Delete(name string) error {
	filter := bson.D{{Key: "name", Value: name}}

	result := database.db.Collection("templates").FindOneAndDelete(context.Background(), filter)
	if result.Err() != nil {
		return fmt.Errorf("error deleting template from database: %w", result.Err())
	}

	return nil
}

func NewTemplateDatabase(connection *mongo.Client) *Template {
	return &Template{connection.Database("templates")}
}
