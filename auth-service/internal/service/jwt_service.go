package service

import (
    "github.com/golang-jwt/jwt/v5"
    "auth-service/internal/model"
    "os"
    "time"
    "errors" // <-- tambahkan ini!
)

func GenerateJWT(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,  // langsung pakai string ID
		"msisdn":   user.Msisdn,
		"username": user.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}


func ParseJWT(tokenStr string) (jwt.MapClaims, error) {
    token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
        return []byte(os.Getenv("JWT_SECRET")), nil
    })
    if err != nil || !token.Valid {
        return nil, err
    }
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, errors.New("invalid claims type")
    }
    return claims, nil
}