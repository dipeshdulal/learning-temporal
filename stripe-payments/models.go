package main

type Order struct {
	ID          string `json:"id"`
	OrderID     string `json:"order_id"`
	Status      string `json:"status"`
	ExecutionID string `json:"execution_id"`
}

type Payment struct {
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
	OrderID   string `json:"order_id"`
}
