package products

import (
	//"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) CreateProduct(c *gin.Context) {
	var input Product
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	ctx := c.Request.Context()
	id, err := h.repo.InsertProduct(ctx, &input)
	if err != nil {
		log.Printf("CreateProduct: repo.InsertProduct error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert"})
		return
	}
	input.ID = id
	c.JSON(http.StatusCreated, input)
}

func (h *Handler) ListProducts(c *gin.Context) {
	ctx := c.Request.Context()
	list, err := h.repo.GetProducts(ctx)
	if err != nil {
		// подробный лог — отправь мне этот лог, если ошибка останется
		log.Printf("ListProducts: repo.GetProducts error: %T: %v", err, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch products"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) GetProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	p, err := h.repo.GetProductByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		log.Printf("GetProduct: repo.GetProductByID error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch product"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) GetPriceHistory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	hist, err := h.repo.GetPriceHistory(ctx, id)
	if err != nil {
		log.Printf("GetPriceHistory: repo.GetPriceHistory error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch history"})
		return
	}
	c.JSON(http.StatusOK, hist)
}
