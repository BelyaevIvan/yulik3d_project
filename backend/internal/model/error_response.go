package model

// ErrorResponse — единый формат тела ошибки для всех эндпоинтов.
// См. spec.md §1.4.
type ErrorResponse struct {
	StatusCode int    `json:"statusCode" example:"400"`
	URL        string `json:"url" example:"/api/v1/items/018f7d3e-4a5b-7c9d-a0b1-c2d3e4f5a6b7"`
	Message    string `json:"message" example:"Item not found"`
	Date       string `json:"date" example:"2026-04-23T18:42:00Z"`
}
