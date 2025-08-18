package products

import "time"

type Product struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	CurrentPrice *float64  `json:"current_price,omitempty"` // nullable
	CreatedAt    time.Time `json:"created_at"`
}

type PriceHistory struct {
	ID         int       `json:"id"`
	ProductID  int       `json:"product_id"`
	Price      float64   `json:"price"`
	RecordedAt time.Time `json:"recorded_at"`
}
