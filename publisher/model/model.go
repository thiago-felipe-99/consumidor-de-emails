//nolint:lll
package model

import (
	"time"

	"github.com/google/uuid"
)

type UserPartial struct {
	Name     string `config:"name"     json:"name"     validate:"required"`
	Email    string `config:"email"    json:"email"    validate:"required,email"`
	Password string `config:"password" json:"password" validate:"required"`
}

type User struct {
	ID         uuid.UUID `json:"id"                  bson:"_id"`
	Name       string    `json:"name"                bson:"name"`
	Email      string    `json:"email"               bson:"email"`
	Password   string    `json:"password,omitempty"  bson:"password"`
	CreateadAt time.Time `json:"createdAt"           bson:"created_at"`
	DeletedAt  time.Time `json:"deletedAt,omitempty" bson:"deleted_at"`
	Roles      []string  `json:"roles,omitempty"     bson:"roles"`
	IsAdmin    bool      `json:"isAdmin,omitempty"   bson:"is_admin"`
	Protected  bool      `json:"protected,omitempty" bson:"protected"`
}

type UserRoles struct {
	Roles []string `json:"roles" validate:"required,min=1"`
}

type Role struct {
	ID            uuid.UUID `json:"id"                      bson:"_id"`
	Name          string    `json:"name"                    bson:"name"`
	CreatedAt     time.Time `json:"createdAt"               bson:"created_at"`
	UserIDCreated uuid.UUID `json:"userIdCreated"           bson:"user_id_created"`
	DeletedAt     time.Time `json:"deletedAt,omitempty"     bson:"deleted_at"`
	UserIDDeleted uuid.UUID `json:"userIdDeleted,omitempty" bson:"user_id_deleted"`
}

type UserSessionPartial struct {
	Name     string `json:"name"     validate:"required_without=Email,excluded_with=Email"`
	Email    string `json:"email"    validate:"required_without=Name,excluded_with=Name,omitempty,email"`
	Password string `json:"password" validate:"required"`
}

type UserSession struct {
	ID         uuid.UUID `json:"id"                  bson:"_id"`
	UserID     uuid.UUID `json:"userId"              bson:"user_id"`
	CreateadAt time.Time `json:"createdAt"           bson:"created_at"`
	Expires    time.Time `json:"expires"             bson:"expires"`
	DeletedAt  time.Time `json:"deletedAt,omitempty" bson:"deleted_at"`
}

type QueuePartial struct {
	Name       string `json:"name"       validate:"required"`
	MaxRetries int64  `json:"maxRetries" validate:"omitempty,min=1"`
}

type Queue struct {
	ID            uuid.UUID `json:"id"                      bson:"_id"`
	Name          string    `json:"name"                    bson:"name"`
	DLX           string    `json:"dlx"                     bson:"dlx"`
	MaxRetries    int64     `json:"maxRetries"              bson:"max_retries"`
	CreatedAt     time.Time `json:"createdAt"               bson:"created_at"`
	UserIDCreated uuid.UUID `json:"userIdCreated"           bson:"user_id_created"`
	DeletedAt     time.Time `json:"deletedAt,omitempty"     bson:"deleted_at"`
	UserIDDeleted uuid.UUID `json:"userIdDeleted,omitempty" bson:"user_id_deleted"`
}

type Receiver struct {
	Name  string `json:"name"  bson:"name"  validate:"required"`
	Email string `json:"email" bson:"email" validate:"required,email"`
}

type TemplateData struct {
	Name string            `json:"name" bson:"name" validate:"required"`
	Data map[string]string `json:"data" bson:"data" validate:"-"`
}

