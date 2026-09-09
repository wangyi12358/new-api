package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	AxonePaygoHeader     = "X-Kovar-Payment-Session"
	AxonePaygoContextKey = "axone_paygo_session_id"

	axonePaygoUsageReserved = "reserved"
	axonePaygoUsageAccrued  = "accrued"
	axonePaygoUsageQueued   = "queued"
	axonePaygoUsageSettled  = "settled"
	axonePaygoUsageRefunded = "refunded"

	axonePaygoChargePending    = "pending"
	axonePaygoChargeProcessing = "processing"
	axonePaygoChargeSucceeded  = "succeeded"
)

var (
	ErrAxonePaygoDisabled             = errors.New("AXOne PayGo is not enabled")
	ErrAxonePaygoSessionNotFound      = errors.New("AXOne PayGo session not found")
	ErrAxonePaygoSessionUnavailable   = errors.New("AXOne PayGo session is not active")
	ErrAxonePaygoInsufficientReserved = errors.New("AXOne PayGo reserved balance is insufficient")
	ErrAxonePaygoIdempotencyConflict  = errors.New("AXOne PayGo idempotency conflict")
	ErrAxonePaygoChargeInProgress     = errors.New("AXOne PayGo charge is already being processed")
)

func IsAxonePaygoReady() bool {
	return setting.AxonePaygoEnabled &&
		strings.TrimSpace(setting.AxoneBaseURL) != "" &&
		strings.TrimSpace(setting.AxoneAccount) != "" &&
		strings.TrimSpace(setting.AxonePassword) != ""
}

func ParseAxoneQ8(value string) (int64, error) {
	amount, err := decimal.NewFromString(strings.TrimSpace(value))
	if err != nil || !amount.IsPositive() {
		return 0, fmt.Errorf("invalid AXOne amount")
	}
	if amount.Exponent() < -8 {
		return 0, fmt.Errorf("AXOne amount supports at most 8 decimal places")
	}
	q8 := amount.Shift(8)
	if !q8.Equal(q8.Truncate(0)) {
		return 0, fmt.Errorf("invalid AXOne q8 amount")
	}
	valueQ8 := q8.IntPart()
	if valueQ8 <= 0 {
		return 0, fmt.Errorf("AXOne amount must be positive")
	}
	return valueQ8, nil
}

func parseAxoneNonNegativeQ8(value string) (int64, error) {
	amount, err := decimal.NewFromString(strings.TrimSpace(value))
	if err != nil || amount.IsNegative() || amount.Exponent() < -8 {
		return 0, fmt.Errorf("invalid AXOne amount")
	}
	q8 := amount.Shift(8)
	if !q8.Equal(q8.Truncate(0)) {
		return 0, fmt.Errorf("invalid AXOne q8 amount")
	}
	return q8.IntPart(), nil
}

func FormatAxoneQ8(value int64) string {
	return decimal.NewFromInt(value).Shift(-8).StringFixed(8)
}

func quotaToAxoneQ8(quota int) (int64, error) {
	if quota <= 0 {
		return 0, nil
	}
	quotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
	if !quotaPerUnit.IsPositive() {
		return 0, fmt.Errorf("invalid QuotaPerUnit")
	}
	return decimal.NewFromInt(int64(quota)).Shift(8).Div(quotaPerUnit).Ceil().IntPart(), nil
}

func paygoCreateHash(walletID string, maxAmountQ8 int64) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(walletID) + "\n" + FormatAxoneQ8(maxAmountQ8)))
	return fmt.Sprintf("%x", sum[:])
}

