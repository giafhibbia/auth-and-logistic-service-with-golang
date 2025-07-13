package repository

import (
	"context"
	"time"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"auth-service/internal/model"
)

type UserRepository struct{ col *mongo.Collection }

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{col: db.Collection("users")}
}

func (r *UserRepository) CreateUser(user *model.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.col.InsertOne(ctx, user)
	return err
}
func (r *UserRepository) FindByUsername(username string) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var user model.User
	err := r.col.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	return &user, err
}
func (r *UserRepository) FindByMsisdn(msisdn string) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var user model.User
	err := r.col.FindOne(ctx, bson.M{"msisdn": msisdn}).Decode(&user)
	return &user, err
}
