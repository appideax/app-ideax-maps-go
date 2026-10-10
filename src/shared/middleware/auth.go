package middleware

import (
	"errors"
	"strings"
	"time"

	"app-ideax-maps-go/config"
	"app-ideax-maps-go/shared/response"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// Contrato con taxi-back-go (modulo maps): firma HS256 con MAPS_JWT_SECRET,
// este emisor y esta audiencia, y un `sub` que identifica la sesion.
const (
	TokenIssuer   = "taxi-back-go"
	TokenAudience = "maps-api"
)

const (
	SubjectLocalKey = "maps_subject"
	CompanyLocalKey = "company_id"
)

type MapClaims struct {
	CompanyID string `json:"company_id,omitempty"`
	jwt.RegisteredClaims
}

// Auth acepta el token de mapas en `Authorization: Bearer` (app movil) o en la
// cookie de sesion (navegador). No se acepta en la URL: quedaria en los logs.
func Auth(cfg config.AuthConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		candidates := tokenCandidates(c, cfg.CookieName)
		if len(candidates) == 0 {
			return response.Error(c, "missing token", fiber.StatusUnauthorized)
		}

		for _, candidate := range candidates {
			claims, err := parseToken(candidate, cfg.JWTSecret)
			if err != nil {
				continue
			}
			c.Locals(SubjectLocalKey, claims.Subject)
			c.Locals(CompanyLocalKey, claims.CompanyID)
			return c.Next()
		}

		return response.Error(c, "invalid or expired token", fiber.StatusUnauthorized)
	}
}

func tokenCandidates(c *fiber.Ctx, cookieName string) []string {
	candidates := make([]string, 0, 2)
	if scheme, token, found := strings.Cut(c.Get(fiber.HeaderAuthorization), " "); found && scheme == "Bearer" {
		if token = strings.TrimSpace(token); token != "" {
			candidates = append(candidates, token)
		}
	}
	if cookie := c.Cookies(cookieName); cookie != "" {
		candidates = append(candidates, cookie)
	}
	return candidates
}

func parseToken(tokenString, secret string) (*MapClaims, error) {
	claims := &MapClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(TokenIssuer),
		jwt.WithAudience(TokenAudience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if claims.Subject == "" {
		return nil, errors.New("missing subject")
	}
	return claims, nil
}

// BuildToken firma un token como lo hace taxi-back-go. Solo lo usan las pruebas.
func BuildToken(secret string, ttl time.Duration, subject, companyID string) (string, error) {
	now := time.Now()
	claims := MapClaims{
		CompanyID: companyID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    TokenIssuer,
			Audience:  jwt.ClaimStrings{TokenAudience},
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
