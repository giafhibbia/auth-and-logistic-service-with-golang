package main

import (
	"logistic-service/internal/handler"
	"logistic-service/internal/middleware"
	"logistic-service/internal/repository"
	"context"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Retrieve MongoDB URI from environment variables
	mongoUri := os.Getenv("MONGO_URI")

	// Establish connection to MongoDB using a background context
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoUri))
	if err != nil {
		// Panic and stop execution if connection fails
		panic(err)
	}

	// Select the database "logisticdb" for the logistics service
	db := client.Database("logisticdb")

	// Initialize repositories for courier rates and shipments
	//courierRepo := repository.NewCourierRateRepository(db)
	shipmentRepo := repository.NewShipmentRepository(db)

	// Initialize Gin router with default middleware (logger & recovery)
	r := gin.Default()

	// Configure CORS to allow access from various origins,
	// and allow common headers and HTTP methods used by frontend
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true, // For production, specify allowed origins explicitly
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour, // Cache preflight requests for 12 hours
	}))

	// Apply JWT authentication middleware to protect all subsequent endpoints
	r.Use(middleware.JWTAuthMiddleware())

	// Endpoint to fetch courier rates
	//r.GET("/courier-rates", handler.GetCourierRates(courierRepo))

	// Endpoint to create a new shipment order
	r.POST("/shipments", handler.CreateShipment(shipmentRepo))

	// Endpoint to update shipment status by tracking number
	r.PATCH("/shipments/:trackingNumber/status", handler.UpdateShipmentStatus(shipmentRepo))

	// Endpoint to track shipment status by tracking number
	r.GET("/shipments/:trackingNumber", handler.TrackShipment(shipmentRepo))

	// Endpoint to get all shipments related to the logged-in user
	r.GET("/shipments", handler.GetShipments(shipmentRepo))

	// Start the HTTP server on port 8082 (logistics service)
	r.Run(":8082")
}
