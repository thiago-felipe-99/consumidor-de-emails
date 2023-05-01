package core

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/thiago-felipe-99/mail/publisher/data"
	"github.com/thiago-felipe-99/mail/publisher/model"
	"github.com/thiago-felipe-99/mail/rabbit"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrInvalidID                = errors.New("was sent a invalid ID")
	ErrInvalidName              = errors.New("was sent a invalid name")
	ErrUserAlreadyExist         = errors.New("user already exist")
	ErrUserDoesNotExist         = errors.New("user does not exist")
	ErrUserWrongPassword        = errors.New("was sent a wrong password")
	ErrUserSessionDoesNotExist  = errors.New("user session does not exist")
	ErrUserIsNotAdmin           = errors.New("user is not admin")
	ErrUserIsProtected          = errors.New("user is protected")
	ErrRoleAlreadyExist         = errors.New("role already exist")
	ErrRoleDoesNotExist         = errors.New("role does not exist")
	ErrQueueAlreadyExist        = errors.New("queue already exist")
	ErrQueueDoesNotExist        = errors.New("queue does not exist")
	ErrBodyValidate             = errors.New("unable to parse body")
	ErrTemplateNameAlreadyExist = errors.New("template name already exist")
	ErrMaxSizeTemplate          = errors.New("template has a max size of 1MB")
	ErrMissingFieldTemplates    = errors.New("missing fields from template")
	ErrTemplateDoesNotExist     = errors.New("template does not exist")
)

const (
	argon2idParamMemory      = 128 * 1024
	argon2idParamIterations  = 2
	argon2idParamSaltLength  = 32
	argon2idParamKeyLength   = 64
	argon2idParamParallelism = 4
)

const maxSizeTemplate = 1024 * 1024

type ModelInvalidError struct {
	invalid validator.ValidationErrors
}

func (err ModelInvalidError) Error() string {
	return err.invalid.Error()
}

func (err ModelInvalidError) Translate(language ut.Translator) string {
	messages := err.invalid.Translate(language)

	messageSend := ""
	for _, message := range messages {
		messageSend += ", " + message
	}

	return messageSend[2:]
}

func validate(validate *validator.Validate, data any) error {
	err := validate.Struct(data)
	if err != nil {
		validationErrs := validator.ValidationErrors{}

		okay := errors.As(err, &validationErrs)
		if !okay {
			return ErrBodyValidate
		}

		return ModelInvalidError{validationErrs}
	}

	return nil
}

type User struct {
	database        *data.User
	validator       *validator.Validate
	argon2id        argon2id.Params
	durationSession time.Duration
}

func (core *User) existByID(userID uuid.UUID) (bool, error) {
	user, err := core.database.GetByID(userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}

		return false, fmt.Errorf("error checking if user exist in database: %w", err)
	}

	return user.DeletedAt.IsZero(), nil
}

func (core *User) ExistByNameOrEmail(name, email string) (bool, error) {
	user, err := core.database.GetByNameOrEmail(name, email)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}

		return false, fmt.Errorf("error checking if user exist in database: %w", err)
	}

	return user.DeletedAt.IsZero(), nil
}

func (core *User) Create(partial model.UserPartial, adminID uuid.UUID) error {
	err := validate(core.validator, partial)
	if err != nil {
		return err
	}

	exist, err := core.ExistByNameOrEmail(partial.Name, partial.Email)
	if err != nil {
		return fmt.Errorf("error checking if user exist in database: %w", err)
	}

	if exist {
		return ErrUserAlreadyExist
	}

	hash, err := argon2id.CreateHash(partial.Password, &core.argon2id)
	if err != nil {
		return fmt.Errorf("error creating password hash: %w", err)
	}

	user := model.User{
		ID:        uuid.New(),
		Name:      partial.Name,
		Email:     partial.Email,
		Password:  hash,
		Roles:     []model.UserRole{},
		CreatedAt: time.Now(),
		CreatedBy: adminID,
		IsAdmin:   false,
		DeletedAt: time.Time{},
		DeletedBy: uuid.UUID{},
	}

	err = core.database.Create(user)
	if err != nil {
		return fmt.Errorf("error creating user in database: %w", err)
	}

	return nil
}

func (core *User) GetByID(userID uuid.UUID) (*model.User, error) {
	exist, err := core.existByID(userID)
	if err != nil {
		return nil, fmt.Errorf("error checking if user exist in database: %w", err)
	}

	if !exist {
		return nil, ErrUserDoesNotExist
	}

	user, err := core.database.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("error getting user from database: %w", err)
	}

	return user, nil
}

