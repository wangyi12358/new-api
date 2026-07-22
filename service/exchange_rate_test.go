package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type exchangeRateRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn exchangeRateRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func exchangeRateJSONResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestExchangeRateServiceCrossValidatesAndCaches(t *testing.T) {
	var primaryCalls atomic.Int32
	var referenceCalls atomic.Int32
	client := &http.Client{Transport: exchangeRateRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case "https://primary.test/latest":
			primaryCalls.Add(1)
			return exchangeRateJSONResponse(`{"result":"success","base":"USD","source":"live","data_updated_at":"2026-07-22T08:33:59Z","rates":{"CNY":6.7734}}`), nil
		case "https://reference.test/latest":
			referenceCalls.Add(1)
			return exchangeRateJSONResponse(`{"date":"2026-07-22","base":"USD","quote":"CNY","rate":6.7649}`), nil
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}, nil
		}
	})}

	service := newExchangeRateService(client, "https://primary.test/latest", "https://reference.test/latest", 5*time.Minute)
	service.now = func() time.Time { return time.Date(2026, 7, 22, 8, 35, 0, 0, time.UTC) }

	first, err := service.get(context.Background())
	require.NoError(t, err)
	require.Equal(t, 6.7734, first.Rate)
	require.Equal(t, 6.7649, first.ReferenceRate)
	require.Equal(t, "exchangerate.dev/live", first.Source)
	require.Equal(t, time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC), first.ReferenceDate)

	second, err := service.get(context.Background())
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, int32(1), primaryCalls.Load())
	require.Equal(t, int32(1), referenceCalls.Load())
}

func TestExchangeRateServiceRejectsLargeSourceDeviation(t *testing.T) {
	client := &http.Client{Transport: exchangeRateRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "primary.test" {
			return exchangeRateJSONResponse(`{"result":"success","base":"USD","source":"live","data_updated_at":"2026-07-22T08:33:59Z","rates":{"CNY":7.1000}}`), nil
		}
		return exchangeRateJSONResponse(`{"date":"2026-07-22","base":"USD","quote":"CNY","rate":6.7600}`), nil
	})}

	service := newExchangeRateService(client, "https://primary.test/latest", "https://reference.test/latest", 5*time.Minute)

	_, err := service.get(context.Background())
	require.ErrorContains(t, err, "汇率来源偏差过大")
}
