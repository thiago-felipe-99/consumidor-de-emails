package controllers

import (
	"errors"
	"log"
	"time"

	ut "github.com/go-playground/universal-translator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/thiago-felipe-99/mail/publisher/core"
	"github.com/thiago-felipe-99/mail/publisher/model"
)

type User struct {
	core       *core.User
	translator *ut.UniversalTranslator
	languages  []string
}

func (controller *User) getTranslator(handler *fiber.Ctx) ut.Translator { //nolint:ireturn
	accept := handler.AcceptsLanguages(controller.languages...)
	if accept == "" {
		accept = controller.languages[0]
	}

	language, _ := controller.translator.GetTranslator(accept)

	return language
}

// Create a user in application
//
//	@Summary		Create user
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Success		201		{object}	sent				"user created successfully"
//	@Failure		400		{object}	sent				"an invalid user param was sent"
//	@Failure		409		{object}	sent				"user already exist"
//	@Failure		500		{object}	sent				"internal server error"
//	@Param			user	body		model.UserPartial	true	"user params"
//	@Router			/user [post]
//	@Description	Create a user in application.
func (controller *User) create(handler *fiber.Ctx) error {
	body := &model.UserPartial{}

	err := handler.BodyParser(body)
	if err != nil {
		return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
	}

	funcCore := func() error { return controller.core.Create(*body) }

	expectErrors := []expectError{
		{core.ErrUserAlreadyExist, fiber.StatusConflict},
	}

	unexpectMessageError := "error creating user"

	okay := okay{"user created", fiber.StatusCreated}

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
}

// Get current user informations
//
//	@Summary		Get user
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	sent	"user informations"
//	@Failure		401	{object}	sent	"unauthorized"
//	@Failure		404	{object}	sent	"user does not exist"
//	@Failure		500	{object}	sent	"internal server error"
//	@Router			/user [get]
//	@Description	Get current user informatios.
func (controller *User) get(handler *fiber.Ctx) error {
	userID, ok := handler.Locals("userID").(uuid.UUID)
	if !ok {
		log.Printf("[ERROR] - error getting user ID")

		return handler.Status(fiber.StatusInternalServerError).
			JSON(sent{"error refreshing session"})
	}

	funcCore := func() (*model.User, error) { return controller.core.Get(userID) }

	expectErrors := []expectError{{core.ErrUserDoesNotExist, fiber.StatusNotFound}}

	unexpectMessageError := "error getting user"

	return callingCoreWithReturn(
		funcCore,
		expectErrors,
		unexpectMessageError,
		handler,
	)
}

// Update user informations
//
//	@Summary		Update user
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	sent				"user updated"
//	@Failure		400		{object}	sent				"an invalid user param was sent"
//	@Failure		401		{object}	sent				"unauthorized"
//	@Failure		404		{object}	sent				"user does not exist"
//	@Failure		500		{object}	sent				"internal server error"
//	@Param			user	body		model.UserPartial	true	"user params"
//	@Router			/user [put]
//	@Description	Update user informatios.
func (controller *User) update(handler *fiber.Ctx) error {
	userID, ok := handler.Locals("userID").(uuid.UUID)
	if !ok {
		log.Printf("[ERROR] - error getting user ID")

		return handler.Status(fiber.StatusInternalServerError).
			JSON(sent{"error updating user"})
	}

	user := &model.UserPartial{}

	err := handler.BodyParser(user)
	if err != nil {
		return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
	}

	funcCore := func() error { return controller.core.Update(userID, *user) }

	expectErrors := []expectError{{core.ErrUserDoesNotExist, fiber.StatusNotFound}}

	unexpectMessageError := "error updating user"

	okay := okay{"user updated", fiber.StatusOK}

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
}

// Delete current user
//
//	@Summary		Delete user
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	sent	"user deleted"
//	@Failure		401	{object}	sent	"unauthorized"
//	@Failure		404	{object}	sent	"user dont exist"
//	@Failure		500	{object}	sent	"internal server error"
//	@Router			/user [delete]
//	@Description	Delete current user.
func (controller *User) delete(handler *fiber.Ctx) error {
	userID, ok := handler.Locals("userID").(uuid.UUID)
	if !ok {
		log.Printf("[ERROR] - error getting user ID")

		return handler.Status(fiber.StatusInternalServerError).
			JSON(sent{"error refreshing session"})
	}

	funcCore := func() error { return controller.core.Delete(userID) }

	expectErrors := []expectError{{core.ErrUserDoesNotExist, fiber.StatusNotFound}}

	unexpectMessageError := "error deleting user"

	okay := okay{"user deleted", fiber.StatusOK}

	handler.Locals("deleteSession", true)

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
}