func (core *User) GetAll() ([]model.User, error) {
	users, err := core.database.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error getting alls users from database: %w", err)
	}

	return users, nil
}

func (core *User) GetByNameOrEmail(name, email string) (*model.User, error) {
	exist, err := core.ExistByNameOrEmail(name, email)
	if err != nil {
		return nil, fmt.Errorf("error checking if user exist in database: %w", err)
	}

	if !exist {
		return nil, ErrUserDoesNotExist
	}

	user, err := core.database.GetByNameOrEmail(name, email)
	if err != nil {
		return nil, fmt.Errorf("error getting user from database: %w", err)
	}

	return user, nil
}

func (core *User) Update(userID uuid.UUID, partial model.UserPartial) error {
	err := validate(core.validator, partial)
	if err != nil {
		return err
	}

	user, err := core.GetByID(userID)
	if err != nil {
		return fmt.Errorf("error checking if user exist in database: %w", err)
	}

	hash, err := argon2id.CreateHash(partial.Password, &core.argon2id)
	if err != nil {
		return fmt.Errorf("error creating password hash: %w", err)
	}

	user.Password = hash

	err = core.database.Update(*user)
	if err != nil {
		return fmt.Errorf("error updating user in database: %w", err)
	}

	return nil
}

func (core *User) Delete(userID uuid.UUID, deleteByID uuid.UUID) error {
	user, err := core.GetByID(userID)
	if err != nil {
		return err
	}

	if user.IsProtected {
		return ErrUserIsProtected
	}

	user.DeletedAt = time.Now()
	user.DeletedBy = deleteByID

	err = core.database.Update(*user)
	if err != nil {
		return fmt.Errorf("error deleting user from database: %w", err)
	}

	return nil
}

func (core *User) IsAdmin(userID uuid.UUID) (bool, error) {
	user, err := core.GetByID(userID)
	if err != nil {
		return false, err
	}

	return user.IsAdmin, nil
}

func (core *User) NewAdmin(userID uuid.UUID) error {
	user, err := core.GetByID(userID)
	if err != nil {
		return err
	}

	user.IsAdmin = true

	err = core.database.Update(*user)
	if err != nil {
		return fmt.Errorf("error updating user in database: %w", err)
	}

	return nil
}

func (core *User) RemoveAdmin(userID uuid.UUID) error {
	user, err := core.GetByID(userID)
	if err != nil {
		return err
	}

	if user.IsProtected {
		return ErrUserIsProtected
	}

	if !user.IsAdmin {
		return ErrUserIsNotAdmin
	}

	user.IsAdmin = false

	err = core.database.Update(*user)
	if err != nil {
		return fmt.Errorf("error updating user in database: %w", err)
	}

	return nil
}

func (core *User) Protected(userID uuid.UUID) error {
	user, err := core.GetByID(userID)
	if err != nil {
		return err
	}

	user.IsAdmin = true
	user.IsProtected = true

	err = core.database.Update(*user)
	if err != nil {
		return fmt.Errorf("error updating user in database: %w", err)
	}

	return nil
}

func (core *User) GetRoles(userID uuid.UUID) ([]model.UserRole, error) {
	user, err := core.GetByID(userID)
	if err != nil {
		return nil, err
	}

	return user.Roles, nil
}

func (core *User) existRole(role string) (bool, error) {
	_, err := core.database.GetRole(role)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}

		return false, fmt.Errorf("error getting role from database: %w", err)
	}

	return true, nil
}

func existsInSlice[T comparable](slice []T, find T) (int, bool) {
	for index, element := range slice {
		if element == find {
			return index, true
		}
	}

	return 0, false
}

func (core *User) existRoles(roles []string) (bool, error) {
	if len(roles) == 0 {
		return true, nil
	}

	rolesRaw, err := core.database.GetAllRoles()
	if err != nil {
		return false, fmt.Errorf("error getting role from database: %w", err)
	}

	rolesNames := make([]string, 0, len(rolesRaw))
	for _, role := range rolesRaw {
		rolesNames = append(rolesNames, role.Name)
	}

	for _, role := range roles {
		if _, exist := existsInSlice(rolesNames, role); !exist {
			return false, nil
		}
	}

	return true, nil
}

