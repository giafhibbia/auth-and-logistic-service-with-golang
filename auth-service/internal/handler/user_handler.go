package handler

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
    "auth-service/internal/model"
    "auth-service/internal/repository"
    "auth-service/internal/service"
    "net/http"
    "strings"
    "os"
    "encoding/json"
    amqp "github.com/rabbitmq/amqp091-go"
    jwt "github.com/golang-jwt/jwt/v5"
    "fmt"
)

type RegisterRequest struct {
    Msisdn   string `json:"msisdn" binding:"required"`
    Name     string `json:"name" binding:"required"`
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

func Register(repo *repository.UserRepository) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req RegisterRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        if !strings.HasPrefix(req.Msisdn, "62") {
            c.JSON(http.StatusBadRequest, gin.H{"error": "msisdn must start with 62"})
            return
        }
        if _, err := repo.FindByUsername(req.Username); err == nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "username exists"})
            return
        }
        if _, err := repo.FindByMsisdn(req.Msisdn); err == nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "msisdn exists"})
            return
        }
        uuidStr := uuid.New().String()
        hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
        user := model.User{
            ID:       uuidStr,
            Msisdn:   req.Msisdn,
            Name:     req.Name,
            Username: req.Username,
            Password: string(hashed),
        }
        if err := repo.CreateUser(&user); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert user"})
            return
        }

        // === Debug RabbitMQ publish ===
        rabbitURL := os.Getenv("RABBITMQ_URL")
        fmt.Println("RABBITMQ_URL:", rabbitURL)
        conn, err := amqp.Dial(rabbitURL)
        if err != nil {
            fmt.Println("[Register] RabbitMQ connection error:", err)
        } else {
            defer conn.Close()
            ch, err := conn.Channel()
            if err != nil {
                fmt.Println("[Register] RabbitMQ channel error:", err)
            } else {
                defer ch.Close()
                body, err := json.Marshal(user)
                if err != nil {
                    fmt.Println("[Register] Marshal user to JSON error:", err)
                } else {
                    err = ch.Publish("", "user.registered", false, false, amqp.Publishing{
                        ContentType: "application/json",
                        Body:        body,
                    })
                    if err != nil {
                        fmt.Println("[Register] RabbitMQ publish error:", err)
                    } else {
                        fmt.Println("[Register] RabbitMQ publish OK (user.registered)")
                    }
                }
            }
        }

        c.JSON(http.StatusCreated, gin.H{"message": "user registered"})
    }
}


type LoginRequest struct {
    Msisdn   string `json:"msisdn" binding:"required"`
    Password string `json:"password" binding:"required"`
}
func Login(repo *repository.UserRepository) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req LoginRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        user, err := repo.FindByMsisdn(req.Msisdn)
        if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
            return
        }
        // Kirim objek user, bukan user.ID
        token, err := service.GenerateJWT(user)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
            return
        }
        c.JSON(http.StatusOK, gin.H{"token": token})
    }
}


func Profile() gin.HandlerFunc {
    return func(c *gin.Context) {
        claimsRaw := c.MustGet("claims")
        claims, ok := claimsRaw.(jwt.MapClaims)
        if !ok {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid claims type"})
            return
        }
        c.JSON(http.StatusOK, claims)
    }
}
