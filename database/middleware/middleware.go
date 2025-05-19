package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tomyedwab/yesterday/users/util"
)

func LoginRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("DUMMY_LOGIN_PROFILE") != "" {
			// For dev only, skip authentication and set a dummy profile
			profileValue := os.Getenv("DUMMY_LOGIN_PROFILE")
			claimValue := util.YesterdayUserClaims{
				Profile: profileValue,
			}
			nextRequest := r.WithContext(context.WithValue(r.Context(), util.ClaimsKey, &claimValue))
			next.ServeHTTP(w, nextRequest)
			return
		}

		// Get bearer token from request
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if !strings.HasPrefix(token, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Decode JWT token
		var claimValue util.YesterdayUserClaims
		tokenString := strings.TrimPrefix(token, "Bearer ")
		claims, err := jwt.ParseWithClaims(tokenString, &claimValue, func(token *jwt.Token) (interface{}, error) {
			return util.LoadJWTSecretKey()
		})
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if the token is valid
		if !claims.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		nextRequest := r.WithContext(context.WithValue(r.Context(), util.ClaimsKey, &claimValue))

		next.ServeHTTP(w, nextRequest)
	}
}

func LogRequests(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Call the next handler
		next.ServeHTTP(w, r)

		// Log the request details
		duration := time.Since(start)
		fmt.Printf("%s - %s %s %s - %v\n",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			r.Proto,
			duration,
		)
	}
}

func EnableCrossOrigin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("ENABLE_CROSS_ORIGIN") != "" {
			// For dev only, enable CORS for all origins
			fmt.Printf("%s - %s %s Enable Cross Origin\n",
				r.RemoteAddr,
				r.Method,
				r.URL.Path,
			)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			// Do not call through to the handler itself, just return immediately
			return
		}

		next.ServeHTTP(w, r)
	}
}

// Combine multiple middleware functions
func Chain(h http.HandlerFunc, middleware ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, m := range middleware {
		h = m(h)
	}
	return h
}

// Apply the default middlewars in the correct order
func ApplyDefault(h http.HandlerFunc) http.HandlerFunc {
	return Chain(
		h,
		LoginRequired,
		EnableCrossOrigin,
		LogRequests,
	)
}
