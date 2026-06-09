package handler

import (
	"github.com/gin-gonic/gin"

	"mini-store-go/backend/internal/http/response"
	uploadservice "mini-store-go/backend/internal/service/upload"
)

type UploadHandler struct {
	service *uploadservice.Service
}

func NewUploadHandler(service *uploadservice.Service) *UploadHandler {
	return &UploadHandler{service: service}
}

func (h *UploadHandler) UploadImage(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		writeBadRequest(c, "missing upload file", err.Error())
		return
	}

	file, err := uploadservice.FileFromHeader(fileHeader)
	if err != nil {
		writeError(c, err)
		return
	}

	uploaded, err := h.service.SaveImage(c.Request.Context(), file)
	if err != nil {
		writeError(c, err)
		return
	}

	response.OK(c, uploaded)
}
