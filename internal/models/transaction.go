package models

import "time"

type Transaction struct {
	TransactionID string
	Date          time.Time
	UserID        string
	Country       string
	Region        string
	ProductID     string
	ProductName   string
	Category      string
	Price         float64
	Quantity      int
	TotalPrice    float64
	Stock         int
	AddedDate     time.Time
}

type CountryRevenue struct {
	Country      string  `json:"country"`
	ProductName  string  `json:"product_name"`
	Category     string  `json:"category"`
	TotalRevenue float64 `json:"total_revenue"`
	Transactions int     `json:"transactions"`
}

type ProductFrequency struct {
	ProductName   string `json:"product_name"`
	Category      string `json:"category"`
	Frequency     int    `json:"frequency"`
	StockQuantity int    `json:"stock_quantity"`
}

type MonthlyData struct {
	Month  string  `json:"month"`
	Volume float64 `json:"volume"`
}

type RegionRevenue struct {
	Region    string  `json:"region"`
	Revenue   float64 `json:"total_revenue"`
	ItemsSold int     `json:"items_sold"`
}