type EmailPartial struct {
	EmailLists     []string      `json:"emailLists,omitempty"     validate:"required_without_all=BlindReceivers Receivers,omitempty,min=1"`
	Receivers      []Receiver    `json:"receivers,omitempty"      validate:"required_without_all=BlindReceivers EmailList,omitempty,min=1"`
	BlindReceivers []Receiver    `json:"blindReceivers,omitempty" validate:"required_without_all=Receivers EmailList,omitempty,min=1"`
	Subject        string        `json:"subject"                  validate:"required"`
	Message        string        `json:"message,omitempty"        validate:"required_without=Template,excluded_with=Template"`
	Template       *TemplateData `json:"template,omitempty"       validate:"required_without=Message,excluded_with=Message"`
	Attachments    []string      `json:"attachments,omitempty"    validate:"-"`
}

type Email struct {
	ID             uuid.UUID     `json:"id"                       bson:"_id"`
	UserID         uuid.UUID     `json:"userId"                   bson:"user_id"`
	EmailLists     []string      `json:"emailLists,omitempty"     bson:"email_lists"`
	Receivers      []Receiver    `json:"receivers,omitempty"      bson:"receivers"`
	BlindReceivers []Receiver    `json:"blindReceivers,omitempty" bson:"blind_receivers"`
	Subject        string        `json:"subject"                  bson:"subject"`
	Message        string        `json:"message,omitempty"        bson:"message"`
	Template       *TemplateData `json:"template,omitempty"       bson:"template"`
	Attachments    []string      `json:"attachments,omitempty"    bson:"attachments"`
}

type EmailListPartial struct {
	Emails      []string `json:"emails"          validate:"required,min=1,dive,email"`
	Name        string   `json:"name"            validate:"required"`
	Description string   `json:"description"     validate:"required"`
	Roles       []string `json:"roles,omitempty" validate:"-"`
}

type EmailList struct {
	ID            uuid.UUID `json:"id"                      bson:"_id"`
	Emails        []string  `json:"emails"                  bson:"emails"`
	Name          string    `json:"name"                    bson:"name"`
	Description   string    `json:"description"             bson:"description"`
	Roles         []string  `json:"roles,omitempty"         bson:"roles"`
	CreatedAt     time.Time `json:"createdAt"               bson:"created_at"`
	UserIDCreated uuid.UUID `json:"userIdCreated"           bson:"user_id_created"`
	DeletedAt     time.Time `json:"deletedAt,omitempty"     bson:"deleted_at"`
	UserIDDeleted uuid.UUID `json:"userIdDeleted,omitempty" bson:"user_id_deleted"`
}

type TemplatePartial struct {
	Name     string   `json:"name"            validate:"required"`
	Template string   `json:"template"        validate:"required"`
	Roles    []string `json:"roles,omitempty" validate:"-"`
}

type Template struct {
	ID            uuid.UUID `json:"id"                      bson:"_id"`
	UserID        uuid.UUID `json:"useId"                   bson:"user_id"`
	Name          string    `json:"name"                    bson:"name"`
	Template      string    `json:"template"                bson:"template"`
	Fields        []string  `json:"fields,omitempty"        bson:"fields"`
	Roles         []string  `json:"roles,omitempty"         bson:"roles"`
	CreatedAt     time.Time `json:"createdAt"               bson:"created_at"`
	UserIDCreated uuid.UUID `json:"userIdCreated"           bson:"user_id_created"`
	DeletedAt     time.Time `json:"deletedAt,omitempty"     bson:"deleted_at"`
	UserIDDeleted uuid.UUID `json:"userIdDeleted,omitempty" bson:"user_id_deleted"`
}

type AttachmentPartial struct {
	Name        string `json:"name"        validate:"required"`
	ContentType string `json:"contentType" validate:"required"`
}

type Attachment struct {
	ID          uuid.UUID `json:"id"          bson:"_id"`
	UserID      uuid.UUID `json:"userId"      bson:"user_id"`
	Name        string    `json:"name"        bson:"name"`
	ContentType string    `json:"contentType" bson:"content_type"`
}

type AttachmentLink struct {
	Name string `json:"name"`
	Link string `json:"link"`
}
