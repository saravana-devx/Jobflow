package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"pulseDashboard/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateUser(c *gin.Context) {
	var body CreateUserRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.MsgInvalidBody)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.svc.CreateUserService(ctx, &body)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			httpx.Error(c, http.StatusConflict, err.Error())
		case errors.Is(err, ErrInvalidEmail):
			httpx.Error(c, http.StatusUnauthorized, err.Error())
		case errors.As(err, new(*WeakPasswordError)):
			httpx.Error(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrHashingPassword), errors.Is(err, ErrToCreateUser):
			httpx.Error(c, http.StatusInternalServerError, err.Error())
		default:
			httpx.Error(c, http.StatusInternalServerError, httpx.MsgInternalError)
		}
		return
	}

	httpx.Success(c, http.StatusCreated, "user created", result)
}

func (h *Handler) LoginUser(c *gin.Context) {
	var body LoginRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.MsgInvalidBody)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	loginResult, err := h.svc.LoginUserService(ctx, &body)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrWrongPassword), errors.Is(err, ErrNotFound):
			httpx.Error(c, http.StatusUnauthorized, err.Error())
		default:
			httpx.Error(c, http.StatusInternalServerError, httpx.MsgInternalError)
		}
		return
	}

	httpx.Success(c, http.StatusOK, "login successful", loginResult)
}

func (h *Handler) RefreshAccessToken(c *gin.Context) {
	refreshToken, err := ExtractBearerToken(c.GetHeader("Authorization"))
	if err != nil {
		httpx.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	tokens, err := h.svc.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRefreshToken),
			errors.Is(err, ErrRefreshTokenReused),
			errors.Is(err, ErrUserNotFound):
			httpx.Error(c, http.StatusUnauthorized, err.Error())
		default:
			httpx.Error(c, http.StatusInternalServerError, httpx.MsgInternalError)
		}
		return
	}

	httpx.Success(c, http.StatusOK, "token refreshed", tokens)
}

func (h *Handler) Logout(c *gin.Context) {
	var body LogoutRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.MsgInvalidBody)
		return
	}

	jti, ok := JTIFromContext(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "missing token context")
		return
	}
	exp, _ := ExpFromContext(c)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.svc.LogoutService(ctx, jti, exp, body.RefreshToken); err != nil {
		switch {
		case errors.Is(err, ErrInvalidRefreshToken),
			errors.Is(err, ErrRefreshTokenReused),
			errors.Is(err, ErrUserNotFound):
			httpx.Error(c, http.StatusUnauthorized, err.Error())
		default:
			httpx.Error(c, http.StatusInternalServerError, httpx.MsgInternalError)
		}
		return
	}

	httpx.Success(c, http.StatusOK, "logout successful", nil)
}

func ExtractBearerToken(header string) (string, error) {
	if header == "" {
		return "", errors.New("missing authorization header")
	}
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("authorization header must be in form: Bearer <token>")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("bearer token is empty")
	}
	return token, nil
}
