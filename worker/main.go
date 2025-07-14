package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// User represents user model in Postgres
type User struct {
	ID         int       `gorm:"primaryKey;autoIncrement" json:"-"`
	ExternalID string    `json:"id"`
	Msisdn     string    `json:"msisdn"`
	Name       string    `json:"name"`
	Username   string    `json:"username"`
	CreatedAt  time.Time `json:"created_at"`
}

// ShipmentItem represents a shipment item in Postgres
type ShipmentItem struct {
	ID         uint   `gorm:"primaryKey;autoIncrement"`
	ShipmentID string `gorm:"index;not null"`
	Name       string
	Quantity   int
	Weight     float64
}

// Shipment represents shipment model with related items
type Shipment struct {
	ID               string         `gorm:"primaryKey;column:id" json:"id"`
	LogisticName     string         `gorm:"column:logistic_name" json:"logistic_name"`
	TrackingNumber   string         `gorm:"column:tracking_number" json:"tracking_number"`
	Status           string         `gorm:"column:status" json:"status"`
	Origin           string         `gorm:"column:origin" json:"origin"`
	Destination      string         `gorm:"column:destination" json:"destination"`
	Notes            string         `gorm:"column:notes" json:"notes"`
	UserID           string         `gorm:"column:user_id" json:"user_id"`
	CreatedAt        time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at" json:"updated_at"`
	SenderName       string         `gorm:"column:sender_name" json:"sender_name"`
	SenderPhone      string         `gorm:"column:sender_phone" json:"sender_phone"`
	SenderAddress    string         `gorm:"column:sender_address" json:"sender_address"`
	RecipientName    string         `gorm:"column:recipient_name" json:"recipient_name"`
	RecipientPhone   string         `gorm:"column:recipient_phone" json:"recipient_phone"`
	RecipientAddress string         `gorm:"column:recipient_address" json:"recipient_address"`
	Items            []ShipmentItem `gorm:"foreignKey:ShipmentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"items"`
}

