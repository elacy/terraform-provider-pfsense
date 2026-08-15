// Package client implements a minimal Go client for the pfSense REST API v2
// (pfrest/pfSense-pkg-RESTAPI).
//
// The v2 API is a REST + GraphQL surface rooted at /api/v2 with a uniform
// response envelope and three authentication methods (Basic, X-API-Key, JWT).
// This package handles the envelope, error mapping, auth (including JWT
// issuance/caching), and the common CRUD + apply verbs used by the provider.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// defaultTimeout is the default HTTP request timeout.
	defaultTimeout = 30 * time.Second

	// defaultJWTTTL is how long we cache an issued JWT before requesting a new
	// one. The API default expiry is 1 hour; we refresh well before that.
	defaultJWTTTL = 45 * time.Minute

	// jwtEndpoint issues a signed JWT when called with Basic auth.
	jwtEndpoint = "/api/v2/auth/jwt"
)

// Config holds the connection and authentication settings for the client.
type Config struct {
	// URL is the base URL of the pfSense instance, e.g. https://192.168.1.1.
	URL string
	// Username and Password enable local-database (Basic) authentication.
	Username string
	Password string
	// APIKey enables X-API-Key authentication. Takes precedence over Basic.
	APIKey string
	// UseJWT causes the client to exchange Basic credentials for a JWT and
	// use bearer auth for subsequent requests. Ignored unless Username is set.
	UseJWT bool
	// SkipTLSVerify disables TLS certificate verification (self-signed boxes).
	SkipTLSVerify bool
	// Timeout overrides the default request timeout.
	Timeout time.Duration
}

// Client is a thread-safe HTTP client for the pfSense REST API v2.
type Client struct {
	cfg      Config
	http     *http.Client
	baseURL  *url.URL
	jwtMu    sync.Mutex
	jwtToken string
	jwtAt    time.Time
}

// New constructs a Client from cfg. It validates and normalises the base URL.
func New(cfg Config) (*Client, error) {
	baseURL, err := url.Parse(strings.TrimRight(cfg.URL, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid pfSense URL %q: %w", cfg.URL, err)
	}
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return nil, fmt.Errorf("pfSense URL must use http or https, got %q", baseURL.Scheme)
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	transport := &http.Transport{}
	if cfg.SkipTLSVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in via provider config
	}
	return &Client{
		cfg:     cfg,
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout, Transport: transport},
	}, nil
}

// Response is the decoded v2 response envelope:
//
//	{"code":200,"status":"ok","response_id":"SUCCESS","message":"","data":{...}}
type Response struct {
	Code       int             `json:"code"`
	Status     string          `json:"status"`
	ResponseID string          `json:"response_id"`
	Message    string          `json:"message"`
	Data       json.RawMessage `json:"data"`
}

// APIError represents a non-success response from the API. It carries the
// HTTP-ish code and the machine-readable response_id so callers (and Terraform
// diagnostics) can surface a stable error identifier.
type APIError struct {
	StatusCode int
	Code       int
	Status     string
	ResponseID string
	Message    string
	Method     string
	Path       string
}

func (e *APIError) Error() string {
	msg := strings.TrimSpace(e.Message)
	switch {
	case msg == "" && e.ResponseID != "":
		return fmt.Sprintf("pfSense API error: %s (code %d, response_id %s)", e.Status, e.Code, e.ResponseID)
	case e.ResponseID != "":
		return fmt.Sprintf("pfSense API error: %s (code %d, response_id %s, message %q)", e.Status, e.Code, e.ResponseID, msg)
	default:
		return fmt.Sprintf("pfSense API error: %s (code %d, message %q)", e.Status, e.Code, msg)
	}
}

// Query is a set of URL query parameters (supports repeated keys).
type Query url.Values

// NewQuery returns an empty Query.
func NewQuery() Query { return Query{} }

// Set sets a single-value query parameter.
func (q Query) Set(key, value string) Query {
	(url.Values)(q).Set(key, value)
	return q
}

// Add appends a query parameter value (use for `__in` arrays etc.).
func (q Query) Add(key, value string) Query {
	(url.Values)(q).Add(key, value)
	return q
}

