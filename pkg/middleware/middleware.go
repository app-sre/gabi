package middleware

import (
	"net/http"
)

type ctxKey string

const (
	contextUserKey      ctxKey = "user"
	contentLengthHeader string = "Content-Length"
	forwardedUserHeader string = "X-Forwarded-User"
)

type Middleware func(http.Handler) http.Handler
