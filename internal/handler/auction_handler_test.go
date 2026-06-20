package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kiarashAlizadeh/herotech/internal/dto"
	"github.com/kiarashAlizadeh/herotech/internal/mocks"
	"github.com/kiarashAlizadeh/herotech/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuctionHandler_PlaceBid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	auctionID := uuid.New()
	bidderID := uuid.New()

	tests := []struct {
		name       string
		pathID     string
		headerID   string
		body       string
		setupMock  func(*mocks.AuctionService)
		wantStatus int
	}{
		{
			name:       "rejects malformed guild header",
			pathID:     auctionID.String(),
			headerID:   "bad-id",
			body:       `{"amount":1000}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects malformed auction id",
			pathID:     "bad-id",
			headerID:   bidderID.String(),
			body:       `{"amount":1000}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects invalid body",
			pathID:     auctionID.String(),
			headerID:   bidderID.String(),
			body:       `{"amount":0}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "returns not found for missing auction",
			pathID:   auctionID.String(),
			headerID: bidderID.String(),
			body:     `{"amount":1000}`,
			setupMock: func(svc *mocks.AuctionService) {
				svc.On("PlaceBid", mock.Anything, auctionID, bidderID, dto.PlaceBidRequest{Amount: 1000}).Return(service.ErrAuctionNotFound).Once()
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "returns bad request for service validation error",
			pathID:   auctionID.String(),
			headerID: bidderID.String(),
			body:     `{"amount":1000}`,
			setupMock: func(svc *mocks.AuctionService) {
				svc.On("PlaceBid", mock.Anything, auctionID, bidderID, dto.PlaceBidRequest{Amount: 1000}).Return(service.ErrBidTooLow).Once()
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "returns internal server error for unexpected failure",
			pathID:   auctionID.String(),
			headerID: bidderID.String(),
			body:     `{"amount":1000}`,
			setupMock: func(svc *mocks.AuctionService) {
				svc.On("PlaceBid", mock.Anything, auctionID, bidderID, dto.PlaceBidRequest{Amount: 1000}).Return(errors.New("db down")).Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:     "places bid successfully",
			pathID:   auctionID.String(),
			headerID: bidderID.String(),
			body:     `{"amount":1000}`,
			setupMock: func(svc *mocks.AuctionService) {
				svc.On("PlaceBid", mock.Anything, auctionID, bidderID, dto.PlaceBidRequest{Amount: 1000}).Return(nil).Once()
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewAuctionService(t)
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}

			router := gin.New()
			router.POST("/auctions/:id/bid", NewAuctionHandler(svc).PlaceBid)

			req := httptest.NewRequest(http.MethodPost, "/auctions/"+tt.pathID+"/bid", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Guild-ID", tt.headerID)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestAuctionHandler_StartAuction_BodyValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := mocks.NewAuctionService(t)
	router := gin.New()
	router.POST("/auctions", NewAuctionHandler(svc).StartAuction)

	req := httptest.NewRequest(http.MethodPost, "/auctions", strings.NewReader(`{"start_price":0}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Guild-ID", uuid.NewString())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
