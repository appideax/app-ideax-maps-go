package middleware

import (
	"fmt"
	"strings"
	"time"

	"app-ideax-maps-go/config"
	"app-ideax-maps-go/shared/response"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const MapClaimsContextKey = "map_claims"

type MapClaims struct {
	UserID    string   `json:"user_id"`
	CompanyID string   `json:"company_id"`
	Roles     []string `json:"roles"`
	RideID    string   `json:"ride_id,omitempty"`
	jwt.RegisteredClaims
}

func extractToken(c *fiber.Ctx, cookieName string) string {
	if cookie := c.Cookies(cookieName); cookie != "" {
		return cookie
	}

	authHeader := c.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	if token := c.Query("token"); token != "" {
		return token
	}

	return ""
}

const (
	expectedIssuer   = "taxi-back-go"
	expectedAudience = "maps-api"
)

func parseAndValidate(tokenString string, secret string) (*MapClaims, error) {
	claims := &MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.Issuer != expectedIssuer {
		return nil, fmt.Errorf("invalid issuer")
	}
	if !audienceContains(claims.Audience, expectedAudience) {
		return nil, fmt.Errorf("invalid audience")
	}
	if claims.UserID == "" && claims.RideID == "" {
		return nil, fmt.Errorf("missing user_id or ride_id")
	}
	return claims, nil
}

func audienceContains(aud jwt.ClaimStrings, value string) bool {
	for _, v := range aud {
		if v == value {
			return true
		}
	}
	return false
}

func Auth(cfg config.JWTConfig, cookieName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := extractToken(c, cookieName)
		if tokenString == "" {
			return response.Error(c, "missing or invalid token", fiber.StatusUnauthorized)
		}

		claims, err := parseAndValidate(tokenString, cfg.Secret)
		if err != nil {
			return response.Error(c, "invalid or expired token", fiber.StatusUnauthorized)
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("company_id", claims.CompanyID)
		c.Locals("roles", claims.Roles)
		c.Locals("ride_id", claims.RideID)
		c.Locals(MapClaimsContextKey, claims)

		return c.Next()
	}
}

func BuildToken(secret string, ttl time.Duration, userID, companyID, rideID string, roles []string) (string, error) {
	now := time.Now().UTC()
	claims := MapClaims{
		UserID:    userID,
		CompanyID: companyID,
		Roles:     roles,
		RideID:    rideID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    expectedIssuer,
			Audience:  jwt.ClaimStrings{expectedAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
