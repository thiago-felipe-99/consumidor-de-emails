package model

import (
	"time"

	"github.com/google/uuid"
)

type UserPartial struct {
	Name     string `json:"name"     bson:"name"     validate:"required_without=Email,excluded_with=Email"`
	Email    string `json:"email"    bson:"email"    validate:"required_without=Name,excluded_with=Name,omitempty,email"`
	Password string `json:"password" bson:"password" validate:"required"`
}

type User struct {
	ID         uuid.UUID `json:"-"                   bson:"_id"        validate:"-"`
	Name       string    `json:"name"                bson:"name"       validate:"required"`
	Email      string    `json:"email"               bson:"email"      validate:"required,email"`
	Password   string    `json:"password,omitempty"  bson:"password"   validate:"required"`
	CreateadAt time.Time `json:"createdAt"           bson:"created_at" validate:"-"`
	DeletedAt  time.Time `json:"deletedAt,omitempty" bson:"deleted_at" validate:"-"`
	Roles      []string  `json:"roles,omitempty"     bson:"roles"      validate:"-"`
	IsAdmin    bool      `json:"isAdmin,omitempty"   bson:"is_admin"   validate:"-"`
}

type UserSession struct {
	ID         uuid.UUID `json:"-"         bson:"_id"        validate:"-"`
	UserID     uuid.UUID `json:"userID"    bson:"user_id"    validate:"-"` //nolint:tagliatelle
	CreateadAt time.Time `json:"createdAt" bson:"created_at" validate:"-"`
	Expires    time.Time `json:"expires"   bson:"expires"    validate:"-"`
	DeletedAt  time.Time `json:"deletedAt" bson:"deleted_at" validate:"-"`
}

type QueuePartial struct {
	Name       string `json:"name"       validate:"required"`
	MaxRetries int64  `json:"maxRetries"`
}

type Queue struct {
	ID            uuid.UUID `json:"-"          bson:"_id"`
	Name          string    `json:"name"       bson:"name"`
	DLX           string    `json:"dlx"        bson:"dlx"`
	MaxRetries    int64     `json:"maxRetries" bson:"max_retries"`
	CreatedAt     time.Time `json:"createdAt"  bson:"created_at"`
	UserIDCreated uuid.UUID `json:"-"          bson:"user_id_created"`
	DeletedAt     time.Time `json:"deletedAt"  bson:"deleted_at"`
	UserIDDeleted uuid.UUID `json:"-"          bson:"user_id_deleted"`
}

type Receiver struct {
	Name  string `json:"name"  bson:"name"  validate:"required"`
	Email string `json:"email" bson:"email" validate:"required,email"`
}

type TemplateData struct {
	Name string            `json:"name" bson:"name" validate:"required"`
	Data map[string]string `json:"data" bson:"data"`
}

//nolint:lll
type Email struct {
	ID             uuid.UUID     `json:"-"              bson:"_id"             validate:"-"`
	Receivers      []Receiver    `json:"receivers"      bson:"receivers"       validate:"required_without=BlindReceivers,omitempty,min=1"`
	BlindReceivers []Receiver    `json:"blindReceivers" bson:"blind_receivers" validate:"required_without=Receivers,omitempty,min=1"`
	Subject        string        `json:"subject"        bson:"subject"         validate:"required"`
	Message        string        `json:"message"        bson:"message"         validate:"required_without=Template,excluded_with=Template"`
	Template       *TemplateData `json:"template"       bson:"template"        validate:"required_without=Message,excluded_with=Message"`
	Attachments    []string      `json:"attachments"    bson:"attachments"`
}

//nolint:lll
type EmailUser struct {
	ID             uuid.UUID     `json:"-"                   bson:"_id"             validate:"-"`
	UserID         uuid.UUID     `json:"-"                   bson:"user_id"         validate:"-"`
	EmailList      string        `json:"emailList,omitempty" bson:"email_list"      validate:"required_without_all=BlindReceivers Receivers"`
	Receivers      []Receiver    `json:"receivers"           bson:"receivers"       validate:"required_without_all=BlindReceivers EmailList,omitempty,min=1"`
	BlindReceivers []Receiver    `json:"blindReceivers"      bson:"blind_receivers" validate:"required_without_all=Receivers EmailList,omitempty,min=1"`
	Subject        string        `json:"subject"             bson:"subject"         validate:"required"`
	Message        string        `json:"message"             bson:"message"         validate:"required_without=Template,excluded_with=Template"`
	Template       *TemplateData `json:"template"            bson:"template"        validate:"required_without=Message,excluded_with=Message"`
	Attachments    []string      `json:"attachments"         bson:"attachments"`
}

type EmailList struct {
	ID            uuid.UUID `json:"-"               bson:"_id"             validate:"-"`
	Emails        []string  `json:"emails"          bson:"emails"          validate:"required,dive,email,required"`
	Name          string    `json:"name"            bson:"name"            validate:"required"`
	Description   string    `json:"description"     bson:"description"     validate:"required"`
	Roles         []string  `json:"roles,omitempty" bson:"roles"           validate:"-"`
	CreatedAt     time.Time `json:"createdAt"       bson:"created_at"      validate:"-"`
	UserIDCreated uuid.UUID `json:"-"               bson:"user_id_created" validate:"-"`
	DeletedAt     time.Time `json:"deletedAt"       bson:"deleted_at"      validate:"-"`
	UserIDDeleted uuid.UUID `json:"-"               bson:"user_id_deleted" validate:"-"`
}

type TemplatePartial struct {
	Name     string   `json:"name"            validate:"required"`
	Template string   `json:"template"        validate:"required"`
	Roles    []string `json:"roles,omitempty" validate:"-"`
}

type Template struct {
	ID            uuid.UUID `json:"-"               bson:"_id"`
	UserID        uuid.UUID `json:"-"               bson:"user_id"`
	Name          string    `json:"name"            bson:"name"`
	Template      string    `json:"template"        bson:"template"`
	Fields        []string  `json:"fields"          bson:"fields"`
	Roles         []string  `json:"roles,omitempty" bson:"roles"`
	CreatedAt     time.Time `json:"createdAt"       bson:"created_at"`
	UserIDCreated uuid.UUID `json:"-"               bson:"user_id_created"`
	DeletedAt     time.Time `json:"deletedAt"       bson:"deleted_at"`
	UserIDDeleted uuid.UUID `json:"-"               bson:"user_id_deleted"`
}

type Attachment struct {
	ID     uuid.UUID `json:"-"    bson:"_id"     validate:"-"`
	UserID uuid.UUID `json:"-"    bson:"user_id" validate:"-"`
	Name   string    `json:"name" bson:"name"    validate:"required"`
}

type AttachmentLink struct {
	Name string `json:"name"`
	Link string `json:"link"`
}
