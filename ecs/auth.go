package ecs

import "net/http"

// Authentication ...
type Authentication interface {
	SetAuth(*http.Request)
}

// BasicAuth ...
type BasicAuth struct {
	username, password string
}

// SetAuth implements Authentication interface
func (ba BasicAuth) SetAuth(req *http.Request) {
	req.SetBasicAuth(ba.username, ba.password)
}

// TokenAuth ...
type TokenAuth struct {
	token string
}

// SetAuth implements Authentication interface
func (ta TokenAuth) SetAuth(req *http.Request) {
	req.Header.Set("X-SDS-AUTH-TOKEN", ta.token)
}

// Check interface
var (
	_ Authentication = &BasicAuth{}
	_ Authentication = &TokenAuth{}
)
