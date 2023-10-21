package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func OrderItem(ctx *gin.Context) {
	c, err := client.NewLazyClient(client.Options{})
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()

	id := ctx.Query("id")
	options := client.StartWorkflowOptions{
		ID:        id,
		TaskQueue: "ORDER_ITEM_QUEUE",
	}
	we, err := c.ExecuteWorkflow(context.Background(), options, StripeWorkflow, OrderState{
		OrderItem: ctx.Query("id"),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"workflowID": we.GetID(),
		"runID":      we.GetRunID(),
	})
}

func main() {
	interruptChannel := worker.InterruptCh()

	go func() {

		db, err := NewDB()
		if err != nil {
			log.Fatal(err)
		}

		db.AutoMigrate(&Order{})

		c, err := client.NewLazyClient(client.Options{})
		if err != nil {
			log.Fatal(err)
		}

		defer c.Close()

		w := worker.New(c, "ORDER_ITEM_QUEUE", worker.Options{})

		a := StripeActivity{
			DB: db,
		}
		w.RegisterActivity(a.CreateOrder)
		w.RegisterWorkflow(StripeWorkflow)

		err = w.Run(interruptChannel)
		if err != nil {
			log.Fatal(err)
		}

	}()

	go func() {
		r := gin.Default()
		r.GET("/order-item", OrderItem)
		err := r.Run(":8081")
		if err != nil {
			log.Fatal(err)
		}
	}()

	<-interruptChannel
}
