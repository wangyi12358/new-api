package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/thanhpk/randstr"
)

type axoneAddressRequest struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	ChainID  string `json:"chain_id"`
}

const (
	axoneOrderExpireSeconds = int64(15 * 60)
	axoneUniqueScale        = int32(4)
	axoneUniqueSlots        = 9000
)

func ListAxoneChains(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	if !isAxoneTopUpEnabled() {
		common.ApiErrorMsg(c, "AXOne stablecoin topup is not enabled")
		return
	}

	client := service.GetAxoneClient()
	chains, err := client.ListChains(c.Request.Context())
	if err != nil {
		common.ApiErrorMsg(c, "Failed to fetch AXOne chains")
		return
	}

	common.ApiSuccess(c, chains)
}

func RequestAxoneAddress(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	if !isAxoneTopUpEnabled() {
		common.ApiErrorMsg(c, "AXOne stablecoin topup is not enabled")
		return
	}

	var req axoneAddressRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	if req.Amount < getMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	chainID := strings.TrimSpace(req.ChainID)
	if currency == "" || chainID == "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	if !containsAxoneCurrency(currency) {
		common.ApiErrorMsg(c, "Unsupported AXOne currency")
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}

	baseMoney := decimal.NewFromFloat(getPayMoney(req.Amount, group)).Round(axoneUniqueScale)
	feeMoney := decimal.Zero
	feePercent := decimal.NewFromFloat(setting.GetAxoneFeePercent())
	if feePercent.IsPositive() {
		feeMoney = baseMoney.Mul(feePercent).Div(decimal.NewFromInt(100)).RoundCeil(axoneUniqueScale)
	}
	paymentMoney := baseMoney.Add(feeMoney).Round(axoneUniqueScale)
	if !paymentMoney.IsPositive() {
		common.ApiErrorMsg(c, "Invalid AXOne payment amount")
		return
	}

	client := service.GetAxoneClient()
	address, err := client.GetWalletAddress(c.Request.Context(), currency, chainID)
	if err != nil {
		common.ApiErrorMsg(c, "Failed to generate AXOne wallet address")
		return
	}

	now := time.Now().Unix()
	if err := model.ExpirePendingAxoneTopUps(now); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("AXOne expire stale orders failed user_id=%d error=%q", id, err.Error()))
	}
	if err := model.ExpireUserPendingAxoneTopUps(id, now); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("AXOne expire user orders failed user_id=%d error=%q", id, err.Error()))
	}

	tradeNo := fmt.Sprintf("AXONE-%d-%d-%s", id, time.Now().UnixMilli(), randstr.String(6))
	uniqueMoney, err := allocateAxoneUniqueMoney(currency, chainID, paymentMoney, now)
	if err != nil {
		common.ApiErrorMsg(c, "Failed to reserve AXOne payment amount")
		return
	}

	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount := decimal.NewFromInt(req.Amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amount = dAmount.Div(dQuotaPerUnit).IntPart()
		if amount < 1 {
			amount = 1
		}
	}

	expireTime := now + axoneOrderExpireSeconds
	topUp := &model.TopUp{
		UserId:          id,
		Amount:          amount,
		Money:           uniqueMoney.InexactFloat64(),
		Fee:             feeMoney.InexactFloat64(),
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodAxone,
		PaymentProvider: model.PaymentProviderAxone,
		AxoneCurrency:   currency,
		AxoneChainID:    chainID,
		AxoneAddress:    address,
		ExpireTime:      expireTime,
		CreateTime:      now,
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("AXOne create order failed user_id=%d trade_no=%s amount=%d error=%q", id, tradeNo, req.Amount, err.Error()))
		common.ApiErrorMsg(c, "Failed to create AXOne topup order")
		return
	}

	common.ApiSuccess(c, gin.H{
		"trade_no":           tradeNo,
		"amount":             req.Amount,
		"base_payment_money": baseMoney.StringFixed(axoneUniqueScale),
		"fee":                feeMoney.StringFixed(axoneUniqueScale),
		"payment_money":      uniqueMoney.StringFixed(axoneUniqueScale),
		"currency":           currency,
		"chain_id":           chainID,
		"address":            address,
		"expires_at":         expireTime,
		"status":             common.TopUpStatusPending,
	})
}

func containsAxoneCurrency(currency string) bool {
	for _, configured := range setting.GetAxoneCurrencies() {
		if configured == currency {
			return true
		}
	}
	return false
}

func allocateAxoneUniqueMoney(currency string, chainID string, baseMoney decimal.Decimal, now int64) (decimal.Decimal, error) {
	for i := 0; i < axoneUniqueSlots; i++ {
		candidate := baseMoney.Add(decimal.NewFromInt(int64(i + 1)).Div(decimal.NewFromInt(10000))).Round(axoneUniqueScale)
		inUse, err := model.IsActiveAxoneTopUpMoneyInUse(currency, chainID, candidate.InexactFloat64(), now)
		if err != nil {
			return decimal.Zero, err
		}
		if !inUse {
			return candidate, nil
		}
	}
	return decimal.Zero, fmt.Errorf("no available axone unique amount")
}
