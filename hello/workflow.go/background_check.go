package workflow

import (
	"hello-temporal/activity"
	"time"

	"go.temporal.io/sdk/workflow"
)

func BackgroundCheck(ctx workflow.Context, ssn string) (string, error) {
	opts := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, opts)
	var ssnTraceResult string
	err := workflow.ExecuteActivity(ctx, activity.SSNTraceActivity, ssn).Get(ctx, &ssnTraceResult)
	if err != nil {
		return "", err
	}

	return ssnTraceResult, nil
}
