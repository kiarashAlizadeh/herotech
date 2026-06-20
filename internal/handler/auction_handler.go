package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kiarashAlizadeh/herotech/internal/dto"
	"github.com/kiarashAlizadeh/herotech/internal/service"
)

type AuctionHandler struct {
	auctionService service.AuctionService
}

func NewAuctionHandler(s service.AuctionService) *AuctionHandler {
	return &AuctionHandler{auctionService: s}
}

// StartAuction godoc
// @Summary      Launch a Legendary item auction
// @Description  Commences a public competitive auction profile cycle sequence restricted exclusively to legendary assets
// @Tags         auctions
// @Accept       json
// @Produce      json
// @Param        X-Guild-ID  header    string                    true  "Seller Guild UUID"
// @Param        request     body      dto.CreateAuctionRequest  true  "Auction listing duration configuration"
// @Success      201  {object}  dto.AuctionResponse
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auctions [post]
func (h *AuctionHandler) StartAuction(c *gin.Context) {
	sellerStr := c.GetHeader("X-Guild-ID")
	sellerID, err := uuid.Parse(sellerStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or malformed X-Guild-ID context header"})
		return
	}

	var req dto.CreateAuctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ItemID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_id is required"})
		return
	}
	if req.StartPrice == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_price is required"})
		return
	}
	if req.Duration == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "duration is required"})
		return
	}

	res, err := h.auctionService.StartAuction(c.Request.Context(), sellerID, req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidStartPrice) ||
			errors.Is(err, service.ErrInvalidDuration) ||
			errors.Is(err, service.ErrInvalidEntityIDs) ||
			errors.Is(err, service.ErrAssociatedItemNotFound) ||
			errors.Is(err, service.ErrItemAlreadyInAuction) ||
			errors.Is(err, service.ErrNonLegendaryAuction) ||
			errors.Is(err, service.ErrNotItemOwner) ||
			errors.Is(err, service.ErrMaxActiveAuctions) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start auction"})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// GetAuction godoc
// @Summary      Get auction board details
// @Description  Retrieves current auction configurations, ends timetables, and leading winner parameters
// @Tags         auctions
// @Produce      json
// @Param        id   path      string  true  "Auction UUID"
// @Success      200  {object}  dto.AuctionResponse
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /auctions/{id} [get]
func (h *AuctionHandler) GetAuction(c *gin.Context) {
	idStr := c.Param("id")
	auctionID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid auction primary index format"})
		return
	}

	res, err := h.auctionService.GetAuction(c.Request.Context(), auctionID)
	if err != nil {
		if errors.Is(err, service.ErrAuctionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup failed internally"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// ListActiveAuctions godoc
// @Summary      List active marketplace auctions
// @Description  Streams current dynamic storefront auction indices filtering active runs
// @Tags         auctions
// @Produce      json
// @Param        page       query     int  false  "Page number"
// @Param        page_size  query     int  false  "Page size"
// @Success      200  {object}  dto.PaginatedResponse[dto.AuctionResponse]
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auctions [get]
func (h *AuctionHandler) ListActiveAuctions(c *gin.Context) {
	req, err := parsePagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.auctionService.ListActiveAuctions(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sync pipeline standings"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// PlaceBid godoc
// @Summary      Submit a new competitive bid
// @Description  Evaluates and appends a bid reservation position. Requires at least 5% gain increment steps.
// @Tags         auctions
// @Accept       json
// @Produce      json
// @Param        X-Guild-ID  header    string               true  "Bidder Guild UUID"
// @Param        id          path      string               true  "Auction UUID"
// @Param        request     body      dto.PlaceBidRequest  true  "Bid financial allocation criteria"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Router       /auctions/{id}/bid [post]
func (h *AuctionHandler) PlaceBid(c *gin.Context) {
	bidderStr := c.GetHeader("X-Guild-ID")
	bidderID, err := uuid.Parse(bidderStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or malformed X-Guild-ID context header"})
		return
	}

	idStr := c.Param("id")
	auctionID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target auction scope format"})
		return
	}

	var req dto.PlaceBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Amount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount is required"})
		return
	}

	err = h.auctionService.PlaceBid(c.Request.Context(), auctionID, bidderID, req)
	if err != nil {
		if errors.Is(err, service.ErrAuctionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrBidTooLow) ||
			errors.Is(err, service.ErrAlreadyHighestBidder) ||
			errors.Is(err, service.ErrInvalidBidAmount) ||
			errors.Is(err, service.ErrInvalidAuctionID) ||
			errors.Is(err, service.ErrInvalidGuildID) ||
			errors.Is(err, service.ErrAuctionNotActive) ||
			errors.Is(err, service.ErrBidOnOwnAuction) ||
			errors.Is(err, service.ErrInsufficientBalance) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to place bid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "bid transaction processed and capital locked successfully"})
}
