package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/http/response"
	productservice "mini-store-go/backend/internal/service/product"
	"mini-store-go/backend/internal/validation"
)

type ProductHandler struct {
	validator *validation.Validator
	service   *productservice.Service
}

func NewProductHandler(validator *validation.Validator, service *productservice.Service) *ProductHandler {
	return &ProductHandler{
		validator: validator,
		service:   service,
	}
}

func (h *ProductHandler) List(c *gin.Context) {
	var filter dto.ProductListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		writeBadRequest(c, "invalid query params", err.Error())
		return
	}
	if err := h.validator.Validate(filter); err != nil {
		writeError(c, err)
		return
	}

	products, meta, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		writeError(c, err)
		return
	}

	response.OK(c, toPagedProducts(products, meta))
}

func (h *ProductHandler) Latest(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "6"))
	products, err := h.service.ListLatest(c.Request.Context(), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toProductResponses(products))
}

func (h *ProductHandler) Featured(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "4"))
	products, err := h.service.ListFeatured(c.Request.Context(), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toProductResponses(products))
}

func (h *ProductHandler) Categories(c *gin.Context) {
	items, err := h.service.ListCategories(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *ProductHandler) GetBySlug(c *gin.Context) {
	product, err := h.service.GetBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toProductResponse(product))
}

func (h *ProductHandler) GetByID(c *gin.Context) {
	product, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toProductResponse(product))
}

func (h *ProductHandler) Create(c *gin.Context) {
	var input dto.UpsertProductInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeBadRequest(c, "invalid request body", err.Error())
		return
	}
	if err := h.validator.Validate(input); err != nil {
		writeError(c, err)
		return
	}

	product, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toProductResponse(product))
}

func (h *ProductHandler) Update(c *gin.Context) {
	var input dto.UpsertProductInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeBadRequest(c, "invalid request body", err.Error())
		return
	}
	if err := h.validator.Validate(input); err != nil {
		writeError(c, err)
		return
	}

	product, err := h.service.Update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toProductResponse(product))
}

func (h *ProductHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
