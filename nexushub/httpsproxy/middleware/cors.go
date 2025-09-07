package middleware

import (
	"net/http"
)

func CorsMiddleware(w http.ResponseWriter, r *http.Request, next func(w http.ResponseWriter, r *http.Request)) {
	// TODO(tom) STOPSHIP: Allow-list certain origins
	w.Header().Set("Access-Control-Allow-Origin", "https://www.yellowstone.localhost:8100")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	next(w, r)
}
