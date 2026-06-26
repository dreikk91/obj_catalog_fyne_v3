package omnicell

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/utils"
)

type Client struct {
	endpoint   string
	login      string
	password   string
	source     string
	httpClient *http.Client
}

type SendRequest struct {
	Phone string
	Text  string
}

type SendResponse struct {
	StatusCode int
	Body       string
}

type messageXML struct {
	XMLName xml.Name   `xml:"message"`
	Service serviceXML `xml:"service"`
	To      string     `xml:"to"`
	Body    bodyXML    `xml:"body"`
}

type serviceXML struct {
	ID     string `xml:"id,attr"`
	Source string `xml:"source,attr"`
	Type   string `xml:"type,attr"`
}

type bodyXML struct {
	ContentType string `xml:"content-type,attr"`
	Text        string `xml:",chardata"`
}

func NewClient(cfg config.OmnicellConfig) *Client {
	return NewClientWithHTTPClient(cfg, &http.Client{Timeout: 20 * time.Second})
}

func NewClientWithHTTPClient(cfg config.OmnicellConfig, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{
		endpoint:   strings.TrimSpace(cfg.Endpoint),
		login:      strings.TrimSpace(cfg.Login),
		password:   cfg.Password,
		source:     strings.TrimSpace(cfg.Source),
		httpClient: httpClient,
	}
}

func (c *Client) SendSMS(ctx context.Context, req SendRequest) (SendResponse, error) {
	if c == nil {
		return SendResponse{}, fmt.Errorf("omnicell client is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	endpoint := strings.TrimSpace(c.endpoint)
	if endpoint == "" {
		return SendResponse{}, fmt.Errorf("omnicell endpoint is empty")
	}
	if strings.TrimSpace(c.login) == "" || strings.TrimSpace(c.password) == "" {
		return SendResponse{}, fmt.Errorf("omnicell credentials are empty")
	}
	if strings.TrimSpace(c.source) == "" {
		return SendResponse{}, fmt.Errorf("omnicell source is empty")
	}

	msisdn, err := NormalizeMSISDN(req.Phone)
	if err != nil {
		return SendResponse{}, err
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return SendResponse{}, fmt.Errorf("sms text is empty")
	}

	payload, err := xml.Marshal(messageXML{
		Service: serviceXML{ID: "single", Source: c.source, Type: "SMS"},
		To:      msisdn,
		Body:    bodyXML{ContentType: "text/plain", Text: text},
	})
	if err != nil {
		return SendResponse{}, fmt.Errorf("build omnicell xml: %w", err)
	}
	payload = append([]byte(xml.Header), payload...)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return SendResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "text/xml; charset=utf-8")
	httpReq.SetBasicAuth(c.login, c.password)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return SendResponse{}, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	result := SendResponse{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	if readErr != nil {
		return result, readErr
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return result, fmt.Errorf("omnicell returned HTTP %d: %s", resp.StatusCode, result.Body)
	}
	return result, nil
}

func NormalizeMSISDN(raw string) (string, error) {
	digits := utils.DigitsOnly(raw)
	switch {
	case len(digits) == 12 && strings.HasPrefix(digits, "380"):
		return digits, nil
	case len(digits) == 10 && strings.HasPrefix(digits, "0"):
		return "38" + digits, nil
	default:
		return "", fmt.Errorf("invalid ukrainian msisdn: %q", raw)
	}
}
