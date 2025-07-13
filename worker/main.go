package main

import (
    "fmt"
    "os"
    "time"
    "context"
    "github.com/rabbitmq/amqp091-go"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    // Test connect ke RabbitMQ
    rabbitUrl := os.Getenv("RABBITMQ_URL")
    conn, err := amqp091.Dial(rabbitUrl)
    if err != nil {
        panic(fmt.Sprintf("worker: Failed to connect to RabbitMQ: %v", err))
    }
    defer conn.Close()
    fmt.Println(" [worker] Connected to RabbitMQ successfully!")

    // Test connect ke Postgres
    pgDsn := os.Getenv("MASTERDB_URL")
    db, err := gorm.Open(postgres.Open(pgDsn), &gorm.Config{})
    if err != nil {
        panic(fmt.Sprintf("worker: Failed to connect to Postgres: %v", err))
    }
    sqlDB, _ := db.DB()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    err = sqlDB.PingContext(ctx)
    if err != nil {
        panic(fmt.Sprintf("worker: Postgres not reachable: %v", err))
    }
    fmt.Println(" [worker] Connected to Postgres successfully!")

    select {}
}
