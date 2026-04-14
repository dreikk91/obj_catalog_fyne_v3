package data

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const kyivstarBaseURL = "https://b2b-api.kyivstar.ua"

var errKyivstarMetadataUnsupported = errors.New("kyivstar: у наданій API-специфікації немає endpoint для запису deviceName/deviceId")

type KyivstarService struct {
	baseURL    string
	httpClient *http.Client
	store      config.KyivstarConfigStore

	mu sync.Mutex
}

type KyivstarOption func(*KyivstarService)

func WithKyivstarHTTPClient(client *http.Client) KyivstarOption {
	return func(s *KyivstarService) {
		if s == nil || client == nil {
			return
		}
		s.httpClient = client
	}
}

func WithKyivstarBaseURL(baseURL string) KyivstarOption {
	return func(s *KyivstarService) {
		if s == nil || strings.TrimSpace(baseURL) == "" {
			return
		}
		s.baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	}
}

func NewKyivstarService(store config.KyivstarConfigStore, opts ...KyivstarOption) *KyivstarService {
	service := &KyivstarService{
		baseURL:    kyivstarBaseURL,
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

func (s *KyivstarService) AuthState() (contracts.KyivstarAuthState, error) {
	cfg := s.loadConfig()
	expiry := cfg.TokenExpiryTime()
	if !expiry.IsZero() && expiry.Before(time.Now().UTC()) && cfg.HasAccessToken() {
		cfg.AccessToken = ""
		cfg.TokenExpiry = ""
		s.saveConfig(cfg)
	}

	state := contracts.KyivstarAuthState{
		ClientID:       strings.TrimSpace(cfg.ClientID),
		UserEmail:      strings.TrimSpace(cfg.UserEmail),
		Configured:     cfg.HasCredentials(),
		Authorized:     cfg.TokenUsableAt(time.Now().UTC()),
		TokenExpiresAt: cfg.TokenExpiryTime(),
	}
	return state, nil
}

func (s *KyivstarService) RefreshToken() (contracts.KyivstarAuthState, error) {
	if _, err := s.ensureAuthorizedToken(); err != nil {
		return contracts.KyivstarAuthState{}, err
	}
	return s.AuthState()
}

func (s *KyivstarService) ClearToken() error {
	cfg := s.loadConfig()
	cfg.AccessToken = ""
	cfg.TokenExpiry = ""
	s.saveConfig(cfg)
	return nil
}

func (s *KyivstarService) GetSIMStatus(msisdn string) (contracts.KyivstarSIMStatus, error) {
	normalized, err := normalizeKyivstarMSISDN(msisdn)
	if err != nil {
		return contracts.KyivstarSIMStatus{}, err
	}

	number, available, err := s.lookupNumber(normalized)
	if err != nil {
		return contracts.KyivstarSIMStatus{}, err
	}

	status := contracts.KyivstarSIMStatus{
		MSISDN:       normalized,
		Available:    available,
		DeviceName:   strings.TrimSpace(number.DeviceName),
		DeviceID:     strings.TrimSpace(number.DeviceID),
		ICCID:        strings.TrimSpace(number.ICCID),
		IMEI:         strings.TrimSpace(number.IMEI),
		TariffPlan:   strings.TrimSpace(number.TariffPlan),
		Account:      strings.TrimSpace(number.Account),
		DataUsage:    strings.TrimSpace(number.DataUsage),
		SMSUsage:     strings.TrimSpace(number.SMSUsage),
		VoiceUsage:   strings.TrimSpace(number.VoiceUsage),
		IsTestPeriod: number.IsTestPeriod,
		IsOnline:     number.IsOnline,
	}
	if !available {
		return status, nil
	}

	numberStatus, err := s.fetchNumberStatus(normalized)
	if err != nil {
		return contracts.KyivstarSIMStatus{}, err
	}
	services, err := s.fetchNumberServices(normalized)
	if err != nil {
		return contracts.KyivstarSIMStatus{}, err
	}

	status.NumberStatus = strings.TrimSpace(numberStatus.Status)
	status.AvailableActions = utils.TrimmedNonEmptyStrings(numberStatus.AvailableActions)
	status.Services = services
	return status, nil
}

func (s *KyivstarService) ListSIMInventory(numbers []string) (map[string]contracts.KyivstarSIMInventoryEntry, error) {
	result := make(map[string]contracts.KyivstarSIMInventoryEntry)

	normalized := make([]string, 0, len(numbers))
	seen := make(map[string]struct{}, len(numbers))
	for _, number := range numbers {
		msisdn, err := normalizeKyivstarMSISDN(number)
		if err != nil {
			continue
		}
		if _, ok := seen[msisdn]; ok {
			continue
		}
		seen[msisdn] = struct{}{}
		normalized = append(normalized, msisdn)
	}
	if len(normalized) == 0 {
		return result, nil
	}

	const batchSize = 100
	for start := 0; start < len(normalized); start += batchSize {
		end := start + batchSize
		if end > len(normalized) {
			end = len(normalized)
		}
		items, err := s.fetchCompanyNumbers(normalized[start:end])
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			msisdn := strings.TrimSpace(item.Number)
			if msisdn == "" {
				continue
			}
			result[msisdn] = contracts.KyivstarSIMInventoryEntry{
				MSISDN:       msisdn,
				Status:       strings.TrimSpace(item.Status),
				DeviceName:   strings.TrimSpace(item.DeviceName),
				DeviceID:     strings.TrimSpace(item.DeviceID),
				IsOnline:     item.IsOnline,
				IsTestPeriod: item.IsTestPeriod,
			}
		}
	}

	return result, nil
}

func (s *KyivstarService) PauseSIM(msisdn string) (contracts.KyivstarSIMOperationResult, error) {
	return s.changeNumberStatus(msisdn, "pause")
}

func (s *KyivstarService) ActivateSIM(msisdn string) (contracts.KyivstarSIMOperationResult, error) {
	return s.changeNumberStatus(msisdn, "activate")
}

func (s *KyivstarService) PauseSIMServices(msisdn string, serviceIDs []string) (contracts.KyivstarSIMOperationResult, error) {
	return s.changeNumberServices(msisdn, serviceIDs, "pause")
}

func (s *KyivstarService) ActivateSIMServices(msisdn string, serviceIDs []string) (contracts.KyivstarSIMOperationResult, error) {
	return s.changeNumberServices(msisdn, serviceIDs, "activate")
}

func (s *KyivstarService) RebootSIM(msisdn string) (contracts.KyivstarSIMResetResult, error) {
	normalized, err := normalizeKyivstarMSISDN(msisdn)
	if err != nil {
		return contracts.KyivstarSIMResetResult{}, err
	}
	if _, err := s.requireAvailableNumber(normalized); err != nil {
		return contracts.KyivstarSIMResetResult{}, err
	}

	cfg := s.loadConfig()
	email := strings.TrimSpace(cfg.UserEmail)
	if email == "" {
		return contracts.KyivstarSIMResetResult{}, errors.New("kyivstar: вкажіть email компанії у налаштуваннях")
	}

	token, err := s.ensureAuthorizedToken()
	if err != nil {
		return contracts.KyivstarSIMResetResult{}, err
	}

	payload := map[string]any{
		"email":   email,
		"numbers": []string{normalized},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return contracts.KyivstarSIMResetResult{}, fmt.Errorf("kyivstar: failed to build reset request: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		s.baseURL+"/rest/iot/company-numbers/reset",
		bytes.NewReader(body),
	)
	if err != nil {
		return contracts.KyivstarSIMResetResult{}, fmt.Errorf("kyivstar: failed to create reset request: %w", err)
	}
	s.applyBearerHeaders(req, token, "application/json")

	if err := s.expectNoContent(req, http.StatusNoContent); err != nil {
		return contracts.KyivstarSIMResetResult{}, err
	}
	return contracts.KyivstarSIMResetResult{
		MSISDN: normalized,
		Email:  email,
	}, nil
}

func (s *KyivstarService) UpdateSIMMetadata(msisdn string, deviceName string, deviceID string) error {
	if _, err := normalizeKyivstarMSISDN(msisdn); err != nil {
		return err
	}
	if strings.TrimSpace(deviceName) == "" && strings.TrimSpace(deviceID) == "" {
		return errors.New("kyivstar: немає даних для запису deviceName/deviceId")
	}
	return errKyivstarMetadataUnsupported
}

func (s *KyivstarService) changeNumberStatus(msisdn string, action string) (contracts.KyivstarSIMOperationResult, error) {
	normalized, err := normalizeKyivstarMSISDN(msisdn)
	if err != nil {
		return contracts.KyivstarSIMOperationResult{}, err
	}
	if _, err := s.requireAvailableNumber(normalized); err != nil {
		return contracts.KyivstarSIMOperationResult{}, err
	}

	token, err := s.ensureAuthorizedToken()
	if err != nil {
		return contracts.KyivstarSIMOperationResult{}, err
	}

	payload := []map[string]string{{
		"number": normalized,
		"action": strings.TrimSpace(action),
	}}
	body, err := json.Marshal(payload)
	if err != nil {
		return contracts.KyivstarSIMOperationResult{}, fmt.Errorf("kyivstar: failed to build status request: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		s.baseURL+"/rest/iot/company-numbers/statuses",
		bytes.NewReader(body),
	)
	if err != nil {
		return contracts.KyivstarSIMOperationResult{}, fmt.Errorf("kyivstar: failed to create status request: %w", err)
	}
	s.applyBearerHeaders(req, token, "application/json")

	if err := s.expectNoContent(req, http.StatusNoContent); err != nil {
		return contracts.KyivstarSIMOperationResult{}, err
	}
	return contracts.KyivstarSIMOperationResult{
		MSISDN:    normalized,
		Operation: strings.TrimSpace(action),
	}, nil
}

func (s *KyivstarService) changeNumberServices(msisdn string, serviceIDs []string, action string) (contracts.KyivstarSIMOperationResult, error) {
	normalized, err := normalizeKyivstarMSISDN(msisdn)
	if err != nil {
		return contracts.KyivstarSIMOperationResult{}, err
	}
	if _, err := s.requireAvailableNumber(normalized); err != nil {
		return contracts.KyivstarSIMOperationResult{}, err
	}

	selected := make([]map[string]string, 0, len(serviceIDs))
	for _, serviceID := range serviceIDs {
		serviceID = strings.TrimSpace(serviceID)
		if serviceID == "" {
			continue
		}
		selected = append(selected, map[string]string{
			"serviceId": serviceID,
			"action":    strings.TrimSpace(action),
		})
	}
	if len(selected) == 0 {
		return contracts.KyivstarSIMOperationResult{}, errors.New("kyivstar: виберіть хоча б один сервіс")
	}

	token, err := s.ensureAuthorizedToken()
	if err != nil {
		return contracts.KyivstarSIMOperationResult{}, err
	}

	payload := []map[string]any{{
		"number":   normalized,
		"services": selected,
	}}
	body, err := json.Marshal(payload)
	if err != nil {
		return contracts.KyivstarSIMOperationResult{}, fmt.Errorf("kyivstar: failed to build service request: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		s.baseURL+"/rest/iot/company-numbers/services",
		bytes.NewReader(body),
	)
	if err != nil {
		return contracts.KyivstarSIMOperationResult{}, fmt.Errorf("kyivstar: failed to create service request: %w", err)
	}
	s.applyBearerHeaders(req, token, "application/json")

	if err := s.expectNoContent(req, http.StatusNoContent); err != nil {
		return contracts.KyivstarSIMOperationResult{}, err
	}
	return contracts.KyivstarSIMOperationResult{
		MSISDN:    normalized,
		Operation: strings.TrimSpace(action),
	}, nil
}

func (s *KyivstarService) requireAvailableNumber(msisdn string) (kyivstarIOTNumber, error) {
	number, available, err := s.lookupNumber(msisdn)
	if err != nil {
		return kyivstarIOTNumber{}, err
	}
	if !available {
		return kyivstarIOTNumber{}, fmt.Errorf("kyivstar: %s відсутній у списку доступних IoT номерів", msisdn)
	}
	return number, nil
}

func (s *KyivstarService) lookupNumber(msisdn string) (kyivstarIOTNumber, bool, error) {
	numbers, err := s.fetchCompanyNumbers([]string{msisdn})
	if err != nil {
		return kyivstarIOTNumber{}, false, err
	}
	for _, number := range numbers {
		if strings.TrimSpace(number.Number) == msisdn {
			return number, true, nil
		}
	}
	return kyivstarIOTNumber{}, false, nil
}

func (s *KyivstarService) fetchCompanyNumbers(numbers []string) ([]kyivstarIOTNumber, error) {
	token, err := s.ensureAuthorizedToken()
	if err != nil {
		return nil, err
	}

	reqURL, err := url.Parse(s.baseURL + "/rest/iot/company-numbers")
	if err != nil {
		return nil, fmt.Errorf("kyivstar: failed to build company-numbers url: %w", err)
	}
	query := reqURL.Query()
	query.Set("size", "100")
	for _, number := range numbers {
		number = strings.TrimSpace(number)
		if number == "" {
			continue
		}
		query.Add("numbers", number)
	}
	reqURL.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("kyivstar: failed to create company-numbers request: %w", err)
	}
	s.applyBearerHeaders(req, token, "")

	var page struct {
		Content []kyivstarIOTNumber `json:"content"`
	}
	if err := s.doJSON(req, &page); err != nil {
		return nil, err
	}
	return page.Content, nil
}

func (s *KyivstarService) fetchNumberStatus(msisdn string) (kyivstarNumberStatus, error) {
	token, err := s.ensureAuthorizedToken()
	if err != nil {
		return kyivstarNumberStatus{}, err
	}

	reqURL, err := url.Parse(s.baseURL + "/rest/iot/company-numbers/statuses")
	if err != nil {
		return kyivstarNumberStatus{}, fmt.Errorf("kyivstar: failed to build statuses url: %w", err)
	}
	query := reqURL.Query()
	query.Set("number", msisdn)
	reqURL.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return kyivstarNumberStatus{}, fmt.Errorf("kyivstar: failed to create status request: %w", err)
	}
	s.applyBearerHeaders(req, token, "")

	var response kyivstarNumberStatus
	if err := s.doJSON(req, &response); err != nil {
		return kyivstarNumberStatus{}, err
	}
	return response, nil
}

func (s *KyivstarService) fetchNumberServices(msisdn string) ([]contracts.KyivstarSIMServiceStatus, error) {
	token, err := s.ensureAuthorizedToken()
	if err != nil {
		return nil, err
	}

	reqURL, err := url.Parse(s.baseURL + "/rest/iot/company-numbers/services")
	if err != nil {
		return nil, fmt.Errorf("kyivstar: failed to build services url: %w", err)
	}
	query := reqURL.Query()
	query.Set("number", msisdn)
	reqURL.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("kyivstar: failed to create services request: %w", err)
	}
	s.applyBearerHeaders(req, token, "")

	var response []kyivstarNumberService
	if err := s.doJSON(req, &response); err != nil {
		return nil, err
	}

	services := make([]contracts.KyivstarSIMServiceStatus, 0, len(response))
	for _, item := range response {
		services = append(services, contracts.KyivstarSIMServiceStatus{
			ServiceID:        strings.TrimSpace(item.ServiceID),
			Name:             strings.TrimSpace(item.Name),
			Status:           strings.TrimSpace(item.Status),
			AvailableActions: utils.TrimmedNonEmptyStrings(item.AvailableActions),
		})
	}
	return services, nil
}

func (s *KyivstarService) ensureAuthorizedToken() (string, error) {
	state, err := s.AuthState()
	if err != nil {
		return "", err
	}
	if state.Authorized {
		cfg := s.loadConfig()
		return strings.TrimSpace(cfg.AccessToken), nil
	}
	return s.fetchAccessToken()
}

func (s *KyivstarService) fetchAccessToken() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := s.loadConfig()
	now := time.Now().UTC()
	if cfg.TokenUsableAt(now) {
		return strings.TrimSpace(cfg.AccessToken), nil
	}
	if !cfg.HasCredentials() {
		return "", errors.New("kyivstar: client_id/client_secret не налаштовані")
	}

	form := url.Values{
		"grant_type": {"client_credentials"},
	}
	req, err := http.NewRequest(
		http.MethodPost,
		s.baseURL+"/idp/oauth2/token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("kyivstar: failed to create token request: %w", err)
	}
	req.SetBasicAuth(strings.TrimSpace(cfg.ClientID), strings.TrimSpace(cfg.ClientSecret))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	var response struct {
		AccessToken string          `json:"access_token"`
		ExpiresIn   json.RawMessage `json:"expires_in"`
	}
	if err := s.doJSON(req, &response); err != nil {
		return "", err
	}
	token := strings.TrimSpace(response.AccessToken)
	if token == "" {
		return "", errors.New("kyivstar: сервер не повернув access token")
	}

	cfg.AccessToken = token
	cfg.TokenExpiry = ""
	if seconds, ok := parseKyivstarExpiresIn(response.ExpiresIn); ok && seconds > 0 {
		expiry := now.Add(time.Duration(seconds) * time.Second)
		if seconds > 60 {
			expiry = expiry.Add(-30 * time.Second)
		}
		cfg.TokenExpiry = expiry.Format(time.RFC3339)
	}
	s.saveConfig(cfg)
	return token, nil
}

