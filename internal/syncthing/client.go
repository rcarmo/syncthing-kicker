package syncthing

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Client struct {
	baseURL *url.URL
	apiKey  string
	hc      *http.Client
}

type ClientOptions struct {
	VerifyTLS      bool
	RequestTimeout time.Duration // 0 means default
}

func NewClient(apiURL, apiKey string, opts ClientOptions) (*Client, error) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid api url: %w", err)
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}

	tr := http.DefaultTransport.(*http.Transport).Clone()
	if u.Scheme == "https" && !opts.VerifyTLS {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	hc := &http.Client{Transport: tr}
	if opts.RequestTimeout > 0 {
		hc.Timeout = opts.RequestTimeout
	}

	return &Client{baseURL: u, apiKey: apiKey, hc: hc}, nil
}

func (c *Client) doJSON(ctx context.Context, method, p string, q url.Values, timeout time.Duration, out any) (int, error) {
	ctx, cancel := withTimeout(ctx, timeout)
	defer cancel()

	u := *c.baseURL
	u.Path = path.Join(c.baseURL.Path, strings.TrimPrefix(p, "/"))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, err
	}

	if resp.StatusCode >= 400 {
		if len(body) == 0 {
			return resp.StatusCode, errors.New("http error")
		}
		return resp.StatusCode, fmt.Errorf("http error: %s", strings.TrimSpace(string(body)))
	}

	if out == nil {
		return resp.StatusCode, nil
	}

	if err := json.Unmarshal(body, out); err != nil {
		return resp.StatusCode, err
	}
	return resp.StatusCode, nil
}

func (c *Client) PostScan(ctx context.Context, folder string, timeout time.Duration) (int, error) {
	q := url.Values{}
	if strings.TrimSpace(folder) != "" && folder != "*" {
		q.Set("folder", folder)
	}
	var ignore any
	return c.doJSON(ctx, http.MethodPost, "/rest/db/scan", q, timeout, &ignore)
}

type FolderStatus struct {
	State       string `json:"state"`
	NeedBytes   int64  `json:"needBytes"`
	InSyncBytes int64  `json:"inSyncBytes"`
}

func (c *Client) FolderStatus(ctx context.Context, folder string, timeout time.Duration) (FolderStatus, int, error) {
	q := url.Values{}
	q.Set("folder", folder)
	var st FolderStatus
	code, err := c.doJSON(ctx, http.MethodGet, "/rest/db/status", q, timeout, &st)
	return st, code, err
}

type Config struct {
	Folders []struct {
		ID string `json:"id"`
	} `json:"folders"`
}

func (c *Client) SystemConfig(ctx context.Context, timeout time.Duration) (Config, int, error) {
	var cfg Config
	code, err := c.doJSON(ctx, http.MethodGet, "/rest/system/config", nil, timeout, &cfg)
	return cfg, code, err
}

func withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
