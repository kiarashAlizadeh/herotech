package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kiarashAlizadeh/herotech/internal/dto"
	"github.com/kiarashAlizadeh/herotech/internal/mocks"
	"github.com/kiarashAlizadeh/herotech/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGuildHandler_CreateGuild(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		body       string
		setupMock  func(*mocks.GuildService)
		wantStatus int
	}{
		{
			name:       "rejects invalid body",
			body:       `{"name":"A","daily_limit":0}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "maps empty name service error",
			body: `{"name":"  ","daily_limit":1000}`,
			setupMock: func(svc *mocks.GuildService) {
				svc.On("CreateGuild", mock.Anything, dto.CreateGuildRequest{Name: "  ", DailyLimit: 1000}).Return(nil, service.ErrEmptyGuildName).Once()
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "returns internal server error for unexpected service failure",
			body: `{"name":"Mages","daily_limit":1000}`,
			setupMock: func(svc *mocks.GuildService) {
				svc.On("CreateGuild", mock.Anything, dto.CreateGuildRequest{Name: "Mages", DailyLimit: 1000}).Return(nil, errors.New("db down")).Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "creates guild successfully",
			body: `{"name":"Mages","daily_limit":1000}`,
			setupMock: func(svc *mocks.GuildService) {
				svc.On("CreateGuild", mock.Anything, dto.CreateGuildRequest{Name: "Mages", DailyLimit: 1000}).Return(&dto.GuildResponse{
					ID:         uuid.New(),
					Name:       "Mages",
					DailyLimit: 1000,
					CreatedAt:  time.Now(),
				}, nil).Once()
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewGuildService(t)
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}

			router := gin.New()
			router.POST("/guilds", NewGuildHandler(svc).CreateGuild)

			req := httptest.NewRequest(http.MethodPost, "/guilds", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestGuildHandler_GetWalletSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	guildID := uuid.New()

	tests := []struct {
		name       string
		pathID     string
		setupMock  func(*mocks.GuildService)
		wantStatus int
	}{
		{
			name:       "rejects malformed guild id",
			pathID:     "bad-id",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "returns not found for missing guild",
			pathID: guildID.String(),
			setupMock: func(svc *mocks.GuildService) {
				svc.On("GetWalletSummary", mock.Anything, guildID).Return(nil, service.ErrGuildNotFound).Once()
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "returns internal error for unexpected failure",
			pathID: guildID.String(),
			setupMock: func(svc *mocks.GuildService) {
				svc.On("GetWalletSummary", mock.Anything, guildID).Return(nil, errors.New("db down")).Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "returns wallet summary",
			pathID: guildID.String(),
			setupMock: func(svc *mocks.GuildService) {
				svc.On("GetWalletSummary", mock.Anything, guildID).Return(&dto.WalletSummaryResponse{
					TotalBalance:     1000,
					ReservedAmount:   200,
					AvailableBalance: 800,
				}, nil).Once()
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewGuildService(t)
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}

			router := gin.New()
			router.GET("/guilds/:id/wallet", NewGuildHandler(svc).GetWalletSummary)

			req := httptest.NewRequest(http.MethodGet, "/guilds/"+tt.pathID+"/wallet", nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
