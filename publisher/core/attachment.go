package core

import (
	"context"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/thiago-felipe-99/mail/publisher/data"
	"github.com/thiago-felipe-99/mail/publisher/model"
)

type Attachment struct {
	database  *data.Attachment
	minio     *minio.Client
	bucket    string
	validator *validator.Validate
	expires   time.Duration
}

func (core *Attachment) Create(
	partial model.AttachmentPartial,
	userID uuid.UUID,
) (*model.AttachmentLink, error) {
	err := validate(core.validator, partial)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	nowString := now.Format("2006-01-02_15-04-05.000")

	attachment := model.Attachment{
		ID:        uuid.New(),
		UserID:    userID,
		CreatedAt: now,
		Name:      partial.Name,
		MinioName: userID.String() + "/" + nowString + "-" + partial.Name,
	}

	link, err := core.minio.PresignedPutObject(
		context.Background(),
		core.bucket,
		attachment.MinioName,
		core.expires,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating minio link: %w", err)
	}

	attachmentLink := model.AttachmentLink{
		Name: attachment.MinioName,
		Link: link.String(),
	}

	err = core.database.Create(attachment)
	if err != nil {
		return nil, fmt.Errorf("error creating attachment in database: %w", err)
	}

	return &attachmentLink, nil
}

func (core *Attachment) Get(attachmentID uuid.UUID) (*model.AttachmentLink, error) {
	exist, err := core.database.Exist(attachmentID)
	if err != nil {
		return nil, fmt.Errorf("error checking if attachment exist in database: %w", err)
	}

	if !exist {
		return nil, ErrAttachmentDoesNotExist
	}

	attachment, err := core.database.Get(attachmentID)
	if err != nil {
		return nil, fmt.Errorf("error getting attachment from database: %w", err)
	}

	link, err := core.minio.PresignedGetObject(
		context.Background(),
		core.bucket,
		attachment.MinioName,
		core.expires,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating minio link: %w", err)
	}

	attachmentLink := model.AttachmentLink{
		Name: attachment.MinioName,
		Link: link.String(),
	}

	return &attachmentLink, nil
}

func (core *Attachment) GetAttachments(userID uuid.UUID) ([]model.Attachment, error) {
	attachments, err := core.database.GetAttachments(userID)
	if err != nil {
		return nil, fmt.Errorf("error getting attachments from database: %w", err)
	}

	return attachments, nil
}

func newAttachment(
	minio *minio.Client,
	bucket string,
	database *data.Attachment,
	validate *validator.Validate,
) *Attachment {
	return &Attachment{
		database:  database,
		minio:     minio,
		bucket:    bucket,
		validator: validate,
		expires:   time.Minute * 30, //nolint:gomnd
	}
}
