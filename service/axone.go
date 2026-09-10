package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
)

type AxoneChain struct {
	ChainID   string `json:"chain_id"`
	ChainName string `json:"chain_name"`
	Symbol    string `json:"symbol"`
}

type AxoneWallet struct {
	ID           string  `json:"id"`
	Currency     string  `json:"currency"`
	Amount       float64 `json:"amount"`
	Price        float64 `json:"price"`
	TotalBalance float64 `json:"total_balance"`
	Percent      float64 `json:"percent"`
	CurrencyIcon string  `json:"currency_icon"`
}

type AxoneWalletListData struct {
	Total   int           `json:"total"`
	Current int           `json:"current"`
	List    []AxoneWallet `json:"list"`
}

type axoneLoginData struct {
	AccessToken           string `json:"accessToken"`
	RefreshToken          string `json:"refreshToken"`
	AccessTokenExpiresAt  int64  `json:"accessTokenExpiresAt"`
	RefreshTokenExpiresAt int64  `json:"refreshTokenExpiresAt"`
}

type axoneTokenState struct {
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  int64
	RefreshTokenExpiresAt int64
}

type axoneResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type axoneChainListData struct {
	Data []AxoneChain `json:"data"`
}

type AxonePaymentOrderRequest struct {
	OrderNo              string `json:"orderNo"`
	Amount               string `json:"amount"`
	Currency             string `json:"currency"`
	Chain                string `json:"chain"`
	PaymentWalletAddress string `json:"paymentWalletAddress"`
}

type AxonePaymentOrderData struct {
	OrderNo      string `json:"orderNo"`
	AxoneOrderNo string `json:"axoneOrderNo"`
	Status       string `json:"status"`
	Amount       string `json:"amount"`
	Currency     string `json:"currency"`
	PayAddress   string `json:"payAddress"`
}

type AxonePaygoSessionData struct {
	SessionID       string  `json:"session_id"`
	Status          string  `json:"status"`
	Currency        string  `json:"currency"`
	ReservedAmount  string  `json:"reserved_amount"`
	ConsumedAmount  string  `json:"consumed_amount"`
	RemainingAmount string  `json:"remaining_amount"`
	LastEventSeq    int64   `json:"last_event_seq"`
	ExpiresAt       string  `json:"expires_at"`
	ClosedAt        *string `json:"closed_at"`
}

type AxonePaygoUsageData struct {
	SessionID          string `json:"session_id"`
	AcceptedThroughSeq int64  `json:"accepted_through_seq"`
	ChargeAmount       string `json:"charge_amount"`
	ConsumedAmount     string `json:"consumed_amount"`
	RemainingAmount    string `json:"remaining_amount"`
}

type axonePaygoCreateSessionRequest struct {
	WalletID  string `json:"wallet_id"`
	MaxAmount string `json:"max_amount"`
}

type axonePaygoUsageRequest struct {
	EventID      string `json:"event_id"`
	EventSeq     int64  `json:"event_seq"`
	ChargeAmount string `json:"charge_amount"`
}

type AxoneClient struct {
	configKey  string
	baseURL    string
	account    string
	password   string
	httpClient *http.Client
	tokenMu    sync.Mutex
	token      axoneTokenState
}

var (
	axoneClientMu sync.Mutex
	axoneClient   *AxoneClient
)

func GetAxoneClient() *AxoneClient {
	baseURL := strings.TrimRight(strings.TrimSpace(setting.AxoneBaseURL), "/")
	account := strings.TrimSpace(setting.AxoneAccount)
	password := setting.AxonePassword
	configKey := baseURL + "\n" + account + "\n" + password

	axoneClientMu.Lock()
	defer axoneClientMu.Unlock()

	if axoneClient != nil && axoneClient.configKey == configKey {
		axoneClient.httpClient = GetHttpClient()
		return axoneClient
	}

	axoneClient = &AxoneClient{
		configKey:  configKey,
		baseURL:    baseURL,
		account:    account,
		password:   password,
		httpClient: GetHttpClient(),
	}
	return axoneClient
}

func (c *AxoneClient) ListChains(ctx context.Context) ([]AxoneChain, error) {
	if err := c.ensureReady(); err != nil {
		return nil, err
	}

	accessToken, err := c.ensureAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	var resp axoneResponse[axoneChainListData]
	if err := c.doJSONRequest(ctx, http.MethodGet, "/web/crypto/chain", nil, accessToken, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, axoneResponseError(resp.Message)
	}
	return resp.Data.Data, nil
}

func (c *AxoneClient) ListWallets(ctx context.Context) (*AxoneWalletListData, error) {
	if err := c.ensureReady(); err != nil {
		return nil, err
	}

	accessToken, err := c.ensureAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	var resp axoneResponse[AxoneWalletListData]
	if err := c.doJSONRequest(ctx, http.MethodGet, "/web/crypto/wallets", nil, accessToken, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, axoneResponseError(resp.Message)
	}
	if resp.Data.List == nil {
		resp.Data.List = []AxoneWallet{}
	}
	return &resp.Data, nil
}

