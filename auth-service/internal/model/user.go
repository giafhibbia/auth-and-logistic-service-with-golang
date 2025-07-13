package model

type User struct {
	ID       string 			`bson:"_id,omitempty" json:"id"`
	Msisdn   string             `bson:"msisdn" json:"msisdn"`
	Name     string             `bson:"name" json:"name"`
	Username string             `bson:"username" json:"username"`
	Password string             `bson:"password" json:"-"`
}