func (controller *User) isAdmin(handler *fiber.Ctx) error {
	userID, ok := handler.Locals("userID").(uuid.UUID)
	if !ok {
		log.Printf("[ERROR] - error getting user ID")

		return handler.Status(fiber.StatusInternalServerError).
			JSON(sent{"error refreshing session"})
	}

	isAdmin, err := controller.core.IsAdmin(userID)
	if err != nil {
		log.Printf("[ERROR] - error getting if user is admin: %s", err)

		return handler.Status(fiber.StatusInternalServerError).
			JSON(sent{"error getting if user is admin"})
	}

	if !isAdmin {
		return handler.Status(fiber.StatusUnauthorized).JSON(sent{"current user is not admin"})
	}

	return handler.Next()
}

// Create a user admin
//
//	@Summary		Create admin
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Success		201		{object}	sent				"admin created successfully"
//	@Failure		401		{object}	sent				"unauthorized"
//	@Failure		404		{object}	sent				"user does not exist"
//	@Failure		500		{object}	sent				"internal server error"
//	@Param			userID	path		string	true	"user id to be promoted to admin"
//	@Router			/user/admin/{userID} [post]
//	@Description	Create a user session and set in the response cookie.
func (controller *User) newAdmin(handler *fiber.Ctx) error {
	userID, err := uuid.Parse(handler.Params("userID"))
	if err != nil {
		return handler.Status(fiber.StatusBadRequest).
			JSON(sent{"was sent a invalid user ID"})
	}

	funcCore := func() error { return controller.core.NewAdmin(userID) }

	expectErrors := []expectError{{core.ErrUserDoesNotExist, fiber.StatusNotFound}}

	unexpectMessageError := "error promoting user"

	okay := okay{"user promoted to admin", fiber.StatusCreated}

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
}

// Create a user session
//
//	@Summary		Create session
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Success		201		{object}	sent				"session created successfully"
//	@Failure		400		{object}	sent				"an invalid user param was sent"
//	@Failure		404		{object}	sent				"user does not exist"
//	@Failure		500		{object}	sent				"internal server error"
//	@Param			user	body		model.UserPartial	true	"user params"
//	@Router			/user/session [post]
//	@Description	Create a user session and set in the response cookie.
func (controller *User) newSession(handler *fiber.Ctx) error {
	body := &model.UserSessionPartial{}

	err := handler.BodyParser(body)
	if err != nil {
		return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
	}

	session := &model.UserSession{}

	funcCore := func() error {
		sessionTemp, err := controller.core.NewSession(*body)
		session = sessionTemp

		return err
	}

	expectErrors := []expectError{
		{core.ErrUserDoesNotExist, fiber.StatusNotFound},
		{core.ErrDifferentPassword, fiber.StatusBadRequest},
	}

	unexpectMessageError := "error creating user session"

	okay := okay{"session created", fiber.StatusCreated}

	err = callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)

	cookie := &fiber.Cookie{
		Name:     "session",
		Value:    "",
		Expires:  time.Now(),
		HTTPOnly: true,
		Secure:   true,
	}

	deleteSession, ok := handler.Locals("deleteSession").(bool)

	if session != nil && !(ok && deleteSession) {
		cookie.Value = session.ID.String()
		cookie.Expires = session.Expires
	}

	handler.Cookie(cookie)

	return err
}

// Refresh a user session
//
//	@Summary		Refresh session
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	sent	"session refreshed successfully"
//	@Failure		401	{object}	sent	"session does not exist"
//	@Failure		500	{object}	sent	"internal server error"
//	@Router			/user/session [put]
//	@Description	Refresh a user session and set in the response cookie.
func (controller *User) refreshSession(handler *fiber.Ctx) error {
	sessionID := handler.Cookies("session", "invalid_session")

	cookie := &fiber.Cookie{
		Name:     "session",
		Value:    "",
		Expires:  time.Now(),
		HTTPOnly: true,
		Secure:   true,
	}

	session, err := controller.core.RefreshSession(sessionID)
	if err != nil {
		handler.Cookie(cookie)

		if errors.Is(err, core.ErrUserSessionDoesNotExist) || errors.Is(err, core.ErrInvalidID) {
			return handler.Status(fiber.StatusUnauthorized).
				JSON(sent{core.ErrUserSessionDoesNotExist.Error()})
		}

		log.Printf("[ERROR] - error refreshing session: %s", err)

		return handler.Status(fiber.StatusInternalServerError).
			JSON(sent{"error refreshing session"})
	}

	if session != nil {
		cookie.Value = session.ID.String()
		cookie.Expires = session.Expires
	}

	handler.Cookie(cookie)
	handler.Locals("userID", session.UserID)

	return handler.Next()
}

