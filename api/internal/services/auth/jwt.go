package auth

import (
	"fmt"
	"time"

	repo "github.com/CoconutOrg/CoconutMaster/internal/adapters/sqlc"

	"github.com/golang-jwt/jwt/v4"
)

func CreateJWT(secret []byte, user *repo.User) (string, error) {
	expiration := time.Second * time.Duration(60)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID":    user.ID,
		"email":     user.Email,
		"expiredAt": time.Now().Add(expiration).Unix(),
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, err
}

func ParseToken(secret []byte, tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		return nil, err
	}

	return token, err
}

func GetClaims(token *jwt.Token) (jwt.MapClaims, bool) {
	claims, ok := token.Claims.(jwt.MapClaims)
	return claims, ok
}

func ValidateClaims(token *jwt.Token, user *repo.User) (bool, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ok, nil
	}

	userIDFloat, ok := claims["userID"].(float64)
	if !ok {
		return false, fmt.Errorf("claim missing")
	}
	email, ok := claims["email"].(string)
	if !ok {
		return false, fmt.Errorf("claim missing")
	}
	expiredFloat, ok := claims["expiredAt"].(float64)
	if !ok {
		return false, fmt.Errorf("claim missing")
	}

	if (int64(userIDFloat) != user.ID) ||
		(email != user.Email) ||
		(int64(expiredFloat) <= time.Now().Unix()) {
		return false, fmt.Errorf("Token expired")
	}

	return true, nil
}

func ValidateToken(secret []byte, tokenString string, user *repo.User) (bool, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return false, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ok, nil
	}

	if (int64(claims["userID"].(float64)) != user.ID) ||
		(claims["email"].(string) != user.Email) ||
		(int64(claims["expiredTime"].(float64)) <= time.Now().Unix()) {
		return false, fmt.Errorf("Token expired")
	}

	return true, nil
}
