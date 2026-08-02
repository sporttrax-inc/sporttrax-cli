// Package api holds the SportTrax public API client. All requests go
// through Client.Get so headers, retries, debug logging, and error mapping
// stay in one place — command handlers and future MCP tool handlers share
// this same typed client.
package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sporttrax-inc/sporttrax-cli/internal/version"
)

const (
	basePath = "/public-api/v1"

	maxAttempts = 3
	// Longest 429 Retry-After the client will sleep through before giving
	// the error back to the caller (the per-minute window is 60s; a longer
	// wait means the daily cap, which sleeping can't fix).
	maxRetryAfter = 90 * time.Second
	// Ceiling on a single response body. The API's own pages are orders of
	// magnitude smaller; the cap bounds memory when the endpoint is
	// hostile or broken rather than trusting it to be well behaved.
	maxResponseBytes = 32 << 20
)

// Client talks to the SportTrax public API.
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
	// Verbose receives request/response debug lines when non-nil.
	Verbose io.Writer
	// Warn receives user-facing notices (e.g. rate-limit waits) when
	// non-nil.
	Warn io.Writer
}

// New returns a Client for baseURL authenticating with token. insecure
// skips TLS certificate verification for self-signed dev instances.
func New(baseURL, token string, insecure bool) *Client {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	if insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // the single sanctioned insecure site (CLAUDE.md): per-env or --insecure opt-in for self-signed dev certs
		}
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTP:    httpClient,
	}
}

// Host returns the host portion of the client's base URL.
func Host(baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid API URL %q: %w", baseURL, err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid API URL %q: missing host (did you include https://?)", baseURL)
	}
	return u.Host, nil
}

// IsLoopbackHost reports whether host (with or without a port) is the local
// machine. Loopback is the one place where the CLI relaxes its transport
// rules: nothing leaves the machine, so plaintext and self-signed certs
// carry no interception risk.
func IsLoopbackHost(host string) bool {
	name := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		name = h
	}
	name = strings.Trim(name, "[]")
	if strings.EqualFold(name, "localhost") {
		return true
	}
	ip := net.ParseIP(name)
	return ip != nil && ip.IsLoopback()
}

// ValidateBaseURL rejects base URLs that would put the bearer token on the
// wire in the clear. Every request carries the caller's personal access
// token, so plaintext HTTP is refused for anything but the local machine.
func ValidateBaseURL(baseURL string) error {
	u, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid API URL %q: %w", baseURL, err)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid API URL %q: missing host (did you include https://?)", baseURL)
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return nil
	case "http":
		if IsLoopbackHost(u.Host) {
			return nil
		}
		return fmt.Errorf(
			"refusing to send your token over plaintext HTTP to %s — use https:// (http is allowed only for localhost)",
			u.Host)
	default:
		return fmt.Errorf("invalid API URL %q: scheme must be https", baseURL)
	}
}

// StatusError is a non-2xx API response mapped to an actionable message.
type StatusError struct {
	StatusCode int
	Message    string
}

func (e *StatusError) Error() string { return e.Message }

// IsAuthError reports whether err is a 401/403 API response.
func IsAuthError(err error) bool {
	var se *StatusError
	return errors.As(err, &se) &&
		(se.StatusCode == http.StatusUnauthorized || se.StatusCode == http.StatusForbidden)
}

// Get performs an authenticated GET of path (relative to /public-api/v1)
// and decodes the JSON response into out (skipped when out is nil). It
// retries transient failures: 429 waits out Retry-After (bounded), 5xx and
// network errors back off briefly.
func (c *Client) Get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.BaseURL + basePath + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	for attempt := 1; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		c.setHeaders(req)

		start := time.Now()
		resp, err := c.HTTP.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.logf("GET %s failed after %s: %v", u, time.Since(start).Round(time.Millisecond), err)
			if attempt < maxAttempts {
				if err := sleep(ctx, time.Duration(attempt)*500*time.Millisecond); err != nil {
					return err
				}
				continue
			}
			return fmt.Errorf("could not reach %s: %w", c.BaseURL, err)
		}
		c.logf("GET %s -> %d (%s)", u, resp.StatusCode, time.Since(start).Round(time.Millisecond))

		switch {
		case resp.StatusCode == http.StatusOK:
			defer resp.Body.Close()
			if out == nil {
				return nil
			}
			// N runs to zero only if the body reached the cap, which
			// distinguishes "too large" from an ordinary decode failure.
			limited := &io.LimitedReader{R: resp.Body, N: maxResponseBytes + 1}
			if err := json.NewDecoder(limited).Decode(out); err != nil {
				if limited.N <= 0 {
					return fmt.Errorf("response from %s exceeded the %d MB limit", c.BaseURL, maxResponseBytes>>20)
				}
				return err
			}
			return nil

		case resp.StatusCode == http.StatusTooManyRequests && attempt < maxAttempts:
			wait := retryAfter(resp)
			resp.Body.Close()
			if wait > maxRetryAfter {
				return statusError(resp.StatusCode, resp.Header, nil)
			}
			c.warnf("rate limited, waiting %s before retrying", wait)
			if err := sleep(ctx, wait); err != nil {
				return err
			}
			continue

		case resp.StatusCode >= 500 && resp.StatusCode != http.StatusServiceUnavailable && attempt < maxAttempts:
			resp.Body.Close()
			if err := sleep(ctx, time.Duration(attempt)*500*time.Millisecond); err != nil {
				return err
			}
			continue

		default:
			defer resp.Body.Close()
			return statusError(resp.StatusCode, resp.Header, resp.Body)
		}
	}
}