func (core *User) CreateRole(partial model.RolePartial, userID uuid.UUID) error {
	err := validate(core.validator, partial)
	if err != nil {
		return err
	}

	exist, err := core.existRole(partial.Name)
	if err != nil {
		return fmt.Errorf("error checking if role exist: %w", err)
	}

	if exist {
		return ErrRoleAlreadyExist
	}

	role := model.Role{
		ID:        uuid.New(),
		Name:      partial.Name,
		CreatedAt: time.Now(),
		CreatedBy: userID,
		DeletedAt: time.Time{},
		DeletedBy: uuid.UUID{},
	}

	err = core.database.CreateRole(role)
	if err != nil {
		return fmt.Errorf("error creating role in database: %w", err)
	}

	return nil
}

func (core *User) HasRoles(userID uuid.UUID, roles []model.UserRole) (bool, error) {
	if len(roles) == 0 {
		return true, nil
	}

	user, err := core.GetByID(userID)
	if err != nil {
		return false, err
	}

	if user.IsAdmin {
		return true, nil
	}

	userRoles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		userRoles = append(userRoles, role.Name)
	}

	for _, role := range roles {
		index, exist := existsInSlice(userRoles, role.Name)
		if !exist {
			return false, nil
		}

		if (role.IsAdmin && !user.Roles[index].IsAdmin) ||
			(role.IsProtected && !user.Roles[index].IsProtected) {
			return false, nil
		}
	}

	return true, nil
}

func (core *User) HasRolesAdmin(userID uuid.UUID, roles []model.RolePartial) (bool, error) {
	if len(roles) == 0 {
		return true, nil
	}

	user, err := core.GetByID(userID)
	if err != nil {
		return false, err
	}

	if user.IsAdmin {
		return true, nil
	}

	userRoles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		userRoles = append(userRoles, role.Name)
	}

	for _, role := range roles {
		index, exist := existsInSlice(userRoles, role.Name)
		if !exist {
			return false, nil
		}

		if !user.Roles[index].IsAdmin {
			return false, nil
		}
	}

	return true, nil
}

func (core *User) AddRoles(roles []model.UserRole, userID uuid.UUID) error {
	if len(roles) == 0 {
		return nil
	}

	user, err := core.GetByID(userID)
	if err != nil {
		return err
	}

	rolesName := make([]string, 0, len(roles))
	for _, role := range roles {
		rolesName = append(rolesName, role.Name)
	}

	exist, err := core.existRoles(rolesName)
	if err != nil {
		return fmt.Errorf("eror checking if roles exist: %w", err)
	}

	if !exist {
		return ErrRoleDoesNotExist
	}

	userRolesName := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		userRolesName = append(userRolesName, role.Name)
	}

	for _, role := range roles {
		index, exist := existsInSlice(userRolesName, role.Name)
		if !exist {
			newRole := model.UserRole{
				Name:        role.Name,
				IsAdmin:     role.IsAdmin || role.IsProtected,
				IsProtected: role.IsProtected,
			}

			user.Roles = append(user.Roles, newRole)
		} else if !user.Roles[index].IsProtected {
			user.Roles[index].IsAdmin = role.IsAdmin || role.IsProtected
			user.Roles[index].IsProtected = role.IsProtected
		}
	}

	err = core.database.Update(*user)
	if err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}

	return nil
}

func (core *User) DeleteRoles(roles []model.RolePartial, userID uuid.UUID, protected bool) error {
	if len(roles) == 0 {
		return nil
	}

	user, err := core.GetByID(userID)
	if err != nil {
		return err
	}

	rolesName := make([]string, 0, len(roles))
	for _, role := range roles {
		rolesName = append(rolesName, role.Name)
	}

	exist, err := core.existRoles(rolesName)
	if err != nil {
		return fmt.Errorf("eror checking if roles exist: %w", err)
	}

	if !exist {
		return ErrRoleDoesNotExist
	}

	userRolesName := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		userRolesName = append(userRolesName, role.Name)
	}

	for _, role := range roles {
		index, exist := existsInSlice(userRolesName, role.Name)
		if exist && (!user.Roles[index].IsProtected || protected) {
			lastIndex := len(user.Roles) - 1

			user.Roles[index] = user.Roles[lastIndex]
			user.Roles = user.Roles[:lastIndex]

			userRolesName[index] = userRolesName[lastIndex]
			userRolesName = userRolesName[:lastIndex]
		}
	}

	err = core.database.Update(*user)
	if err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}

	return nil
}