func (s *KyivstarService) doJSON(req *http.Request, out any) error {
	if s == nil || s.httpClient == nil {
		return errors.New("kyivstar: http client is not configured")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("kyivstar: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return decodeKyivstarAPIError(resp)
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("kyivstar: failed to read response body: %w", err)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("kyivstar: failed to decode response: %w", err)
	}
	return nil
}

func (s *KyivstarService) expectNoContent(req *http.Request, expected ...int) error {
	if s == nil || s.httpClient == nil {
		return errors.New("kyivstar: http client is not configured")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("kyivstar: request failed: %w", err)
	}
	defer resp.Body.Close()

	for _, code := range expected {
		if resp.StatusCode == code {
			_, _ = io.Copy(io.Discard, resp.Body)
			return nil
		}
	}
	return decodeKyivstarAPIError(resp)
}

func (s *KyivstarService) applyBearerHeaders(req *http.Request, token string, contentType string) {
	if req == nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", strings.TrimSpace(contentType))
	}
}

func (s *KyivstarService) loadConfig() config.KyivstarConfig {
	if s == nil || s.store == nil {
		return config.KyivstarConfig{}
	}
	return s.store.LoadKyivstarConfig()
}

func (s *KyivstarService) saveConfig(cfg config.KyivstarConfig) {
	if s == nil || s.store == nil {
		return
	}
	s.store.SaveKyivstarConfig(cfg)
}

