package jobs

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"pulseDashboard/internal/auth"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateJob(c *gin.Context) {
	var body CreateJobRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	userID, ok := auth.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	body.UserID = userID

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.svc.CreateJobService(ctx, &body)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidJobInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": ErrToCreateJob.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "job created",
		"data":    result,
	})
}

func (h *Handler) CreateJobs(c *gin.Context) {
	var body []CreateJobRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	userID, ok := auth.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	// body.UserID = userID

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.svc.CreateJobsService(ctx, &body, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidJobInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": ErrToCreateJob.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "multiple job created",
		"data":    result,
	})
}

func (h *Handler) GetJobById(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job id is required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.svc.GetJobByIdService(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrJobNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": ErrJobNotFound.Error()})
		case errors.Is(err, ErrInvalidJobInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": ErrToGetJob.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "job fetched",
		"data":    result,
	})
}

func (h *Handler) GetAllJobs(c *gin.Context) {
}

func (h *Handler) UpdateJob(c *gin.Context) {
}

func (h *Handler) DeleteJob(c *gin.Context) {
}
