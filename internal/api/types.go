package api

// ConcurrencyConfig represents worker config update request body
type ConcurrencyConfig struct {
	Workers int `json:"workers"`
}