// Encode returns the URL-encoded query string ("" when empty).
func (q Query) Encode() string { return (url.Values)(q).Encode() }

// Do performs an HTTP request against path with optional query params and a
// JSON body (nil body -> no body). It returns the decoded envelope. A nil
// body with a non-2xx envelope returns an *APIError. JWT auth is handled
// transparently: an expired/invalid token is re-issued once and the request
// retried.
func (c *Client) Do(ctx context.Context, method, path string, query Query, body any) (*Response, error) {
	reqURL := *c.baseURL
	reqURL.Path = strings.TrimSuffix(c.baseURL.Path, "/") + "/" + strings.TrimPrefix(path, "/")
	if len(query) > 0 {
		reqURL.RawQuery = query.Encode()
	}

	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
	}

	resp, err := c.doOnce(ctx, method, reqURL.String(), bodyBytes)
	if err != nil {
		// Recoverable auth failure: refresh JWT and retry exactly once.
		if c.cfg.UseJWT && c.isAuthError(err) {
			tflog.Debug(ctx, "JWT may have expired; refreshing and retrying", map[string]any{"path": path})
			c.invalidateJWT()
			if jerr := c.issueJWT(ctx); jerr != nil {
				return nil, jerr
			}
			resp, err = c.doOnce(ctx, method, reqURL.String(), bodyBytes)
		}
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// doOnce performs a single request without JWT retry logic.
func (c *Client) doOnce(ctx context.Context, method, fullURL string, body []byte) (*Response, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	if err := c.setAuth(ctx, req); err != nil {
		return nil, err
	}

	tflog.Debug(ctx, "pfSense API request", map[string]any{"method": method, "url": fullURL})

	httpResp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing %s %s: %w", method, fullURL, err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(httpResp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// The API always speaks JSON, even for errors. A 401 with an empty body
	// (e.g. a proxy) is still an auth error we can classify.
	if len(bytes.TrimSpace(respBytes)) == 0 {
		return nil, &APIError{StatusCode: httpResp.StatusCode, Code: httpResp.StatusCode, Status: "empty response", Method: method, Path: fullURL}
	}

	var envelope Response
	if err := json.Unmarshal(respBytes, &envelope); err != nil {
		return nil, fmt.Errorf("decoding response envelope: %w (body: %.500s)", err, string(respBytes))
	}

	if httpResp.StatusCode >= 400 || envelope.Code >= 400 {
		status := envelope.Status
		if status == "" {
			status = strings.ToLower(http.StatusText(httpResp.StatusCode))
		}
		apiErr := &APIError{
			StatusCode: httpResp.StatusCode,
			Code:       envelope.Code,
			Status:     status,
			ResponseID: envelope.ResponseID,
			Message:    envelope.Message,
			Method:     method,
			Path:       fullURL,
		}
		if apiErr.Code == 0 {
			apiErr.Code = httpResp.StatusCode
		}
		return nil, apiErr
	}

	return &envelope, nil
}

// setAuth applies the configured authentication scheme to the request.
func (c *Client) setAuth(ctx context.Context, req *http.Request) error {
	switch {
	case c.cfg.APIKey != "":
		req.Header.Set("X-API-Key", c.cfg.APIKey)
	case c.cfg.Username != "":
		if c.cfg.UseJWT {
			token, err := c.jwt(ctx)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		}
		req.Header.Set("Authorization", "Basic "+basicAuth(c.cfg.Username, c.cfg.Password))
	}
	return nil
}

// jwt returns a cached JWT, issuing a new one if none is present or it is stale.
func (c *Client) jwt(ctx context.Context) (string, error) {
	c.jwtMu.Lock()
	defer c.jwtMu.Unlock()
	if c.jwtToken != "" && time.Since(c.jwtAt) < defaultJWTTTL {
		return c.jwtToken, nil
	}
	if err := c.issueJWT(ctx); err != nil {
		return "", err
	}
	return c.jwtToken, nil
}

// issueJWT exchanges Basic credentials for a JWT. Callers must hold jwtMu.
func (c *Client) issueJWT(ctx context.Context) error {
	fullURL := *c.baseURL
	fullURL.Path = strings.TrimSuffix(c.baseURL.Path, "/") + jwtEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL.String(), nil)
	if err != nil {
		return fmt.Errorf("building JWT request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+basicAuth(c.cfg.Username, c.cfg.Password))

	httpResp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("requesting JWT: %w", err)
	}
	defer httpResp.Body.Close()
	respBytes, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("reading JWT response: %w", err)
	}

	var envelope struct {
		Code    int    `json:"code"`
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &envelope); err != nil {
		return fmt.Errorf("decoding JWT response: %w", err)
	}
	if httpResp.StatusCode >= 400 || envelope.Code >= 400 || envelope.Data.Token == "" {
		return &APIError{StatusCode: httpResp.StatusCode, Code: envelope.Code, Status: envelope.Status, Message: envelope.Message, Method: http.MethodPost, Path: fullURL.String()}
	}

	c.jwtToken = envelope.Data.Token
	c.jwtAt = time.Now()
	return nil
}

