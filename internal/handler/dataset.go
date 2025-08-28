package handler

import (
	"awesomeEval/internal/codes"
	"awesomeEval/internal/models"
	"awesomeEval/internal/service/dataset"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type DatasetHandler struct {
	DatasetService *dataset.Service
}

func NewDatasetHandler(service *dataset.Service) *DatasetHandler {
	return &DatasetHandler{
		DatasetService: service,
	}
}

func (h *DatasetHandler) CreateDataset(c *gin.Context) {
	uid, ok := c.Get("user_id")
	if !ok {
		c.JSON(401, gin.H{"code": codes.CommonUnauthorizedCode, "message": "user_id missing"})
		return
	}
	var userID int
	switch v := uid.(type) {
	case int:
		userID = v
	case int64:
		userID = int(v)
	case string:
		id, err := strconv.Atoi(v)
		if err != nil {
			c.JSON(500, gin.H{"code": codes.CommonInternalErrorCode, "message": "user_id type error"})
			return
		}
		userID = id
	default:
		userIdStr := fmt.Sprintf("%v", uid)
		id, err := strconv.Atoi(userIdStr)
		if err != nil {
			c.JSON(500, gin.H{"code": codes.CommonInternalErrorCode, "message": "user_id type error"})
			return
		}
		userID = id
	}

	var createDatasetRequest models.CreateDatasetRequest
	createDatasetRequest.Name = c.PostForm("name")
	createDatasetRequest.Description = c.PostForm("description")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"code": codes.CommonInvalidParamCode, "message": "file missing"})
		return
	}
	defer file.Close()

	if err := h.DatasetService.CreateDataset(file, header, &createDatasetRequest, int64(userID), c.Request.Context()); err != nil {
		c.JSON(500, gin.H{"code": codes.CommonInternalErrorCode, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Dataset created successfully"})
}

func (h *DatasetHandler) ListDatasets(c *gin.Context) {
	// obtain userID from token
	userId, _ := c.Get("user_id")
	user := userId.(string)
	userID, err := strconv.Atoi(user)
	if err != nil {
		c.JSON(400, codes.CommonInvalidParamCode)
		return
	}
	datasets, err := h.DatasetService.ListDatasets(int64(userID), c.Request.Context())
	if err != nil {
		c.JSON(500, codes.CommonInternalErrorCode)
		return
	}
	c.JSON(200, gin.H{"datasets": datasets})
}

func (h *DatasetHandler) ListDatasetItems(c *gin.Context) {
	// obtain dataset_id from query
	datasetID, err := strconv.Atoi(c.Param("dataset_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, codes.CommonInvalidParamCode)
	}
	// obtain user_id from context
	user, _ := c.Get("user_id")
	userIdStr := user.(string)
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, codes.CommonInvalidParamCode)
	}
	// obtain pageNum and pageSize
	pageNumStr := c.Param("page")
	pageSizeStr := c.Param("page_size")
	pageNum_, err := strconv.Atoi(pageNumStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, codes.CommonInvalidParamCode)
		return
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, codes.CommonInvalidParamCode)
		return
	}
	result, err := h.DatasetService.ListDatasetItems(datasetID, int64(userId), pageNum_, pageSize, c.Request.Context())
	if err != nil {
		c.JSON(500, codes.CommonInternalErrorCode)
		return
	}
	c.JSON(http.StatusOK, gin.H{"datasetItems": result})
}
