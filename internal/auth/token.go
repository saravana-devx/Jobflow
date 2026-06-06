package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"jobflow/internal/config"
)

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
)

// JWT settings are validated centrally in config.Load; these just read the
// already-loaded singleton.
func getJWTSecret() []byte   { return []byte(config.Get().AccessSecret) }
func getJWTIssuer() string   { return config.Get().JWTIssuer }
func getJWTAudience() string { return config.Get().JWTAudience }

func CreateToken(userID string) (string, error) {
	jti, err := generateJTI()
	if err != nil {
		return "", err
	}
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer:    getJWTIssuer(),
			Subject:   userID,
			Audience:  jwt.ClaimStrings{getJWTAudience()},
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
		})
	return token.SignedString(getJWTSecret())
}

func generateJTI() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken returns a deterministic SHA-256 hex digest of the token.
// Deterministic hashing lets us look up refresh-token rows by hash. Bcrypt is
// the wrong tool here — it's salted/random and is for low-entropy passwords.
// Refresh tokens already carry 256 bits of entropy from crypto/rand.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func GenerateRefreshToken() (rawToken string, hashedToken string, err error) {
	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return "", "", err
	}
	rawToken = hex.EncodeToString(bytes)
	hashedToken = HashToken(rawToken)
	return rawToken, hashedToken, nil
}

type AccessClaims struct {
	UserID    string
	JTI       string
	ExpiresAt time.Time
}

func ParseAccessToken(tokenString string) (*AccessClaims, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return getJWTSecret(), nil
	},
		jwt.WithIssuer(getJWTIssuer()),
		jwt.WithAudience(getJWTAudience()),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.Subject == "" || claims.ID == "" {
		return nil, fmt.Errorf("missing required claims")
	}
	if claims.ExpiresAt == nil {
		return nil, fmt.Errorf("missing exp claim")
	}
	return &AccessClaims{
		UserID:    claims.Subject,
		JTI:       claims.ID,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}