func providerPayload(value any) string {
	data, err := common.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func parseAxoneTime(value string) int64 {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed.Unix()
}

func applyProviderSession(target *model.AxonePaygoSession, data *AxonePaygoSessionData) error {
	if target == nil || data == nil {
		return fmt.Errorf("empty AXOne PayGo session response")
	}
	providerSessionID := strings.TrimSpace(data.SessionID)
	if providerSessionID == "" {
		return fmt.Errorf("AXOne returned empty PayGo session id")
	}
	if target.SessionID != "" && target.SessionID != providerSessionID {
		return fmt.Errorf("AXOne returned a mismatched PayGo session id")
	}
	reservedQ8, err := parseAxoneNonNegativeQ8(data.ReservedAmount)
	if err != nil {
		return fmt.Errorf("invalid reserved amount: %w", err)
	}
	consumedQ8 := int64(0)
	if strings.TrimSpace(data.ConsumedAmount) != "" {
		consumedQ8, err = parseAxoneNonNegativeQ8(data.ConsumedAmount)
		if err != nil {
			return fmt.Errorf("invalid consumed amount: %w", err)
		}
	}
	target.SessionID = providerSessionID
	target.Currency = strings.ToUpper(strings.TrimSpace(data.Currency))
	target.Status = strings.ToLower(strings.TrimSpace(data.Status))
	target.ReservedQ8 = reservedQ8
	target.ProviderConsumedQ8 = consumedQ8
	// Provider state can lag a locally queued idempotent usage event. Never move
	// the locally allocated sequence backwards, or a later event could reuse it.
	if data.LastEventSeq > target.LastEventSeq {
		target.LastEventSeq = data.LastEventSeq
	}
	target.ExpiresAt = parseAxoneTime(data.ExpiresAt)
	if data.ClosedAt != nil {
		target.ClosedAt = parseAxoneTime(*data.ClosedAt)
	}
	target.ProviderPayload = providerPayload(data)
	target.UpdatedAt = time.Now().Unix()
	return nil
}

func CreateAxonePaygoSession(ctx context.Context, userID int, walletID string, maxAmount string, idempotencyKey string) (*model.AxonePaygoSession, error) {
	if !IsAxonePaygoReady() {
		return nil, ErrAxonePaygoDisabled
	}
	walletID = strings.TrimSpace(walletID)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if userID <= 0 || walletID == "" || idempotencyKey == "" {
		return nil, fmt.Errorf("user, wallet_id and Idempotency-Key are required")
	}
	maxAmountQ8, err := ParseAxoneQ8(maxAmount)
	if err != nil {
		return nil, err
	}
	requestHash := paygoCreateHash(walletID, maxAmountQ8)

	existing := &model.AxonePaygoSession{}
	err = model.DB.Where("user_id = ? AND create_idempotency_key = ?", userID, idempotencyKey).First(existing).Error
	if err == nil {
		if existing.CreateRequestHash != requestHash {
			return nil, ErrAxonePaygoIdempotencyConflict
		}
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	providerSession, err := GetAxoneClient().CreatePaygoSession(ctx, walletID, FormatAxoneQ8(maxAmountQ8), idempotencyKey)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	localSession := &model.AxonePaygoSession{
		UserID:               userID,
		WalletID:             walletID,
		CreateIdempotencyKey: idempotencyKey,
		CreateRequestHash:    requestHash,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := applyProviderSession(localSession, providerSession); err != nil {
		return nil, err
	}
	if err := model.DB.Create(localSession).Error; err != nil {
		// A provider retry may have returned the original session while another
		// application request persisted it first.
		if lookupErr := model.DB.Where("user_id = ? AND create_idempotency_key = ?", userID, idempotencyKey).First(existing).Error; lookupErr == nil {
			if existing.CreateRequestHash != requestHash {
				return nil, ErrAxonePaygoIdempotencyConflict
			}
			return existing, nil
		}
		return nil, err
	}
	return localSession, nil
}

func GetAxonePaygoSession(ctx context.Context, userID int, sessionID string, refresh bool) (*model.AxonePaygoSession, error) {
	session := &model.AxonePaygoSession{}
	if err := model.DB.Where("session_id = ? AND user_id = ?", strings.TrimSpace(sessionID), userID).First(session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAxonePaygoSessionNotFound
		}
		return nil, err
	}
	if !refresh || !IsAxonePaygoReady() {
		return session, nil
	}
	providerSession, err := GetAxoneClient().GetPaygoSession(ctx, session.SessionID)
	if err != nil {
		return nil, err
	}
	if err := applyProviderSession(session, providerSession); err != nil {
		return nil, err
	}
	if err := model.DB.Save(session).Error; err != nil {
		return nil, err
	}
	return session, nil
}

func ListAxonePaygoSessions(userID int, limit int) ([]*model.AxonePaygoSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	sessions := make([]*model.AxonePaygoSession, 0)
	err := model.DB.Where("user_id = ?", userID).Order("id desc").Limit(limit).Find(&sessions).Error
	return sessions, err
}

func ReserveAxonePaygoRequest(userID int, sessionID string, requestID string, quota int) (int64, error) {
	estimatedQ8, err := quotaToAxoneQ8(quota)
	if err != nil {
		return 0, err
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return 0, fmt.Errorf("request id is required for AXOne PayGo")
	}
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		existing := &model.AxonePaygoUsage{}
		lookupErr := tx.Where("request_id = ?", requestID).First(existing).Error
		if lookupErr == nil {
			if existing.UserID != userID || existing.SessionID != sessionID || existing.EstimatedQ8 != estimatedQ8 {
				return ErrAxonePaygoIdempotencyConflict
			}
			return nil
		}
		if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			return lookupErr
		}

		session := &model.AxonePaygoSession{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("session_id = ? AND user_id = ?", sessionID, userID).First(session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAxonePaygoSessionNotFound
			}
			return err
		}
		if session.Status != "active" || (session.ExpiresAt > 0 && session.ExpiresAt <= time.Now().Unix()) {
			return ErrAxonePaygoSessionUnavailable
		}
		remainingQ8 := session.ReservedQ8 - session.AccruedQ8 - session.InFlightQ8
		if estimatedQ8 > remainingQ8 {
			return ErrAxonePaygoInsufficientReserved
		}

		now := time.Now().Unix()
		usage := &model.AxonePaygoUsage{
			SessionID:   sessionID,
			UserID:      userID,
			RequestID:   requestID,
			EstimatedQ8: estimatedQ8,
			Status:      axonePaygoUsageReserved,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := tx.Create(usage).Error; err != nil {
			return err
		}
		return tx.Model(session).Updates(map[string]any{
			"in_flight_q8": gorm.Expr("in_flight_q8 + ?", estimatedQ8),
			"updated_at":   now,
		}).Error
	})
	return estimatedQ8, err
}