type Queue struct {
	core       *core.Queue
	translator *ut.UniversalTranslator
	languages  []string
}

func (controller *Queue) getTranslator(handler *fiber.Ctx) ut.Translator { //nolint:ireturn
	accept := handler.AcceptsLanguages(controller.languages...)
	if accept == "" {
		accept = controller.languages[0]
	}

	language, _ := controller.translator.GetTranslator(accept)

	return language
}

// Create a RabbitMQ queue with DLX
//
//	@Summary		Creating queue
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Success		201		{object}	sent				"create queue successfully"
//	@Failure		400		{object}	sent				"an invalid queue param was sent"
//	@Failure		409		{object}	sent				"queue already exist"
//	@Failure		500		{object}	sent				"internal server error"
//	@Param			queue	body		model.QueuePartial	true	"queue params"
//	@Router			/email/queue [post]
//	@Description	Create a RabbitMQ queue with DLX.
func (controller *Queue) create(handler *fiber.Ctx) error {
	body := &model.QueuePartial{
		MaxRetries: 10, //nolint:gomnd
	}

	err := handler.BodyParser(body)
	if err != nil {
		return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
	}

	funcCore := func() error { return controller.core.Create(*body) }

	expectErrors := []expectError{{core.ErrQueueAlreadyExist, fiber.StatusConflict}}

	unexpectMessageError := "error creating queue"

	okay := okay{"queue created", fiber.StatusCreated}

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
}

// Get all RabbitMQ queues
//
//	@Summary		Get queues
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}		model.Queue	"all queues"
//	@Failure		500	{object}	sent		"internal server error"
//	@Router			/email/queue [get]
//	@Description	Get all RabbitMQ queues.
func (controller *Queue) getAll(handler *fiber.Ctx) error {
	return callingCoreWithReturn(
		controller.core.GetAll,
		[]expectError{},
		"error getting all queues",
		handler,
	)
}

// Delete a queue with DLX
//
//	@Summary		Delete queues
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	sent	"queue deleted"
//	@Failure		404		{object}	sent	"queue dont exist"
//	@Failure		500		{object}	sent	"internal server error"
//	@Param			name	path		string	true	"queue name"
//	@Router			/email/queue/{name} [delete]
//	@Description	Delete a queue with DLX.
func (controller *Queue) delete(handler *fiber.Ctx) error {
	funcCore := func() error { return controller.core.Delete(handler.Params("name")) }

	expectErrors := []expectError{{core.ErrQueueDoesNotExist, fiber.StatusNotFound}}

	unexpectMessageError := "error deleting queue"

	okay := okay{"queue deleted", fiber.StatusOK}

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
}

// Sends an email to the RabbitMQ queue
//
//	@Summary		Sends email
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	sent		"email sent successfully"
//	@Failure		400		{object}	sent		"an invalid email param was sent"
//	@Failure		404		{object}	sent		"queue does not exist"
//	@Failure		500		{object}	sent		"internal server error"
//	@Param			name	path		string		true	"queue name"
//	@Param			queue	body		model.Email	true	"email"
//	@Router			/email/queue/{name}/send [post]
//	@Description	Sends an email to the RabbitMQ queue.
func (controller *Queue) sendEmail(handler *fiber.Ctx) error {
	body := &model.Email{}

	err := handler.BodyParser(body)
	if err != nil {
		return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
	}

	funcCore := func() error { return controller.core.SendEmail(handler.Params("name"), *body) }

	expectErrors := []expectError{
		{core.ErrQueueDoesNotExist, fiber.StatusNotFound},
		{core.ErrMissingFieldTemplates, fiber.StatusBadRequest},
		{core.ErrTemplateDoesNotExist, fiber.StatusBadRequest},
	}

	unexpectMessageError := "error sending email"

	okay := okay{"email sent", fiber.StatusOK}

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
}

type Template struct {
	core       *core.Template
	translator *ut.UniversalTranslator
	languages  []string
}

