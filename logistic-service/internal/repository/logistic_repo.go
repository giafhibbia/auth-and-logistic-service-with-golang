package repository

import (
	"context"
	"time"

	"logistic-service/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ShipmentRepository handles CRUD operations on the "shipments" MongoDB collection
type ShipmentRepository struct {
	col *mongo.Collection
}

// NewShipmentRepository creates a new ShipmentRepository bound to the "shipments" collection
func NewShipmentRepository(db *mongo.Database) *ShipmentRepository {
	return &ShipmentRepository{col: db.Collection("shipments")}
}

// Insert inserts a new shipment document into the shipments collection
func (r *ShipmentRepository) Insert(shipment *model.Shipment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.col.InsertOne(ctx, shipment)
	return err
}

// UpdateStatus updates the status and updated_at timestamp of a shipment identified by trackingNumber
func (r *ShipmentRepository) UpdateStatus(trackingNumber string, status string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First find the shipment
	var shipment model.Shipment
	err := r.col.FindOne(ctx, bson.M{"trackingnumber": trackingNumber}).Decode(&shipment)
	if err != nil {
		return err
	}

	// Copy nested to flat fields before update
	updateFields := bson.M{
		"status":          status,
		"updated_at":      time.Now().Unix(),
		"sendername":      shipment.Sender.Name,
		"senderphone":     shipment.Sender.Phone,
		"senderaddress":   shipment.Sender.Address,
		"recipientname":   shipment.Recipient.Name,
		"recipientphone":  shipment.Recipient.Phone,
		"recipientaddress": shipment.Recipient.Address,
	}

	_, err = r.col.UpdateOne(ctx, bson.M{"trackingnumber": trackingNumber}, bson.M{"$set": updateFields})
	return err
}


// FindByTrackingNumber retrieves a shipment document by its tracking number.
// Returns (nil, nil) if shipment not found.
func (r *ShipmentRepository) FindByTrackingNumber(trackingNumber string) (*model.Shipment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var shipment model.Shipment
	err := r.col.FindOne(ctx, bson.M{"trackingnumber": trackingNumber}).Decode(&shipment)
	if err == mongo.ErrNoDocuments {
		return nil, nil // Tidak ditemukan
	}
	return &shipment, err
}


// FindByOriginDestination finds courier rates by origin_name and destination_name
func (r *ShipmentRepository) FindByOriginDestination(origin, destination string) ([]*model.Shipment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{"origin_name": origin, "destination_name": destination}
	cursor, err := r.col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	var results []*model.Shipment
	err = cursor.All(ctx, &results)
	return results, err
}

// FindByUserID retrieves all shipment documents associated with a given user ID
func (r *ShipmentRepository) FindByUserID(userID string) ([]*model.Shipment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Filter query MongoDB dengan user_id
	filter := bson.M{"userid": userID}

	// Cari semua dokumen shipment dengan user_id tersebut
	cursor, err := r.col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Decode hasil cursor ke slice hasil []*model.Shipment
	var results []*model.Shipment
	err = cursor.All(ctx, &results)
	return results, err
}

