package handler

import (
	"logistic-service/internal/model"
	"logistic-service/internal/repository"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	jwt "github.com/golang-jwt/jwt/v5"
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

// CreateShipment menerima channel RabbitMQ sebagai argumen tambahan
func CreateShipment(repo *repository.ShipmentRepository, ch *amqp.Channel) gin.HandlerFunc {
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

		userID, ok := claims["user_id"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id in claims"})
			return
		}

		input.UserID = userID

		if err := repo.Insert(&input); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create shipment"})
			return
		}

		// Publish ke RabbitMQ pakai channel yang sudah ada
		body, err := json.Marshal(input)
		if err != nil {
			log.Printf("[CreateShipment] Marshal shipment error: %v", err)
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
				log.Printf("[CreateShipment] RabbitMQ publish error: %v", err)
			} else {
				log.Printf("[CreateShipment] Published shipment.created for tracking number: %s", input.TrackingNumber)
			}
		}

		c.JSON(http.StatusCreated, input)
	}
}

// UpdateShipmentStatus menerima channel RabbitMQ sebagai argumen tambahan
func UpdateShipmentStatus(repo *repository.ShipmentRepository, ch *amqp.Channel) gin.HandlerFunc {
	return func(c *gin.Context) {
		trackingNumber := c.Param("trackingNumber")
		var req struct {
			Status string `json:"status" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := repo.UpdateStatus(trackingNumber, req.Status)
		if err != nil {
			log.Printf("UpdateStatus error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
			return
		}


		shipment, err := repo.FindByTrackingNumber(trackingNumber)
		if err != nil || shipment == nil {
			log.Printf("[UpdateShipmentStatus] Warning: failed to find shipment after update: %v", err)
		} else {
			body, err := json.Marshal(shipment)
			if err != nil {
				log.Printf("[UpdateShipmentStatus] Marshal shipment error: %v", err)
			} else {
				err = ch.Publish(
					"",
					"shipment.updated",
					false,
					false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        body,
					},
				)
				if err != nil {
					log.Printf("[UpdateShipmentStatus] RabbitMQ publish error: %v", err)
				} else {
					log.Printf("[UpdateShipmentStatus] Published shipment.updated for tracking number: %s", trackingNumber)
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "status updated"})
	}
}



// TrackShipment handles GET /shipments/:trackingNumber
func TrackShipment(repo *repository.ShipmentRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		trackingNumber := c.Param("trackingNumber")

		shipment, err := repo.FindByTrackingNumber(trackingNumber)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch shipment"})
			return
		}
		if shipment == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "shipment not found"})
			return
		}

		c.JSON(http.StatusOK, shipment)
	}
}

// GetShipments handles GET /shipments to fetch all shipments for logged-in user
func GetShipments(repo *repository.ShipmentRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ambil claim JWT yang sudah disimpan oleh middleware JWTAuthMiddleware
		claimsRaw, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Casting claim ke tipe jwt.MapClaims
		claims, ok := claimsRaw.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		// Ambil user_id dari claim
		userID, ok := claims["user_id"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id in claims"})
			return
		}

		// Query MongoDB untuk mendapatkan shipment yang terkait dengan user_id ini
		shipments, err := repo.FindByUserID(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch shipments"})
			return
		}

		// Kirimkan response JSON berisi list shipment
		c.JSON(http.StatusOK, shipments)
	}
}
