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

func (database *mongo[T]) create(data T) error {
	_, err := database.collection.InsertOne(context.Background(), data)
	if err != nil {
		return fmt.Errorf("error creating data in database: %w", err)
	}

	return nil
}

func (database *mongo[T]) exist(filter bson.D) (bool, error) {
	data := new(T)

	err := database.collection.FindOne(context.Background(), filter).Decode(data)
	if err != nil {
		if errors.Is(err, mongodb.ErrNoDocuments) {
			return false, nil
		}

		return false, fmt.Errorf("error checking if data exist on database: %w", err)
	}

	return true, nil
}

func (database *mongo[T]) get(filter bson.D) (*T, error) {
	data := new(T)

	err := database.collection.FindOne(context.Background(), filter).Decode(data)
	if err != nil {
		return nil, fmt.Errorf("error getting data from database: %w", err)
	}

	return data, nil
}

func (database *mongo[T]) getMultiples(filter bson.D) ([]T, error) {
	data := []T{}

	cursor, err := database.collection.Find(context.Background(), filter)
	if err != nil {
		return nil, fmt.Errorf("error getting data from database: %w", err)
	}

	err = cursor.All(context.Background(), &data)
	if err != nil {
		return nil, fmt.Errorf("error parsing data: %w", err)
	}

	return data, nil
}

func (database *mongo[T]) getAll() ([]T, error) {
	return database.getMultiples(bson.D{})
}

func (database *mongo[T]) update(dataID uuid.UUID, update bson.D) error {
	_, err := database.collection.UpdateByID(context.Background(), dataID, update)
	if err != nil {
		return fmt.Errorf("error getting data from database: %w", err)
	}

	return nil
}

func createMongoDatabase[T any](client *mongodb.Client, database, collection string) *mongo[T] {
	return &mongo[T]{client.Database(database).Collection(collection)}
}

type User struct {
	users    *mongo[model.User]
	sessions *mongo[model.UserSession]
	roles    *mongo[model.Role]
}

func (database *User) Create(user model.User) error {
	return database.users.create(user)
}

func (database *User) ExistByID(userID uuid.UUID) (bool, error) {
	filter := bson.D{
		{Key: "_id", Value: userID},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	return database.users.exist(filter)
}

func (database *User) ExistByNameOrEmail(name, email string) (bool, error) {
	filter := bson.D{
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "name", Value: name}},
			bson.D{{Key: "email", Value: email}},
		}},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	return database.users.exist(filter)
}

func (database *User) GetByID(userID uuid.UUID) (*model.User, error) {
	filter := bson.D{
		{Key: "_id", Value: userID},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	return database.users.get(filter)
}

func (database *User) GetByNameOrEmail(name, email string) (*model.User, error) {
	filter := bson.D{
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "name", Value: name}},
			bson.D{{Key: "email", Value: email}},
		}},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	return database.users.get(filter)
}

func (database *User) GetAll() ([]model.User, error) {
	return database.users.getAll()
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

	return database.users.update(user.ID, update)
}

func (database *User) SaveSession(session model.UserSession) error {
	return database.sessions.create(session)
}

func (database *User) ExistSession(sessionID uuid.UUID) (bool, error) {
	filter := bson.D{
		{Key: "_id", Value: sessionID},
		{Key: "deleted_at", Value: bson.D{{Key: "$gt", Value: time.Now()}}},
	}

	return database.sessions.exist(filter)
}

func (database *User) GetSession(sessionID uuid.UUID) (*model.UserSession, error) {
	filter := bson.D{
		{Key: "_id", Value: sessionID},
		{Key: "deleted_at", Value: bson.D{{Key: "$gt", Value: time.Now()}}},
	}

	session := new(model.UserSession)

	err := database.sessions.collection.FindOne(context.Background(), filter).Decode(session)
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

	return database.sessions.update(session.ID, update)
}

func (database *User) CreateRole(role model.Role) error {
	return database.roles.create(role)
}

