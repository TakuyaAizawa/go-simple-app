package middleware

import (
    "net/http"
    "go-simple-app/internal/handlers"
)

func AuthRequired(next http.HandlerFunc) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
        _, err := handlers.GetUserID(r)
        if err != nil {
            http.Error(w, "ログインが必要です", http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}
