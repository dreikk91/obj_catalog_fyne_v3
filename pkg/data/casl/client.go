package casl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type APIClient struct {
	baseURL    string
	pultID     int64
	email      string
	pass       string
	httpClient *http.Client

	authMu sync.Mutex
	mu     sync.RWMutex

	token  string
	wsURL  string
	userID string
}

func NewAPIClient(baseURL string, token string, pultID int64, credentials ...string) *APIClient {
	email := ""
	pass := ""
	if len(credentials) > 0 {
		email = strings.TrimSpace(credentials[0])
	}
	if len(credentials) > 1 {
		pass = strings.TrimSpace(credentials[1])
	}

	return &APIClient{
		baseURL: normalizeBaseURL(baseURL),
		pultID:  pultID,
		email:   email,
		pass:    pass,
		token:   strings.TrimSpace(token),
		httpClient: &http.Client{
			Timeout: HTTPTimeout,
		},
	}
}

func (c *APIClient) PostCommand(ctx context.Context, payload map[string]any, out any, requireAuth bool) error {
	return c.postCommandWithRetry(ctx, payload, out, requireAuth, true)
}

func (c *APIClient) postCommandWithRetry(ctx context.Context, payload map[string]any, out any, requireAuth bool, allowRelogin bool) error {
	requestPayload := make(map[string]any)
	for k, v := range payload {
		requestPayload[k] = v
	}

	if requireAuth {
		token, err := c.EnsureToken(ctx)
		if err != nil {
			return err
		}
		if _, exists := requestPayload["token"]; !exists || requestPayload["token"] == "" {
			requestPayload["token"] = token
		}
	}

	body, status, err := c.doJSONRequest(ctx, CommandPath, requestPayload)
	if err != nil {
		return err
	}

	if !statusIsOK(status.Status) {
		if requireAuth && allowRelogin && isAuthError(status.Error) && c.canRelogin() {
			if reloginErr := c.RefreshToken(ctx, true); reloginErr != nil {
				return fmt.Errorf("casl relogin failed: %w", reloginErr)
			}
			return c.postCommandWithRetry(ctx, payload, out, requireAuth, false)
		}
		return fmt.Errorf("casl command %v status=%q error=%q", payload["type"], status.Status, status.Error)
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("casl decode response: %w", err)
	}
	return nil
}

func (c *APIClient) doJSONRequest(ctx context.Context, path string, payload any) ([]byte, StatusOnlyResponse, error) {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, StatusOnlyResponse{}, fmt.Errorf("casl marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(requestBody))
	if err != nil {
		return nil, StatusOnlyResponse{}, fmt.Errorf("casl create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, StatusOnlyResponse{}, fmt.Errorf("casl request failed: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, StatusOnlyResponse{}, fmt.Errorf("casl read response: %w", readErr)
	}

	var status StatusOnlyResponse
	_ = json.Unmarshal(body, &status)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if strings.TrimSpace(status.Error) != "" {
			return nil, status, fmt.Errorf("casl http %d: %s", resp.StatusCode, status.Error)
		}
		return nil, status, fmt.Errorf("casl unexpected http status: %d", resp.StatusCode)
	}

	return body, status, nil
}

func (c *APIClient) doJSONGet(ctx context.Context, path string) ([]byte, StatusOnlyResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, StatusOnlyResponse{}, fmt.Errorf("casl create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, StatusOnlyResponse{}, fmt.Errorf("casl request failed: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, StatusOnlyResponse{}, fmt.Errorf("casl read response: %w", readErr)
	}

	var status StatusOnlyResponse
	_ = json.Unmarshal(body, &status)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if strings.TrimSpace(status.Error) != "" {
			return nil, status, fmt.Errorf("casl http %d: %s", resp.StatusCode, status.Error)
		}
		return nil, status, fmt.Errorf("casl unexpected http status: %d", resp.StatusCode)
	}

	return body, status, nil
}

func (c *APIClient) EnsureToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	token := strings.TrimSpace(c.token)
	c.mu.RUnlock()
	if token != "" {
		return token, nil
	}

	if !c.canRelogin() {
		return "", fmt.Errorf("casl: token is empty and credentials are not configured")
	}

	if err := c.RefreshToken(ctx, false); err != nil {
		return "", err
	}

	c.mu.RLock()
	token = strings.TrimSpace(c.token)
	c.mu.RUnlock()
	return token, nil
}