func (core *User) NewSession(partial model.UserSessionPartial) (*model.UserSession, error) {
	err := validate(core.validator, partial)
	if err != nil {
		return nil, err
	}

	exist, err := core.ExistByNameOrEmail(partial.Name, partial.Email)
	if err != nil {
		return nil, fmt.Errorf("error checking if user exist in database: %w", err)
	}

	if !exist {
		return nil, ErrUserDoesNotExist
	}

	user, err := core.database.GetByNameOrEmail(partial.Name, partial.Email)
	if err != nil {
		return nil, fmt.Errorf("error getting user in database: %w", err)
	}

	equals, err := argon2id.ComparePasswordAndHash(partial.Password, user.Password)
	if err != nil {
		return nil, fmt.Errorf("error comparing password with hash: %w", err)
	}

	if !equals {
		return nil, ErrUserWrongPassword
	}

	session := model.UserSession{
		ID:        uuid.New(),
		UserID:    user.ID,
		CreateaAt: time.Now(),
		Expires:   time.Now().Add(core.durationSession),
		DeletedAt: time.Now().Add(core.durationSession),
	}

	err = core.database.SaveSession(session)
	if err != nil {
		return nil, fmt.Errorf("error saving session in database: %w", err)
	}

	return &session, nil
}

func (core *User) RefreshSession(sessionID string) (*model.UserSession, error) {
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrInvalidID
	}

	exist, err := core.database.ExistSession(sessionUUID)
	if err != nil {
		return nil, fmt.Errorf("error checking if session exist in database: %w", err)
	}

	if !exist {
		return nil, ErrUserSessionDoesNotExist
	}

	currentSession, err := core.database.GetSession(sessionUUID)
	if err != nil {
		return nil, fmt.Errorf("error getting session from database: %w", err)
	}

	if currentSession.DeletedAt.Before(time.Now()) {
		return nil, ErrUserSessionDoesNotExist
	}

	currentSession.DeletedAt = time.Now()

	exist, err = core.existByID(currentSession.UserID)
	if err != nil {
		return nil, fmt.Errorf("error checking if user exist in database: %w", err)
	}

	if !exist {
		return nil, ErrUserSessionDoesNotExist
	}

	err = core.database.UpdateSession(*currentSession)
	if err != nil {
		return nil, fmt.Errorf("error updating session in database: %w", err)
	}

	newSession := model.UserSession{
		ID:        uuid.New(),
		UserID:    currentSession.UserID,
		CreateaAt: time.Now(),
		Expires:   time.Now().Add(core.durationSession),
		DeletedAt: time.Now().Add(core.durationSession),
	}

	err = core.database.SaveSession(newSession)
	if err != nil {
		return nil, fmt.Errorf("error saving session in database: %w", err)
	}

	return &newSession, nil
}

func NewUser(
	database *data.User,
	validate *validator.Validate,
	durationSession time.Duration,
) *User {
	return &User{
		database:  database,
		validator: validate,
		argon2id: argon2id.Params{
			Memory:      argon2idParamMemory,
			Iterations:  argon2idParamIterations,
			Parallelism: argon2idParamParallelism,
			SaltLength:  argon2idParamSaltLength,
			KeyLength:   argon2idParamKeyLength,
		},
		durationSession: durationSession,
	}
}

func dlxName(name string) string {
	return name + "-dlx"
}

type Queue struct {
	template  *Template
	rabbit    *rabbit.Rabbit
	database  *data.Queue
	validator *validator.Validate
}

func (core *Queue) Create(partial model.QueuePartial) error {
	err := validate(core.validator, partial)
	if err != nil {
		return err
	}

	queue := model.Queue{
		ID:         uuid.New(),
		Name:       partial.Name,
		DLX:        dlxName(partial.Name),
		MaxRetries: partial.MaxRetries,
		CreatedAt:  time.Now(),
	}

	queueExist, err := core.database.Exist(queue.Name)
	if err != nil {
		return fmt.Errorf("error checking if queue exist in database: %w", err)
	}

	dlxExist, err := core.database.Exist(queue.Name)
	if err != nil {
		return fmt.Errorf("error checking if dlx queue exist in database: %w", err)
	}

	if queueExist || dlxExist {
		return ErrQueueAlreadyExist
	}

	err = core.rabbit.CreateQueueWithDLX(queue.Name, queue.DLX, queue.MaxRetries)
	if err != nil {
		return fmt.Errorf("error creating queue: %w", err)
	}

	err = core.database.Create(queue)
	if err != nil {
		return fmt.Errorf("error creating queue in database: %w", err)
	}

	return nil
}