func main() {
	dsn := os.Getenv("MASTERDB_URL")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("worker: Failed to connect to Postgres: %v", err))
	}

	// Auto migrate schema
	if err := db.AutoMigrate(&User{}, &Shipment{}, &ShipmentItem{}); err != nil {
		panic(fmt.Sprintf("worker: Failed to migrate schema: %v", err))
	}
	log.Println("[worker] Migrated Postgres schema successfully!")

	rabbitURL := os.Getenv("RABBITMQ_URL")

	var conn *amqp.Connection
	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(rabbitURL)
		if err == nil {
			log.Println("[worker] Connected to RabbitMQ successfully!")
			break
		}
		log.Println("[worker] Waiting for RabbitMQ... retry in 3s")
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		panic(fmt.Sprintf("worker: Failed to connect to RabbitMQ: %v", err))
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		panic(fmt.Sprintf("worker: Failed to open RabbitMQ channel: %v", err))
	}
	defer ch.Close()

	// Declare queues
	queues := []string{"user.registered", "shipment.created", "shipment.updated"}
	for _, q := range queues {
		_, err = ch.QueueDeclare(
			q,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			panic(fmt.Sprintf("worker: Failed to declare queue %s: %v", q, err))
		}
	}

	// Consume user.registered asynchronously
	go func() {
		msgs, err := ch.Consume("user.registered", "", true, false, false, false, nil)
		if err != nil {
			log.Printf("Error consuming user.registered: %v", err)
			return
		}
		for msg := range msgs {
			var payload struct {
				ID       string `json:"id"`
				Msisdn   string `json:"msisdn"`
				Name     string `json:"name"`
				Username string `json:"username"`
			}
			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				log.Println("Unmarshal user.registered failed:", err)
				continue
			}
			user := User{
				ExternalID: payload.ID,
				Msisdn:     payload.Msisdn,
				Name:       payload.Name,
				Username:   payload.Username,
				CreatedAt:  time.Now(),
			}
			if err := db.Create(&user).Error; err != nil {
				log.Println("Failed to insert user to Postgres:", err)
			} else {
				log.Println("Inserted user to Postgres:", user.Username)
			}
		}
	}()

	// Consume shipment.created asynchronously
	go func() {
		msgs, err := ch.Consume("shipment.created", "", true, false, false, false, nil)
		if err != nil {
			log.Printf("Error consuming shipment.created: %v", err)
			return
		}
		for msg := range msgs {
			log.Println("Received message on shipment.created")
			var payload struct {
				LogisticName   string `json:"logistic_name"`
				TrackingNumber string `json:"tracking_number"`
				Status         string `json:"status"`
				Origin         string `json:"origin"`
				Destination    string `json:"destination"`
				Sender         struct {
					Name    string `json:"name"`
					Phone   string `json:"phone"`
					Address string `json:"address"`
				} `json:"sender"`
				Recipient struct {
					Name    string `json:"name"`
					Phone   string `json:"phone"`
					Address string `json:"address"`
				} `json:"recipient"`
				Items []struct {
					Name     string  `json:"name"`
					Quantity int     `json:"quantity"`
					Weight   float64 `json:"weight"`
				} `json:"items"`
				Notes  string `json:"notes"`
				UserID string `json:"user_id"`
			}

			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				log.Println("Unmarshal shipment.created failed:", err)
				continue
			}

			shipment := Shipment{
				ID:               payload.TrackingNumber,
				LogisticName:     payload.LogisticName,
				TrackingNumber:   payload.TrackingNumber,
				Status:           payload.Status,
				Origin:           payload.Origin,
				Destination:      payload.Destination,
				SenderName:       payload.Sender.Name,
				SenderPhone:      payload.Sender.Phone,
				SenderAddress:    payload.Sender.Address,
				RecipientName:    payload.Recipient.Name,
				RecipientPhone:   payload.Recipient.Phone,
				RecipientAddress: payload.Recipient.Address,
				Notes:            payload.Notes,
				UserID:           payload.UserID,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}

			for _, itm := range payload.Items {
				shipment.Items = append(shipment.Items, ShipmentItem{
					Name:       itm.Name,
					Quantity:   itm.Quantity,
					Weight:     itm.Weight,
					ShipmentID: shipment.ID,
				})
			}

			if err := db.Session(&gorm.Session{FullSaveAssociations: true}).Create(&shipment).Error; err != nil {
				log.Println("Failed to insert shipment to Postgres:", err)
			} else {
				log.Println("Inserted shipment to Postgres with tracking number:", shipment.TrackingNumber)
			}
		}
	}()

	// Consume shipment.updated asynchronously
	go func() {
		msgs, err := ch.Consume("shipment.updated", "", true, false, false, false, nil)
		if err != nil {
			log.Printf("Error consuming shipment.updated: %v", err)
			return
		}
		for msg := range msgs {
			log.Println("Received message on shipment.updated")

			var payload struct {
				ID               string         `json:"id"`
				TrackingNumber   string         `json:"tracking_number"`
				Status           string         `json:"status"`
				Origin           string         `json:"origin"`
				Destination      string         `json:"destination"`
				Notes            string         `json:"notes"`
				UserID           string         `json:"user_id"`
				SenderName       string         `json:"sender_name"`
				SenderPhone      string         `json:"sender_phone"`
				SenderAddress    string         `json:"sender_address"`
				RecipientName    string         `json:"recipient_name"`
				RecipientPhone   string         `json:"recipient_phone"`
				RecipientAddress string         `json:"recipient_address"`
				UpdatedAt        time.Time      `json:"updated_at"`
			}

			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				log.Printf("Failed to unmarshal shipment.updated message: %v", err)
				continue
			}

			var shipment Shipment
			result := db.Where("tracking_number = ?", payload.TrackingNumber).First(&shipment)
			if result.Error != nil {
				if result.Error == gorm.ErrRecordNotFound {
					log.Printf("Shipment not found for update: %s", payload.TrackingNumber)
					continue
				}
				log.Printf("Error finding shipment for update: %v", result.Error)
				continue
			}

			shipment.Status = payload.Status
			shipment.Notes = payload.Notes
			shipment.Origin = payload.Origin
			shipment.Destination = payload.Destination
			shipment.UserID = payload.UserID
			shipment.SenderName = payload.SenderName
			shipment.SenderPhone = payload.SenderPhone
			shipment.SenderAddress = payload.SenderAddress
			shipment.RecipientName = payload.RecipientName
			shipment.RecipientPhone = payload.RecipientPhone
			shipment.RecipientAddress = payload.RecipientAddress
			shipment.UpdatedAt = time.Now()

			if err := db.Save(&shipment).Error; err != nil {
				log.Printf("Failed to update shipment in Postgres: %v", err)
			} else {
				log.Printf("Successfully updated shipment in Postgres: %s", shipment.TrackingNumber)
			}
		}
	}()

	// Prevent main from exiting so all goroutines keep running
	select {}
}
