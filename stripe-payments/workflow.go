package main

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

type OrderState struct {
	OrderItem string
}

func StripeWorkflow(ctx workflow.Context, state OrderState) error {
	var a StripeActivity
	return workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Second * 2,
	}), a.CreateOrder, state.OrderItem).Get(ctx, nil)

	// create order with status created
	// create payment with status created
	// call stripe and create payment link

	// if payment succeded
	// update order status to created
	// send email
	// update payment to success and create transaction
	// if payment failed
	// update payment with failed
	// update order status to failed
	// send email, payment failed

	// check payment status in 15 minutes, if not completed, fail it.

}
