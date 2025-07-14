package main

import (
	"context"
	"log"
	"logistic-service/internal/handler"
	"logistic-service/internal/middleware"
	"logistic-service/internal/repository"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Get MongoDB URI from environment variables
	mongoUri := os.Getenv("MONGO_URI")
	if mongoUri == "" {
		log.Fatal("MONGO_URI environment variable not set")
	}

	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoUri))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Ping MongoDB to verify connection
	if err := client.Ping(context.Background(), nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	// Select the database "logisticdb"
	db := client.Database("logisticdb")

	// Create shipment repository instance
	shipmentRepo := repository.NewShipmentRepository(db)

	// Connect to RabbitMQ
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Fatal("RABBITMQ_URL environment variable not set")
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// Open RabbitMQ channel (used for publishing)
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open RabbitMQ channel: %v", err)
	}
	defer ch.Close()

	// Initialize Gin router with default middleware (logger & recovery)
	r := gin.Default()

	// Setup CORS middleware (adjust for your production environment)
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true, // Replace with allowed origins in production
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Apply JWT authentication middleware to protect all routes below
	r.Use(middleware.JWTAuthMiddleware())

	// Register routes with injected repository and RabbitMQ channel
	r.POST("/shipments", handler.CreateShipment(shipmentRepo, ch))
	r.PATCH("/shipments/:trackingNumber/status", handler.UpdateShipmentStatus(shipmentRepo, ch))
	r.GET("/shipments/:trackingNumber", handler.TrackShipment(shipmentRepo))
	r.GET("/shipments", handler.GetShipments(shipmentRepo))

	// Start the HTTP server on port 8082
	log.Println("Starting logistics service on :8082")
	if err := r.Run(":8082"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