func (c *AxoneClient) GetWalletAddress(ctx context.Context, currency string, chainID string) (string, error) {
	if err := c.ensureReady(); err != nil {
		return "", err
	}

	accessToken, err := c.ensureAccessToken(ctx)
	if err != nil {
		return "", err
	}

	query := url.Values{}
	query.Set("currency", strings.ToUpper(strings.TrimSpace(currency)))
	query.Set("chain_ids", strings.TrimSpace(chainID))

	var resp axoneResponse[struct {
		Adress string `json:"adress"`
	}]
	if err := c.doJSONRequest(ctx, http.MethodGet, "/web/crypto/chain/address/v2?"+query.Encode(), nil, accessToken, &resp); err != nil {
		return "", err
	}
	if resp.Code != 0 {
		return "", axoneResponseError(resp.Message)
	}

	address := strings.TrimSpace(resp.Data.Adress)
	if address == "" {
		return "", fmt.Errorf("axone returned empty address")
	}
	return address, nil
}

func (c *AxoneClient) CreatePaymentOrder(ctx context.Context, request AxonePaymentOrderRequest) (*AxonePaymentOrderData, error) {
	if err := c.ensurePaymentOrderReady(); err != nil {
		return nil, err
	}
	accessToken, err := c.ensureAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	request.OrderNo = strings.TrimSpace(request.OrderNo)
	request.Amount = strings.TrimSpace(request.Amount)
	request.Currency = strings.ToUpper(strings.TrimSpace(request.Currency))
	request.Chain = strings.TrimSpace(request.Chain)
	request.PaymentWalletAddress = strings.TrimSpace(request.PaymentWalletAddress)

	var resp axoneResponse[AxonePaymentOrderData]
	if err := c.doSignedJSONRequest(ctx, http.MethodPost, "/api/v1/payment/orders", request, accessToken, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, axoneResponseError(resp.Message)
	}
	if strings.TrimSpace(resp.Data.PayAddress) == "" {
		return nil, fmt.Errorf("axone returned empty pay address")
	}
	return &resp.Data, nil
}

func (c *AxoneClient) CreatePaygoSession(ctx context.Context, walletID string, maxAmount string, idempotencyKey string) (*AxonePaygoSessionData, error) {
	if err := c.ensureReady(); err != nil {
		return nil, err
	}
	accessToken, err := c.ensureAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	payload := axonePaygoCreateSessionRequest{
		WalletID:  strings.TrimSpace(walletID),
		MaxAmount: strings.TrimSpace(maxAmount),
	}
	var resp axoneResponse[AxonePaygoSessionData]
	if err := c.doJSONRequestWithHeaders(ctx, http.MethodPost, "/web/agen-pay/sessions", payload, accessToken, map[string]string{
		"Idempotency-Key": strings.TrimSpace(idempotencyKey),
	}, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, axoneResponseError(resp.Message)
	}
	if strings.TrimSpace(resp.Data.SessionID) == "" {
		return nil, fmt.Errorf("axone returned empty paygo session id")
	}
	return &resp.Data, nil
}

func (c *AxoneClient) SubmitPaygoUsage(ctx context.Context, sessionID string, eventID string, eventSeq int64, chargeAmount string, idempotencyKey string) (*AxonePaygoUsageData, error) {
	if err := c.ensureReady(); err != nil {
		return nil, err
	}
	accessToken, err := c.ensureAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	payload := axonePaygoUsageRequest{
		EventID:      strings.TrimSpace(eventID),
		EventSeq:     eventSeq,
		ChargeAmount: strings.TrimSpace(chargeAmount),
	}
	path := "/web/agen-pay/sessions/" + url.PathEscape(strings.TrimSpace(sessionID)) + "/usage"
	var resp axoneResponse[AxonePaygoUsageData]
	if err := c.doJSONRequestWithHeaders(ctx, http.MethodPost, path, payload, accessToken, map[string]string{
		"Idempotency-Key": strings.TrimSpace(idempotencyKey),
	}, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, axoneResponseError(resp.Message)
	}
	return &resp.Data, nil
}

func (c *AxoneClient) GetPaygoSession(ctx context.Context, sessionID string) (*AxonePaygoSessionData, error) {
	if err := c.ensureReady(); err != nil {
		return nil, err
	}
	accessToken, err := c.ensureAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	path := "/web/agen-pay/sessions/" + url.PathEscape(strings.TrimSpace(sessionID))
	var resp axoneResponse[AxonePaygoSessionData]
	if err := c.doJSONRequest(ctx, http.MethodGet, path, nil, accessToken, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, axoneResponseError(resp.Message)
	}
	return &resp.Data, nil
}

func (c *AxoneClient) ClosePaygoSession(ctx context.Context, sessionID string, idempotencyKey string) (*AxonePaygoSessionData, error) {
	if err := c.ensureReady(); err != nil {
		return nil, err
	}
	accessToken, err := c.ensureAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	path := "/web/agen-pay/sessions/" + url.PathEscape(strings.TrimSpace(sessionID)) + "/close"
	var resp axoneResponse[AxonePaygoSessionData]
	if err := c.doJSONRequestWithHeaders(ctx, http.MethodPost, path, nil, accessToken, map[string]string{
		"Idempotency-Key": strings.TrimSpace(idempotencyKey),
	}, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, axoneResponseError(resp.Message)
	}
	return &resp.Data, nil
}

