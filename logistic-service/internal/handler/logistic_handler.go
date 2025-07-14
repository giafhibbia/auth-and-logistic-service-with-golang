package handler

import (
	"logistic-service/internal/model"
	"logistic-service/internal/repository"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	jwt "github.com/golang-jwt/jwt/v5"
	"os"
    "log"
    "encoding/json"
    amqp "github.com/rabbitmq/amqp091-go"
)

// GetCourierRates handles GET /courier-rates?origin_name=...&destination_name=...
// Requires valid JWT. Returns courier rates filtered by origin and destination.
func GetCourierRates(repo *repository.ShipmentRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Query("origin_name")
		destination := c.Query("destination_name")

		if origin == "" || destination == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "origin_name and destination_name are required"})
			return
		}

		rates, err := repo.FindByOriginDestination(origin, destination)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch courier rates"})
			return
		}

		c.JSON(http.StatusOK, rates)
	}
}

func CreateShipment(repo *repository.ShipmentRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input model.Shipment

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validasi duplikat tracking_number
		existingShipment, err := repo.FindByTrackingNumber(input.TrackingNumber)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check existing shipment"})
			return
		}
		if existingShipment != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tracking_number already exists"})
			return
		}

		input.ID = uuid.New().String()
		now := time.Now()
		input.CreatedAt = now
		input.UpdatedAt = now

		claimsRaw, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		claims, ok := claimsRaw.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		// Debug print seluruh claims
		log.Printf("[DEBUG] Claims: %+v\n", claims)

		// Ambil user_id dari claim
		userID, ok := claims["user_id"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id in claims"})
			return
		}

		// Debug print userID
		log.Printf("[DEBUG] userID from claims: %s\n", userID)

		input.UserID = userID

		if err := repo.Insert(&input); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create shipment"})
			return
		}

		// Publish event ke RabbitMQ (optional)
		rabbitURL := os.Getenv("RABBITMQ_URL")
		conn, err := amqp.Dial(rabbitURL)
		if err != nil {
			log.Println("[CreateShipment] RabbitMQ connection error:", err)
		} else {
			defer conn.Close()
			ch, err := conn.Channel()
			if err != nil {
				log.Println("[CreateShipment] RabbitMQ channel error:", err)
			} else {
				defer ch.Close()
				body, err := json.Marshal(input)
				if err != nil {
					log.Println("[CreateShipment] Marshal shipment to JSON error:", err)
				} else {
					err = ch.Publish(
						"",
						"shipment.created",
						false,
						false,
						amqp.Publishing{
							ContentType: "application/json",
							Body:        body,
						},
					)
					if err != nil {
						log.Println("[CreateShipment] RabbitMQ publish error:", err)
					} else {
						log.Println("[CreateShipment] RabbitMQ publish OK (shipment.created)")
					}
				}
			}
		}

		// Response sukses dengan data shipment yang baru dibuat
		c.JSON(http.StatusCreated, input)
	}
}




// UpdateShipmentStatus handles PATCH /shipments/:trackingNumber/status to update shipment status
func UpdateShipmentStatus(repo *repository.ShipmentRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		trackingNumber := c.Param("trackingNumber")
		var req struct {
			Status string `json:"status" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := repo.UpdateStatus(trackingNumber, req.Status); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "status updated"})
	}
}

// TrackShipment handles GET /shipments/:trackingNumber to get shipment details by tracking number
func TrackShipment(repo *repository.ShipmentRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		trackingNumber := c.Param("trackingNumber")
		shipment, err := repo.FindByTrackingNumber(trackingNumber)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "shipment not found"})
			return
		}
		c.JSON(http.StatusOK, shipment)
	}
}

// GetShipments handles GET /shipments to fetch all shipments for logged-in user
func GetShipments(repo *repository.ShipmentRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsRaw, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		claims, ok := claimsRaw.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}
		userID, ok := claims["user_id"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id in claims"})
			return
		}

		shipments, err := repo.FindByUserID(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch shipments"})
			return
		}
		c.JSON(http.StatusOK, shipments)
	}
}
