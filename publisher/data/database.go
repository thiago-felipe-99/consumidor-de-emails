package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/thiago-felipe-99/mail/publisher/model"
	"go.mongodb.org/mongo-driver/bson"
	mongodb "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongo[T any] struct {
	collection *mongodb.Collection
}

func (database *mongo[any]) create(data any) error {
	_, err := database.collection.InsertOne(context.Background(), data)
	if err != nil {
		return fmt.Errorf("error creating data in database: %w", err)
	}

	return nil
}

func (database *mongo[T]) existByID(id uuid.UUID) (bool, error) {
	filter := bson.D{
		{Key: "_id", Value: id},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	data := new(T)

	err := database.collection.FindOne(context.Background(), filter).Decode(data)
	if err != nil {
		if errors.Is(err, mongodb.ErrNoDocuments) {
			return false, nil
		}

		return false, fmt.Errorf("error getting data from database: %w", err)
	}

	return true, nil
}

func (database *mongo[T]) getByID(id uuid.UUID) (*T, error) {
	filter := bson.D{
		{Key: "_id", Value: id},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	data := new(T)

	err := database.collection.FindOne(context.Background(), filter).Decode(data)
	if err != nil {
		return nil, fmt.Errorf("error getting data from database: %w", err)
	}

	return data, nil
}

func (database *mongo[T]) getByFieldsOr(fields map[string]any) (*T, error) {
	fieldsBson := bson.A{}

	for key, value := range fields {
		fieldsBson = append(fieldsBson, bson.D{{Key: key, Value: value}})
	}

	filter := bson.D{
		{Key: "$or", Value: fieldsBson},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	data := new(T)

	err := database.collection.FindOne(context.Background(), filter).Decode(data)
	if err != nil {
		return nil, fmt.Errorf("error getting data from database: %w", err)
	}

	return data, nil
}

func (database *mongo[T]) update(dataID uuid.UUID, fields map[string]any) (*T, error) {
	fieldsBson := bson.D{}

	for key, value := range fields {
		fieldsBson = append(fieldsBson, bson.E{Key: key, Value: value})
	}

	update := bson.D{{Key: "$set", Value: fieldsBson}}

	data := new(T)

	_, err := database.collection.UpdateByID(context.Background(), dataID, update)
	if err != nil {
		return nil, fmt.Errorf("error getting data from database: %w", err)
	}

	return data, nil
}

func (database *mongo[T]) getAll() ([]T, error) {
	data := []T{}

	cursor, err := database.collection.Find(context.Background(), bson.D{})
	if err != nil {
		return nil, fmt.Errorf("error getting all data from database: %w", err)
	}

	err = cursor.All(context.Background(), &data)
	if err != nil {
		return nil, fmt.Errorf("error parsing data: %w", err)
	}

	return data, nil
}

type User struct {
	users    *mongo[model.User]
	sessions *mongo[model.UserSession]
	roles    *mongo[model.Role]
}

func (database *User) Create(user model.User) error {
	err := database.users.create(user)
	if err != nil {
		return fmt.Errorf("error creating user: %w", err)
	}

	return nil
}

func (database *User) GetByID(userID uuid.UUID) (*model.User, error) {
	user, err := database.users.getByID(userID)
	if err != nil {
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	return user, nil
}

func (database *User) GetByNameOrEmail(name, email string) (*model.User, error) {
	filter := map[string]any{
		"name":  name,
		"email": email,
	}

	user, err := database.users.getByFieldsOr(filter)
	if err != nil {
		return nil, fmt.Errorf("error getting user from database: %w", err)
	}

	return user, nil
}

func (database *User) GetAll() ([]model.User, error) {
	users, err := database.users.getAll()
	if err != nil {
		return nil, fmt.Errorf("error parsing users: %w", err)
	}

	return users, nil
}

func (database *User) Update(user model.User) error {
	update := map[string]any{
		"password":   user.Password,
		"deleted_at": user.DeletedAt,
		"deleted_by": user.DeletedBy,
		"is_admin":   user.IsAdmin,
		"protected":  user.IsProtected,
		"roles":      user.Roles,
	}

	_, err := database.users.update(user.ID, update)
	if err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}

	return nil
}

func (database *User) SaveSession(session model.UserSession) error {
	err := database.sessions.create(session)
	if err != nil {
		return fmt.Errorf("error creating user session in database: %w", err)
	}

	return nil
}

func (database *User) ExistSession(sessionID uuid.UUID) (bool, error) {
	exist, err := database.sessions.existByID(sessionID)
	if err != nil {
		return false, fmt.Errorf("error checking if exist session: %w", err)
	}

	return exist, nil
}

func (database *User) GetSession(sessionID uuid.UUID) (*model.UserSession, error) {
	session, err := database.sessions.getByID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("error getting session from database: %w", err)
	}

	return session, nil
}

func (database *User) UpdateSession(session model.UserSession) error {
	update := map[string]any{"deleted_at": session.DeletedAt}

	_, err := database.sessions.update(session.ID, update)
	if err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}

	return nil
}

func (database *User) CreateRole(role model.Role) error {
	err := database.roles.create(role)
	if err != nil {
		return fmt.Errorf("error creating role: %w", err)
	}

	return nil
}

func (database *User) GetRole(name string) (*model.Role, error) {
	filter := map[string]any{"name": name}

	role, err := database.roles.getByFieldsOr(filter)
	if err != nil {
		return nil, fmt.Errorf("error getting role from database: %w", err)
	}

	return role, nil
}

func (database *User) GetAllRoles() ([]model.Role, error) {
	roles, err := database.roles.getAll()
	if err != nil {
		return nil, fmt.Errorf("error getting all roles: %w", err)
	}

	return roles, nil
}

func NewUserDatabase(client *mongodb.Client) *User {
	return &User{
		&mongo[model.User]{client.Database("users").Collection("users")},
		&mongo[model.UserSession]{client.Database("users").Collection("sessions")},
		&mongo[model.Role]{client.Database("users").Collection("roles")},
	}
}

type Queue struct {
	db *mongodb.Database
}

func (database *Queue) Create(queue model.Queue) error {
	_, err := database.db.Collection("queues").InsertOne(context.Background(), queue)
	if err != nil {
		return fmt.Errorf("error creating queue in database: %w", err)
	}

	return nil
}

func (database *Queue) Get(name string) (*model.Queue, error) {
	filter := bson.D{
		{Key: "name", Value: name},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	queue := &model.Queue{}

	err := database.db.Collection("queues").FindOne(context.Background(), filter).Decode(queue)
	if err != nil {
		return nil, fmt.Errorf("error getting queue from database: %w", err)
	}

	return queue, nil
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
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	count, err := database.db.Collection("queues").CountDocuments(context.Background(), filter)
	if err != nil {
		return false, fmt.Errorf("error verify if queue exist in database: %w", err)
	}

	return count >= 1, nil
}

func (database *Queue) Update(queue model.Queue) error {
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "deleted_at", Value: queue.DeletedAt},
			{Key: "deleted_by", Value: queue.DeletedBy},
		}},
	}

	_, err := database.db.Collection("queues").UpdateByID(context.Background(), queue.ID, update)
	if err != nil {
		return fmt.Errorf("error updating queue from database: %w", err)
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

func NewQueueDatabase(client *mongodb.Client) *Queue {
	return &Queue{client.Database("email")}
}

type Template struct {
	db *mongodb.Database
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

func NewTemplateDatabase(client *mongodb.Client) *Template {
	return &Template{client.Database("templates")}
}

func NewMongoClient(uri string) (*mongodb.Client, error) {
	connection, err := mongodb.Connect(context.Background(), options.Client().ApplyURI(uri))
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

func NewDatabases(client *mongodb.Client) *Databases {
	return &Databases{
		User:     NewUserDatabase(client),
		Queue:    NewQueueDatabase(client),
		Template: NewTemplateDatabase(client),
	}
}
