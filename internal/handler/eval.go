package handler

import (
	"awesomeEval/internal/codes"
	"awesomeEval/internal/models"
	"awesomeEval/internal/service/eval"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type EvalHandler struct {
	EvalService *eval.Service
}

func NewEvalHandler(evalService *eval.Service) *EvalHandler {
	return &EvalHandler{EvalService: evalService}
}

func (h *EvalHandler) CreateBatchEvalJob(c *gin.Context) {
	// obtain user_id
	var userID int
	uid, _ := c.Get("user_id")
	switch uid.(type) {
	case int:
		userID = uid.(int)
	case int64:
		userID = int(uid.(int64))
	case string:
		id, err := strconv.Atoi(uid.(string))
		if err != nil {
			c.JSON(500, codes.CommonInternalErrorCode)
			return
		}
		userID = id
	default:
		userIdStr := fmt.Sprintf("%v", uid)
		id, err := strconv.Atoi(userIdStr)
		if err != nil {
			c.JSON(500, codes.CommonInternalErrorCode)
			return
		}
		userID = id
	}
	var evalRequest models.EvalBatchTaskRequest
	if err := c.ShouldBindBodyWithJSON(&evalRequest); err != nil {
		c.JSON(500, codes.CommonInternalErrorCode)
	}
	err := h.EvalService.CreateBatchEvalJob(c.Request.Context(), userID, &evalRequest)
	if err != nil {
		c.JSON(500, codes.CommonInternalErrorCode)
	}
	c.JSON(200, gin.H{"message": "eval batch job created successfully!"})
}

func (h *EvalHandler) PreviewEval(c *gin.Context) {
	// stream
	// obtain user_id
	var userID int
	uid, _ := c.Get("user_id")
	switch uid.(type) {
	case int:
		userID = uid.(int)
	case int64:
		userID = int(uid.(int64))
	case string:
		id, err := strconv.Atoi(uid.(string))
		if err != nil {
			c.JSON(500, codes.CommonInternalErrorCode)
			return
		}
		userID = id
	default:
		userIdStr := fmt.Sprintf("%v", uid)
		id, err := strconv.Atoi(userIdStr)
		if err != nil {
			c.JSON(500, codes.CommonInternalErrorCode)
			return
		}
		userID = id
	}
	if userID <= 0 {

	}
	var evalRequest models.EvalBatchTaskRequest
	if err := c.ShouldBindBodyWithJSON(&evalRequest); err != nil {
		c.JSON(http.StatusBadRequest, codes.CommonInvalidParamCode)
	}
	if &evalRequest == nil || evalRequest.DatasetItem > 5 {
		c.JSON(500, codes.CommonInvalidParamCode)
	}
	c.Writer.Header().Set("Content-Type", "event/stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	h.EvalService.PreviewEval(c.Writer, c.Request.Context(), &evalRequest)
}
