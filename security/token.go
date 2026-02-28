package security

import (
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secret []byte
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{
		secret: []byte(secret),
	}
}

type Claims struct {
	jwt.RegisteredClaims
	UserId string `json:"user_id"`
}

func (s *JWTService) GenerateToken(id int64) (string, error) {
	now := time.Now()
	claims := Claims{
		UserId: strconv.FormatInt(id, 10),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *JWTService) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	if claims == nil {
		return nil, jwt.ErrSignatureInvalid
	}
	issuedAt := claims.IssuedAt.Time

	// Check if the token is expired. token is valid for 3 minutes
	if time.Now().After(issuedAt.Add(time.Minute * 3)) {
		return nil, jwt.ErrTokenExpired
	}

	return claims, nil
}
