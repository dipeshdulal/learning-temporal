package main

import (
	"time"

	"github.com/stripe/stripe-go/v76"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type OrderState struct {
	OrderItem string
}

func StripeWorkflow(ctx workflow.Context, state OrderState) error {

	actOpts := workflow.ActivityOptions{
		StartToCloseTimeout: time.Second * 2,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 5,
		},
	}

	ctx, cancelWorkflow := workflow.WithCancel(ctx)

	var a StripeActivity
	var plink *stripe.PaymentLink

	// get workflow run id
	info := workflow.GetInfo(ctx)
	executionID := info.WorkflowExecution.RunID

	workflow.SetQueryHandler(ctx, "GetPaymentLink", func() (string, error) {
		return plink.URL, nil
	})

	// create order with status created
	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, actOpts), a.CreateOrder, state.OrderItem, executionID).Get(ctx, nil)
	if err != nil {
		return err
	}

	// call stripe and create payment link
	err = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, actOpts), a.CreatePaymentLink, state.OrderItem).Get(ctx, &plink)
	if err != nil {
		return err
	}

	// create payment with status created
	err = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, actOpts), a.CreatePayment, plink.ID, state.OrderItem).Get(ctx, nil)
	if err != nil {
		return err
	}

	// if payment succeded
	successChan := workflow.GetSignalChannel(ctx, "payment_success")
	workflow.Go(ctx, func(ctx workflow.Context) {
		successChan.Receive(ctx, nil)
		err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, actOpts), a.ClosePaymentLink, plink.ID).Get(ctx, nil)
		if err != nil {
			workflow.GetLogger(ctx).Error("error closing payment link", "error", err)
		}

		err = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, actOpts), a.PaymentSuccess, plink.ID).Get(ctx, nil)
		if err != nil {
			workflow.GetLogger(ctx).Error("error updating payment", "error", err)
		}

		cancelWorkflow()
	})

	// if payment failed
	failedChan := workflow.GetSignalChannel(ctx, "payment_failed")
	workflow.Go(ctx, func(ctx workflow.Context) {
		failedChan.Receive(ctx, nil)

		err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, actOpts), a.PaymentFailed, plink.ID).Get(ctx, nil)
		if err != nil {
			workflow.GetLogger(ctx).Error("error updating payment", "error", err)
		}

		cancelWorkflow()
	})

	// check payment status in 15 minutes, if not completed, fail it.
	workflow.Sleep(ctx, time.Minute*3)
	return workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, actOpts), a.PaymentCheckAndFail, plink.ID).Get(ctx, nil)
}
