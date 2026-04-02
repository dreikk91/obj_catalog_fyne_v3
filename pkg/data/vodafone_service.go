package data

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	vodafoneClientID     = "web-myvodafone-mw"
	vodafoneClientSecret = "F3jfj5J)9;"
	vodafoneBaseURL      = "https://mw-api.vodafone.ua"
	vodafoneAppVersion   = "3.1.17"
	vodafoneIOTCategory  = "14038376"
)

var errVodafoneAuthRequired = errors.New("vodafone: потрібна авторизація через SMS-код у налаштуваннях")

type VodafoneService struct {
	baseURL    string
	httpClient *http.Client
	store      config.VodafoneConfigStore

	mu              sync.Mutex
	availableSIMs   map[string]vodafoneSubscriber
	availableSIMsAt time.Time
}

type VodafoneOption func(*VodafoneService)

func WithVodafoneHTTPClient(client *http.Client) VodafoneOption {
	return func(s *VodafoneService) {
		if s == nil || client == nil {
			return
		}
		s.httpClient = client
	}
}

func WithVodafoneBaseURL(baseURL string) VodafoneOption {
	return func(s *VodafoneService) {
		if s == nil || strings.TrimSpace(baseURL) == "" {
			return
		}
		s.baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	}
}

func NewVodafoneService(store config.VodafoneConfigStore, opts ...VodafoneOption) *VodafoneService {
	service := &VodafoneService{
		baseURL:    vodafoneBaseURL,
		httpClient: &http.Client{Timeout: 20 * time.Second},
		store:      store,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}
	return service
}

func (s *VodafoneService) AuthState() (contracts.VodafoneAuthState, error) {
	cfg := s.loadConfig()
	expiry := cfg.TokenExpiryTime()
	if !expiry.IsZero() && expiry.Before(time.Now().UTC()) && cfg.HasAccessToken() {
		cfg.AccessToken = ""
		cfg.TokenExpiry = ""
		s.saveConfig(cfg)
	}

	state := contracts.VodafoneAuthState{
		Phone:          strings.TrimSpace(cfg.Phone),
		TokenExpiresAt: cfg.TokenExpiryTime(),
	}
	state.Authorized = cfg.TokenUsableAt(time.Now().UTC())
	return state, nil
}

