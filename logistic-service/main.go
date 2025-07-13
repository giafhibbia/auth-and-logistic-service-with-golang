package main

import (
    "context"
    "fmt"
    "os"
    "time"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
    mongoUri := os.Getenv("MONGO_URI")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoUri))
    if err != nil {
        panic(fmt.Sprintf("logistic-service: Failed to connect to MongoDB: %v", err))
    }
    err = client.Ping(ctx, nil)
    if err != nil {
        panic(fmt.Sprintf("logistic-service: MongoDB not reachable: %v", err))
    }
    fmt.Println(" [logistic-service] Connected to MongoDB successfully!")
    select {}
}
