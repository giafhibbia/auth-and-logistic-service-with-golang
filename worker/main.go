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

// User represents the user model mapped to the database table for storing user data
type User struct {
	ID         int       `gorm:"primaryKey;autoIncrement" json:"-"` // Primary key, auto increment, hidden in JSON
	ExternalID string    `json:"id"`                                // UUID coming from RabbitMQ or other microservice
	Msisdn     string    `json:"msisdn"`                            // User's phone number
	Name       string    `json:"name"`                              // User's full name
	Username   string    `json:"username"`                          // User's username
	CreatedAt  time.Time `json:"created_at"`                       // Timestamp when the user was created
}

// Logistic represents the logistic (courier) information
type Logistic struct {
	ID              string    `gorm:"primaryKey"`       // Primary key ID for logistic company
	LogisticName    string                              // Name of logistic company
	Amount          int                                 // Price or amount for the service
	DestinationName string                              // Destination location name
	OriginName      string                              // Origin location name
	Duration        string                              // Estimated delivery duration
	CreatedAt       time.Time                           // Timestamp of record creation
}

// ShipmentItem represents a single item within a shipment order
type ShipmentItem struct {
	ID         uint    `gorm:"primaryKey;autoIncrement"`          // Primary key, auto increment
	ShipmentID string  `gorm:"index;not null"`                    // Foreign key to Shipment.ID, indexed and required
	Name       string                              // Item name
	Quantity   int                                 // Quantity of the item
	Weight     float64                             // Weight per item (kg)
}

// Shipment represents the main shipment order data including metadata and nested items
type Shipment struct {
	ID               string         `gorm:"primaryKey;column:id" json:"id"`                    // UUID, primary key
	LogisticName     string         `gorm:"column:logistic_name" json:"logistic_name"`         // Name of the logistic company
	TrackingNumber   string         `gorm:"column:tracking_number" json:"tracking_number"`     // Unique tracking number
	Status           string         `gorm:"column:status" json:"status"`                        // Shipment status
	Origin           string         `gorm:"column:origin" json:"origin"`                        // Origin location
	Destination      string         `gorm:"column:destination" json:"destination"`             // Destination location
	Notes            string         `gorm:"column:notes" json:"notes"`                          // Optional notes for shipment
	UserID           string         `gorm:"column:user_id" json:"user_id"`                      // User ID of the creator (indexed)
	CreatedAt        time.Time      `gorm:"column:created_at" json:"created_at"`                // Created timestamp
	UpdatedAt        time.Time      `gorm:"column:updated_at" json:"updated_at"`                // Last updated timestamp
	SenderName       string         `gorm:"column:sender_name" json:"sender_name"`              // Sender's name
	SenderPhone      string         `gorm:"column:sender_phone" json:"sender_phone"`            // Sender's phone number
	SenderAddress    string         `gorm:"column:sender_address" json:"sender_address"`        // Sender's address
	RecipientName    string         `gorm:"column:recipient_name" json:"recipient_name"`        // Recipient's name
	RecipientPhone   string         `gorm:"column:recipient_phone" json:"recipient_phone"`      // Recipient's phone number
	RecipientAddress string         `gorm:"column:recipient_address" json:"recipient_address"`  // Recipient's address
	Items            []ShipmentItem `gorm:"foreignKey:ShipmentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"items"` // Shipment items with cascading update/delete
}

