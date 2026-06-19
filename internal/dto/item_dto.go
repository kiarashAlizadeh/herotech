package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateItemRequest struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	ListPrice *int64 `json:"list_price,omitempty"`
}

type ItemResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	OwnerID   uuid.UUID `json:"owner_id"`
	BasePrice int64     `json:"base_price"`
	ListPrice *int64    `json:"list_price,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
