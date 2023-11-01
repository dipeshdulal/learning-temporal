package main

import (
	"os"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentlink"
	"gorm.io/gorm"
)

type StripeActivity struct {
	DB *gorm.DB
}

func NewStripeActivity(db *gorm.DB) *StripeActivity {
	stripeKey := os.Getenv("STRIPE_KEY")
	stripe.Key = stripeKey
	return &StripeActivity{
		DB: db,
	}
}

func (s *StripeActivity) CreateOrder(id string, executionID string) error {
	return s.DB.Create(&Order{
		ID:          id,
		OrderID:     id,
		Status:      "created",
		ExecutionID: executionID,
	}).Error
}

func (s *StripeActivity) CreatePaymentLink(orderId string) (*stripe.PaymentLink, error) {
	params := &stripe.PaymentLinkParams{
		LineItems: []*stripe.PaymentLinkLineItemParams{
			{
				Price:    stripe.String("price_1O63rDJ5mEeVfKaR3PNvI0xZ"),
				Quantity: stripe.Int64(1),
			},
		},
		AfterCompletion: &stripe.PaymentLinkAfterCompletionParams{
			Type: stripe.String("redirect"),
			Redirect: &stripe.PaymentLinkAfterCompletionRedirectParams{
				URL: stripe.String("http://localhost:8081/complete"),
			},
		},
		Metadata: map[string]string{
			"order_id": orderId,
		},
	}

	result, err := paymentlink.New(params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *StripeActivity) PaymentSuccess(paymentId string) error {
	// find payment
	var payment Payment
	err := s.DB.Where("payment_id = ?", paymentId).First(&payment).Error
	if err != nil {
		return err
	}

	// update payment status to success
	err = s.DB.Model(&Payment{}).Where("payment_id = ?", paymentId).Update("status", "success").Error
	if err != nil {
		return err
	}

	// update order status to success
	err = s.DB.Model(&Order{}).Where("order_id = ?", payment.OrderID).Update("status", "success").Error
	if err != nil {
		return err
	}

	return err
}

func (s *StripeActivity) PaymentFailed(paymentId string) error {
	// find payment
	var payment Payment
	err := s.DB.Where("payment_id = ?", paymentId).First(&payment).Error
	if err != nil {
		return err
	}

	// update payment status to failed
	err = s.DB.Model(&Payment{}).Where("payment_id = ?", paymentId).Update("status", "failed").Error
	if err != nil {
		return err
	}

	// update order status to failed
	err = s.DB.Model(&Order{}).Where("order_id = ?", payment.OrderID).Update("status", "failed").Error
	if err != nil {
		return err
	}

	return nil
}

func (s *StripeActivity) PaymentCheckAndFail(paymentId string) error {

	if err := s.ClosePaymentLink(paymentId); err != nil {
		return err
	}

	// find payment
	var payment Payment
	err := s.DB.Where("payment_id = ?", paymentId).First(&payment).Error
	if err != nil {
		return err
	}

	if payment.Status != "success" {
		// update payment status to failed
		err = s.DB.Model(&Payment{}).Where("payment_id = ?", paymentId).Update("status", "failed").Error
		if err != nil {
			return err
		}

		// update order status to failed
		err = s.DB.Model(&Order{}).Where("order_id = ?", payment.OrderID).Update("status", "failed").Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *StripeActivity) ClosePaymentLink(paymentId string) error {
	_, err := paymentlink.Update(paymentId, &stripe.PaymentLinkParams{
		Active: stripe.Bool(false),
	})
	return err
}

func (s *StripeActivity) CreatePayment(paymentId, orderId string) error {
	return s.DB.Create(&Payment{
		PaymentID: paymentId,
		Status:    "created",
		OrderID:   orderId,
	}).Error
}
