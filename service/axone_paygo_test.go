package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type axoneRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn axoneRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func setupAxonePaygoTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	previousDB := model.DB
	previousQuotaPerUnit := common.QuotaPerUnit
	previousMode := setting.AxonePaygoChargeMode
	previousThreshold := setting.AxonePaygoChargeThreshold

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AxonePaygoSession{}, &model.AxonePaygoUsage{}, &model.AxonePaygoCharge{}))
	model.DB = db
	common.QuotaPerUnit = 500_000
	setting.AxonePaygoChargeMode = setting.AxonePaygoChargeModeThreshold
	setting.AxonePaygoChargeThreshold = "2.00000000"

	t.Cleanup(func() {
		model.DB = previousDB
		common.QuotaPerUnit = previousQuotaPerUnit
		setting.AxonePaygoChargeMode = previousMode
		setting.AxonePaygoChargeThreshold = previousThreshold
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestAxoneQ8Parsing(t *testing.T) {
	q8, err := ParseAxoneQ8("1.23000000")
	require.NoError(t, err)
	require.Equal(t, int64(123_000_000), q8)
	require.Equal(t, "1.23000000", FormatAxoneQ8(q8))

	_, err = ParseAxoneQ8("0")
	require.Error(t, err)
	_, err = ParseAxoneQ8("0.000000001")
	require.Error(t, err)
}

func TestAxonePaygoReserveFinalizeAndRefund(t *testing.T) {
	db := setupAxonePaygoTestDB(t)
	now := time.Now().Unix()
	require.NoError(t, db.Create(&model.AxonePaygoSession{
		SessionID:  "aps_test",
		UserID:     7,
		WalletID:   "wallet_test",
		Currency:   "USDC",
		Status:     "active",
		ReservedQ8: 200_000_000,
		ExpiresAt:  now + 3600,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error)

	reservedQ8, err := ReserveAxonePaygoRequest(7, "aps_test", "req_1", 500_000)
	require.NoError(t, err)
	require.Equal(t, int64(100_000_000), reservedQ8)

	_, err = ReserveAxonePaygoRequest(7, "aps_test", "req_2", 1_000_000)
	require.ErrorIs(t, err, ErrAxonePaygoInsufficientReserved)

	require.NoError(t, FinalizeAxonePaygoRequest(context.Background(), "req_1", 250_000))
	var session model.AxonePaygoSession
	require.NoError(t, db.Where("session_id = ?", "aps_test").First(&session).Error)
	require.Zero(t, session.InFlightQ8)
	require.Equal(t, int64(50_000_000), session.AccruedQ8)
	require.Equal(t, int64(50_000_000), session.PendingQ8)

	_, err = ReserveAxonePaygoRequest(7, "aps_test", "req_3", 100_000)
	require.NoError(t, err)
	require.NoError(t, RefundAxonePaygoRequest("req_3"))
	require.NoError(t, db.Where("session_id = ?", "aps_test").First(&session).Error)
	require.Zero(t, session.InFlightQ8)
}

func TestEnqueueAxonePaygoChargeRejectsConcurrentProcessing(t *testing.T) {
	db := setupAxonePaygoTestDB(t)
	now := time.Now().Unix()
	require.NoError(t, db.Create(&model.AxonePaygoSession{SessionID: "aps_busy", UserID: 7, Status: "active", PendingQ8: 1, CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&model.AxonePaygoCharge{
		SessionID: "aps_busy", EventID: "event_busy", EventSeq: 1, AmountQ8: 1,
		Status: axonePaygoChargeProcessing, IdempotencyKey: "event_busy", CreatedAt: now, UpdatedAt: now,
	}).Error)

	charge, err := enqueueAxonePaygoCharge("aps_busy", true)
	require.Nil(t, charge)
	require.ErrorIs(t, err, ErrAxonePaygoChargeInProgress)
}

func TestAxonePaygoClientContract(t *testing.T) {
	requests := make([]string, 0, 4)
	httpClient := &http.Client{Transport: axoneRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
		require.NotEmpty(t, r.Header.Get("Idempotency-Key"))
		requests = append(requests, r.Method+" "+r.URL.Path)

		var response any
		switch r.URL.Path {
		case "/web/agen-pay/sessions":
			response = axoneResponse[AxonePaygoSessionData]{Data: AxonePaygoSessionData{SessionID: "aps_1", Status: "active", Currency: "USDC", ReservedAmount: "1.00000000", ConsumedAmount: "0.00000000"}}
		case "/web/agen-pay/sessions/aps_1/usage":
			response = axoneResponse[AxonePaygoUsageData]{Data: AxonePaygoUsageData{SessionID: "aps_1", AcceptedThroughSeq: 1, ChargeAmount: "0.10000000", ConsumedAmount: "0.10000000"}}
		case "/web/agen-pay/sessions/aps_1/close":
			response = axoneResponse[AxonePaygoSessionData]{Data: AxonePaygoSessionData{SessionID: "aps_1", Status: "closed", Currency: "USDC", ReservedAmount: "1.00000000", ConsumedAmount: "0.10000000"}}
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		data, err := common.Marshal(response)
		require.NoError(t, err)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(data)),
			Request:    r,
		}, nil
	})}

	client := &AxoneClient{
		baseURL: "https://axone.test", account: "account", password: "password", httpClient: httpClient,
		token: axoneTokenState{AccessToken: "access-token", AccessTokenExpiresAt: time.Now().Add(time.Hour).UnixMilli()},
	}
	created, err := client.CreatePaygoSession(context.Background(), "wallet_1", "1.00000000", "create-key")
	require.NoError(t, err)
	require.Equal(t, "aps_1", created.SessionID)
	usage, err := client.SubmitPaygoUsage(context.Background(), "aps_1", "event_1", 1, "0.10000000", "event-key")
	require.NoError(t, err)
	require.Equal(t, int64(1), usage.AcceptedThroughSeq)
	closed, err := client.ClosePaygoSession(context.Background(), "aps_1", "close-key")
	require.NoError(t, err)
	require.Equal(t, "closed", closed.Status)
	require.Equal(t, []string{
		"POST /web/agen-pay/sessions",
		"POST /web/agen-pay/sessions/aps_1/usage",
		"POST /web/agen-pay/sessions/aps_1/close",
	}, requests)
}
