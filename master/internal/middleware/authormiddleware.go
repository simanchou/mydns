package middleware

import (
	"fmt"
	"net/http"
)

type AuthorMiddleware struct {
}

func NewAuthorMiddleware() *AuthorMiddleware {
	return &AuthorMiddleware{}
}

var (
	ApiKey    string
	ApiSecret string
)

func (m *AuthorMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authorKeyFromReq := r.Header.Get("Authorization")
		if authorKeyFromReq != fmt.Sprintf("sso-key %s:%s", ApiKey, ApiSecret) {
			w.WriteHeader(401)
			w.Write([]byte("Not Allow"))
		} else {
			next(w, r)
		}
	}
}