// Me is the typed view of GET /me (token introspection) used for display;
// Raw callers get the verbatim payload via the second return of MeInfo.
type Me struct {
	User struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Role string `json:"role"`
	} `json:"user"`
	RateLimit struct {
		PerMinute *int `json:"per_minute"`
		PerDay    *int `json:"per_day"`
	} `json:"rate_limit"`
}

// MeInfo fetches token introspection: who the token belongs to and what
// access applies. The raw payload passes server fields through verbatim.
func (c *Client) MeInfo(ctx context.Context) (Me, json.RawMessage, error) {
	raw, err := c.GetRaw(ctx, "/me")
	if err != nil {
		return Me{}, nil, err
	}
	var me Me
	if err := json.Unmarshal(raw, &me); err != nil {
		return Me{}, nil, fmt.Errorf("unexpected /me payload: %w", err)
	}
	return me, raw, nil
}

// Validate checks that the token authenticates against the public API.
func (c *Client) Validate(ctx context.Context) error {
	_, _, err := c.MeInfo(ctx)
	return err
}

// setHeaders applies auth plus the X-SPORTTRAX-* client identification
// headers every SportTrax client sends; this one identifies itself with
// device type "cli". No persistent device ID is sent — a deliberate
// omission for a public CLI.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent",
		"sporttrax-cli/"+version.Version+" ("+runtime.GOOS+"/"+runtime.GOARCH+")")
	req.Header.Set("X-SPORTTRAX-CLIENT-DEVICE-TYPE", "cli")
	req.Header.Set("X-SPORTTRAX-CLI-VERSION", version.Version)
	req.Header.Set("X-SPORTTRAX-CLIENT-NOW", time.Now().Format(time.RFC3339))
}

func (c *Client) logf(format string, args ...any) {
	if c.Verbose != nil {
		fmt.Fprintf(c.Verbose, "[debug] "+format+"\n", args...)
	}
}

func (c *Client) warnf(format string, args ...any) {
	if c.Warn != nil {
		fmt.Fprintf(c.Warn, format+"\n", args...)
	}
}

func statusError(code int, header http.Header, body io.Reader) error {
	if header.Get("X-SPORTTRAX-DOWN") == "1" {
		return &StatusError{code, "SportTrax is down for maintenance — try again shortly"}
	}
	msg := ""
	if body != nil {
		msg = serverMessage(body, "")
	}
	switch code {
	case http.StatusUnauthorized:
		return &StatusError{code, "token was rejected (401): it may be invalid, revoked, or the account's email is unverified"}
	case http.StatusForbidden:
		if msg == "" {
			msg = "check that the token was created with the public-api permission"
		}
		return &StatusError{code, "access denied (403): " + msg}
	case http.StatusNotFound:
		return &StatusError{code, "not found (404)"}
	case http.StatusUnprocessableEntity:
		if msg == "" {
			msg = "validation failed"
		}
		return &StatusError{code, msg + " (422)"}
	case http.StatusTooManyRequests:
		s := "rate limited (429)"
		if ra := header.Get("Retry-After"); ra != "" {
			s += ", retry after " + ra + "s"
		}
		return &StatusError{code, s}
	case http.StatusServiceUnavailable:
		return &StatusError{code, "the public API is currently disabled (503)"}
	default:
		if msg == "" {
			msg = http.StatusText(code)
		}
		return &StatusError{code, fmt.Sprintf("unexpected response %d: %s", code, msg)}
	}
}

func retryAfter(resp *http.Response) time.Duration {
	if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && s > 0 {
		return time.Duration(s) * time.Second
	}
	return 2 * time.Second
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// serverMessage extracts the API's {"message": "..."} error body, falling
// back to fallback when absent.
func serverMessage(body io.Reader, fallback string) string {
	var payload struct {
		Message string `json:"message"`
	}
	data, err := io.ReadAll(io.LimitReader(body, 4096))
	if err == nil {
		if json.Unmarshal(data, &payload) == nil && payload.Message != "" {
			return payload.Message
		}
	}
	return fallback
}
