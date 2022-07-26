package middlewares

import (
	"context"
	"net/http"
	"strings"

	"ucrm/app/auth"
	"ucrm/app/config"

	"github.com/dgrijalva/jwt-go/v4"

	"ucrm/pkg/logger"
)

type AuthGuard struct {
	cfg config.JWTConfig
}

func NewAuthGuard(cfg config.JWTConfig) *AuthGuard {
	return &AuthGuard{
		cfg: cfg,
	}
}

func (ag *AuthGuard) Next() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := strings.Split(r.Header.Get("Authorization"), "Bearer ")
			if len(authHeader) != 2 {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Malformed Token"))
				return
			} else {
				jwtToken := authHeader[1]
				customClaims := &auth.Claims{}

				token, err := jwt.ParseWithClaims(jwtToken, customClaims, func(token *jwt.Token) (interface{}, error) {
					return []byte(ag.cfg.SigningKey), nil
				})
				if err != nil || !token.Valid {
					logger.Logger.Error("[AuthGuard] Error :%s", err.Error())
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Unauthorized"))
					return
				}

				ctx := context.WithValue(r.Context(), auth.ContextUserKey, customClaims.Id)
				next.ServeHTTP(w, r.WithContext(ctx))
			}
		})
	}
}
