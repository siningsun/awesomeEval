package handler

import (
	"awesomeEval/internal/codes"
	"awesomeEval/internal/models"
	"awesomeEval/internal/service/eval"
	"fmt"
	"github.com/gin-gonic/gin"
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

}
