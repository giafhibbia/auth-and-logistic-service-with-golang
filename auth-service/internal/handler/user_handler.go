package handler

import (
	"github.com/gin-gonic/gin"
	"auth-service/internal/model"
	"auth-service/internal/repository"
	"auth-service/internal/service"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
	"encoding/json"
	"os"
	amqp "github.com/rabbitmq/amqp091-go"
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
		hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		user := model.User{
			Msisdn:   req.Msisdn,
			Name:     req.Name,
			Username: req.Username,
			Password: string(hashed),
		}
		repo.CreateUser(&user)
		// Publish to RabbitMQ
		conn, _ := amqp.Dial(os.Getenv("RABBITMQ_URL"))
		ch, _ := conn.Channel()
		defer ch.Close()
		body, _ := json.Marshal(user)
		ch.Publish("", "user.registered", false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
		c.JSON(http.StatusCreated, gin.H{"message": "user registered"})
	}
}
