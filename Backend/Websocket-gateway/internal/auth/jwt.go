package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"github.com/yourcompany/websocket-gateway/internal/config"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token expired")
	ErrInvalidIssuer    = errors.New("invalid issuer")
	ErrMissingUserID    = errors.New("missing user id")
)

// Claims represents the JWT claims
type Claims struct {
	UserID    string `json:"user_id"`
	DeviceID  string `json:"device_id"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

// Authenticator handles JWT authentication
type Authenticator struct {
	secret     []byte
	issuer     string
	expiry     time.Duration
	logger     *zap.Logger
}

// NewAuthenticator creates a new JWT authenticator
func NewAuthenticator(cfg *config.Config, logger *zap.Logger) *Authenticator {
	return &Authenticator{
		secret: []byte(cfg.Auth.JWTSecret),
		issuer: "websocket-gateway",
		expiry: cfg.Auth.TokenExpiry,
		logger: logger,
	}
}

// ValidateToken validates a JWT token and returns claims
func (a *Authenticator) ValidateToken(tokenString string) (*Claims, error) {
	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.secret, nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	
	// Validate claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Validate issuer
		if claims.Issuer != a.issuer {
			return nil, ErrInvalidIssuer
		}
		
		// Validate user ID
		if claims.UserID == "" {
			return nil, ErrMissingUserID
		}
		
		return claims, nil
	}
	
	return nil, ErrInvalidToken
}

// GenerateToken generates a new JWT token for a user
func (a *Authenticator) GenerateToken(userID, deviceID string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		DeviceID: deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(a.expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    a.issuer,
			ID:        fmt.Sprintf("%s-%d", userID, now.UnixNano()),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.secret)
}

// ExtractFromContext extracts authentication from HTTP request context
func (a *Authenticator) ExtractFromContext(ctx context.Context) (*Claims, error) {
	// Get token from context
	tokenValue := ctx.Value("token")
	if tokenValue == nil {
		return nil, ErrInvalidToken
	}
	
	tokenString, ok := tokenValue.(string)
	if !ok {
		return nil, ErrInvalidToken
	}
	
	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	
	return a.ValidateToken(tokenString)
}

// Middleware creates an HTTP middleware for JWT authentication
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Try WebSocket protocol header
			authHeader = r.Header.Get("Sec-WebSocket-Protocol")
		}
		
		if authHeader == "" {
			http.Error(w, "authorization header required", http.StatusUnauthorized)
			return
		}
		
		// Remove "Bearer " prefix
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		
		// Validate token
		claims, err := a.ValidateToken(tokenString)
		if err != nil {
			a.logger.Debug("jwt validation failed", 
				zap.Error(err),
				zap.String("token", tokenString[:min(len(tokenString), 20)]))
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		
		// Add claims to context
		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
