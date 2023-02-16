package middleware

import (
	"net/http"
)

type ctxKey string

const (
	ContextKeyUser  ctxKey = "user"
	ContextKeyQuery ctxKey = "query"
)

const (
	contentLengthHeader = "Content-Length"
	forwardedUserHeader = "X-Forwarded-User"
)

type Middleware func(http.Handler) http.Handler
