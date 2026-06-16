package jobs

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"jobflow/internal/auth"
	"jobflow/internal/httpx"
)

// msgJobIDRequired is jobs-specific, so it stays in this package rather than
// in the shared httpx message set.
const msgJobIDRequired = "job id is required"

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateJob(c *gin.Context) {
	var body CreateJobRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.MsgInvalidBody)
		return
	}

	userID, ok := auth.UserIDFromContext(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, httpx.MsgUnauthorized)
		return
	}

	body.UserID = userID

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.svc.CreateJobService(ctx, &body)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidJobInput):
			httpx.Error(c, http.StatusBadRequest, err.Error())
		default:
			httpx.Error(c, http.StatusInternalServerError, ErrToCreateJob.Error())
		}
		return
	}

	httpx.Success(c, http.StatusOK, "job created", result)
}

func (h *Handler) CreateJobs(c *gin.Context) {
	var body []CreateJobRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.MsgInvalidBody)
		return
	}

	userID, ok := auth.UserIDFromContext(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, httpx.MsgUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.svc.CreateJobsService(ctx, &body, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidJobInput):
			httpx.Error(c, http.StatusBadRequest, err.Error())
		default:
			httpx.Error(c, http.StatusInternalServerError, ErrToCreateJobs.Error())
		}
		return
	}

	httpx.Success(c, http.StatusOK, "jobs created", result)
}

func (h *Handler) GetJobById(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		httpx.Error(c, http.StatusBadRequest, msgJobIDRequired)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.svc.GetJobByIdService(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrJobNotFound):
			httpx.Error(c, http.StatusNotFound, ErrJobNotFound.Error())
		case errors.Is(err, ErrInvalidJobInput):
			httpx.Error(c, http.StatusBadRequest, err.Error())
		default:
			httpx.Error(c, http.StatusInternalServerError, ErrToGetJob.Error())
		}
		return
	}

	httpx.Success(c, http.StatusOK, "job fetched", result)
}

func (h *Handler) GetAllJobs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	userID, ok := auth.UserIDFromContext(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, httpx.MsgUnauthorized)
		return
	}

	result, err := h.svc.GetAllJobsService(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrJobNotFound):
			httpx.Error(c, http.StatusNotFound, ErrJobNotFound.Error())
		default:
			httpx.Error(c, http.StatusInternalServerError, ErrToGetAllJobs.Error())
		}
		return
	}

	httpx.Success(c, http.StatusOK, "jobs fetched", result)
}

func (h *Handler) UpdateJob(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		httpx.Error(c, http.StatusBadRequest, msgJobIDRequired)
		return
	}

	var body UpdateJobRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.MsgInvalidBody)
		return
	}

	userID, ok := auth.UserIDFromContext(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, httpx.MsgUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.svc.UpdateJobService(ctx, id, userID, &body)
	if err != nil {
		switch {
		case errors.Is(err, ErrJobNotFound):
			httpx.Error(c, http.StatusNotFound, ErrJobNotFound.Error())
		case errors.Is(err, ErrInvalidJobInput):
			httpx.Error(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrUnauthorized):
			httpx.Error(c, http.StatusForbidden, ErrUnauthorized.Error())
		default:
			httpx.Error(c, http.StatusInternalServerError, ErrToUpdateJob.Error())
		}
		return
	}

	httpx.Success(c, http.StatusOK, "job updated", result)
}

func (h *Handler) DeleteJob(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		httpx.Error(c, http.StatusBadRequest, msgJobIDRequired)
		return
	}

	userID, ok := auth.UserIDFromContext(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, httpx.MsgUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err := h.svc.DeleteJobService(ctx, id, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrJobNotFound):
			httpx.Error(c, http.StatusNotFound, ErrJobNotFound.Error())
		case errors.Is(err, ErrUnauthorized):
			httpx.Error(c, http.StatusForbidden, ErrUnauthorized.Error())
		default:
			httpx.Error(c, http.StatusInternalServerError, ErrToDeleteJob.Error())
		}
		return
	}

	httpx.Success(c, http.StatusOK, "job deleted", nil)
}
