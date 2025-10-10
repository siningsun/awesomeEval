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
	userId, err := h.getUserId(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, codes.UserNotFoundCode)
	}
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, codes.UserNotFoundCode)
	}
	var evalRequest models.EvalBatchTaskRequest
	if err := c.ShouldBindBodyWithJSON(&evalRequest); err != nil {
		c.JSON(500, codes.CommonInternalErrorCode)
	}
	err = h.EvalService.CreateBatchEvalJob(c.Request.Context(), userId, &evalRequest)
	if err != nil {
		c.JSON(500, codes.CommonInternalErrorCode)
	}
	c.JSON(200, gin.H{"message": "eval batch job created successfully!"})
}

func (h *EvalHandler) PreviewEval(c *gin.Context) {
	userId, err := h.getUserId(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, codes.UserNotFoundCode)
	}
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, codes.CommonUnauthorizedCode)
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

func (h *EvalHandler) getUserId(c *gin.Context) (int, error) {
	var userId int
	uid, _ := c.Get("user_id")
	switch uid.(type) {
	case string:
		id, err := strconv.Atoi(uid.(string))
		if err != nil {
			return 0, err
		}
		userId = id
	case int:
		userId = uid.(int)
	case int64:
		userId = int(uid.(int64))
	default:
		userIdStr := fmt.Sprintf("%v", uid)
		id, err := strconv.Atoi(userIdStr)
		if err != nil {
			return 0, err
		}
		userId = id
	}
	return userId, nil
}
