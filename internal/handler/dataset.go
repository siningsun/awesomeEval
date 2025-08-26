package handler

import (
	"awesomeEval/internal/codes"
	"awesomeEval/internal/models"
	"awesomeEval/internal/service/dataset"
	"github.com/gin-gonic/gin"
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
	// obtain userID from token
	user := c.GetString("userID")
	userID, err := strconv.Atoi(user)
	if err != nil {
		c.JSON(400, codes.CommonInvalidParamCode)
		return
	}
	var createDatasetRequest models.CreateDatasetRequest
	if err := c.ShouldBindJSON(&createDatasetRequest); err != nil {
		c.JSON(400, codes.CommonInvalidParamCode)
		return
	}
	// obtain file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, codes.CommonInvalidParamCode)
		return
	}
	defer file.Close()
	err = h.DatasetService.CreateDataset(file, header, &createDatasetRequest, int64(userID), c.Request.Context())
	if err != nil {
		c.JSON(500, codes.CommonInternalErrorCode)
		return
	}
	c.JSON(200, gin.H{"message": "Dataset created successfully"})
}