func decodeKyivstarAPIError(resp *http.Response) error {
	if resp == nil {
		return errors.New("kyivstar: empty http response")
	}

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		log.Debug().Err(readErr).Msg("kyivstar: failed to read error response body")
	}
	var payload struct {
		Message          string `json:"message"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &payload)

	message := strings.TrimSpace(payload.Message)
	if message == "" {
		message = strings.TrimSpace(payload.ErrorDescription)
	}
	if message == "" {
		message = strings.TrimSpace(payload.Error)
	}
	if message == "" {
		message = strings.TrimSpace(string(body))
	}
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}
	return fmt.Errorf("kyivstar: %s (HTTP %d)", message, resp.StatusCode)
}

func normalizeKyivstarMSISDN(raw string) (string, error) {
	digits := digitsOnlyKyivstar(raw)
	switch {
	case len(digits) == 10 && strings.HasPrefix(digits, "0") && isKyivstarLocalCode(digits[1:3]):
		return "38" + digits, nil
	case len(digits) == 12 && strings.HasPrefix(digits, "380") && isKyivstarLocalCode(digits[3:5]):
		return digits, nil
	default:
		return "", errors.New("kyivstar: номер має починатися з 067/068/096/097/098/077 або 38067/38068/38096/38097/38098/38077")
	}
}

func digitsOnlyKyivstar(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isKyivstarLocalCode(code string) bool {
	switch strings.TrimSpace(code) {
	case "67", "68", "96", "97", "98", "77":
		return true
	default:
		return false
	}
}

func parseKyivstarExpiresIn(raw json.RawMessage) (int64, bool) {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return 0, false
	}

	var asInt int64
	if err := json.Unmarshal(raw, &asInt); err == nil {
		return asInt, true
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		asString = strings.TrimSpace(asString)
		if asString == "" {
			return 0, false
		}
		value, err := strconv.ParseInt(asString, 10, 64)
		if err != nil {
			return 0, false
		}
		return value, true
	}

	return 0, false
}

type kyivstarIOTNumber struct {
	Number       string `json:"number"`
	ICCID        string `json:"iccid"`
	IMEI         string `json:"imei"`
	TariffPlan   string `json:"tariffPlan"`
	Status       string `json:"status"`
	Account      string `json:"account"`
	DeviceName   string `json:"deviceName"`
	Group        string `json:"group"`
	DeviceID     string `json:"deviceId"`
	DataUsage    string `json:"dataUsage"`
	SMSUsage     string `json:"smsUsage"`
	VoiceUsage   string `json:"voiceUsage"`
	IsTestPeriod bool   `json:"isTestPeriod"`
	IsOnline     bool   `json:"isOnline"`
}

type kyivstarNumberStatus struct {
	Status           string   `json:"status"`
	AvailableActions []string `json:"availableActions"`
}

type kyivstarNumberService struct {
	Status           string   `json:"status"`
	ServiceID        string   `json:"serviceId"`
	Name             string   `json:"name"`
	AvailableActions []string `json:"availableActions"`
}
