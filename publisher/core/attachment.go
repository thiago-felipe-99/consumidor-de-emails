package core

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/thiago-felipe-99/mail/publisher/data"
	"github.com/thiago-felipe-99/mail/publisher/model"
)

type Attachment struct {
	database     *data.Attachment
	minio        *minio.Client
	bucket       string
	validator    *validator.Validate
	expires      time.Duration
	maxEntrySize int
}

func (core *Attachment) createUploadURL(path string, size int) (*model.AttachmentURL, error) {
	policy := minio.NewPostPolicy()

	err := policy.SetBucket(core.bucket)
	if err != nil {
		return nil, fmt.Errorf("error setting POST policy key 'Bucket': %w", err)
	}

	err = policy.SetKey(path)
	if err != nil {
		return nil, fmt.Errorf("error setting POST policy key 'Key': %w", err)
	}

	err = policy.SetExpires(time.Now().UTC().Add(core.expires))
	if err != nil {
		return nil, fmt.Errorf("error setting POST policy key 'Expires': %w", err)
	}

	err = policy.SetContentLengthRange(1, int64(size))
	if err != nil {
		return nil, fmt.Errorf("error setting POST policy key 'ContentLengthRange': %w", err)
	}

	url, formData, err := core.minio.PresignedPostPolicy(context.Background(), policy)
	if err != nil {
		return nil, fmt.Errorf("error creating minio link: %w", err)
	}

	attachmentURL := model.AttachmentURL{
		URL:      url.String(),
		FormData: formData,
	}

	return &attachmentURL, nil
}

func (core *Attachment) Create(
	partial model.AttachmentPartial,
	userID uuid.UUID,
) (*model.AttachmentURL, error) {
	err := validate(core.validator, partial)
	if err != nil {
		return nil, err
	}

	if partial.Size > core.maxEntrySize {
		return nil, ErrMaxSizeAttachment
	}

	now := time.Now().UTC()
	nowString := now.Format("2006-01-02_15-04-05.000")

	attachment := model.Attachment{
		ID:              uuid.New(),
		UserID:          userID,
		CreatedAt:       now,
		Name:            partial.Name,
		MinioName:       userID.String() + "/" + nowString + "-" + partial.Name,
		ContentType:     partial.ContentType,
		Size:            partial.Size,
		ConfirmedUpload: false,
	}

	err = core.database.Create(attachment)
	if err != nil {
		return nil, fmt.Errorf("error creating attachment in database: %w", err)
	}

	return core.createUploadURL(attachment.MinioName, attachment.Size)
}

func (core *Attachment) get(attachmentID uuid.UUID, userID uuid.UUID) (*model.Attachment, error) {
	exist, err := core.database.Exist(attachmentID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking if attachment exist in database: %w", err)
	}

	if !exist {
		return nil, ErrAttachmentDoesNotExist
	}

	attachment, err := core.database.Get(attachmentID, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting attachment from database: %w", err)
	}

	return attachment, nil
}

func (core *Attachment) RefreshUploadURL(
	attachmentID uuid.UUID,
	userID uuid.UUID,
) (*model.AttachmentURL, error) {
	attachment, err := core.get(attachmentID, userID)
	if err != nil {
		return nil, err
	}

	return core.createUploadURL(attachment.MinioName, attachment.Size)
}

func (core *Attachment) Get(
	attachmentID uuid.UUID,
	userID uuid.UUID,
) (*model.AttachmentURL, error) {
	attachment, err := core.get(attachmentID, userID)
	if err != nil {
		return nil, err
	}

	fileName := "filename=\"" + attachment.Name + "\""

	link, err := core.minio.PresignedGetObject(
		context.Background(),
		core.bucket,
		attachment.MinioName,
		core.expires,
		url.Values{
			"response-content-disposition": {"attachment; " + fileName},
			"response-content-type":        {attachment.ContentType},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating minio link: %w", err)
	}

	attachmentURL := model.AttachmentURL{
		URL: link.String(),
	}

	return &attachmentURL, nil
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
	maxEntrySize int,
) *Attachment {
	return &Attachment{
		database:     database,
		minio:        minio,
		bucket:       bucket,
		validator:    validate,
		expires:      time.Minute * 30,           //nolint:gomnd
		maxEntrySize: maxEntrySize * 1000 * 1000, //nolint:gomnd
	}
}
