package models

import "time"

type Inventory struct {
	ID        int       `json:"id"`
	ProductID int       `json:"product_id"`
	Quantity  int       `json:"quantity"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateInventoryRequest struct {
	Quantity int `json:"quantity"`
}
