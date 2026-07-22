package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	usdCNYPrimaryURL       = "https://api.exchangerate.dev/v1/latest/USD?symbols=CNY"
	usdCNYReferenceURL     = "https://api.frankfurter.dev/v2/rate/USD/CNY"
	usdCNYCacheTTL         = 5 * time.Minute
	usdCNYRequestTimeout   = 5 * time.Second
	usdCNYMaxRateDeviation = 0.01
)

type ExchangeRateQuote struct {
	Base            string    `json:"base"`
	Quote           string    `json:"quote"`
	Rate            float64   `json:"rate"`
	Source          string    `json:"source"`
	SourceTimestamp time.Time `json:"source_timestamp"`
	ReferenceRate   float64   `json:"reference_rate"`
	ReferenceSource string    `json:"reference_source"`
	ReferenceDate   time.Time `json:"reference_date"`
	FetchedAt       time.Time `json:"fetched_at"`
}

type exchangeRateService struct {
	client       *http.Client
	primaryURL   string
	referenceURL string
	cacheTTL     time.Duration
	now          func() time.Time

	mu        sync.Mutex
	cached    ExchangeRateQuote
	cachedTil time.Time
}

var usdCNYRates = newExchangeRateService(
	&http.Client{Timeout: usdCNYRequestTimeout},
	usdCNYPrimaryURL,
	usdCNYReferenceURL,
	usdCNYCacheTTL,
)

func newExchangeRateService(client *http.Client, primaryURL, referenceURL string, cacheTTL time.Duration) *exchangeRateService {
	return &exchangeRateService{
		client:       client,
		primaryURL:   primaryURL,
		referenceURL: referenceURL,
		cacheTTL:     cacheTTL,
		now:          time.Now,
	}
}

func GetUSDToCNYExchangeRate(ctx context.Context) (ExchangeRateQuote, error) {
	return usdCNYRates.get(ctx)
}

func (s *exchangeRateService) get(ctx context.Context) (ExchangeRateQuote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if s.cached.Rate > 0 && now.Before(s.cachedTil) {
		return s.cached, nil
	}

	primary, err := s.fetchPrimary(ctx)
	if err != nil {
		return ExchangeRateQuote{}, fmt.Errorf("获取实时 USD/CNY 汇率失败: %w", err)
	}
	reference, err := s.fetchReference(ctx)
	if err != nil {
		return ExchangeRateQuote{}, fmt.Errorf("校验 USD/CNY 汇率失败: %w", err)
	}

	deviation := math.Abs(primary.Rate-reference.Rate) / reference.Rate
	if deviation > usdCNYMaxRateDeviation {
		return ExchangeRateQuote{}, fmt.Errorf(
			"USD/CNY 汇率来源偏差过大: realtime=%.6f reference=%.6f deviation=%.2f%%",
			primary.Rate,
			reference.Rate,
			deviation*100,
		)
	}

	primary.ReferenceRate = reference.Rate
	primary.ReferenceSource = reference.Source
	primary.ReferenceDate = reference.SourceTimestamp
	primary.FetchedAt = now
	s.cached = primary
	s.cachedTil = now.Add(s.cacheTTL)
	return primary, nil
}

func (s *exchangeRateService) fetchPrimary(ctx context.Context) (ExchangeRateQuote, error) {
	var response struct {
		Result        string             `json:"result"`
		Base          string             `json:"base"`
		Source        string             `json:"source"`
		Timestamp     string             `json:"timestamp"`
		DataUpdatedAt string             `json:"data_updated_at"`
		Rates         map[string]float64 `json:"rates"`
	}
	if err := s.getJSON(ctx, s.primaryURL, &response); err != nil {
		return ExchangeRateQuote{}, err
	}
	if response.Result != "success" || response.Base != "USD" || response.Rates["CNY"] <= 0 {
		return ExchangeRateQuote{}, errors.New("实时汇率响应无效")
	}

	timestamp := response.DataUpdatedAt
	if timestamp == "" {
		timestamp = response.Timestamp
	}
	parsedTimestamp, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return ExchangeRateQuote{}, fmt.Errorf("实时汇率时间无效: %w", err)
	}

	return ExchangeRateQuote{
		Base:            "USD",
		Quote:           "CNY",
		Rate:            response.Rates["CNY"],
		Source:          "exchangerate.dev/" + response.Source,
		SourceTimestamp: parsedTimestamp.UTC(),
	}, nil
}

func (s *exchangeRateService) fetchReference(ctx context.Context) (ExchangeRateQuote, error) {
	var response struct {
		Date  string  `json:"date"`
		Base  string  `json:"base"`
		Quote string  `json:"quote"`
		Rate  float64 `json:"rate"`
	}
	if err := s.getJSON(ctx, s.referenceURL, &response); err != nil {
		return ExchangeRateQuote{}, err
	}
	if response.Base != "USD" || response.Quote != "CNY" || response.Rate <= 0 {
		return ExchangeRateQuote{}, errors.New("参考汇率响应无效")
	}
	referenceDate, err := time.Parse("2006-01-02", response.Date)
	if err != nil {
		return ExchangeRateQuote{}, fmt.Errorf("参考汇率日期无效: %w", err)
	}

	return ExchangeRateQuote{
		Base:            "USD",
		Quote:           "CNY",
		Rate:            response.Rate,
		Source:          "frankfurter.dev",
		SourceTimestamp: referenceDate.UTC(),
	}, nil
}

func (s *exchangeRateService) getJSON(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "new-api/exchange-rate")

	response, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("汇率服务返回 HTTP %d", response.StatusCode)
	}
	if err := common.DecodeJson(response.Body, target); err != nil {
		return fmt.Errorf("解析汇率响应失败: %w", err)
	}
	return nil
}
