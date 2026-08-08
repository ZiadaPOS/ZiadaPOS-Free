package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Pagination meta block attached to list endpoints.
type Pagination struct {
	TotalCount int64 `json:"total_count"`
	Page       int   `json:"current_page"`
	PageSize   int   `json:"page_size"`
	TotalPages int64 `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// Envelope is the standard response body for every endpoint.
type Envelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
	Meta    *Pagination `json:"meta,omitempty"`
}

func write(c *gin.Context, status int, body Envelope) {
	c.JSON(status, body)
}

// OK returns a 200 with data and a default "OK" message.
func OK(c *gin.Context, data interface{}) {
	write(c, http.StatusOK, Envelope{Success: true, Message: "OK", Data: data})
}

// Message returns a 200 with just a message.
func Message(c *gin.Context, status int, message string) {
	write(c, status, Envelope{Success: status < 400, Message: message})
}

// Created returns a 201 with data.
func Created(c *gin.Context, data interface{}, message string) {
	if message == "" {
		message = "Created"
	}
	write(c, http.StatusCreated, Envelope{Success: true, Message: message, Data: data})
}

// NoContent returns a 204 with an empty body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// List returns a 200 with data plus a pagination meta block.
func List(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	totalPages := int64(0)
	if pageSize > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}
	write(c, http.StatusOK, Envelope{
		Success: true,
		Message: "OK",
		Data:    data,
		Meta: &Pagination{
			TotalCount: total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
			HasNext:    page < int(totalPages),
			HasPrev:    page > 1,
		},
	})
}

// Error writes an error envelope. errors may be a string, a map of
// field -> messages, or any other value.
func Error(c *gin.Context, status int, message string, errs interface{}) {
	write(c, status, Envelope{Success: false, Message: message, Errors: errs})
}

// BadRequest is a shorthand for a 400 validation error.
func BadRequest(c *gin.Context, message string, errs interface{}) {
	Error(c, http.StatusBadRequest, message, errs)
}

// Unauthorized is a shorthand for a 401.
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message, nil)
}

// Forbidden is a shorthand for a 403.
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message, nil)
}

// NotFound is a shorthand for a 404.
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "Not found"
	}
	Error(c, http.StatusNotFound, message, nil)
}

// Conflict is a shorthand for a 409.
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, message, nil)
}

// Internal is a shorthand for a 500.
func Internal(c *gin.Context, message string) {
	if message == "" {
		message = "Internal server error"
	}
	Error(c, http.StatusInternalServerError, message, nil)
}
