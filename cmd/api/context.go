package main

import (
	"context"
	"net/http"

	"github.com/Crocmagnon/greenlight/internal/data"
)

type contextKey string

const userContextKey = contextKey("user")

func (*application) contextSetUser(r *http.Request, user *data.User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userContextKey, user))
}

func (*application) contextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}

	return user
}
