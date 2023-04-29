package main

import (
	"time"

	"github.com/google/uuid"
)

type receiver struct {
	Name  string `json:"name"  bson:"name"  validate:"required"`
	Email string `json:"email" bson:"email" validate:"required,email"`
}

type template struct {
	Name string            `json:"name" bson:"name" validate:"required"`
	Data map[string]string `json:"data" bson:"data"`
}

//nolint:lll
type emailModel struct {
	ID             uuid.UUID  `json:"-"              bson:"_id"             validate:"-"`
	Receivers      []receiver `json:"receivers"      bson:"receivers"       validate:"required_without=BlindReceivers,omitempty,min=1"`
	BlindReceivers []receiver `json:"blindReceivers" bson:"blind_receivers" validate:"required_without=Receivers,omitempty,min=1"`
	Subject        string     `json:"subject"        bson:"subject"         validate:"required"`
	Message        string     `json:"message"        bson:"message"         validate:"required_without=Template,excluded_with=Template"`
	Template       *template  `json:"template"       bson:"template"        validate:"required_without=Message,excluded_with=Message"`
	Attachments    []string   `json:"attachments"    bson:"attachments"`
}

type queueModel struct {
	ID         uuid.UUID `json:"-"          bson:"_id"`
	Name       string    `json:"name"       bson:"name"`
	DLX        string    `json:"dlx"        bson:"dlx"`
	MaxRetries int64     `json:"maxRetries" bson:"max_retries"`
	CreatedAt  time.Time `json:"createdAt"  bson:"created_at"`
}
