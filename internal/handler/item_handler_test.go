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

func TestItemHandler_BuyItemDirectly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	itemID := uuid.New()
	buyerID := uuid.New()

	tests := []struct {
		name       string
		pathID     string
		headerID   string
		setupMock  func(*mocks.ItemService)
		wantStatus int
	}{
		{
			name:       "rejects malformed guild header",
			pathID:     itemID.String(),
			headerID:   "bad-id",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects malformed item id",
			pathID:     "bad-id",
			headerID:   buyerID.String(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "returns not found for missing item",
			pathID:   itemID.String(),
			headerID: buyerID.String(),
			setupMock: func(svc *mocks.ItemService) {
				svc.On("BuyItemDirectly", mock.Anything, itemID, buyerID).Return(service.ErrItemNotFound).Once()
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "returns bad request on service error",
			pathID:   itemID.String(),
			headerID: buyerID.String(),
			setupMock: func(svc *mocks.ItemService) {
				svc.On("BuyItemDirectly", mock.Anything, itemID, buyerID).Return(service.ErrInsufficientGold).Once()
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "returns internal server error on unexpected failure",
			pathID:   itemID.String(),
			headerID: buyerID.String(),
			setupMock: func(svc *mocks.ItemService) {
				svc.On("BuyItemDirectly", mock.Anything, itemID, buyerID).Return(errors.New("db down")).Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:     "buys item successfully",
			pathID:   itemID.String(),
			headerID: buyerID.String(),
			setupMock: func(svc *mocks.ItemService) {
				svc.On("BuyItemDirectly", mock.Anything, itemID, buyerID).Return(nil).Once()
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewItemService(t)
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}

			router := gin.New()
			router.POST("/items/:id/buy", NewItemHandler(svc).BuyItemDirectly)

			req := httptest.NewRequest(http.MethodPost, "/items/"+tt.pathID+"/buy", nil)
			req.Header.Set("X-Guild-ID", tt.headerID)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestItemHandler_ListItem_BodyValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := mocks.NewItemService(t)
	svc.On("ListItem", mock.Anything, mock.Anything, mock.Anything).Return(nil, service.ErrInvalidItemType).Once()

	router := gin.New()
	router.POST("/items", NewItemHandler(svc).ListItem)

	req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{"name":"A","type":"invalid"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Guild-ID", uuid.NewString())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestItemHandler_ListItem_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ownerID := uuid.New()
	price := int64(120)
	reqBody := `{"name":"Potion","type":"common","list_price":120}`
	expectedReq := dto.CreateItemRequest{Name: "Potion", Type: "common", ListPrice: &price}

	svc := mocks.NewItemService(t)
	svc.On("ListItem", mock.Anything, ownerID, expectedReq).Return(&dto.ItemResponse{
		ID:        uuid.New(),
		Name:      "Potion",
		Type:      "common",
		OwnerID:   ownerID,
		BasePrice: 100,
		ListPrice: &price,
	}, nil).Once()

	router := gin.New()
	router.POST("/items", NewItemHandler(svc).ListItem)

	req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Guild-ID", ownerID.String())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestItemHandler_ListAvailable_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	invalidType := "invalid"
	svc := mocks.NewItemService(t)
	svc.On("ListAvailable", mock.Anything, &invalidType, dto.PaginationRequest{Page: 1, PageSize: 20}).Return(nil, service.ErrInvalidItemType).Once()

	router := gin.New()
	router.GET("/items", NewItemHandler(svc).ListAvailable)

	req := httptest.NewRequest(http.MethodGet, "/items?type=invalid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
