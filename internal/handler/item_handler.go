package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kiarashAlizadeh/herotech/internal/dto"
	"github.com/kiarashAlizadeh/herotech/internal/service"
)

type ItemHandler struct {
	itemService service.ItemService
}

func NewItemHandler(s service.ItemService) *ItemHandler {
	return &ItemHandler{itemService: s}
}

// CreateItem godoc
// @Summary      Mint a new inventory item
// @Description  Mints a dynamic asset directly into the guild's private inventory vault ('sold' state initially)
// @Tags         items
// @Accept       json
// @Produce      json
// @Param        X-Guild-ID  header    string                 true  "Owner Guild UUID"
// @Param        request     body      dto.CreateItemRequest  true  "Item metadata specs"
// @Success      201  {object}  dto.ItemResponse
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /items [post]
func (h *ItemHandler) CreateItem(c *gin.Context) {
	ownerStr := c.GetHeader("X-Guild-ID")
	ownerID, err := uuid.Parse(ownerStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or malformed X-Guild-ID context header"})
		return
	}

	var req dto.CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type is required"})
		return
	}

	res, err := h.itemService.CreateItem(c.Request.Context(), ownerID, req)
	if err != nil {
		if errors.Is(err, service.ErrBlankItemName) ||
			errors.Is(err, service.ErrInvalidItemNameLength) ||
			errors.Is(err, service.ErrInvalidEntityIDs) ||
			errors.Is(err, service.ErrInvalidItemType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mint asset into guild inventory"})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// ListForSale godoc
// @Summary      List an item on the marketplace
// @Description  Exposes a non-legendary inventory item to the public market catalog with a fixed price
// @Tags         items
// @Accept       json
// @Produce      json
// @Param        X-Guild-ID  header    string  true  "Owner Guild UUID"
// @Param        id          path      string  true  "Item UUID"
// @Param        request     body      dto.ListItemRequest  true  "Market listing pricing" // 🛠️ FIX: Now points to dto package
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /items/{id}/list [post]
func (h *ItemHandler) ListForSale(c *gin.Context) {
	ownerStr := c.GetHeader("X-Guild-ID")
	ownerID, err := uuid.Parse(ownerStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or malformed X-Guild-ID context header"})
		return
	}

	idStr := c.Param("id")
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item identifier format"})
		return
	}

	var req dto.ListItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ListPrice <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "list_price must be greater than zero"})
		return
	}

	err = h.itemService.ListNonLegendaryItem(c.Request.Context(), itemID, ownerID, req.ListPrice)
	if err != nil {
		if errors.Is(err, service.ErrItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrInvalidEntityIDs) ||
			errors.Is(err, service.ErrInvalidListPrice) ||
			errors.Is(err, service.ErrNotItemOwner) ||
			errors.Is(err, service.ErrItemNotAvailable) ||
			errors.Is(err, service.ErrLegendaryPriceAllowed) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list asset into public marketplace"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "item successfully exposed to the marketplace storefront"})
}

// GetItem godoc
// @Summary      Get item profile details
// @Description  Retrieves individual asset evaluation logs by its primary identifier key
// @Tags         items
// @Produce      json
// @Param        id    path      string  true  "Item UUID"
// @Success      200  {object}  dto.ItemResponse
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /items/{id} [get]
func (h *ItemHandler) GetItem(c *gin.Context) {
	idStr := c.Param("id")
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item identifier format"})
		return
	}

	res, err := h.itemService.GetItem(c.Request.Context(), itemID)
	if err != nil {
		if errors.Is(err, service.ErrItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrInvalidEntityIDs) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal look up error"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// ListAvailable godoc
// @Summary      List all available catalog items
// @Description  Fetches currently unpurchased inventory orders with optional filtering by class tier type
// @Tags         items
// @Produce      json
// @Param        type       query     string  false  "Filter by type (common, rare, legendary)"
// @Param        page       query     int     false  "Page number"
// @Param        page_size  query     int     false  "Page size"
// @Success      200  {object}  dto.PaginatedResponse[dto.ItemResponse]
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /items [get]
func (h *ItemHandler) ListAvailable(c *gin.Context) {
	req, err := parsePagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var typeFilter *string
	t := c.Query("type")
	if t != "" {
		typeFilter = &t
	}

	res, err := h.itemService.ListAvailable(c.Request.Context(), typeFilter, req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidItemType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stream inventory updates"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// BuyItemDirectly godoc
// @Summary      Purchase item directly (Limit Order)
// @Description  Triggers instant direct transaction fulfillment loop logic for Common or Rare asset categories
// @Tags         items
// @Param        X-Guild-ID  header    string  true  "Buyer Guild UUID"
// @Param        id          path      string  true  "Item UUID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /items/{id}/buy [post]
func (h *ItemHandler) BuyItemDirectly(c *gin.Context) {
	buyerStr := c.GetHeader("X-Guild-ID")
	buyerID, err := uuid.Parse(buyerStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or malformed X-Guild-ID context header"})
		return
	}

	idStr := c.Param("id")
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target item asset format"})
		return
	}

	err = h.itemService.BuyItemDirectly(c.Request.Context(), itemID, buyerID)
	if err != nil {
		if errors.Is(err, service.ErrItemNotFound) || errors.Is(err, service.ErrGuildNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrInvalidEntityIDs) ||
			errors.Is(err, service.ErrItemNotAvailable) ||
			errors.Is(err, service.ErrPurchaseOwnItem) ||
			errors.Is(err, service.ErrInsufficientGold) ||
			errors.Is(err, service.ErrDailyLimitExceeded) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to buy item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "immediate order filled and item transferred successfully"})
}
