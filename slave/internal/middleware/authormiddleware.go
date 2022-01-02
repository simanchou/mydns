package middleware

import (
	"net/http"
	"strings"
)

type AuthorMiddleware struct {
}

func NewAuthorMiddleware() *AuthorMiddleware {
	return &AuthorMiddleware{}
}

var (
	MasterIp string
)

func (m *AuthorMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if MasterIp != strings.Split(r.RemoteAddr, ":")[0] {
			w.WriteHeader(401)
			w.Write([]byte("Not Allow, Special Master Allow Only!"))
		} else {
			next(w, r)
		}
	}
}
