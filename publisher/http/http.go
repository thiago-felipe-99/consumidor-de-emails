//nolint:wrapcheck
package http

import (
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/pt"
	"github.com/go-playground/locales/pt_BR"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	ptTranslations "github.com/go-playground/validator/v10/translations/pt"
	pt_br_translations "github.com/go-playground/validator/v10/translations/pt_BR"
	"github.com/gofiber/fiber/v2"
	"github.com/thiago-felipe-99/mail/rabbit"
)

func createTranslator(validate *validator.Validate) (*ut.UniversalTranslator, error) {
	translator := ut.New(en.New(), pt.New(), pt_BR.New())

	enTrans, _ := translator.GetTranslator("en")

	err := en_translations.RegisterDefaultTranslations(validate, enTrans)
	if err != nil {
		return nil, err
	}

	ptTrans, _ := translator.GetTranslator("pt")

	err = ptTranslations.RegisterDefaultTranslations(validate, ptTrans)
	if err != nil {
		return nil, err
	}

	ptBRTrans, _ := translator.GetTranslator("pt_BR")

	err = pt_br_translations.RegisterDefaultTranslations(validate, ptBRTrans)
	if err != nil {
		return nil, err
	}

	return translator, nil
}

func CreateServer(rabbit *rabbit.Rabbit, queues *rabbit.Queues) (*fiber.App, error) {
	app := fiber.New()

	validate := validator.New()

	translator, err := createTranslator(validate)
	if err != nil {
		return nil, err
	}

	queue := queue{
		rabbit:     rabbit,
		queues:     queues,
		validate:   validate,
		translator: translator,
		languages:  []string{"en", "pt_BR", "pt"},
	}

	app.Post("/email/queue", queue.create())
	app.Post("/email/queue/:name/add", queue.send())

	return app, nil
}