func (controller *Template) getTranslator(handler *fiber.Ctx) ut.Translator { //nolint:ireturn
	accept := handler.AcceptsLanguages(controller.languages...)
	if accept == "" {
		accept = controller.languages[0]
	}

	language, _ := controller.translator.GetTranslator(accept)

	return language
}

// Create a email template
//
//	@Summary		Creating template
//	@Tags			template
//	@Accept			json
//	@Produce		json
//	@Success		201			{object}	sent					"create template successfully"
//	@Failure		400			{object}	sent					"an invalid template param was sent"
//	@Failure		409			{object}	sent					"template name already exist"
//	@Failure		500			{object}	sent					"internal server error"
//	@Param			template	body		model.TemplatePartial	true	"template params"
//	@Router			/email/template [post]
//	@Description	Create a email template.
func (controller *Template) create(handler *fiber.Ctx) error {
	body := &model.TemplatePartial{}

	err := handler.BodyParser(body)
	if err != nil {
		return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
	}

	funcCore := func() error { return controller.core.Create(*body) }

	expectErrors := []expectError{
		{core.ErrTemplateNameAlreadyExist, fiber.StatusConflict},
		{core.ErrMaxSizeTemplate, fiber.StatusBadRequest},
	}

	unexpectMessageError := "error creating template"

	okay := okay{"template created", fiber.StatusCreated}

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
}

// Delete all email templates
//
//	@Summary		Get templates
//	@Tags			template
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}		model.Template	"all templates"
//	@Failure		500	{object}	sent			"internal server error"
//	@Router			/email/template [get]
//	@Description	Delete all email templates.
func (controller *Template) getAll(handler *fiber.Ctx) error {
	return callingCoreWithReturn(
		controller.core.GetAll,
		[]expectError{},
		"error getting all templates",
		handler,
	)
}

// Get a email template
//
//	@Summary		Get template
//	@Tags			template
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	model.Template	"all templates"
//	@Success		404		{array}		sent			"template does not exist"
//	@Failure		500		{object}	sent			"internal server error"
//	@Param			name	path		string			true	"template name"
//	@Router			/email/template/{name} [get]
//	@Description	Get a email template.
func (controller *Template) get(handler *fiber.Ctx) error {
	coreFunc := func() (*model.Template, error) { return controller.core.Get(handler.Params("name")) }

	expectErros := []expectError{{core.ErrTemplateDoesNotExist, fiber.StatusNotFound}}

	return callingCoreWithReturn(coreFunc, expectErros, "error getting template", handler)
}

// Update a email template
//
//	@Summary		Update template
//	@Tags			template
//	@Accept			json
//	@Produce		json
//	@Success		200			{object}	sent					"template updated"
//	@Failure		400			{object}	sent					"an invalid template param was sent"
//	@Failure		404			{object}	sent					"template does not exist"
//	@Failure		500			{object}	sent					"internal server error"
//	@Param			name		path		string					true	"template name"
//	@Param			template	body		model.TemplatePartial	true	"template params"
//	@Router			/email/template/{name} [put]
//	@Description	Update a email template.
func (controller *Template) update(handler *fiber.Ctx) error {
	body := &model.TemplatePartial{}

	err := handler.BodyParser(body)
	if err != nil {
		return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
	}

	funcCore := func() error { return controller.core.Update(handler.Params("name"), *body) }

	expectErrors := []expectError{
		{core.ErrTemplateDoesNotExist, fiber.StatusNotFound},
		{core.ErrMaxSizeTemplate, fiber.StatusBadRequest},
	}

	unexpectMessageError := "error updating template"

	okay := okay{"template updated", fiber.StatusOK}

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
}

// Delete a email template
//
//	@Summary		Delete template
//	@Tags			template
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	sent	"template deleted"
//	@Failure		404		{object}	sent	"template does not exist"
//	@Failure		500		{object}	sent	"internal server error"
//	@Param			name	path		string	true	"template name"
//	@Router			/email/template/{name} [delete]
//	@Description	Delete a email template.
func (controller *Template) delete(handler *fiber.Ctx) error {
	funcCore := func() error { return controller.core.Delete(handler.Params("name")) }

	expectErrors := []expectError{{core.ErrTemplateDoesNotExist, fiber.StatusNotFound}}

	unexpectMessageError := "error deleting template"

	okay := okay{"template deleted", fiber.StatusOK}

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
}
