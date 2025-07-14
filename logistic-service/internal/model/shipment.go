// model/shipment.go
package model

import (
	"time"
	"gorm.io/gorm"
)
// ShipmentItem represents a single item in a shipment order.
// Contains the item name, quantity, and weight (in kg).
type ShipmentItem struct {
	Name   string  `bson:"name" json:"name"`     // Name of the item
	Qty    int     `bson:"qty" json:"qty"`       // Quantity of the item
	Weight float64 `bson:"weight" json:"weight"` // Weight per item in kilograms
}

// Shipment represents the main shipment order data.
// It stores information about the shipment, sender, recipient, and related metadata.
type Shipment struct {
	ID             string         `json:"id"`
	LogisticName   string         `json:"logistic_name"`
	TrackingNumber string         `json:"tracking_number"`
	Status         string         `json:"status"`
	Origin         string         `json:"origin"`
	Destination    string         `json:"destination"`
	Notes          string         `json:"notes,omitempty"`
	UserID         string         `gorm:"index;column:user_id" json:"user_id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`

	// Flat columns for sender and recipient info
	SenderName       string `gorm:"column:sender_name"`
	SenderPhone      string `gorm:"column:sender_phone"`
	SenderAddress    string `gorm:"column:sender_address"`
	RecipientName    string `gorm:"column:recipient_name"`
	RecipientPhone   string `gorm:"column:recipient_phone"`
	RecipientAddress string `gorm:"column:recipient_address"`

	// Nested structs for JSON bind/unbind
	Sender    ShipmentPerson `gorm:"-" json:"sender"`    // ignore by GORM
	Recipient ShipmentPerson `gorm:"-" json:"recipient"` // ignore by GORM

	Items []ShipmentItem `json:"items"`
}

func (s *Shipment) BeforeSave(tx *gorm.DB) (err error) {
	s.SenderName = s.Sender.Name
	s.SenderPhone = s.Sender.Phone
	s.SenderAddress = s.Sender.Address
	s.RecipientName = s.Recipient.Name
	s.RecipientPhone = s.Recipient.Phone
	s.RecipientAddress = s.Recipient.Address
	return nil
}



// ShipmentPerson represents a person involved in shipment (sender or recipient).
// Useful if you want to group person-related fields as a nested struct.
type ShipmentPerson struct {
	Name    string `bson:"name" json:"name"`       // Person's full name
	Phone   string `bson:"phone" json:"phone"`     // Person's phone number
	Address string `bson:"address" json:"address"` // Person's address
}