func (c *AxoneClient) ensureReady() error {
	if c.baseURL == "" {
		return fmt.Errorf("axone base url is empty")
	}
	if c.account == "" {
		return fmt.Errorf("axone account is empty")
	}
	if c.password == "" {
		return fmt.Errorf("axone password is empty")
	}
	if c.httpClient == nil {
		c.httpClient = http.DefaultClient
	}
	return nil
}

func (c *AxoneClient) ensurePaymentOrderReady() error {
	return c.ensureReady()
}

func (c *AxoneClient) ensureAccessToken(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	nowMs := time.Now().Add(30 * time.Second).UnixMilli()
	if c.token.AccessToken != "" && c.token.AccessTokenExpiresAt > nowMs {
		return c.token.AccessToken, nil
	}

	if c.token.RefreshToken != "" && c.token.RefreshTokenExpiresAt > nowMs {
		if err := c.refreshLocked(ctx); err == nil {
			return c.token.AccessToken, nil
		}
	}

	if err := c.loginLocked(ctx); err != nil {
		return "", err
	}
	return c.token.AccessToken, nil
}

func (c *AxoneClient) loginLocked(ctx context.Context) error {
	payload := map[string]string{
		"loginMethod": "email",
		"email":       c.account,
		"password":    md5Hex(c.password),
		"loginType":   "api",
	}

	var resp axoneResponse[axoneLoginData]
	if err := c.doJSONRequest(ctx, http.MethodPost, "/web/user/login", payload, "", &resp); err != nil {
		return err
	}
	if resp.Code != 0 {
		return axoneResponseError(resp.Message)
	}
	if strings.TrimSpace(resp.Data.AccessToken) == "" {
		return fmt.Errorf("axone login returned empty access token")
	}
	c.token = axoneTokenState{
		AccessToken:           resp.Data.AccessToken,
		RefreshToken:          resp.Data.RefreshToken,
		AccessTokenExpiresAt:  resp.Data.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: resp.Data.RefreshTokenExpiresAt,
	}
	return nil
}

func (c *AxoneClient) refreshLocked(ctx context.Context) error {
	payload := map[string]string{
		"refreshToken": c.token.RefreshToken,
	}

	var resp axoneResponse[axoneLoginData]
	if err := c.doJSONRequest(ctx, http.MethodPost, "/web/user/refresh", payload, "", &resp); err != nil {
		return err
	}
	if resp.Code != 0 {
		return axoneResponseError(resp.Message)
	}
	if strings.TrimSpace(resp.Data.AccessToken) == "" || strings.TrimSpace(resp.Data.RefreshToken) == "" {
		return fmt.Errorf("axone refresh returned empty token")
	}
	c.token = axoneTokenState{
		AccessToken:           resp.Data.AccessToken,
		RefreshToken:          resp.Data.RefreshToken,
		AccessTokenExpiresAt:  resp.Data.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: resp.Data.RefreshTokenExpiresAt,
	}
	return nil
}

func (c *AxoneClient) doJSONRequest(ctx context.Context, method string, path string, payload any, accessToken string, target any) error {
	return c.doJSONRequestWithHeaders(ctx, method, path, payload, accessToken, nil, target)
}

func (c *AxoneClient) doJSONRequestWithHeaders(ctx context.Context, method string, path string, payload any, accessToken string, headers map[string]string, target any) error {
	fullURL := c.baseURL + path

	var body io.Reader
	if payload != nil {
		bodyBytes, err := common.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	for key, value := range headers {
		if strings.TrimSpace(value) != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return axoneHTTPError(resp.StatusCode, bodyBytes)
	}

	return common.DecodeJson(resp.Body, target)
}

func (c *AxoneClient) doSignedJSONRequest(ctx context.Context, method string, path string, payload any, accessToken string, target any) error {
	fullURL := c.baseURL + path

	bodyBytes, err := common.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", buildAxoneSignature(timestamp, bodyBytes, setting.AxoneWebhookSecret))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return axoneHTTPError(resp.StatusCode, bodyBytes)
	}

	return common.DecodeJson(resp.Body, target)
}

func buildAxoneSignature(timestamp string, body []byte, secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "new-api"
	}
	mac := hmac.New(sha512.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func axoneResponseError(message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "AXOne request failed"
	}
	return fmt.Errorf("%s", message)
}

func axoneHTTPError(statusCode int, bodyBytes []byte) error {
	var resp struct {
		Message string `json:"message"`
	}
	if err := common.Unmarshal(bodyBytes, &resp); err == nil {
		if message := strings.TrimSpace(resp.Message); message != "" {
			return fmt.Errorf("%s", message)
		}
	}

	body := strings.TrimSpace(string(bodyBytes))
	if body != "" {
		return fmt.Errorf("%s", body)
	}
	return fmt.Errorf("AXOne request failed with status %d", statusCode)
}

func md5Hex(value string) string {
	sum := md5.Sum([]byte(value))
	return fmt.Sprintf("%x", sum)
}
