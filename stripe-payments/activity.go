package main

import "gorm.io/gorm"

type StripeActivity struct {
	DB *gorm.DB
}

func (s *StripeActivity) CreateOrder(id string) error {
	return s.DB.Create(&Order{
		ID:      id,
		OrderID: "order-1",
		Status:  "created",
	}).Error
}