func (database *User) GetRole(name string) (*model.Role, error) {
	filter := bson.D{
		{Key: "name", Value: name},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	return database.roles.get(filter)
}

func (database *User) GetAllRoles() ([]model.Role, error) {
	return database.roles.getAll()
}

func newUserDatabase(client *mongodb.Client) *User {
	return &User{
		createMongoDatabase[model.User](client, "users", "users"),
		createMongoDatabase[model.UserSession](client, "users", "sessions"),
		createMongoDatabase[model.Role](client, "users", "roles"),
	}
}

type Queue struct {
	queues *mongo[model.Queue]
	emails *mongo[model.Email]
}

func (database *Queue) Create(queue model.Queue) error {
	return database.queues.create(queue)
}

func (database *Queue) Get(name string) (*model.Queue, error) {
	filter := bson.D{
		{Key: "name", Value: name},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	return database.queues.get(filter)
}

func (database *Queue) GetAll() ([]model.Queue, error) {
	return database.queues.getAll()
}

func (database *Queue) Exist(name string) (bool, error) {
	filter := bson.D{
		{Key: "name", Value: name},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	return database.queues.exist(filter)
}

func (database *Queue) Update(queue model.Queue) error {
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "deleted_at", Value: queue.DeletedAt},
			{Key: "deleted_by", Value: queue.DeletedBy},
		}},
	}

	return database.queues.update(queue.ID, update)
}

func (database *Queue) SaveEmail(email model.Email) error {
	return database.emails.create(email)
}

func newQueueDatabase(client *mongodb.Client) *Queue {
	return &Queue{
		createMongoDatabase[model.Queue](client, "email", "queues"),
		createMongoDatabase[model.Email](client, "email", "sent"),
	}
}

type Template struct {
	templates *mongo[model.Template]
}

func (database *Template) Create(template model.Template) error {
	return database.templates.create(template)
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

	return database.templates.update(template.ID, update)
}

func (database *Template) Exist(name string) (bool, error) {
	filter := bson.D{
		{Key: "name", Value: name},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	return database.templates.exist(filter)
}

func (database *Template) Get(name string) (*model.Template, error) {
	filter := bson.D{
		{Key: "name", Value: name},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	return database.templates.get(filter)
}

func (database *Template) GetByUser(userID uuid.UUID) ([]model.Template, error) {
	filter := bson.D{
		{Key: "created_by", Value: userID},
		{Key: "deleted_at", Value: bson.D{{Key: "$eq", Value: time.Time{}}}},
	}

	return database.templates.getMultiples(filter)
}

func (database *Template) GetAll() ([]model.Template, error) {
	return database.templates.getAll()
}

func newTemplateDatabase(client *mongodb.Client) *Template {
	return &Template{
		createMongoDatabase[model.Template](client, "template", "templates"),
	}
}

type Attachment struct {
	attachment *mongo[model.Attachment]
}

func (database *Attachment) Create(attachment model.Attachment) error {
	return database.attachment.create(attachment)
}

func (database *Attachment) Exist(id uuid.UUID) (bool, error) {
	filter := bson.D{{Key: "_id", Value: id}}

	return database.attachment.exist(filter)
}

func (database *Attachment) Get(id uuid.UUID) (*model.Attachment, error) {
	filter := bson.D{{Key: "_id", Value: id}}

	return database.attachment.get(filter)
}

func (database *Attachment) GetAttachments(userID uuid.UUID) ([]model.Attachment, error) {
	filter := bson.D{{Key: "user_id", Value: userID}}

	return database.attachment.getMultiples(filter)
}

func newAttachmenteDatabase(client *mongodb.Client) *Attachment {
	return &Attachment{
		createMongoDatabase[model.Attachment](client, "attachment", "attachments"),
	}
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
	*Attachment
}

func NewDatabases(client *mongodb.Client) *Databases {
	return &Databases{
		User:       newUserDatabase(client),
		Queue:      newQueueDatabase(client),
		Template:   newTemplateDatabase(client),
		Attachment: newAttachmenteDatabase(client),
	}
}
