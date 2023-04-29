package model

import (
	"time"

	"github.com/google/uuid"
)

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

type QueuePartial struct {
	Name       string `json:"name"       validate:"required"`
	MaxRetries int64  `json:"maxRetries"`
}

type Queue struct {
	ID         uuid.UUID `json:"-"          bson:"_id"`
	Name       string    `json:"name"       bson:"name"`
	DLX        string    `json:"dlx"        bson:"dlx"`
	MaxRetries int64     `json:"maxRetries" bson:"max_retries"`
	CreatedAt  time.Time `json:"createdAt"  bson:"created_at"`
}

type TemplatePartial struct {
	Name     string `json:"name"     validate:"required"`
	Template string `json:"template" validate:"required"`
}

type Template struct {
	ID       uuid.UUID `json:"-"        bson:"_id"`
	Name     string    `json:"name"     bson:"name"`
	Template string    `json:"template" bson:"template"`
	Fields   []string  `json:"fields"   bson:"fields"`
}
