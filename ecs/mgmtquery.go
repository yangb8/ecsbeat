package ecs

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/logp"
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

// PerformRequest sends request to ECS
func (e *MgmtClient) PerformRequest(method, scheme, port, uri string, body io.Reader, bodyLength int64, headers http.Header, auth Authentication, vdc string) (*http.Response, int, error) {
	h, err := e.ecs.NextAvailableNode(vdc)
	if err != nil {
		return nil, 0, err
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
		return nil, 0, err
	}

	// Set content length
	req.ContentLength = bodyLength

	// Set header
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	logp.Info("[%s] performing request %s %s", e.Name, req.Method, req.URL)

	auth.SetAuth(req)

	resp, err := e.client.Do(req)
	if err != nil {
		logp.Warn("[%s] error while performing request %s %s: %s", e.Name, req.Method, req.URL, err)
		if err, ok := err.(net.Error); ok && err.Timeout() {
			e.ecs.BlockNode(host, e.blockDur)
		}
		return nil, 0, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode <= 219 {
		return resp, resp.StatusCode, nil
	}

	// Close the body on failure response
	resp.Body.Close()
	logp.Warn("[%s] error response %s %s %s", e.Name, method, req.URL, resp.Status)
	return nil, resp.StatusCode, fmt.Errorf("[%s]: %s %s", e.Name, method, resp.Status)
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
		if resp, _, err = e.PerformRequest("GET", "https", "", "/login", nil, 0, http.Header{}, &BasicAuth{e.username, e.password}, ""); err != nil {
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
	var status int
	for i := 0; i < 3; i++ {
		if e.token == nil || e.token.Expired() {
			if err = e.MgmtLogin(); err != nil {
				continue
			}
		}
		token, _ := e.token.Get()
		if resp, status, err = e.PerformRequest(method, scheme, port, uri, body, bodyLength, headers, &TokenAuth{token}, vdc); err != nil {
			if status == 401 {
				e.token.ForceExpire()
			}
		} else {
			return resp, err
		}
	}
	return resp, err
}

// MgmtLogout logs out ECS mgmt interface
func (e *MgmtClient) MgmtLogout(token string) error {
	resp, _, err := e.PerformRequest("GET", "https", "", "/logout", nil, 0, http.Header{}, &TokenAuth{token}, "")
	if err != nil {
		logp.Info("logout failed [%s]", err)
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
