/*

package jwt

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

// GenerateJWT generates a JWT token for the account
func GenerateJWT(accountNumber int64) (string, error) {
	claims := &jwt.MapClaims{
		"expiresAt":     time.Now().Add(time.Minute * 15).Unix(),
		"accountnumber": accountNumber,
	}

	mySigningKey := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(mySigningKey))
}

// ValidateJWT validates the JWT token from the request header
func ValidateJWT(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
}

// JWTauthMiddleware validates the JWT token and adds account information to the context
func JWTauthMiddleware(handlerFunc http.HandlerFunc, s Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Middleware Authenticating request\n")
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			permissionDenied(w)
			return
		}

		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		token, err := ValidateJWT(tokenString)
		if err != nil || !token.Valid {
			permissionDenied(w)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		accountNumber := int64(claims["accountnumber"].(float64))

		account, err := s.GetAccountByNumber(int(accountNumber))
		if err != nil {
			permissionDenied(w)
			return
		}

		// Add account to the request context
		ctx := context.WithValue(r.Context(), "account", account)
		handlerFunc(w, r.WithContext(ctx))
	}
}

func permissionDenied(w http.ResponseWriter) {
	writeJson(w, http.StatusUnauthorized, APIError{Error: "Permission Denied"})
}

*/