func (s *VodafoneService) RequestLoginSMS(phone string) error {
	msisdn, err := normalizeVodafoneMSISDN(phone)
	if err != nil {
		return err
	}

	guestToken, err := s.getGuestToken()
	if err != nil {
		return err
	}

	payload := map[string]string{
		"receiver":        msisdn,
		"receiverTypeKey": "PHONE-NUMBER",
		"typeKey":         "MYVF-LOGIN-IOS",
		"langKey":         "uk",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("vodafone: failed to build otp request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.baseURL+"/otp/api/one-time-password/secured", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("vodafone: failed to create otp request: %w", err)
	}
	s.applyHeaders(req, guestToken, "", "application/json")

	if err := s.expectNoContent(req, http.StatusOK, http.StatusCreated); err != nil {
		return err
	}
	return nil
}

func (s *VodafoneService) VerifyLogin(phone string, code string) (contracts.VodafoneAuthState, error) {
	msisdn, err := normalizeVodafoneMSISDN(phone)
	if err != nil {
		return contracts.VodafoneAuthState{}, err
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return contracts.VodafoneAuthState{}, errors.New("vodafone: введіть SMS-код")
	}

	form := url.Values{
		"username": {msisdn},
		"password": {code},
	}
	req, err := http.NewRequest(
		http.MethodPost,
		s.baseURL+"/uaa/oauth/token?grant_type=password&Profile=MYVODAFONE",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return contracts.VodafoneAuthState{}, fmt.Errorf("vodafone: failed to create verify request: %w", err)
	}
	req.SetBasicAuth(vodafoneClientID, vodafoneClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Profile", "MYVODAFONE-LOGIN-PUK")

	var resp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := s.doJSON(req, &resp); err != nil {
		return contracts.VodafoneAuthState{}, err
	}
	if strings.TrimSpace(resp.AccessToken) == "" {
		return contracts.VodafoneAuthState{}, errors.New("vodafone: сервер не повернув access token")
	}

	expiry := resolveVodafoneTokenExpiry(resp.AccessToken, resp.ExpiresIn)
	cfg := s.loadConfig()
	cfg.Phone = msisdn
	cfg.AccessToken = strings.TrimSpace(resp.AccessToken)
	if !expiry.IsZero() {
		cfg.TokenExpiry = expiry.Format(time.RFC3339)
	} else {
		cfg.TokenExpiry = ""
	}
	s.saveConfig(cfg)
	s.invalidateAvailableSIMsCache()
	return s.AuthState()
}

func (s *VodafoneService) ClearLogin() error {
	cfg := s.loadConfig()
	cfg.AccessToken = ""
	cfg.TokenExpiry = ""
	s.saveConfig(cfg)
	s.invalidateAvailableSIMsCache()
	return nil
}

func (s *VodafoneService) GetSIMStatus(msisdn string) (contracts.VodafoneSIMStatus, error) {
	normalized, err := normalizeVodafoneMSISDN(msisdn)
	if err != nil {
		return contracts.VodafoneSIMStatus{}, err
	}

	subscriber, available, err := s.lookupSubscriber(normalized)
	if err != nil {
		return contracts.VodafoneSIMStatus{}, err
	}
	status := contracts.VodafoneSIMStatus{
		MSISDN:         normalized,
		Available:      available,
		SubscriberName: subscriber.Description,
	}
	if !available {
		return status, nil
	}

	connectivity, err := s.fetchConnectivityStatus(normalized)
	if err != nil {
		return contracts.VodafoneSIMStatus{}, err
	}
	lastEvent, err := s.fetchLastEvent(normalized)
	if err != nil {
		return contracts.VodafoneSIMStatus{}, err
	}
	status.Connectivity = connectivity
	status.LastEvent = lastEvent
	return status, nil
}

func (s *VodafoneService) RebootSIM(msisdn string) (contracts.VodafoneSIMRebootResult, error) {
	normalized, err := normalizeVodafoneMSISDN(msisdn)
	if err != nil {
		return contracts.VodafoneSIMRebootResult{}, err
	}
	if _, err := s.requireAvailableSubscriber(normalized); err != nil {
		return contracts.VodafoneSIMRebootResult{}, err
	}
	token, err := s.ensureAuthorizedToken()
	if err != nil {
		return contracts.VodafoneSIMRebootResult{}, err
	}

	payload := map[string]any{
		"processing": map[string]any{
			"maxParallelExecutions": 1,
		},
		"order": map[string]any{
			"@type":    "SYNCHLR-CHANGE-M2M",
			"category": "M2M",
			"productOrderItem": []map[string]any{
				{
					"action": "add",
					"type":   "synchlr-change-m2m",
					"characteristic": []map[string]string{
						{
							"name":      "msisdn",
							"value":     normalized,
							"valueType": "string",
						},
					},
				},
			},
			"characteristic": []any{},
			"relatedParty":   []any{},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return contracts.VodafoneSIMRebootResult{}, fmt.Errorf("vodafone: failed to build reboot request: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		s.baseURL+"/order/tmf-api/productOrderingManagement/v4/productOrder/autoSubmit",
		bytes.NewReader(body),
	)
	if err != nil {
		return contracts.VodafoneSIMRebootResult{}, fmt.Errorf("vodafone: failed to create reboot request: %w", err)
	}
	s.applyHeaders(req, token, "", "application/json")

	var resp struct {
		ID    string `json:"id"`
		State string `json:"state"`
	}
	if err := s.doJSON(req, &resp); err != nil {
		return contracts.VodafoneSIMRebootResult{}, err
	}
	return contracts.VodafoneSIMRebootResult{
		OrderID: strings.TrimSpace(resp.ID),
		State:   strings.TrimSpace(resp.State),
	}, nil
}

func (s *VodafoneService) UpdateSIMMetadata(msisdn string, name string, comment string) error {
	normalized, err := normalizeVodafoneMSISDN(msisdn)
	if err != nil {
		return err
	}
	if _, err := s.requireAvailableSubscriber(normalized); err != nil {
		return err
	}
	token, err := s.ensureAuthorizedToken()
	if err != nil {
		return err
	}

	updates := []struct {
		characteristic string
		value          string
	}{
		{characteristic: "MYVF:B2B:SUBSCRIBER-NAME", value: strings.TrimSpace(name)},
		{characteristic: "MYVF:B2B:SUBSCRIBER-COMMENT", value: strings.TrimSpace(comment)},
	}

	performed := false
	for _, item := range updates {
		if item.value == "" {
			continue
		}
		if err := s.patchSubscriberCharacteristic(token, normalized, item.characteristic, item.value); err != nil {
			return err
		}
		performed = true
	}
	if !performed {
		return errors.New("vodafone: немає даних для запису в name/comment")
	}
	return nil
}

func (s *VodafoneService) patchSubscriberCharacteristic(token string, msisdn string, name string, value string) error {
	payload := []map[string]any{
		{
			"op":   "replace",
			"path": "/characteristic/-",
			"value": map[string]string{
				"name":  name,
				"value": value,
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("vodafone: failed to build metadata patch: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPatch,
		s.baseURL+"/customer/api/customerManagement/v3/customer/self?childMsisdn="+url.QueryEscape(msisdn),
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("vodafone: failed to create metadata patch: %w", err)
	}
	s.applyHeaders(req, token, "B2B-CHARACTERISTIC", "application/json-patch+json")

	if err := s.expectNoContent(req, http.StatusOK); err != nil {
		return err
	}
	return nil
}

func (s *VodafoneService) fetchConnectivityStatus(msisdn string) (contracts.VodafoneConnectivityStatus, error) {
	token, err := s.ensureAuthorizedToken()
	if err != nil {
		return contracts.VodafoneConnectivityStatus{}, err
	}

	req, err := http.NewRequest(
		http.MethodGet,
		s.baseURL+"/customer/api/customerManagement/v3/customer/self?msisdn="+url.QueryEscape(msisdn),
		nil,
	)
	if err != nil {
		return contracts.VodafoneConnectivityStatus{}, fmt.Errorf("vodafone: failed to create connectivity request: %w", err)
	}
	s.applyHeaders(req, token, "CONNECTIVITY-CHECK-BY-MSISDN", "application/json-patch+json")

	var resp struct {
		RelatedParty []struct {
			ID              string               `json:"id"`
			Characteristics []vodafoneNamedValue `json:"characteristics"`
		} `json:"relatedParty"`
	}
	if err := s.doJSON(req, &resp); err != nil {
		return contracts.VodafoneConnectivityStatus{}, err
	}
	if len(resp.RelatedParty) == 0 {
		return contracts.VodafoneConnectivityStatus{}, nil
	}

	values := make(map[string]string)
	for _, item := range resp.RelatedParty[0].Characteristics {
		values[item.Name] = item.Value
	}

	connectionTime, _ := parseVodafoneTime("02-01-2006 15:04:05", values["connectionTime"])
	return contracts.VodafoneConnectivityStatus{
		OperationStatus:   strings.TrimSpace(values["status"]),
		SIMStatus:         strings.TrimSpace(values["statusSIM"]),
		BaseStationStatus: strings.TrimSpace(values["statusBS"]),
		LBSStatusKey:      strings.TrimSpace(values["lbsStatusKey"]),
		ConnectionTime:    connectionTime,
		ConnectionTimeRaw: strings.TrimSpace(values["connectionTime"]),
	}, nil
}

func (s *VodafoneService) fetchLastEvent(msisdn string) (contracts.VodafoneLastEvent, error) {
	token, err := s.ensureAuthorizedToken()
	if err != nil {
		return contracts.VodafoneLastEvent{}, err
	}

	req, err := http.NewRequest(
		http.MethodGet,
		s.baseURL+"/customer/api/customerManagement/v3/customer/self?msisdn="+url.QueryEscape(msisdn),
		nil,
	)
	if err != nil {
		return contracts.VodafoneLastEvent{}, fmt.Errorf("vodafone: failed to create last-event request: %w", err)
	}
	s.applyHeaders(req, token, "LASTEVENT-MSISDN-M2M", "application/json-patch+json")

	var resp []struct {
		RelatedParty []struct {
			ID              string               `json:"id"`
			Characteristics []vodafoneNamedValue `json:"characteristics"`
			Characterictics []vodafoneNamedValue `json:"characterictics"`
		} `json:"relatedParty"`
	}
	if err := s.doJSON(req, &resp); err != nil {
		return contracts.VodafoneLastEvent{}, err
	}
	if len(resp) == 0 || len(resp[0].RelatedParty) == 0 {
		return contracts.VodafoneLastEvent{}, nil
	}

	values := make(map[string]string)
	party := resp[0].RelatedParty[0]
	for _, item := range party.Characteristics {
		values[item.Name] = item.Value
	}
	for _, item := range party.Characterictics {
		values[item.Name] = item.Value
	}

	eventTime, _ := parseVodafoneTime(time.RFC3339, values["lastEventTime"])
	return contracts.VodafoneLastEvent{
		CallType:     strings.TrimSpace(values["callType"]),
		EventTime:    eventTime,
		EventTimeRaw: strings.TrimSpace(values["lastEventTime"]),
	}, nil
}

func (s *VodafoneService) requireAvailableSubscriber(msisdn string) (vodafoneSubscriber, error) {
	subscriber, ok, err := s.lookupSubscriber(msisdn)
	if err != nil {
		return vodafoneSubscriber{}, err
	}
	if !ok {
		return vodafoneSubscriber{}, fmt.Errorf("vodafone: номер %s відсутній у списку доступних IoT SIM", msisdn)
	}
	return subscriber, nil
}

func (s *VodafoneService) lookupSubscriber(msisdn string) (vodafoneSubscriber, bool, error) {
	subscribers, err := s.listAvailableSubscribers(context.Background())
	if err != nil {
		return vodafoneSubscriber{}, false, err
	}
	subscriber, ok := subscribers[msisdn]
	return subscriber, ok, nil
}

func (s *VodafoneService) listAvailableSubscribers(ctx context.Context) (map[string]vodafoneSubscriber, error) {
	s.mu.Lock()
	if len(s.availableSIMs) > 0 && time.Since(s.availableSIMsAt) < 5*time.Minute {
		cached := cloneVodafoneSubscribers(s.availableSIMs)
		s.mu.Unlock()
		return cached, nil
	}
	s.mu.Unlock()

	token, err := s.ensureAuthorizedToken()
	if err != nil {
		return nil, err
	}

	subscribers := make(map[string]vodafoneSubscriber)
	const limit = 100
	for offset := 0; ; offset += limit {
		reqURL := fmt.Sprintf(
			"%s/customer/api/customerManagement/v3/customer?offset=%d&limit=%d&sort=+msisdn&category.id=%s&fields=description",
			s.baseURL,
			offset,
			limit,
			vodafoneIOTCategory,
		)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("vodafone: failed to create customers request: %w", err)
		}
		s.applyHeaders(req, token, "GET-FILTERED-BILLING-ACCOUNT-TAGS", "application/json")

		var resp []struct {
			ID      string `json:"id"`
			Account struct {
				ID string `json:"id"`
			} `json:"account"`
			RelatedParty []struct {
				ID              string               `json:"id"`
				Characteristics []vodafoneNamedValue `json:"characteristics"`
				Characterictics []vodafoneNamedValue `json:"characterictics"`
			} `json:"relatedParty"`
		}
		if err := s.doJSON(req, &resp); err != nil {
			return nil, err
		}

		for _, item := range resp {
			for _, party := range item.RelatedParty {
				msisdnValue := strings.TrimSpace(party.ID)
				if msisdnValue == "" {
					continue
				}
				description := ""
				for _, pair := range party.Characteristics {
					if subtle.ConstantTimeCompare([]byte(pair.Name), []byte("phoneDescription")) == 1 {
						description = strings.TrimSpace(pair.Value)
					}
				}
				for _, pair := range party.Characterictics {
					if subtle.ConstantTimeCompare([]byte(pair.Name), []byte("phoneDescription")) == 1 {
						description = strings.TrimSpace(pair.Value)
					}
				}
				subscribers[msisdnValue] = vodafoneSubscriber{
					MSISDN:      msisdnValue,
					AccountID:   strings.TrimSpace(item.Account.ID),
					Description: description,
				}
			}
		}
		if len(resp) < limit {
			break
		}
	}

	s.mu.Lock()
	s.availableSIMs = cloneVodafoneSubscribers(subscribers)
	s.availableSIMsAt = time.Now()
	s.mu.Unlock()
	return subscribers, nil
}

func (s *VodafoneService) getGuestToken() (string, error) {
	form := url.Values{
		"username": {""},
		"password": {""},
	}
	req, err := http.NewRequest(
		http.MethodPost,
		s.baseURL+"/uaa/oauth/token?grant_type=client_credentials&Profile=MYVODAFONE",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("vodafone: failed to create guest-token request: %w", err)
	}
	req.SetBasicAuth(vodafoneClientID, vodafoneClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp struct {
		AccessToken string `json:"access_token"`
	}
	if err := s.doJSON(req, &resp); err != nil {
		return "", err
	}
	if strings.TrimSpace(resp.AccessToken) == "" {
		return "", errors.New("vodafone: не вдалося отримати guest token")
	}
	return strings.TrimSpace(resp.AccessToken), nil
}

func (s *VodafoneService) ensureAuthorizedToken() (string, error) {
	state, err := s.AuthState()
	if err != nil {
		return "", err
	}
	if !state.Authorized {
		return "", errVodafoneAuthRequired
	}
	return strings.TrimSpace(s.loadConfig().AccessToken), nil
}

func (s *VodafoneService) doJSON(req *http.Request, out any) error {
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vodafone: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		_ = s.ClearLogin()
		return errVodafoneAuthRequired
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("vodafone: %s", msg)
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("vodafone: failed to decode response: %w", err)
	}
	return nil
}

func (s *VodafoneService) expectNoContent(req *http.Request, expected ...int) error {
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vodafone: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		_ = s.ClearLogin()
		return errVodafoneAuthRequired
	}
	for _, statusCode := range expected {
		if resp.StatusCode == statusCode {
			return nil
		}
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = resp.Status
	}
	return fmt.Errorf("vodafone: %s", msg)
}

func (s *VodafoneService) applyHeaders(req *http.Request, token string, profile string, contentType string) {
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
	req.Header.Set("X-App-Version", vodafoneAppVersion)
	req.Header.Set("X-Device-Source", "Windows OS")
	req.Header.Set("X-Dev-Mode", "true")
	req.Header.Set("Accept-Language", "uk")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://b2b.vodafone.ua")
	req.Header.Set("Referer", "https://b2b.vodafone.ua/")
	if strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", strings.TrimSpace(contentType))
	}
	if strings.TrimSpace(profile) != "" {
		req.Header.Set("Profile", strings.TrimSpace(profile))
	}
}

func (s *VodafoneService) loadConfig() config.VodafoneConfig {
	if s == nil || s.store == nil {
		return config.VodafoneConfig{}
	}
	return s.store.LoadVodafoneConfig()
}

func (s *VodafoneService) saveConfig(cfg config.VodafoneConfig) {
	if s == nil || s.store == nil {
		return
	}
	s.store.SaveVodafoneConfig(cfg)
}

func (s *VodafoneService) invalidateAvailableSIMsCache() {
	s.mu.Lock()
	s.availableSIMs = nil
	s.availableSIMsAt = time.Time{}
	s.mu.Unlock()
}

type vodafoneSubscriber struct {
	MSISDN      string
	AccountID   string
	Description string
}

type vodafoneNamedValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func cloneVodafoneSubscribers(src map[string]vodafoneSubscriber) map[string]vodafoneSubscriber {
	out := make(map[string]vodafoneSubscriber, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func parseVodafoneTime(layout string, raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	return time.ParseInLocation(layout, raw, time.UTC)
}

func resolveVodafoneTokenExpiry(token string, expiresIn int64) time.Time {
	if exp, ok := jwtTokenExpiry(token); ok {
		return exp.UTC()
	}
	if expiresIn > 0 {
		return time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
	}
	return time.Time{}
}

func jwtTokenExpiry(token string) (time.Time, bool) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) < 2 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp <= 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0).UTC(), true
}

func normalizeVodafoneMSISDN(raw string) (string, error) {
	digits := digitsOnlyVodafone(raw)
	if digits == "" {
		return "", errors.New("vodafone: номер SIM порожній")
	}

	switch {
	case len(digits) == 12 && strings.HasPrefix(digits, "380"):
		return digits, nil
	case len(digits) == 10 && strings.HasPrefix(digits, "0"):
		return "38" + digits, nil
	case len(digits) == 11 && strings.HasPrefix(digits, "80"):
		return "3" + digits, nil
	case len(digits) == 13 && strings.HasPrefix(digits, "0380"):
		return digits[1:], nil
	case len(digits) == 9:
		return "380" + digits, nil
	default:
		return "", fmt.Errorf("vodafone: некоректний формат номера %q", raw)
	}
}

func digitsOnlyVodafone(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
