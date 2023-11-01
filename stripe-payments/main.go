package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"gorm.io/gorm"
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

func GetPaymentLink(ctx *gin.Context) {
	c, err := client.NewLazyClient(client.Options{})
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()

	id := ctx.Query("id")

	db, _ := ctx.MustGet("db").(*gorm.DB)

	var order Order
	err = db.Where("order_id = ?", id).First(&order).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var url string
	val, err := c.QueryWorkflow(context.Background(), id, order.ExecutionID, "GetPaymentLink")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = val.Get(&url)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}

func CompletePayment(ctx *gin.Context) {
	item, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println(item)
}

func SuccessPayment(ctx *gin.Context) {
	c, err := client.NewLazyClient(client.Options{})
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()

	id := ctx.Query("id")

	var order Order
	db, _ := ctx.MustGet("db").(*gorm.DB)
	err = db.Where("order_id = ?", id).First(&order).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = c.SignalWorkflow(context.Background(), id, order.ExecutionID, "payment_success", nil)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
}

func FailPayment(ctx *gin.Context) {
	c, err := client.NewLazyClient(client.Options{})
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()

	id := ctx.Query("id")

	var order Order
	db, _ := ctx.MustGet("db").(*gorm.DB)
	err = db.Where("order_id = ?", id).First(&order).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = c.SignalWorkflow(context.Background(), id, order.ExecutionID, "payment_failed", nil)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	interruptChannel := worker.InterruptCh()

	// get stripe key from env
	stripeKey := os.Getenv("STRIPE_KEY")
	if stripeKey == "" {
		log.Fatal("STRIPE_KEY is not set")
	}

	go func() {

		db, err := NewDB()
		if err != nil {
			log.Fatal(err)
		}

		db.AutoMigrate(&Order{})
		db.AutoMigrate(&Payment{})

		c, err := client.NewLazyClient(client.Options{})
		if err != nil {
			log.Fatal(err)
		}

		defer c.Close()

		w := worker.New(c, "ORDER_ITEM_QUEUE", worker.Options{})

		a := NewStripeActivity(db)
		w.RegisterActivity(a.CreatePayment)
		w.RegisterActivity(a.CreatePaymentLink)
		w.RegisterActivity(a.ClosePaymentLink)
		w.RegisterActivity(a.CreateOrder)
		w.RegisterActivity(a.PaymentSuccess)
		w.RegisterActivity(a.PaymentFailed)
		w.RegisterActivity(a.PaymentCheckAndFail)
		w.RegisterWorkflow(StripeWorkflow)

		err = w.Run(interruptChannel)
		if err != nil {
			log.Fatal(err)
		}

	}()

	go func() {
		r := gin.Default()
		db, err := NewDB()
		if err != nil {
			log.Fatal(err)
		}

		r.Use(func(c *gin.Context) {
			c.Set("db", db)
			c.Next()
		})

		r.GET("/order-item", OrderItem)
		r.GET("/order-link", GetPaymentLink)
		r.GET("/complete", CompletePayment)
		r.GET("/success_payment", SuccessPayment)
		r.GET("/fail_payment", FailPayment)
		err = r.Run(":8081")
		if err != nil {
			log.Fatal(err)
		}
	}()

	<-interruptChannel
}
