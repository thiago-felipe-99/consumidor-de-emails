package controllers

import (
	"log"

	ut "github.com/go-playground/universal-translator"
	"github.com/gofiber/fiber/v2"
	"github.com/thiago-felipe-99/mail/publisher/core"
	"github.com/thiago-felipe-99/mail/publisher/model"
)

type EmailList struct {
	core       *core.EmailList
	translator *ut.UniversalTranslator
	languages  []string
}

func (controller *EmailList) getTranslator(handler *fiber.Ctx) ut.Translator { //nolint:ireturn
	accept := handler.AcceptsLanguages(controller.languages...)
	if accept == "" {
		accept = controller.languages[0]
	}

	language, _ := controller.translator.GetTranslator(accept)

	return language
}

// Create a email list to user.
//
//	@Summary		Creating email list
//	@Tags			emailList
//	@Accept			json
//	@Produce		json
//	@Success		201			{object}	sent					"create email list successfully"
//	@Failure		400			{object}	sent					"an invalid email list param was sent"
//	@Failure		401			{object}	sent					"user session has expired"
//	@Failure		409			{object}	sent					"email list already exist"
//	@Failure		500			{object}	sent					"internal server error"
//	@Param			emailList	body		model.EmailListPartial	true	"email list params"
//	@Router			/email/list [post]
//	@Description	Create a email list to user.
func (controller *EmailList) create(handler *fiber.Ctx) error {
	userID, ok := handler.Locals("userID").(model.ID)
	if !ok {
		log.Printf("[ERROR] - error getting user ID")

		return handler.Status(fiber.StatusInternalServerError).
			JSON(sent{"error refreshing session"})
	}

	body := &model.EmailListPartial{}

	err := handler.BodyParser(body)
	if err != nil {
		return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
	}

	funcCore := func() error { return controller.core.Create(userID, *body) }

	expectErrors := []expectError{{core.ErrEmailListAlreadyExist, fiber.StatusConflict}}

	unexpectMessageError := "error creating email list"

	okay := okay{"email list created", fiber.StatusCreated}

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
}
