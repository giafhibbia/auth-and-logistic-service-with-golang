package main

import (
    "context"
    "os"
    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"

    "auth-service/internal/handler"
    "auth-service/internal/repository"
    "auth-service/internal/middleware"
)

func main() {
    mongoUri := os.Getenv("MONGO_URI")
    client, _ := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoUri))
    db := client.Database("authdb")
    repo := repository.NewUserRepository(db)

    r := gin.Default()
    r.POST("/register", handler.Register(repo))
    r.POST("/login", handler.Login(repo))
    r.Run(":8081")
}