func (c *Client) invalidateJWT() {
	c.jwtMu.Lock()
	c.jwtToken = ""
	c.jwtAt = time.Time{}
	c.jwtMu.Unlock()
}

// isAuthError reports whether an error is an authentication failure (HTTP 401
// or a 403 the API attributes to a stale credential). We only retry 401s to
// avoid masking genuine permission errors.
func (c *Client) isAuthError(err error) bool {
	var apiErr *APIError
	if ok := asAPIError(err, &apiErr); ok {
		return apiErr.StatusCode == http.StatusUnauthorized || apiErr.Code == http.StatusUnauthorized
	}
	return false
}

// basicAuth base64-encodes username:password.
func basicAuth(user, pass string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
}

// --- CRUD helpers -----------------------------------------------------------

// List performs a GET against a plural endpoint and decodes `data` as an array
// of raw JSON objects (one per element).
func (c *Client) List(ctx context.Context, path string, query Query) ([]json.RawMessage, error) {
	resp, err := c.Do(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		return nil, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, fmt.Errorf("expected array data from %s, got %s: %w", path, string(resp.Data), err)
	}
	return items, nil
}

// Get performs a GET against a singular endpoint and decodes `data` as a single
// JSON object. A 404 (NotFoundError) returns ErrNotFound.
func (c *Client) Get(ctx context.Context, path string, query Query) (json.RawMessage, error) {
	resp, err := c.Do(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		return nil, ErrNotFound
	}
	return resp.Data, nil
}

// Create POSTs body to a singular endpoint and returns the created object.
func (c *Client) Create(ctx context.Context, path string, body any) (json.RawMessage, error) {
	resp, err := c.Do(ctx, http.MethodPost, path, nil, body)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// Update PATCHes body to a singular endpoint. The body must include the
// object's `id` (see findByKey) plus any control parameters such as `apply`.
// It returns the updated object.
func (c *Client) Update(ctx context.Context, path string, body any) (json.RawMessage, error) {
	resp, err := c.Do(ctx, http.MethodPatch, path, nil, body)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// Delete issues a DELETE to a singular endpoint addressed by query (?id=N).
func (c *Client) Delete(ctx context.Context, path string, query Query) error {
	_, err := c.Do(ctx, http.MethodDelete, path, query, nil)
	return err
}

// Apply posts to an "apply" endpoint to push pending configuration changes to
// the running system (e.g. /api/v2/firewall/apply). Some resources apply
// immediately and do not require this.
func (c *Client) Apply(ctx context.Context, path string, body any) error {
	_, err := c.Do(ctx, http.MethodPost, path, nil, body)
	return err
}

func asAPIError(err error, target **APIError) bool {
	if e, ok := err.(*APIError); ok {
		*target = e
		return true
	}
	return false
}

func isNotFound(err error) bool {
	var apiErr *APIError
	if asAPIError(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound ||
			apiErr.Code == http.StatusNotFound ||
			apiErr.ResponseID == "NOT_FOUND"
	}
	return false
}
