package main

import (
	"github.com/razzie/beepboop"
)

func AuthMiddleware(username, password string) beepboop.Middleware {
	return func(r *beepboop.PageRequest) *beepboop.View {
		return nil
	}
}