func (c *APIClient) RefreshToken(ctx context.Context, force bool) error {
	if !c.canRelogin() {
		return fmt.Errorf("casl: credentials are not configured")
	}

	c.authMu.Lock()
	defer c.authMu.Unlock()

	if force {
		c.mu.Lock()
		c.token = ""
		c.mu.Unlock()
	} else {
		c.mu.RLock()
		if c.token != "" {
			c.mu.RUnlock()
			return nil
		}
		c.mu.RUnlock()
	}

	payload := map[string]any{
		"email":   c.email,
		"pwd":     c.pass,
		"pult_id": strconv.FormatInt(c.pultID, 10),
		"captcha": "",
	}

	body, _, err := c.doJSONRequest(ctx, LoginPath, payload)
	if err != nil {
		return err
	}

	var resp LoginResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("casl decode login response: %w", err)
	}

	if !statusIsOK(resp.Status) {
		return fmt.Errorf("casl login status=%q error=%q", resp.Status, resp.Error)
	}

	c.mu.Lock()
	c.token = strings.TrimSpace(resp.Token)
	c.wsURL = strings.TrimSpace(resp.WSURL)
	c.userID = strings.TrimSpace(resp.UserID)
	c.mu.Unlock()

	return nil
}

func (c *APIClient) canRelogin() bool {
	return strings.TrimSpace(c.email) != "" && strings.TrimSpace(c.pass) != ""
}

func (c *APIClient) GetSessionInfo() (token, wsURL, userID string, pultID int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token, c.wsURL, c.userID, c.pultID
}

func (c *APIClient) BaseURL() string {
	return c.baseURL
}

// Specialized methods

func (c *APIClient) GetCaptchaConfig(ctx context.Context) (CaptchaConfig, error) {
	body, status, err := c.doJSONGet(ctx, CaptchaShowPath)
	if err != nil {
		return CaptchaConfig{}, err
	}

	var resp CaptchaConfig
	if err := json.Unmarshal(body, &resp); err != nil {
		return CaptchaConfig{}, fmt.Errorf("casl decode captchaShow response: %w", err)
	}

	if !statusIsOK(resp.Status) && !statusIsOK(status.Status) {
		errText := strings.TrimSpace(resp.Error)
		if errText == "" {
			errText = strings.TrimSpace(status.Error)
		}
		return CaptchaConfig{}, fmt.Errorf("casl captchaShow status=%q error=%q", resp.Status, errText)
	}

	return resp, nil
}

func (c *APIClient) ReadPults(ctx context.Context, skip int, limit int) ([]Pult, error) {
	payload := map[string]any{"type": "read_pult", "skip": skip, "limit": limit}
	var resp ReadPultResponse
	if err := c.PostCommand(ctx, payload, &resp, false); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *APIClient) ReadGuardObjects(ctx context.Context, skip int, limit int) ([]GrdObject, error) {
	payload := map[string]any{"type": "read_grd_object", "skip": skip, "limit": limit}
	var resp ReadGrdObjectResponse
	if err := c.PostCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *APIClient) ReadUsers(ctx context.Context, skip int, limit int) ([]User, error) {
	payload := map[string]any{"type": "read_user", "skip": skip, "limit": limit}
	var resp ReadUserResponse
	if err := c.PostCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *APIClient) ReadDevices(ctx context.Context, skip int, limit int) ([]Device, error) {
	payload := map[string]any{"type": "read_device", "skip": skip, "limit": limit}
	var resp ReadDeviceResponse
	if err := c.PostCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *APIClient) ReadBasketCount(ctx context.Context) (int, error) {
	payload := map[string]any{"type": "read_count_in_basket"}
	var resp BasketResponse
	if err := c.PostCommand(ctx, payload, &resp, true); err != nil {
		return 0, err
	}
	return resp.Count, nil
}

func (c *APIClient) GetStatisticReport(ctx context.Context, name string, limit int) ([]map[string]any, error) {
	payload := map[string]any{
		"type": "get_statistic",
		"name": name,
	}
	if limit > 0 {
		payload["limit"] = limit
	}

	var resp struct {
		Status string           `json:"status"`
		Data   []map[string]any `json:"data"`
		Error  string           `json:"error"`
	}
	if err := c.PostCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func normalizeBaseURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = DefaultBaseURL
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	return strings.TrimRight(value, "/")
}

func statusIsOK(status string) bool {
	value := strings.ToLower(strings.TrimSpace(status))
	return value == "" || value == "ok"
}

func isAuthError(raw string) bool {
	value := strings.ToUpper(strings.TrimSpace(raw))
	return strings.Contains(value, "TOKEN") || strings.Contains(value, "AUTH") || strings.Contains(value, "UNAUTHORIZED") || value == "WRONG_FORMAT"
}
