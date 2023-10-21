package main

import (
	"hello-temporal/activity"
	"hello-temporal/workflow.go"
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	clientOptions := client.Options{
		HostPort: "localhost:7233",
	}
	temporalClient, err := client.Dial(clientOptions)
	if err != nil {
		log.Fatalln("unable to create temporal client", err)
	}

	defer temporalClient.Close()

	w := worker.New(temporalClient, "hello", worker.Options{})
	w.RegisterWorkflow(workflow.BackgroundCheck)
	w.RegisterActivity(activity.SSNTraceActivity)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("unable to start worker", err)
	}

}
