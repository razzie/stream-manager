package main

import (
	"net/http"

	"github.com/razzie/beepboop"
)

func DummyMiddleware(r *beepboop.PageRequest) *beepboop.View {
	return nil
}

func AuthMiddleware(username, password string) beepboop.Middleware {
	if len(username) == 0 && len(password) == 0 {
		return DummyMiddleware
	}
	return func(r *beepboop.PageRequest) *beepboop.View {
		reqUser, reqPass, ok := r.Request.BasicAuth()
		if !ok || reqUser != username || reqPass != password {
			return r.ErrorView("Unauthorized", http.StatusUnauthorized, beepboop.WithHeader("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`))
		}
		return nil
	}
}
