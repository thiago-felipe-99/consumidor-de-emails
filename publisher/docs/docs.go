// Code generated by swaggo/swag. DO NOT EDIT.

package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/email/queue": {
            "get": {
                "description": "Get all RabbitMQ queues.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "queue"
                ],
                "summary": "Get queues",
                "responses": {
                    "200": {
                        "description": "all queues",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/model.Queue"
                            }
                        }
                    },
                    "500": {
                        "description": "internal server error",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    }
                }
            },
            "post": {
                "description": "Create a RabbitMQ queue with DLX.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "queue"
                ],
                "summary": "Creating queue",
                "parameters": [
                    {
                        "description": "queue params",
                        "name": "queue",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/model.QueuePartial"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "create queue successfully",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "400": {
                        "description": "an invalid queue param was sent",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "409": {
                        "description": "queue already exist",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "500": {
                        "description": "internal server error",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    }
                }
            }
        },
        "/email/queue/{name}": {
            "delete": {
                "description": "Delete a queue with DLX.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "queue"
                ],
                "summary": "Delete queues",
                "parameters": [
                    {
                        "type": "string",
                        "description": "queue name",
                        "name": "name",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "queue deleted",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "404": {
                        "description": "queue dont exist",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "500": {
                        "description": "internal server error",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    }
                }
            }
        },
        "/email/queue/{name}/send": {
            "post": {
                "description": "Sends an email to the RabbitMQ queue.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "queue"
                ],
                "summary": "Sends email",
                "parameters": [
                    {
                        "type": "string",
                        "description": "queue name",
                        "name": "name",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "email",
                        "name": "queue",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/model.Email"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "email sent successfully",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "400": {
                        "description": "an invalid email param was sent",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "404": {
                        "description": "queue does not exist",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "500": {
                        "description": "internal server error",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    }
                }
            }
        },
        "/email/template": {
            "get": {
                "description": "Delete all email templates.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "template"
                ],
                "summary": "Get templates",
                "responses": {
                    "200": {
                        "description": "all templates",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/model.Template"
                            }
                        }
                    },
                    "500": {
                        "description": "internal server error",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    }
                }
            },
            "post": {
                "description": "Create a email template.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "template"
                ],
                "summary": "Creating template",
                "parameters": [
                    {
                        "description": "template params",
                        "name": "template",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/model.TemplatePartial"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "create template successfully",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "400": {
                        "description": "an invalid template param was sent",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "409": {
                        "description": "template name already exist",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "500": {
                        "description": "internal server error",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    }
                }
            }
        },
        "/email/template/{name}": {
            "get": {
                "description": "Get a email template.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "template"
                ],
                "summary": "Get template",
                "parameters": [
                    {
                        "type": "string",
                        "description": "template name",
                        "name": "name",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "all templates",
                        "schema": {
                            "$ref": "#/definitions/model.Template"
                        }
                    },
                    "404": {
                        "description": "template does not exist",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/controllers.sent"
                            }
                        }
                    },
                    "500": {
                        "description": "internal server error",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    }
                }
            },
            "put": {
                "description": "Update a email template.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "template"
                ],
                "summary": "Update template",
                "parameters": [
                    {
                        "type": "string",
                        "description": "template name",
                        "name": "name",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "template params",
                        "name": "template",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/model.TemplatePartial"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "template updated",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "400": {
                        "description": "an invalid template param was sent",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "404": {
                        "description": "template does not exist",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "500": {
                        "description": "internal server error",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    }
                }
            },
            "delete": {
                "description": "Delete a email template.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "template"
                ],
                "summary": "Delete template",
                "parameters": [
                    {
                        "type": "string",
                        "description": "template name",
                        "name": "name",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "template deleted",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "404": {
                        "description": "template does not exist",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "500": {
                        "description": "internal server error",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    }
                }
            }
        },
        "/user": {
            "post": {
                "description": "Create a user in application.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "user"
                ],
                "summary": "Create user",
                "parameters": [
                    {
                        "description": "user params",
                        "name": "queue",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/model.User"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "user created successfully",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "400": {
                        "description": "an invalid user param was sent",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "409": {
                        "description": "user already exist",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "500": {
                        "description": "internal server error",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    }
                }
            }
        },
        "/user/session": {
            "put": {
                "description": "Refresh a user session and set in the response cookie.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "user"
                ],
                "summary": "Refresh session",
                "responses": {
                    "200": {
                        "description": "session refreshed successfully",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "401": {
                        "description": "session does not exist",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "500": {
                        "description": "internal server error",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    }
                }
            },
            "post": {
                "description": "Create a user session and set in the response cookie.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "user"
                ],
                "summary": "Create session",
                "parameters": [
                    {
                        "description": "user params",
                        "name": "queue",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/model.UserPartial"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "session created successfully",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "400": {
                        "description": "an invalid user param was sent",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "404": {
                        "description": "user does not exist",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    },
                    "500": {
                        "description": "internal server error",
                        "schema": {
                            "$ref": "#/definitions/controllers.sent"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "controllers.sent": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string"
                }
            }
        },
        "model.Email": {
            "type": "object",
            "required": [
                "subject"
            ],
            "properties": {
                "attachments": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "blindReceivers": {
                    "type": "array",
                    "minItems": 1,
                    "items": {
                        "$ref": "#/definitions/model.Receiver"
                    }
                },
                "message": {
                    "type": "string"
                },
                "receivers": {
                    "type": "array",
                    "minItems": 1,
                    "items": {
                        "$ref": "#/definitions/model.Receiver"
                    }
                },
                "subject": {
                    "type": "string"
                },
                "template": {
                    "$ref": "#/definitions/model.TemplateData"
                }
            }
        },
        "model.Queue": {
            "type": "object",
            "properties": {
                "createdAt": {
                    "type": "string"
                },
                "dlx": {
                    "type": "string"
                },
                "maxRetries": {
                    "type": "integer"
                },
                "name": {
                    "type": "string"
                }
            }
        },
        "model.QueuePartial": {
            "type": "object",
            "required": [
                "name"
            ],
            "properties": {
                "maxRetries": {
                    "type": "integer"
                },
                "name": {
                    "type": "string"
                }
            }
        },
        "model.Receiver": {
            "type": "object",
            "required": [
                "email",
                "name"
            ],
            "properties": {
                "email": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                }
            }
        },
        "model.Template": {
            "type": "object",
            "properties": {
                "fields": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "name": {
                    "type": "string"
                },
                "template": {
                    "type": "string"
                }
            }
        },
        "model.TemplateData": {
            "type": "object",
            "required": [
                "name"
            ],
            "properties": {
                "data": {
                    "type": "object",
                    "additionalProperties": {
                        "type": "string"
                    }
                },
                "name": {
                    "type": "string"
                }
            }
        },
        "model.TemplatePartial": {
            "type": "object",
            "required": [
                "name",
                "template"
            ],
            "properties": {
                "name": {
                    "type": "string"
                },
                "template": {
                    "type": "string"
                }
            }
        },
        "model.User": {
            "type": "object",
            "required": [
                "email",
                "name",
                "password"
            ],
            "properties": {
                "email": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                },
                "password": {
                    "type": "string"
                }
            }
        },
        "model.UserPartial": {
            "type": "object",
            "required": [
                "password"
            ],
            "properties": {
                "email": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                },
                "password": {
                    "type": "string"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8080",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "Publisher Emails",
	Description:      "This is an api that publishes emails in RabbitMQ.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
