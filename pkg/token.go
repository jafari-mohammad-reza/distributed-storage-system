package pkg

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateApiKey(email, agent string) (string, error) {
	claims := jwt.MapClaims{}
	claims["email"] = email
	claims["agent"] = agent
	claims["iat"] = time.Now().Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
func DecodeToken(token string) (jwt.MapClaims, error) {
	t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, err
	}
	return claims, nil
}
