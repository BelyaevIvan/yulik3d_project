// Package migrations предоставляет встроенные SQL-миграции для goose.
// Файлы .sql в этой папке автоматически попадают в бинарник.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