func (core *Queue) GetAll() ([]model.Queue, error) {
	queues, err := core.database.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error getting all queues: %w", err)
	}

	return queues, nil
}

func (core *Queue) Delete(name string) error {
	if len(name) == 0 {
		return ErrInvalidName
	}

	exist, err := core.database.Exist(name)
	if err != nil {
		return fmt.Errorf("error checking if queue exist: %w", err)
	}

	if !exist {
		return ErrQueueDoesNotExist
	}

	err = core.rabbit.DeleteQueueWithDLX(name, dlxName(name))
	if err != nil {
		return fmt.Errorf("error deleting queue from RabbitMQ: %w", err)
	}

	err = core.database.Delete(name)
	if err != nil {
		return fmt.Errorf("error deleting queue from database: %w", err)
	}

	return nil
}

func (core *Queue) SendEmail(queue string, partial model.EmailPartial, userID uuid.UUID) error {
	if len(queue) == 0 {
		return ErrInvalidName
	}

	err := validate(core.validator, partial)
	if err != nil {
		return err
	}

	queueExist, err := core.database.Exist(queue)
	if err != nil {
		return fmt.Errorf("error checking if queue exist: %w", err)
	}

	if !queueExist {
		return ErrQueueDoesNotExist
	}

	if partial.Template != nil {
		exist, err := core.template.Exist(partial.Template.Name)
		if err != nil {
			return fmt.Errorf("error checking if template exist: %w", err)
		}

		if !exist {
			return ErrTemplateDoesNotExist
		}

		fields, err := core.template.GetFields(partial.Template.Name)
		if err != nil {
			return fmt.Errorf("error getting templates fields: %w", err)
		}

		for _, field := range fields {
			if _, found := partial.Template.Data[field]; !found {
				return ErrMissingFieldTemplates
			}
		}
	}

	// add logic to get emails from mail list

	err = core.rabbit.SendMessage(context.Background(), queue, partial)
	if err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}

	email := model.Email{
		ID:             uuid.New(),
		UserID:         userID,
		EmailLists:     partial.EmailLists,
		Receivers:      partial.Receivers,
		BlindReceivers: partial.BlindReceivers,
		Subject:        partial.Subject,
		Message:        partial.Message,
		Template:       partial.Template,
		Attachments:    partial.Attachments,
		SentAt:         time.Now(),
	}

	err = core.database.SaveEmail(email)
	if err != nil {
		return fmt.Errorf("error saving email in database: %w", err)
	}

	return nil
}

func NewQueue(
	template *Template,
	rabbit *rabbit.Rabbit,
	database *data.Queue,
	validate *validator.Validate,
) *Queue {
	return &Queue{
		template:  template,
		rabbit:    rabbit,
		database:  database,
		validator: validate,
	}
}

type Template struct {
	minio       *minio.Client
	bucket      string
	database    *data.Template
	validate    *validator.Validate
	regexFields *regexp.Regexp
}

func (core *Template) getFields(template string) []string {
	fieldsRaw := core.regexFields.FindAllString(template, -1)

	fields := make([]string, 0, len(fieldsRaw))

	existField := func(fields []string, find string) bool {
		for _, field := range fields {
			if field == find {
				return true
			}
		}

		return false
	}

	for _, field := range fieldsRaw {
		field = strings.Trim(field, "{} ")

		if !existField(fields, field) {
			fields = append(fields, field)
		}
	}

	return fields
}

func (core *Template) Create(partial model.TemplatePartial, userID uuid.UUID) error {
	err := validate(core.validate, partial)
	if err != nil {
		return err
	}

	if len(partial.Template) > maxSizeTemplate {
		return ErrMaxSizeTemplate
	}

	exist, err := core.database.Exist(partial.Name)
	if err != nil {
		return fmt.Errorf("error checking if template exist: %w", err)
	}

	if exist {
		return ErrTemplateNameAlreadyExist
	}

	template := model.Template{
		ID:        uuid.New(),
		Name:      partial.Name,
		Template:  partial.Template,
		Fields:    core.getFields(partial.Template),
		Roles:     []string{},
		CreatedAt: time.Now(),
		CreatedBy: userID,
		DeletedAt: time.Time{},
		DeletedBy: uuid.UUID{},
	}

	templateReader := strings.NewReader(template.Template)

	_, err = core.minio.PutObject(
		context.Background(),
		core.bucket,
		template.Name,
		templateReader,
		templateReader.Size(),
		minio.PutObjectOptions{
			ContentType: "text/markdown",
		},
	)
	if err != nil {
		return fmt.Errorf("error creating template in Minio: %w", err)
	}

	err = core.database.Create(template)
	if err != nil {
		return fmt.Errorf("error creating template in database: %w", err)
	}

	return nil
}

