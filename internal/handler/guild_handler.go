package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kiarashAlizadeh/herotech/internal/dto"
	"github.com/kiarashAlizadeh/herotech/internal/service"
)

type GuildHandler struct {
	guildService service.GuildService
}

func NewGuildHandler(s service.GuildService) *GuildHandler {
	return &GuildHandler{guildService: s}
}

// CreateGuild godoc
// @Summary      Register a new guild
// @Description  Creates a new guild profile with a defined daily spending limit
// @Tags         guilds
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateGuildRequest true "Guild payload specs"
// @Success      201  {object}  dto.GuildResponse
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /guilds [post]
func (h *GuildHandler) CreateGuild(c *gin.Context) {
	var req dto.CreateGuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if req.DailyLimit == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "daily_limit is required"})
		return
	}

	res, err := h.guildService.CreateGuild(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrEmptyGuildName) ||
			errors.Is(err, service.ErrInvalidGuildNameLength) ||
			errors.Is(err, service.ErrInvalidDailyLimit) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal processing failure"})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// ListGuilds godoc
// @Summary      List all registered guilds
// @Description  Retrieves all guild profiles present in the ledger system
// @Tags         guilds
// @Produce      json
// @Param        page       query     int  false  "Page number"
// @Param        page_size  query     int  false  "Page size"
// @Success      200  {object}  dto.PaginatedResponse[dto.GuildResponse]
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /guilds [get]
func (h *GuildHandler) ListGuilds(c *gin.Context) {
	req, err := parsePagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.guildService.ListGuilds(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch guilds"})
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetWalletSummary godoc
// @Summary      Fetch guild wallet summary
// @Description  Retrieves financial metrics including total balance, reserved funds, and liquid available gold
// @Tags         guilds
// @Produce      json
// @Param        id   path      string  true  "Guild UUID"
// @Success      200  {object}  dto.WalletSummaryResponse
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /guilds/{id}/wallet [get]
func (h *GuildHandler) GetWalletSummary(c *gin.Context) {
	idStr := c.Param("id")
	guildID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild identity formatting"})
		return
	}

	res, err := h.guildService.GetWalletSummary(c.Request.Context(), guildID)
	if err != nil {
		if errors.Is(err, service.ErrGuildNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrInvalidGuildID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate wallet snapshot"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// DepositGold godoc
// @Summary      Deposit gold into guild wallet
// @Description  Increments a specific guild's gold balance ledger sequence
// @Tags         guilds
// @Accept       json
// @Produce      json
// @Param        id      path      string                  true  "Guild UUID"
// @Param        request body      dto.DepositGoldRequest  true  "Deposit amount parameters"
// @Success      200  {object}  dto.GuildResponse
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /guilds/{id}/wallet/deposit [post]
func (h *GuildHandler) DepositGold(c *gin.Context) {
	idStr := c.Param("id")
	guildID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild identity formatting"})
		return
	}

	var req dto.DepositGoldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Amount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount is required"})
		return
	}

	res, err := h.guildService.DepositGold(c.Request.Context(), guildID, req)
	if err != nil {
		if errors.Is(err, service.ErrGuildNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrInvalidGuildID) || errors.Is(err, service.ErrInvalidDepositAmount) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "deposit mutation failed to execute"})
		return
	}

	c.JSON(http.StatusOK, res)
}
