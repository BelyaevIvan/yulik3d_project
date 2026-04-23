// Package docs — stub-файл, чтобы код компилировался до запуска `swag init`.
// После выполнения `make swagger` (или `swag init -g cmd/main.go --output internal/generated/docs`)
// этот файл будет перезаписан реальной спецификацией на основе аннотаций в handler/*.go.
//
// Этот файл в .gitignore — не коммитится.
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
  "swagger": "2.0",
  "info": {
    "title": "yulik3d API",
    "description": "Stub spec. Run `+"`make swagger`"+` to regenerate.",
    "version": "1.0"
  },
  "basePath": "/api/v1",
  "paths": {}
}`

// SwaggerInfo — минимальная метаинформация. swag init перезапишет файл полностью.
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "",
	BasePath:         "/api/v1",
	Schemes:          []string{},
	Title:            "yulik3d API",
	Description:      "Stub spec. Run `make swagger` to regenerate.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
