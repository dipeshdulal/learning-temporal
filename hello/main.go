package main

import (
	"context"
	"log"

	"hello-temporal/workflow.go"

	"go.temporal.io/sdk/client"
)

func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatal(err)
	}

	workflowRun, err := c.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
		TaskQueue: "hello",
	}, workflow.BackgroundCheck, "param")
	if err != nil {
		log.Fatal(err)
	}

	var result string
	err = workflowRun.Get(context.Background(), &result)
	if err != nil {
		log.Fatal(err)
	}
}
