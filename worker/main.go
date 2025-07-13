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

type User struct {
    ID        int       `gorm:"primaryKey;autoIncrement" json:"-"`
    ExternalID string   `json:"id"`           // UUID dari RabbitMQ/microservice
    Msisdn    string    `json:"msisdn"`
    Name      string    `json:"name"`
    Username  string    `json:"username"`
    CreatedAt time.Time `json:"created_at"`
}



type Logistic struct {
	ID              string    `gorm:"primaryKey"`
	LogisticName    string
	Amount          int
	DestinationName string
	OriginName      string
	Duration        string
	CreatedAt       time.Time
}

func main() {
	dsn := os.Getenv("MASTERDB_URL")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("worker: Failed to connect to Postgres: %v", err))
	}

	if err := db.AutoMigrate(&User{}, &Logistic{}); err != nil {
		panic(fmt.Sprintf("AutoMigrate error: %v", err))
	}
	fmt.Println(" [worker] Migrated Postgres schema successfully!")

	// === Retry connect RabbitMQ ===
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

	ch, err := conn.Channel()
	if err != nil {
		panic(fmt.Sprintf("worker: Failed to open RabbitMQ channel: %v", err))
	}
	defer ch.Close()

	// --- WAJIB: Declare queue sebelum Consume ---
	_, err = ch.QueueDeclare(
		"user.registered", // name
		true,              // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		panic(fmt.Sprintf("worker: Failed to declare queue user.registered: %v", err))
	}

	_, err = ch.QueueDeclare(
		"shipment.created", // name
		true,               // durable
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		panic(fmt.Sprintf("worker: Failed to declare queue shipment.created: %v", err))
	}

	// === Listen user.registered ===
	go func() {
		msgs, err := ch.Consume("user.registered", "", true, false, false, false, nil)
		if err != nil {
			log.Printf("Error consume user.registered: %v", err)
			return
		}
		for msg := range msgs {
			var payload struct {
				ID       string `json:"id"` // UUID dari RabbitMQ
				Msisdn   string `json:"msisdn"`
				Name     string `json:"name"`
				Username string `json:"username"`
			}
			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				log.Println("Unmarshal failed:", err)
				continue
			}

			user := User{
				ExternalID: payload.ID, // UUID dari microservice
				Msisdn:     payload.Msisdn,
				Name:       payload.Name,
				Username:   payload.Username,
				CreatedAt:  time.Now(),
			}
			if err := db.Create(&user).Error; err != nil {
				log.Println("Failed insert user to masterdb:", err)
			} else {
				log.Println("Insert user to masterdb:", user.Username)
			}
		}

	}()

	// === Listen shipment.created ===
	msgs, err := ch.Consume("shipment.created", "", true, false, false, false, nil)
	if err != nil {
		panic(fmt.Sprintf("Error consume shipment.created: %v", err))
	}
	for msg := range msgs {
		var logistic Logistic
		if err := json.Unmarshal(msg.Body, &logistic); err != nil {
			log.Println("Unmarshal logistic failed:", err)
			continue
		}
		logistic.CreatedAt = time.Now()
		if err := db.Create(&logistic).Error; err != nil {
			log.Println("Failed insert logistic to masterdb:", err)
		} else {
			log.Println("Insert logistic to masterdb:", logistic.LogisticName)
		}
	}
}
