package response

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ivancabzon/task-service/pkg/apperr"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": ErrorBody{Code: code, Message: message}})
}

func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		status := statusFromSentinel(appErr.Err)
		c.AbortWithStatusJSON(status, gin.H{
			"error": ErrorBody{Code: appErr.Code, Message: appErr.Message},
		})
		return
	}

	status := statusFromSentinel(err)
	code, msg := codeFromStatus(status)

	if status >= 500 {
		log.Printf("[ERROR] internal: %v", err)
	}
	c.AbortWithStatusJSON(status, gin.H{
		"error": ErrorBody{Code: code, Message: msg},
	})
}

func statusFromSentinel(err error) int {
	switch {
	case errors.Is(err, apperr.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, apperr.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, apperr.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, apperr.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, apperr.ErrValidation):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func codeFromStatus(status int) (string, string) {
	switch status {
	case http.StatusBadRequest:
		return "bad_request", "bad request"
	case http.StatusUnauthorized:
		return "unauthorized", "unauthorized"
	case http.StatusForbidden:
		return "forbidden", "forbidden"
	case http.StatusNotFound:
		return "not_found", "resource not found"
	case http.StatusConflict:
		return "conflict", "conflict"
	default:
		return "internal", "internal server error"
	}
}
