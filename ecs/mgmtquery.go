package ecs

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// NewMgmtClient ...
func NewMgmtClient(name, username, password string, ecs *Ecs, reqTimeout, blockDur time.Duration) *MgmtClient {
	return &MgmtClient{
		Name:     name,
		username: username,
		password: password,
		ecs:      ecs,
		blockDur: blockDur,
		mutex:    &sync.Mutex{},
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: reqTimeout,
		},
	}
}

// MgmtClient defines client for ECS mgmt
type MgmtClient struct {
	Name     string
	username string
	password string
	ecs      *Ecs
	blockDur time.Duration
	token    *Token
	mutex    *sync.Mutex
	client   *http.Client
}

// PerformRequest sends request to ECS
func (e *MgmtClient) PerformRequest(method, uri string, body io.Reader, bodyLength int64, headers http.Header, auth Authentication, vdc string) (*http.Response, error) {
	return e.PerformRequestBase(method, "https", "", uri, body, bodyLength, headers, auth, vdc)
}

func parseHostPort(s string) (hostname, port string) {
	pair := strings.Split(s, ":")
	if len(pair) > 0 {
		hostname = pair[0]
	}
	if len(pair) > 1 {
		port = pair[1]
	}
	return
}

// PerformRequestBase sends request to ECS
func (e *MgmtClient) PerformRequestBase(method, scheme, port, uri string, body io.Reader, bodyLength int64, headers http.Header, auth Authentication, vdc string) (*http.Response, error) {
	h, err := e.ecs.NextAvailableNode(vdc)
	if err != nil {
		return nil, err
	}

	host, pport := parseHostPort(h)
	if port == "" {
		if pport != "" {
			port = pport
		} else {
			port = "4443"
		}
	}

	req, err := http.NewRequest(method, scheme+"://"+host+":"+port+uri, body)
	if err != nil {
		return nil, err
	}

	// Set content length
	req.ContentLength = bodyLength

	// Set header
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	log.Printf("[%s] performing request %s %s", e.Name, req.Method, req.URL)

	auth.SetAuth(req)

	resp, err := e.client.Do(req)
	if err != nil {
		log.Printf("[%s] error while performing request %s %s: %s", e.Name, req.Method, req.URL, err)
		if err, ok := err.(net.Error); ok && err.Timeout() {
			e.ecs.BlockNode(host, e.blockDur)
		}
		return nil, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode <= 219 {
		return resp, nil
	}

	// Close the body on failure response
	resp.Body.Close()
	return nil, fmt.Errorf("[%s]: %s %s", e.Name, method, resp.Status)
}

// MgmtLogin to login ECS mgmt interface
func (e *MgmtClient) MgmtLogin() (err error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.token != nil && !e.token.Expired() {
		return nil
	}
	var (
		resp  *http.Response
		token string
	)
	// retry for login
	i := 0
	for ; i < 3; i++ {
		if resp, err = e.PerformRequest("GET", "/login", nil, 0, http.Header{}, &BasicAuth{e.username, e.password}, ""); err != nil {
			continue
		}
		resp.Body.Close()
		token = resp.Header.Get("X-Sds-Auth-Token")
		if len(token) > 0 {
			break
		}
	}
	if i == 3 {
		return err
	}
	// first time login
	if e.token == nil {
		e.token = NewToken(token)
	} else {
		prevToken := e.token.Refresh(token)
		// release previous token to avoid token leak
		go e.MgmtLogout(prevToken)
	}
	return nil
}

// GetQuery sends Get request to ECS
func (e *MgmtClient) GetQuery(uri string, vdc string) (*http.Response, error) {
	return e.GetQueryBase("https", "", uri, vdc)
}

// GetQueryBase sends Get request to ECS
func (e *MgmtClient) GetQueryBase(scheme, port, uri, vdc string) (resp *http.Response, err error) {
	return e.QueryBaseWithRetry("GET", scheme, port, uri, nil, 0, http.Header{}, vdc)
}

// PostQuery sends Post request to ECS
func (e *MgmtClient) PostQuery(uri string, body io.Reader, bodyLength int64, headers http.Header, vdc string) (*http.Response, error) {
	return e.PostQueryBase("https", "", uri, body, bodyLength, headers, vdc)
}

// PostQueryBase sends Get request to ECS
func (e *MgmtClient) PostQueryBase(scheme, port, uri string, body io.Reader, bodyLength int64, headers http.Header, vdc string) (resp *http.Response, err error) {
	return e.QueryBaseWithRetry("POST", scheme, port, uri, body, bodyLength, headers, vdc)
}

// QueryBaseWithRetry does the general query to ECS with retry
func (e *MgmtClient) QueryBaseWithRetry(method, scheme, port, uri string, body io.Reader, bodyLength int64, headers http.Header, vdc string) (resp *http.Response, err error) {
	for i := 0; i < 3; i++ {
		if resp, err = e.QueryBase(method, scheme, port, uri, body, bodyLength, headers, vdc); err == nil {
			return resp, err
		}
	}
	return resp, err
}

// QueryBase does the general query to ECS
func (e *MgmtClient) QueryBase(method, scheme, port, uri string, body io.Reader, bodyLength int64, headers http.Header, vdc string) (*http.Response, error) {
	if e.token == nil || e.token.Expired() {
		if err := e.MgmtLogin(); err != nil {
			return nil, err
		}
	}
	token, _ := e.token.Get()
	return e.PerformRequestBase(method, scheme, port, uri, body, bodyLength, headers, &TokenAuth{token}, vdc)
}

// MgmtLogout logs out ECS mgmt interface
func (e *MgmtClient) MgmtLogout(token string) error {
	resp, err := e.PerformRequest("GET", "/logout", nil, 0, http.Header{}, &TokenAuth{token}, "")
	if err != nil {
		log.Printf("logout failed [%s]", err)
		return err
	}
	resp.Body.Close()
	return nil
}

// Close removes token and set pointer to nil
func (e *MgmtClient) Close() {
	e.mutex.Lock()
	if e.token != nil {
		if prev, expired := e.token.Get(); !expired {
			e.MgmtLogout(prev)
		}
	}
	e.token = nil
	e.mutex.Unlock()
	e = nil
}
