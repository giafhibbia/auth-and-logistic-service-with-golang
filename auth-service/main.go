package main

import (
    "context"
    "os"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/cors"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"

    "auth-service/internal/handler"
    "auth-service/internal/repository"
    "auth-service/internal/middleware"
)

func main() {
    mongoUri := os.Getenv("MONGO_URI")
    client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoUri))
    if err != nil {
        panic(err)
    }
    db := client.Database("authdb")
    repo := repository.NewUserRepository(db)

    r := gin.Default()

    // Tambahkan konfigurasi CORS
   r.Use(cors.New(cors.Config{
    AllowAllOrigins:  true,
    AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
}))


    r.POST("/register", handler.Register(repo))
    r.POST("/login", handler.Login(repo))
    r.GET("/profile", middleware.JWTAuthMiddleware(), handler.Profile())

    r.Run(":8081")
}
