package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ivancabzon/task-service/internal/model"
	"github.com/ivancabzon/task-service/internal/service"
	"github.com/ivancabzon/task-service/pkg/response"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register — POST /api/v1/register
//
//	@Summary      Регистрация
//	@Tags         auth
//	@Accept       json
//	@Produce      json
//	@Param        request body model.RegisterRequest true "Данные пользователя"
//	@Success      201 {object} model.User
//	@Failure      400 {object} response.ErrorBody
//	@Failure      409 {object} response.ErrorBody
//	@Router       /register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	user, err := h.auth.Register(c.Request.Context(), req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Created(c, user)
}

// Login — POST /api/v1/login
//
//	@Summary      Аутентификация
//	@Tags         auth
//	@Accept       json
//	@Produce      json
//	@Param        request body model.LoginRequest true "Email + пароль"
//	@Success      200 {object} model.LoginResponse
//	@Failure      400 {object} response.ErrorBody
//	@Failure      401 {object} response.ErrorBody
//	@Router       /login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	resp, err := h.auth.Login(c.Request.Context(), req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.OK(c, resp)
}
