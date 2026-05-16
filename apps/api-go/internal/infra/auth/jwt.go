package auth

import (
	"fmt"
	"strconv"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

type JWTTokenService struct {
	secretKey      []byte
	algorithm      string
	expiresMinutes int
}

func NewJWTTokenService(cfg config.Config) JWTTokenService {
	return JWTTokenService{
		secretKey:      []byte(cfg.JWTSecretKey),
		algorithm:      cfg.JWTAlgorithm,
		expiresMinutes: cfg.JWTExpiresMinutes,
	}
}

func (service JWTTokenService) CreateAccessToken(userID int64) (string, error) {
	if service.algorithm != "HS256" {
		return "", fmt.Errorf("unsupported JWT algorithm: %s", service.algorithm)
	}

	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(service.expiresMinutes) * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(service.secretKey)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

func (service JWTTokenService) GetUserID(tokenValue string) (int64, bool) {
	claims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenValue, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected jwt signing method")
		}
		return service.secretKey, nil
	})
	if err != nil || !token.Valid || claims.Subject == "" {
		return 0, false
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return 0, false
	}
	return userID, true
}
