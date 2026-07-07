package controller

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/thanhpk/randstr"
)

type axoneAddressRequest struct {
	Amount               int64  `json:"amount"`
	Currency             string `json:"currency"`
	ChainID              string `json:"chain_id"`
	PaymentWalletAddress string `json:"payment_wallet_address"`
}

type axoneWebhookPayload struct {
	Event                string `json:"event"`
	OrderNo              string `json:"orderNo"`
	AxoneOrderNo         string `json:"axoneOrderNo"`
	Amount               string `json:"amount"`
	Currency             string `json:"currency"`
	PaymentWalletAddress string `json:"paymentWalletAddress"`
	PayAddress           string `json:"payAddress"`
	Status               string `json:"status"`
	WriteOffStatus       string `json:"writeOffStatus"`
	PaidAt               string `json:"paidAt"`
}

const (
	axoneMoneyScale = int32(2)
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
		logger.LogError(c.Request.Context(), fmt.Sprintf("AXOne list chains failed error=%q", err.Error()))
		common.ApiErrorMsg(c, err.Error())
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
	paymentWalletAddress := strings.TrimSpace(req.PaymentWalletAddress)
	if currency == "" || chainID == "" || paymentWalletAddress == "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	if !containsAxoneCurrency(currency) {
		common.ApiErrorMsg(c, "Unsupported AXOne currency")
		return
	}

	client := service.GetAxoneClient()
	axoneChain, err := resolveAxonePaymentChain(c.Request.Context(), client, chainID)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("AXOne resolve chain failed chain_id=%s error=%q", chainID, err.Error()))
		common.ApiErrorMsg(c, err.Error())
		return
	}

	id := c.GetInt("id")
	baseMoney := decimal.NewFromInt(req.Amount).Round(axoneMoneyScale)
	feeMoney := decimal.Zero
	feePercent := decimal.NewFromFloat(setting.GetAxoneFeePercent())
	if feePercent.IsPositive() {
		feeMoney = baseMoney.Mul(feePercent).Div(decimal.NewFromInt(100)).RoundCeil(axoneMoneyScale)
	}
	paymentMoney := baseMoney.Add(feeMoney).Round(axoneMoneyScale)
	if !paymentMoney.IsPositive() {
		common.ApiErrorMsg(c, "Invalid AXOne payment amount")
		return
	}

	tradeNo := fmt.Sprintf("AXONE-%d-%d-%s", id, time.Now().UnixMilli(), randstr.String(6))

	topUp := &model.TopUp{
		UserId:                    id,
		Amount:                    req.Amount,
		Money:                     paymentMoney.InexactFloat64(),
		Fee:                       feeMoney.InexactFloat64(),
		TradeNo:                   tradeNo,
		PaymentMethod:             model.PaymentMethodAxone,
		PaymentProvider:           model.PaymentProviderAxone,
		AxoneCurrency:             currency,
		AxoneChainID:              chainID,
		AxonePaymentWalletAddress: paymentWalletAddress,
		CreateTime:                time.Now().Unix(),
		Status:                    common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("AXOne create order failed user_id=%d trade_no=%s amount=%d error=%q", id, tradeNo, req.Amount, err.Error()))
		common.ApiErrorMsg(c, "Failed to create AXOne topup order")
		return
	}

	axoneOrder, err := client.CreatePaymentOrder(c.Request.Context(), service.AxonePaymentOrderRequest{
		OrderNo:              tradeNo,
		Amount:               paymentMoney.StringFixed(axoneMoneyScale),
		Currency:             currency,
		Chain:                axoneChain,
		PaymentWalletAddress: paymentWalletAddress,
	})
	if err != nil {
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		logger.LogError(c.Request.Context(), fmt.Sprintf("AXOne create provider order failed user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
		common.ApiErrorMsg(c, err.Error())
		return
	}

	topUp.AxoneOrderNo = strings.TrimSpace(axoneOrder.AxoneOrderNo)
	topUp.AxoneAddress = strings.TrimSpace(axoneOrder.PayAddress)
	topUp.ProviderPayload = common.GetJsonString(axoneOrder)
	if err := topUp.Update(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("AXOne update order failed user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
		common.ApiErrorMsg(c, "Failed to save AXOne payment order")
		return
	}

	common.ApiSuccess(c, gin.H{
		"trade_no":               tradeNo,
		"axone_order_no":         topUp.AxoneOrderNo,
		"amount":                 req.Amount,
		"base_payment_money":     baseMoney.StringFixed(axoneMoneyScale),
		"fee":                    feeMoney.StringFixed(axoneMoneyScale),
		"payment_money":          paymentMoney.StringFixed(axoneMoneyScale),
		"display_fee":            feeMoney.StringFixed(2),
		"display_payment_money":  paymentMoney.StringFixed(2),
		"currency":               currency,
		"chain_id":               chainID,
		"address":                topUp.AxoneAddress,
		"payment_wallet_address": paymentWalletAddress,
		"status":                 common.TopUpStatusPending,
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

func resolveAxonePaymentChain(ctx context.Context, client *service.AxoneClient, selectedChain string) (string, error) {
	selectedChain = strings.TrimSpace(selectedChain)
	if selectedChain == "" {
		return "", fmt.Errorf("empty chain")
	}

	chains, err := client.ListChains(ctx)
	if err != nil {
		return "", err
	}
	for _, chain := range chains {
		if chain.ChainID == selectedChain || strings.EqualFold(chain.ChainName, selectedChain) {
			chainName := strings.TrimSpace(chain.ChainName)
			if chainName == "" {
				return "", fmt.Errorf("empty chain name")
			}
			return chainName, nil
		}
	}
	return "", fmt.Errorf("chain not found")
}

func AxoneWebhook(c *gin.Context) {
	if !setting.AxoneEnabled {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("AXOne webhook rejected reason=disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": 1, "message": "disabled"})
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("AXOne webhook read body failed path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "bad request"})
		return
	}

	var payload axoneWebhookPayload
	if err := common.Unmarshal(bodyBytes, &payload); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("AXOne webhook parse failed path=%q client_ip=%s error=%q body=%q", c.Request.RequestURI, c.ClientIP(), err.Error(), string(bodyBytes)))
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "bad request"})
		return
	}

	timestamp := strings.TrimSpace(c.GetHeader("X-Timestamp"))
	signature := strings.TrimSpace(c.GetHeader("X-Signature"))
	if !verifyAxoneWebhookSignature(timestamp, payload, bodyBytes, signature) {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("AXOne webhook signature invalid path=%q client_ip=%s signature=%q body=%q", c.Request.RequestURI, c.ClientIP(), signature, string(bodyBytes)))
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "message": "invalid signature"})
		return
	}

	if payload.Event != "payment.success" || payload.Status != "Paid" || payload.WriteOffStatus != "WrittenOff" {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("AXOne webhook ignored event=%s status=%s write_off_status=%s order_no=%s client_ip=%s", payload.Event, payload.Status, payload.WriteOffStatus, payload.OrderNo, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		return
	}

	if err := model.RechargeAxone(
		payload.OrderNo,
		payload.AxoneOrderNo,
		payload.Amount,
		payload.Currency,
		payload.PayAddress,
		payload.PaymentWalletAddress,
		string(bodyBytes),
		c.ClientIP(),
	); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("AXOne webhook recharge failed order_no=%s axone_order_no=%s client_ip=%s error=%q", payload.OrderNo, payload.AxoneOrderNo, c.ClientIP(), err.Error()))
		c.JSON(http.StatusOK, gin.H{"code": 1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func verifyAxoneWebhookSignature(timestamp string, payload axoneWebhookPayload, rawBody []byte, signature string) bool {
	publicKeyConfig := strings.TrimSpace(setting.AxoneWebhookPublicKey)
	if publicKeyConfig == "" || timestamp == "" || signature == "" {
		return false
	}

	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}
	publicKey, err := parseAxoneWebhookPublicKey(publicKeyConfig)
	if err != nil {
		return false
	}

	bodyBytes, err := common.Marshal(payload)
	if err == nil && verifyAxoneWebhookSignaturePayload(publicKey, timestamp, bodyBytes, signatureBytes) {
		return true
	}
	return verifyAxoneWebhookSignaturePayload(publicKey, timestamp, rawBody, signatureBytes)
}

func verifyAxoneWebhookSignaturePayload(publicKey *rsa.PublicKey, timestamp string, body []byte, signature []byte) bool {
	signPayload := append([]byte(timestamp+"."), body...)
	digest := sha256.Sum256(signPayload)
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature) == nil
}

func parseAxoneWebhookPublicKey(publicKeyConfig string) (*rsa.PublicKey, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(publicKeyConfig, `\n`, "\n"))
	if !strings.Contains(normalized, "-----BEGIN") {
		decoded, err := base64.StdEncoding.DecodeString(normalized)
		if err == nil {
			decodedValue := strings.TrimSpace(string(decoded))
			if strings.Contains(decodedValue, "-----BEGIN") {
				normalized = decodedValue
			}
		}
	}

	block, _ := pem.Decode([]byte(normalized))
	if block == nil {
		return nil, fmt.Errorf("invalid public key pem")
	}
	parsedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	publicKey, ok := parsedKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not rsa")
	}
	return publicKey, nil
}
