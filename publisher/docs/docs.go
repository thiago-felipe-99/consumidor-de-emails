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
                "description": "Getting all RabbitMQ queues.",
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
                "description": "Creating a RabbitMQ queue with DLX.",
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
                    "200": {
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
                "responses": {
                    "204": {
                        "description": "queue deleted",
                        "schema": {
                            "type": "onject"
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
                    "$ref": "#/definitions/model.Template"
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