func (core *Template) Update(name string, partial model.TemplatePartial) error {
	err := validate(core.validate, partial)
	if err != nil {
		return err
	}

	if len(partial.Template) > maxSizeTemplate {
		return ErrMaxSizeTemplate
	}

	exist, err := core.database.Exist(name)
	if err != nil {
		return fmt.Errorf("error checking if template exist: %w", err)
	}

	if !exist {
		return ErrTemplateDoesNotExist
	}

	template, err := core.database.Get(name)
	if err != nil {
		return fmt.Errorf("error getting template ID: %w", err)
	}

	template.Template = partial.Template
	template.Fields = core.getFields(partial.Template)

	templateReader := strings.NewReader(template.Template)

	_, err = core.minio.PutObject(
		context.Background(),
		core.bucket,
		template.Name,
		templateReader,
		templateReader.Size(),
		minio.PutObjectOptions{
			ContentType: "text/markdown",
		},
	)
	if err != nil {
		return fmt.Errorf("error updating template in Minio: %w", err)
	}

	err = core.database.Update(*template)
	if err != nil {
		return fmt.Errorf("error updating template in database: %w", err)
	}

	return nil
}

func (core *Template) Exist(name string) (bool, error) {
	exist, err := core.database.Exist(name)
	if err != nil {
		return false, fmt.Errorf("error checking if template exist in database: %w", err)
	}

	return exist, nil
}

func (core *Template) GetFields(name string) ([]string, error) {
	template, err := core.database.Get(name)
	if err != nil {
		return nil, fmt.Errorf("error getting template from database: %w", err)
	}

	return template.Fields, nil
}

func (core *Template) GetAll() ([]model.Template, error) {
	templates, err := core.database.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error getting templates from database: %w", err)
	}

	return templates, nil
}

func (core *Template) Get(name string) (*model.Template, error) {
	if len(name) == 0 {
		return nil, ErrInvalidName
	}

	exist, err := core.database.Exist(name)
	if err != nil {
		return nil, fmt.Errorf("error checking if template exist: %w", err)
	}

	if !exist {
		return nil, ErrTemplateDoesNotExist
	}

	template, err := core.database.Get(name)
	if err != nil {
		return nil, fmt.Errorf("error getting template from database: %w", err)
	}

	return template, nil
}

func (core *Template) Delete(name string, userID uuid.UUID) error {
	if len(name) == 0 {
		return ErrInvalidName
	}

	template, err := core.Get(name)
	if err != nil {
		return err
	}

	err = core.minio.RemoveObject(
		context.Background(),
		core.bucket,
		template.Name,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("error deleting template from Minio: %w", err)
	}

	template.DeletedAt = time.Now()
	template.DeletedBy = userID

	err = core.database.Update(*template)
	if err != nil {
		return fmt.Errorf("error deleting template from database: %w", err)
	}

	return nil
}

func NewTemplate(
	database *data.Template,
	minio *minio.Client,
	bucket string,
	validate *validator.Validate,
) *Template {
	return &Template{
		database:    database,
		minio:       minio,
		bucket:      bucket,
		validate:    validate,
		regexFields: regexp.MustCompile(`{{ *(\w|\d)+ *}}`),
	}
}

type Cores struct {
	*User
	*Queue
	*Template
}

func NewCores(
	databases *data.Databases,
	validate *validator.Validate,
	sessionDuration time.Duration,
	rabbit *rabbit.Rabbit,
	minio *minio.Client,
	bukcetTemplate string,
) *Cores {
	template := NewTemplate(databases.Template, minio, bukcetTemplate, validate)

	return &Cores{
		User:     NewUser(databases.User, validate, sessionDuration),
		Template: template,
		Queue:    NewQueue(template, rabbit, databases.Queue, validate),
	}
}
