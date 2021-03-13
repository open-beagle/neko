package session

import (
	"fmt"
	"strings"
	"net/http"

	"demodesk/neko/internal/types"
)

func (manager *SessionManagerCtx) Authenticate(r *http.Request) (types.Session, error) {
	token, ok := getToken(r)
	if !ok {
		return nil, fmt.Errorf("no authentication provided")
	}

	session, ok := manager.Get(token)
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	return session, nil
}

func getToken(r *http.Request) (string, bool) {
	// get from Cookie
	cookie, err := r.Cookie("neko-token")
	if err == nil {
		return cookie.Value, true
	}

	// get from Header
	reqToken := r.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer ")
	if len(splitToken) == 2 {
		return strings.TrimSpace(splitToken[1]), true
	}

	// get from URL
	token := r.URL.Query().Get("token")
	if token != "" {
		return token, true
	}

	return "", false
}
