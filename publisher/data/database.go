package data

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/thiago-felipe-99/mail/publisher/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type User struct {
	db *mongo.Database
}

func (database *User) Create(user model.User) error {
	_, err := database.db.Collection("users").InsertOne(context.Background(), user)
	if err != nil {
		return fmt.Errorf("error creating user in database: %w", err)
	}

	return nil
}

func (database *User) GetByID(userID uuid.UUID) (*model.User, error) {
	filter := bson.D{
		{Key: "_id", Value: userID},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	user := &model.User{}

	err := database.db.Collection("users").FindOne(context.Background(), filter).Decode(user)
	if err != nil {
		return nil, fmt.Errorf("error getting user from database: %w", err)
	}

	return user, nil
}

func (database *User) GetByNameOrEmail(name, email string) (*model.User, error) {
	filter := bson.D{
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "name", Value: name}},
			bson.D{{Key: "email", Value: email}},
		}},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	user := &model.User{}

	err := database.db.Collection("users").FindOne(context.Background(), filter).Decode(user)
	if err != nil {
		return nil, fmt.Errorf("error getting user from database: %w", err)
	}

	return user, nil
}

func (database *User) GetAll() ([]model.User, error) {
	users := []model.User{}

	cursor, err := database.db.Collection("users").Find(context.Background(), bson.D{})
	if err != nil {
		return nil, fmt.Errorf("error getting all users from database: %w", err)
	}

	err = cursor.All(context.Background(), &users)
	if err != nil {
		return nil, fmt.Errorf("error parsing users: %w", err)
	}

	return users, nil
}

func (database *User) Update(user model.User) error {
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "password", Value: user.Password},
			{Key: "deleted_at", Value: user.DeletedAt},
			{Key: "deleted_by", Value: user.DeletedBy},
			{Key: "is_admin", Value: user.IsAdmin},
			{Key: "protected", Value: user.IsProtected},
			{Key: "roles", Value: user.Roles},
		}},
	}

	_, err := database.db.Collection("users").UpdateByID(context.Background(), user.ID, update)
	if err != nil {
		return fmt.Errorf("error deleting user from database: %w", err)
	}

	return nil
}

func (database *User) SaveSession(session model.UserSession) error {
	_, err := database.db.Collection("sessions").InsertOne(context.Background(), session)
	if err != nil {
		return fmt.Errorf("error creating user session in database: %w", err)
	}

	return nil
}

func (database *User) ExistSession(sessionID uuid.UUID) (bool, error) {
	filter := bson.D{{Key: "_id", Value: sessionID}}

	count, err := database.db.Collection("sessions").CountDocuments(context.Background(), filter)
	if err != nil {
		return false, fmt.Errorf("error counting sessions in database: %w", err)
	}

	return count > 0, nil
}

func (database *User) GetSession(sessionID uuid.UUID) (*model.UserSession, error) {
	filter := bson.D{{Key: "_id", Value: sessionID}}

	session := &model.UserSession{}

	err := database.db.Collection("sessions").FindOne(context.Background(), filter).Decode(session)
	if err != nil {
		return nil, fmt.Errorf("error getting session from database: %w", err)
	}

	return session, nil
}

func (database *User) UpdateSession(session model.UserSession) error {
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "deleted_at", Value: session.DeletedAt},
		}},
	}

	_, err := database.db.Collection("sessions").
		UpdateByID(context.Background(), session.ID, update)
	if err != nil {
		return fmt.Errorf("error updating session in database: %w", err)
	}

	return nil
}

func (database *User) CreateRole(role model.Role) error {
	_, err := database.db.Collection("roles").InsertOne(context.Background(), role)
	if err != nil {
		return fmt.Errorf("error creating role in database: %w", err)
	}

	return nil
}

func (database *User) GetRole(name string) (*model.Role, error) {
	filter := bson.D{{Key: "name", Value: name}}

	role := &model.Role{}

	err := database.db.Collection("roles").FindOne(context.Background(), filter).Decode(role)
	if err != nil {
		return nil, fmt.Errorf("error getting role from database: %w", err)
	}

	return role, nil
}

func (database *User) GetAllRoles() ([]model.Role, error) {
	filter := bson.D{}

	role := []model.Role{}

	cursor, err := database.db.Collection("roles").Find(context.Background(), filter)
	if err != nil {
		return nil, fmt.Errorf("error getting role from database: %w", err)
	}

	err = cursor.All(context.Background(), &role)
	if err != nil {
		return nil, fmt.Errorf("error getting all roles: %w", err)
	}

	return role, nil
}

func NewUserDatabase(client *mongo.Client) *User {
	return &User{client.Database("user")}
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

func NewQueueDatabase(client *mongo.Client) *Queue {
	return &Queue{client.Database("email")}
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
			{Key: "createdAt", Value: template.CreatedAt},
			{Key: "createdBy", Value: template.CreatedBy},
			{Key: "deletedAt", Value: template.DeletedAt},
			{Key: "deletedBy", Value: template.DeletedBy},
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
	filter := bson.D{
		{Key: "name", Value: name},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	count, err := database.db.Collection("templates").CountDocuments(context.Background(), filter)
	if err != nil {
		return false, fmt.Errorf("error getting template from database: %w", err)
	}

	return count >= 1, nil
}

func (database *Template) Get(name string) (*model.Template, error) {
	filter := bson.D{
		{Key: "name", Value: name},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	template := &model.Template{}

	err := database.db.Collection("templates").
		FindOne(context.Background(), filter).
		Decode(template)
	if err != nil {
		return nil, fmt.Errorf("error getting template from database: %w", err)
	}

	return template, nil
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

func NewTemplateDatabase(client *mongo.Client) *Template {
	return &Template{client.Database("templates")}
}

func NewMongoClient(uri string) (*mongo.Client, error) {
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

type Databases struct {
	*User
	*Queue
	*Template
}

func NewDatabases(client *mongo.Client) *Databases {
	return &Databases{
		User:     NewUserDatabase(client),
		Queue:    NewQueueDatabase(client),
		Template: NewTemplateDatabase(client),
	}
}