func ResizeAxonePaygoRequest(requestID string, targetQuota int) error {
	targetQ8, err := quotaToAxoneQ8(targetQuota)
	if err != nil {
		return err
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		usage := &model.AxonePaygoUsage{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("request_id = ?", requestID).First(usage).Error; err != nil {
			return err
		}
		if usage.Status != axonePaygoUsageReserved || targetQ8 == usage.EstimatedQ8 {
			return nil
		}
		deltaQ8 := targetQ8 - usage.EstimatedQ8
		session := &model.AxonePaygoSession{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("session_id = ? AND user_id = ?", usage.SessionID, usage.UserID).First(session).Error; err != nil {
			return err
		}
		remainingQ8 := session.ReservedQ8 - session.AccruedQ8 - session.InFlightQ8
		if deltaQ8 > 0 && deltaQ8 > remainingQ8 {
			return ErrAxonePaygoInsufficientReserved
		}
		now := time.Now().Unix()
		if err := tx.Model(usage).Updates(map[string]any{"estimated_q8": targetQ8, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(session).Updates(map[string]any{
			"in_flight_q8": gorm.Expr("in_flight_q8 + ?", deltaQ8),
			"updated_at":   now,
		}).Error
	})
}

func FinalizeAxonePaygoRequest(ctx context.Context, requestID string, actualQuota int) error {
	actualQ8, err := quotaToAxoneQ8(actualQuota)
	if err != nil {
		return err
	}
	var sessionID string
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		usage := &model.AxonePaygoUsage{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("request_id = ?", requestID).First(usage).Error; err != nil {
			return err
		}
		sessionID = usage.SessionID
		if usage.Status != axonePaygoUsageReserved {
			return nil
		}
		session := &model.AxonePaygoSession{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("session_id = ? AND user_id = ?", usage.SessionID, usage.UserID).First(session).Error; err != nil {
			return err
		}
		availableAfterRelease := session.ReservedQ8 - session.AccruedQ8 - session.InFlightQ8 + usage.EstimatedQ8
		if actualQ8 > availableAfterRelease {
			return ErrAxonePaygoInsufficientReserved
		}
		now := time.Now().Unix()
		usageStatus := axonePaygoUsageAccrued
		if actualQ8 == 0 {
			usageStatus = axonePaygoUsageSettled
		}
		if err := tx.Model(usage).Updates(map[string]any{
			"actual_q8":  actualQ8,
			"status":     usageStatus,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(session).Updates(map[string]any{
			"in_flight_q8": gorm.Expr("in_flight_q8 - ?", usage.EstimatedQ8),
			"accrued_q8":   gorm.Expr("accrued_q8 + ?", actualQ8),
			"pending_q8":   gorm.Expr("pending_q8 + ?", actualQ8),
			"updated_at":   now,
		}).Error
	})
	if err != nil || actualQ8 == 0 {
		return err
	}
	if flushErr := FlushAxonePaygoSession(ctx, sessionID, false); flushErr != nil {
		// The usage and its provider idempotency key are durable. Do not fail
		// settlement after the AI response has already been served.
		common.SysLog("AXOne PayGo charge queued for retry: " + flushErr.Error())
	}
	return nil
}

func RefundAxonePaygoRequest(requestID string) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		usage := &model.AxonePaygoUsage{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("request_id = ?", requestID).First(usage).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if usage.Status != axonePaygoUsageReserved {
			return nil
		}
		now := time.Now().Unix()
		if err := tx.Model(usage).Updates(map[string]any{"status": axonePaygoUsageRefunded, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&model.AxonePaygoSession{}).Where("session_id = ?", usage.SessionID).Updates(map[string]any{
			"in_flight_q8": gorm.Expr("in_flight_q8 - ?", usage.EstimatedQ8),
			"updated_at":   now,
		}).Error
	})
}

func axonePaygoThresholdQ8() (int64, error) {
	return ParseAxoneQ8(setting.AxonePaygoChargeThreshold)
}

func enqueueAxonePaygoCharge(sessionID string, force bool) (*model.AxonePaygoCharge, error) {
	var charge *model.AxonePaygoCharge
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		existing := &model.AxonePaygoCharge{}
		if err := tx.Where("session_id = ? AND status = ?", sessionID, axonePaygoChargePending).Order("event_seq asc").First(existing).Error; err == nil {
			charge = existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Where("session_id = ? AND status = ?", sessionID, axonePaygoChargeProcessing).First(existing).Error; err == nil {
			return ErrAxonePaygoChargeInProgress
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		session := &model.AxonePaygoSession{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("session_id = ?", sessionID).First(session).Error; err != nil {
			return err
		}
		if session.PendingQ8 <= 0 {
			return nil
		}
		if !force && setting.GetAxonePaygoChargeMode() == setting.AxonePaygoChargeModeThreshold {
			thresholdQ8, err := axonePaygoThresholdQ8()
			if err != nil {
				return err
			}
			if session.PendingQ8 < thresholdQ8 {
				return nil
			}
		}

		nextSeq := session.LastEventSeq + 1
		eventID := fmt.Sprintf("kovar-%s-%d", session.SessionID, nextSeq)
		now := time.Now().Unix()
		charge = &model.AxonePaygoCharge{
			SessionID:      session.SessionID,
			EventID:        eventID,
			EventSeq:       nextSeq,
			AmountQ8:       session.PendingQ8,
			Status:         axonePaygoChargePending,
			IdempotencyKey: eventID,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := tx.Create(charge).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.AxonePaygoUsage{}).
			Where("session_id = ? AND status = ?", session.SessionID, axonePaygoUsageAccrued).
			Updates(map[string]any{"status": axonePaygoUsageQueued, "charge_event_id": eventID, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(session).Updates(map[string]any{
			"pending_q8":     0,
			"last_event_seq": nextSeq,
			"updated_at":     now,
		}).Error
	})
	return charge, err
}

func processAxonePaygoCharge(ctx context.Context, charge *model.AxonePaygoCharge) error {
	if charge == nil || charge.Status == axonePaygoChargeSucceeded {
		return nil
	}
	claimed := model.DB.Model(&model.AxonePaygoCharge{}).
		Where("id = ? AND status = ?", charge.Id, axonePaygoChargePending).
		Updates(map[string]any{"status": axonePaygoChargeProcessing, "updated_at": time.Now().Unix()})
	if claimed.Error != nil {
		return claimed.Error
	}
	if claimed.RowsAffected == 0 {
		return nil
	}

	result, err := GetAxoneClient().SubmitPaygoUsage(
		ctx,
		charge.SessionID,
		charge.EventID,
		charge.EventSeq,
		FormatAxoneQ8(charge.AmountQ8),
		charge.IdempotencyKey,
	)
	if err != nil {
		markAxonePaygoChargePending(charge.Id, err)
		return err
	}
	if result == nil || strings.TrimSpace(result.SessionID) != charge.SessionID || result.AcceptedThroughSeq < charge.EventSeq {
		err = fmt.Errorf("AXOne returned an invalid usage acknowledgement")
		markAxonePaygoChargePending(charge.Id, err)
		return err
	}
	providerConsumedQ8, err := parseAxoneNonNegativeQ8(result.ConsumedAmount)
	if err != nil {
		markAxonePaygoChargePending(charge.Id, err)
		return err
	}
	now := time.Now().Unix()
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AxonePaygoCharge{}).Where("id = ?", charge.Id).Updates(map[string]any{
			"status":           axonePaygoChargeSucceeded,
			"attempts":         gorm.Expr("attempts + 1"),
			"last_error":       "",
			"provider_payload": providerPayload(result),
			"updated_at":       now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.AxonePaygoUsage{}).Where("charge_event_id = ?", charge.EventID).Updates(map[string]any{
			"status":     axonePaygoUsageSettled,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&model.AxonePaygoSession{}).Where("session_id = ?", charge.SessionID).Updates(map[string]any{
			"provider_consumed_q8": providerConsumedQ8,
			"updated_at":           now,
		}).Error
	})
	if err != nil {
		// AXOne may already have accepted the event. Returning it to the durable
		// outbox is safe because the provider request is idempotent.
		markAxonePaygoChargePending(charge.Id, err)
	}
	return err
}

func markAxonePaygoChargePending(chargeID int64, cause error) {
	message := ""
	if cause != nil {
		message = cause.Error()
	}
	_ = model.DB.Model(&model.AxonePaygoCharge{}).Where("id = ?", chargeID).Updates(map[string]any{
		"status":     axonePaygoChargePending,
		"attempts":   gorm.Expr("attempts + 1"),
		"last_error": message,
		"updated_at": time.Now().Unix(),
	}).Error
}

func FlushAxonePaygoSession(ctx context.Context, sessionID string, force bool) error {
	for {
		charge, err := enqueueAxonePaygoCharge(sessionID, force)
		if err != nil || charge == nil {
			return err
		}
		if err := processAxonePaygoCharge(ctx, charge); err != nil {
			return err
		}
		// Per-request mode and close flush all locally accrued amounts. Threshold
		// mode stops naturally when the remaining tail is below the threshold.
		if !force && setting.GetAxonePaygoChargeMode() == setting.AxonePaygoChargeModeThreshold {
			return nil
		}
	}
}

func CloseAxonePaygoSession(ctx context.Context, userID int, sessionID string, idempotencyKey string) (*model.AxonePaygoSession, error) {
	localSession, err := GetAxonePaygoSession(ctx, userID, sessionID, false)
	if err != nil {
		return nil, err
	}
	if localSession.InFlightQ8 > 0 {
		return nil, fmt.Errorf("AXOne PayGo session still has in-flight AI requests")
	}
	if err := FlushAxonePaygoSession(ctx, localSession.SessionID, true); err != nil {
		return nil, fmt.Errorf("flush AXOne PayGo charges before close: %w", err)
	}
	providerSession, err := GetAxoneClient().ClosePaygoSession(ctx, localSession.SessionID, strings.TrimSpace(idempotencyKey))
	if err != nil {
		return nil, err
	}
	if err := applyProviderSession(localSession, providerSession); err != nil {
		return nil, err
	}
	if err := model.DB.Save(localSession).Error; err != nil {
		return nil, err
	}
	return localSession, nil
}

func StartAxonePaygoSettlementTask() {
	gopool.Go(func() {
		// Recover charges left in processing when a previous process stopped
		// after claiming the durable outbox row.
		_ = model.DB.Model(&model.AxonePaygoCharge{}).
			Where("status = ? AND updated_at < ?", axonePaygoChargeProcessing, time.Now().Add(-time.Minute).Unix()).
			Updates(map[string]any{"status": axonePaygoChargePending, "updated_at": time.Now().Unix()}).Error

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if !IsAxonePaygoReady() {
				continue
			}
			runAxonePaygoSettlementPass()
		}
	})
}

func runAxonePaygoSettlementPass() {
	_ = model.DB.Model(&model.AxonePaygoCharge{}).
		Where("status = ? AND updated_at < ?", axonePaygoChargeProcessing, time.Now().Add(-time.Minute).Unix()).
		Updates(map[string]any{"status": axonePaygoChargePending, "updated_at": time.Now().Unix()}).Error

	charges := make([]*model.AxonePaygoCharge, 0)
	if err := model.DB.Where("status = ?", axonePaygoChargePending).Order("id asc").Limit(50).Find(&charges).Error; err != nil {
		common.SysLog("failed to load AXOne PayGo charges: " + err.Error())
		return
	}
	for _, charge := range charges {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err := processAxonePaygoCharge(ctx, charge)
		cancel()
		if err != nil {
			common.SysLog("failed to settle AXOne PayGo charge: " + err.Error())
		}
	}

	sessions := make([]*model.AxonePaygoSession, 0)
	query := model.DB.Where("status = ? AND pending_q8 > 0", "active")
	if setting.GetAxonePaygoChargeMode() == setting.AxonePaygoChargeModeThreshold {
		thresholdQ8, err := axonePaygoThresholdQ8()
		if err != nil {
			common.SysLog("invalid AXOne PayGo charge threshold: " + err.Error())
			return
		}
		query = query.Where("pending_q8 >= ?", thresholdQ8)
	}
	if err := query.Order("id asc").Limit(50).Find(&sessions).Error; err != nil {
		common.SysLog("failed to load AXOne PayGo sessions: " + err.Error())
		return
	}
	for _, session := range sessions {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err := FlushAxonePaygoSession(ctx, session.SessionID, false)
		cancel()
		if err != nil {
			common.SysLog("failed to flush AXOne PayGo session: " + err.Error())
		}
	}
}

type AxoneFunding struct {
	userID           int
	sessionID        string
	requestID        string
	preConsumedQuota int
}

func (a *AxoneFunding) Source() string { return BillingSourceAxone }

func (a *AxoneFunding) PreConsume(amount int) error {
	_, err := ReserveAxonePaygoRequest(a.userID, a.sessionID, a.requestID, amount)
	if err == nil {
		a.preConsumedQuota = amount
	}
	return err
}

func (a *AxoneFunding) Settle(delta int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return FinalizeAxonePaygoRequest(ctx, a.requestID, a.preConsumedQuota+delta)
}

func (a *AxoneFunding) Refund() error {
	return RefundAxonePaygoRequest(a.requestID)
}

func (a *AxoneFunding) Reserve(targetQuota int) error {
	if err := ResizeAxonePaygoRequest(a.requestID, targetQuota); err != nil {
		return err
	}
	a.preConsumedQuota = targetQuota
	return nil
}

func (a *AxoneFunding) RollbackReserve(delta int) {
	targetQuota := a.preConsumedQuota - delta
	if targetQuota < 0 {
		targetQuota = 0
	}
	if err := ResizeAxonePaygoRequest(a.requestID, targetQuota); err != nil {
		common.SysLog("error rolling back AXOne PayGo reservation: " + err.Error())
		return
	}
	a.preConsumedQuota = targetQuota
}