func main() {
	// Get Postgres connection string from environment variable
	dsn := os.Getenv("MASTERDB_URL")

	// Open connection to Postgres using GORM
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("worker: Failed to connect to Postgres: %v", err))
	}

	// Automatically migrate schema (creates/updates tables as needed)
	if err := db.AutoMigrate(&User{}, &Logistic{}, &Shipment{}, &ShipmentItem{}); err != nil {
		panic(fmt.Sprintf("AutoMigrate error: %v", err))
	}
	fmt.Println(" [worker] Migrated Postgres schema successfully!")

	// Attempt to connect to RabbitMQ with retry logic (max 10 tries, 3 seconds apart)
	var conn *amqp.Connection
	rabbitURL := os.Getenv("RABBITMQ_URL")
	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(rabbitURL)
		if err == nil {
			fmt.Println(" [worker] Connected to RabbitMQ successfully!")
			break
		}
		fmt.Println(" [worker] Waiting for RabbitMQ... retry in 3s")
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		panic(fmt.Sprintf("worker: Failed to connect to RabbitMQ: %v", err))
	}
	defer conn.Close()

	// Open a channel on RabbitMQ connection
	ch, err := conn.Channel()
	if err != nil {
		panic(fmt.Sprintf("worker: Failed to open RabbitMQ channel: %v", err))
	}
	defer ch.Close()

	// Declare durable queue for user registration events
	_, err = ch.QueueDeclare(
		"user.registered", // queue name
		true,              // durable (survives RabbitMQ restart)
		false,             // auto-delete
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		panic(fmt.Sprintf("worker: Failed to declare queue user.registered: %v", err))
	}

	// Declare durable queue for shipment creation events
	_, err = ch.QueueDeclare(
		"shipment.created",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(fmt.Sprintf("worker: Failed to declare queue shipment.created: %v", err))
	}

	// Start asynchronous goroutine to consume user.registered messages
	go func() {
		msgs, err := ch.Consume(
			"user.registered", // queue
			"",                // consumer tag, empty for auto-generated
			true,              // auto-acknowledge
			false,             // exclusive
			false,             // no-local (not supported by RabbitMQ)
			false,             // no-wait
			nil,               // args
		)
		if err != nil {
			log.Printf("Error consuming user.registered: %v", err)
			return
		}
		for msg := range msgs {
			// Define expected JSON payload structure for user registration
			var payload struct {
				ID       string `json:"id"`
				Msisdn   string `json:"msisdn"`
				Name     string `json:"name"`
				Username string `json:"username"`
			}
			// Parse JSON payload
			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				log.Println("Unmarshal failed:", err)
				continue
			}

			// Map payload to User struct
			user := User{
				ExternalID: payload.ID,
				Msisdn:     payload.Msisdn,
				Name:       payload.Name,
				Username:   payload.Username,
				CreatedAt:  time.Now(),
			}
			// Insert user data into Postgres
			if err := db.Create(&user).Error; err != nil {
				log.Println("Failed insert user to masterdb:", err)
			} else {
				log.Println("Insert user to masterdb:", user.Username)
			}
		}
	}()

	// Consume shipment.created messages synchronously (blocking)
	msgs, err := ch.Consume(
		"shipment.created", // queue
		"",
		true,  // auto-acknowledge
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		panic(fmt.Sprintf("Error consume shipment.created: %v", err))
	}
	log.Println("Worker started consuming shipment.created")

	// Loop to process each shipment.created message
	for msg := range msgs {
		log.Println("Received message on shipment.created")

		// Define expected JSON structure for shipment.created payload
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
			UserID string `json:"user_id"` // User ID of the creator, can be empty
		}

		// Log raw message body for debugging
		log.Println("=== RAW MESSAGE BEGIN ===")
		log.Printf("%s\n", string(msg.Body))
		log.Println("=== RAW MESSAGE END ===")

		// Unmarshal JSON message into the payload struct
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			log.Println("Unmarshal shipment failed:", err)
			continue // skip to next message on error
		}

		// Log parsed payload for debugging
		log.Printf("Parsed payload: %+v\n", payload)

		// Map payload to Shipment struct for saving in DB
		shipment := Shipment{
			ID:               payload.TrackingNumber, // Use tracking number as unique ID
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
		}

		// Append shipment items to the shipment
		for _, itm := range payload.Items {
			shipment.Items = append(shipment.Items, ShipmentItem{
				Name:       itm.Name,
				Quantity:   itm.Quantity,
				Weight:     itm.Weight,
				ShipmentID: shipment.ID, // Link item to shipment via FK
			})
		}

		// Insert shipment along with items into the database with full save of associations
		if err := db.Session(&gorm.Session{FullSaveAssociations: true}).Create(&shipment).Error; err != nil {
			log.Println("Failed insert shipment to masterdb:", err)
		} else {
			log.Println("Insert shipment to masterdb with tracking number:", shipment.TrackingNumber)
		}
	}
}
