package core

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
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

func (core *Attachment) confirmUploads() {
	events := core.minio.ListenBucketNotification(
		context.Background(),
		core.bucket,
		"",
		"",
		[]string{"s3:ObjectCreated:*"},
	)

	for event := range events {
		if event.Err != nil {
			log.Printf("[ERROR] - Error processing minio event: %s", event.Err)

			continue
		}

		for _, resource := range event.Records {
			attachment, err := core.database.GetByMinioName(resource.S3.Object.Key)
			if err != nil {
				log.Printf("[ERROR] - Unable do get attachment from database: %s", err)

				continue
			}

			attachment.ConfirmedUpload = true

			err = core.database.Update(*attachment)
			if err != nil {
				log.Printf("[ERROR] - Error updating attachment on database: %s", err)

				continue
			}

			log.Printf("[INFO] - Upload confirmed: %s", resource.S3.Object.Key)
		}
	}
}

func (core *Attachment) createUploadURL(
	attachmentID uuid.UUID,
	path string,
	size int,
) (*model.AttachmentURL, error) {
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
		return nil, fmt.Errorf("error creating minio url: %w", err)
	}

	attachmentURL := model.AttachmentURL{
		ID:       attachmentID,
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

	return core.createUploadURL(attachment.ID, attachment.MinioName, attachment.Size)
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

	if attachment.ConfirmedUpload {
		return nil, ErrUploadAlreadyConfirmed
	}

	return core.createUploadURL(attachment.ID, attachment.MinioName, attachment.Size)
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

	url, err := core.minio.PresignedGetObject(
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
		return nil, fmt.Errorf("error creating minio url: %w", err)
	}

	attachmentURL := model.AttachmentURL{
		ID:  attachment.ID,
		URL: url.String(),
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

func (core *Attachment) ConfirmUpload(attachmentID uuid.UUID, userID uuid.UUID) error {
	attachment, err := core.get(attachmentID, userID)
	if err != nil {
		return err
	}

	if attachment.ConfirmedUpload {
		return nil
	}

	_, err = core.minio.StatObject(
		context.Background(),
		core.bucket,
		attachment.MinioName,
		minio.StatObjectOptions{},
	)
	if err != nil {
		errorResponse := minio.ErrorResponse{}
		if errors.As(err, &errorResponse) && errorResponse.StatusCode == fiber.StatusNotFound {
			return ErrAttachmentDoesNotExistOnMinio
		}

		return fmt.Errorf("error verifying if Attachment exist on minio: %w", err)
	}

	attachment.ConfirmedUpload = true

	err = core.database.Update(*attachment)
	if err != nil {
		return fmt.Errorf("error updating attachment on database: %w", err)
	}

	return nil
}

func newAttachment(
	minio *minio.Client,
	bucket string,
	database *data.Attachment,
	validate *validator.Validate,
	maxEntrySize int,
) *Attachment {
	attachment := &Attachment{
		database:     database,
		minio:        minio,
		bucket:       bucket,
		validator:    validate,
		expires:      time.Minute * 30,           //nolint:gomnd
		maxEntrySize: maxEntrySize * 1000 * 1000, //nolint:gomnd
	}

	go attachment.confirmUploads()

	return attachment
}
