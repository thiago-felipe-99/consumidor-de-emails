//nolint:lll
package model

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ID uuid.UUID

func (id ID) String() string {
	return uuid.UUID(id).String()
}

func (id ID) MarshalKey() (string, error) {
	return hex.EncodeToString(id[:]), nil
}

func (id *ID) UnmarshalKey(key string) error {
	idUUID, err := uuid.Parse(key)
	if err != nil {
		return fmt.Errorf("error parsing ID: %w", err)
	}

	*id = ID(idUUID)

	return nil
}

func NewID() ID {
	return ID(uuid.New())
}

func ParseID(id string) (ID, error) {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return ID(uuid.UUID{}), fmt.Errorf("error parsing ID: %w", err)
	}

	return ID(idUUID), nil
}

type UserPartial struct {
	Name     string `config:"name"     json:"name"     validate:"required"`
	Email    string `config:"email"    json:"email"    validate:"required,email"`
	Password string `config:"password" json:"password" validate:"required"`
}

type User struct {
	ID          ID        `json:"id"                    bson:"_id"`
	Name        string    `json:"name"                  bson:"name"`
	Email       string    `json:"email"                 bson:"email"`
	Password    string    `json:"password,omitempty"    bson:"password"`
	CreatedAt   time.Time `json:"createdAt"             bson:"created_at"`
	CreatedBy   ID        `json:"createdBy"             bson:"created_by"`
	DeletedAt   time.Time `json:"deletedAt,omitempty"   bson:"deleted_at"`
	DeletedBy   ID        `json:"deletedBy,omitempty"   bson:"deleted_by"`
	IsAdmin     bool      `json:"isAdmin,omitempty"     bson:"is_admin"`
	IsProtected bool      `json:"isProtected,omitempty" bson:"is_protected"`
}

type UserSessionPartial struct {
	Name     string `json:"name"     validate:"required_without=Email,excluded_with=Email"`
	Email    string `json:"email"    validate:"required_without=Name,excluded_with=Name,omitempty,email"`
	Password string `json:"password" validate:"required"`
}

type UserSession struct {
	ID        ID        `json:"id"                  bson:"_id"`
	UserID    ID        `json:"userId"              bson:"user_id"`
	CreateaAt time.Time `json:"createdAt"           bson:"created_at"`
	Expires   time.Time `json:"expires"             bson:"expires"`
	DeletedAt time.Time `json:"deletedAt,omitempty" bson:"deleted_at"`
}

type QueuePartial struct {
	Name       string `json:"name"       validate:"required"`
	MaxRetries int64  `json:"maxRetries" validate:"omitempty,min=1"`
}

type Queue struct {
	ID         ID        `json:"id"                  bson:"_id"`
	Name       string    `json:"name"                bson:"name"`
	DLX        string    `json:"dlx"                 bson:"dlx"`
	MaxRetries int64     `json:"maxRetries"          bson:"max_retries"`
	CreatedAt  time.Time `json:"createdAt"           bson:"created_at"`
	CreatedBy  ID        `json:"createdBy"           bson:"created_by"`
	DeletedAt  time.Time `json:"deletedAt,omitempty" bson:"deleted_at"`
	DeletedBy  ID        `json:"deletedBy,omitempty" bson:"deleted_by"`
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
	ID             ID            `json:"id"                       bson:"_id"`
	UserID         ID            `json:"userId"                   bson:"user_id"`
	EmailLists     []string      `json:"emailLists,omitempty"     bson:"email_lists"`
	Receivers      []Receiver    `json:"receivers,omitempty"      bson:"receivers"`
	BlindReceivers []Receiver    `json:"blindReceivers,omitempty" bson:"blind_receivers"`
	Subject        string        `json:"subject"                  bson:"subject"`
	Message        string        `json:"message,omitempty"        bson:"message"`
	Template       *TemplateData `json:"template,omitempty"       bson:"template"`
	Attachments    []string      `json:"attachments,omitempty"    bson:"attachments"`
	SentAt         time.Time     `json:"sentAt"                   bson:"sent_at"`
}

type EmailListPartial struct {
	Emails      []string `json:"emails"      validate:"required,min=1,dive,email"`
	Name        string   `json:"name"        validate:"required"`
	EmailAlias  string   `json:"emailAlias"  validate:"required,email"`
	Description string   `json:"description" validate:"required"`
}

type EmailList struct {
	ID          ID            `json:"id"                  bson:"_id"`
	Emails      map[ID]string `json:"emails"              bson:"emails"`
	Name        string        `json:"name"                bson:"name"`
	EmailAlias  string        `json:"emailAlias"          bson:"email_alias"`
	Description string        `json:"description"         bson:"description"`
	CreatedAt   time.Time     `json:"createdAt"           bson:"created_at"`
	CreatedBy   ID            `json:"createdBy"           bson:"created_by"`
	DeletedAt   time.Time     `json:"deletedAt,omitempty" bson:"deleted_at"`
	DeletedBy   ID            `json:"deletedBy,omitempty" bson:"deleted_by"`
}

type TemplatePartial struct {
	Name     string `json:"name"     validate:"required"`
	Template string `json:"template" validate:"required"`
}

type Template struct {
	ID        ID        `json:"id"                  bson:"_id"`
	Name      string    `json:"name"                bson:"name"`
	Template  string    `json:"template"            bson:"template"`
	Fields    []string  `json:"fields,omitempty"    bson:"fields"`
	CreatedAt time.Time `json:"createdAt"           bson:"created_at"`
	CreatedBy ID        `json:"createdBy"           bson:"created_by"`
	DeletedAt time.Time `json:"deletedAt,omitempty" bson:"deleted_at"`
	DeletedBy ID        `json:"deletedBy,omitempty" bson:"deleted_by"`
}

type AttachmentPartial struct {
	Name        string `json:"name"        validate:"required"`
	ContentType string `json:"contentType" validate:"required"`
	Size        int    `json:"size"        validate:"required,min=1"`
}

type Attachment struct {
	ID              ID        `json:"id"              bson:"_id"`
	UserID          ID        `json:"userId"          bson:"user_id"`
	CreatedAt       time.Time `json:"createdAt"       bson:"created_at"`
	Name            string    `json:"name"            bson:"name"`
	ContentType     string    `json:"contentType"     bson:"content_type"`
	Size            int       `json:"size"            bson:"size"`
	MinioName       string    `json:"minioName"       bson:"minio_name"`
	ConfirmedUpload bool      `json:"confirmedUpload" bson:"confirmed_upload"`
}

type AttachmentURL struct {
	ID        ID                `json:"id"`
	MinioName string            `json:"minioName"`
	URL       string            `json:"url"`
	FormData  map[string]string `json:"formData,omitempty"`
}
