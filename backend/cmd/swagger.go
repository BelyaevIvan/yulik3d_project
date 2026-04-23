package main

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	"yulik3d/config"
	// Импорт генерируется командой `swag init -g cmd/main.go --output internal/generated/docs`.
	// До первого запуска `swag init` используется stub (см. файл), чтобы код компилировался.
	_ "yulik3d/internal/generated/docs"
)

// mountSwagger — подключает Swagger UI на /swagger/*.
// Работает с session cookie автоматически: когда пользователь выполняет
// /api/v1/auth/login через Swagger UI, браузер сам сохраняет cookie (same-origin)
// и шлёт её во все последующие запросы — отдельная кнопка Authorize не нужна.
func mountSwagger(mux *http.ServeMux, cfg config.Config) {
	swaggerUI := httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.PersistAuthorization(true),
	)

	// /swagger и /swagger/ редиректим на /swagger/index.html — httpSwagger
	// сам index.html без имени файла не отдаёт.
	redirect := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusFound)
	}
	mux.HandleFunc("GET /swagger", redirect)

	// Для /swagger/* — сначала проверяем «пустой» путь, остальное отдаём UI.
	mux.HandleFunc("GET /swagger/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/swagger/" || r.URL.Path == "/swagger" {
			redirect(w, r)
			return
		}
		swaggerUI(w, r)
	})
}
