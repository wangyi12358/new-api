package controller

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
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

const (
	alipayGatewayURL  = "https://openapi.alipay.com/gateway.do"
	alipaySignType    = "RSA2"
	alipayCharset     = "utf-8"
	alipayVersion     = "1.0"
	alipayProductCode = "FAST_INSTANT_TRADE_PAY"
)

var alipayAdaptor = &AlipayAdaptor{}

type AlipayPayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
}

type AlipayAdaptor struct{}

func (*AlipayAdaptor) RequestAmount(c *gin.Context, req *AlipayPayRequest) {
	if req.Amount < getAlipayMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getAlipayMinTopup())})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}

	payMoney, _, err := getAlipayPayMoney(c.Request.Context(), float64(req.Amount), group)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("获取支付宝实时汇率失败 user_id=%d amount=%d error=%q", id, req.Amount, err.Error()))
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "error", "data": "暂时无法获取 USD/CNY 实时汇率，请稍后重试"})
		return
	}
	if payMoney <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success", "data": strconv.FormatFloat(payMoney, 'f', 2, 64)})
}

func (*AlipayAdaptor) RequestPay(c *gin.Context, req *AlipayPayRequest) {
	if req.PaymentMethod != model.PaymentMethodAlipay {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "不支持的支付渠道"})
		return
	}
	if req.Amount < getAlipayMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("充值数量不能小于 %d", getAlipayMinTopup()), "data": 10})
		return
	}
	if req.Amount > 10000 {
		c.JSON(http.StatusOK, gin.H{"message": "充值数量不能大于 10000", "data": 10})
		return
	}

	id := c.GetInt("id")
	user, err := model.GetUserById(id, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "用户不存在"})
		return
	}
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}

	reference := fmt.Sprintf("new-api-alipay-ref-%d-%d-%s", user.Id, time.Now().UnixMilli(), randstr.String(4))
	referenceId := "ref_" + common.Sha1([]byte(reference))
	payMoney, exchangeRate, err := getAlipayPayMoney(c.Request.Context(), float64(req.Amount), group)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("获取支付宝实时汇率失败 user_id=%d amount=%d error=%q", id, req.Amount, err.Error()))
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "暂时无法获取 USD/CNY 实时汇率，请稍后重试", "data": nil})
		return
	}

	payLink, err := genAlipayLink(c.Request.Context(), referenceId, payMoney)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝创建支付链接失败 user_id=%d trade_no=%s amount=%d error=%q", id, referenceId, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	exchangeRatePayload, err := common.Marshal(exchangeRate)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("序列化支付宝汇率快照失败 user_id=%d trade_no=%s error=%q", id, referenceId, err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "创建订单失败", "data": nil})
		return
	}

	topUp := &model.TopUp{
		UserId:          id,
		Amount:          req.Amount,
		Money:           payMoney,
		TradeNo:         referenceId,
		PaymentMethod:   model.PaymentMethodAlipay,
		PaymentProvider: model.PaymentProviderAlipay,
		ProviderPayload: string(exchangeRatePayload),
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝创建充值订单失败 user_id=%d trade_no=%s amount=%d error=%q", id, referenceId, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝充值订单创建成功 user_id=%d trade_no=%s amount_usd=%d money_cny=%.2f usd_cny_rate=%.6f rate_source=%q", id, referenceId, req.Amount, payMoney, exchangeRate.Rate, exchangeRate.Source))
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": payLink,
		},
	})
}

func RequestAlipayAmount(c *gin.Context) {
	var req AlipayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	alipayAdaptor.RequestAmount(c, &req)
}

func RequestAlipayPay(c *gin.Context) {
	var req AlipayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	alipayAdaptor.RequestPay(c, &req)
}

func AlipayNotify(c *gin.Context) {
	ctx := c.Request.Context()
	if !isAlipayWebhookEnabled() {
		logger.LogWarn(ctx, fmt.Sprintf("支付宝回调被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.String(http.StatusForbidden, "failure")
		return
	}

	if err := c.Request.ParseForm(); err != nil {
		logger.LogError(ctx, fmt.Sprintf("支付宝回调解析表单失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.String(http.StatusBadRequest, "failure")
		return
	}

	params := make(map[string]string, len(c.Request.PostForm))
	for key, values := range c.Request.PostForm {
		if len(values) == 0 {
			continue
		}
		params[key] = values[0]
	}

	logger.LogInfo(ctx, fmt.Sprintf("支付宝回调收到请求 path=%q client_ip=%s body=%q", c.Request.RequestURI, c.ClientIP(), common.GetJsonString(params)))

	if !verifyAlipayNotify(params) {
		logger.LogWarn(ctx, fmt.Sprintf("支付宝回调验签失败 path=%q client_ip=%s trade_no=%q out_trade_no=%q", c.Request.RequestURI, c.ClientIP(), params["trade_no"], params["out_trade_no"]))
		c.String(http.StatusBadRequest, "failure")
		return
	}

	if appID := strings.TrimSpace(params["app_id"]); appID != "" && appID != strings.TrimSpace(setting.AlipayAppID) {
		logger.LogWarn(ctx, fmt.Sprintf("支付宝回调应用ID不匹配 app_id=%q expected_app_id=%q trade_no=%q client_ip=%s", appID, setting.AlipayAppID, params["out_trade_no"], c.ClientIP()))
		c.String(http.StatusBadRequest, "failure")
		return
	}

	referenceId := strings.TrimSpace(params["out_trade_no"])
	if referenceId == "" {
		logger.LogWarn(ctx, fmt.Sprintf("支付宝回调缺少订单号 client_ip=%s body=%q", c.ClientIP(), common.GetJsonString(params)))
		c.String(http.StatusBadRequest, "failure")
		return
	}

	tradeStatus := strings.TrimSpace(params["trade_status"])
	if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
		logger.LogInfo(ctx, fmt.Sprintf("支付宝回调忽略未完成支付 trade_no=%s trade_status=%s client_ip=%s", referenceId, tradeStatus, c.ClientIP()))
		c.String(http.StatusOK, "success")
		return
	}

	LockOrder(referenceId)
	defer UnlockOrder(referenceId)

	if err := model.RechargeAlipay(referenceId, params["total_amount"], c.ClientIP()); err != nil {
		logger.LogError(ctx, fmt.Sprintf("支付宝充值处理失败 trade_no=%s alipay_trade_no=%s client_ip=%s error=%q", referenceId, params["trade_no"], c.ClientIP(), err.Error()))
		c.String(http.StatusInternalServerError, "failure")
		return
	}

	logger.LogInfo(ctx, fmt.Sprintf("支付宝充值成功 trade_no=%s alipay_trade_no=%s total_amount=%s buyer_pay_amount=%s client_ip=%s", referenceId, params["trade_no"], params["total_amount"], params["buyer_pay_amount"], c.ClientIP()))
	c.String(http.StatusOK, "success")
}

func genAlipayLink(ctx context.Context, referenceId string, payMoney float64) (string, error) {
	privateKey, err := parseRSAPrivateKey(setting.AlipayPrivateKey)
	if err != nil {
		return "", fmt.Errorf("支付宝应用私钥无效: %w", err)
	}

	notifyURL := strings.TrimSpace(setting.AlipayNotifyURL)
	if notifyURL == "" {
		notifyURL = strings.TrimRight(service.GetCallbackAddress(), "/") + "/api/alipay/notify"
	}

	returnURL := strings.TrimSpace(setting.AlipayReturnURL)
	if returnURL == "" {
		returnURL = paymentReturnPath("/console/topup")
	}

	bizContentBytes, err := common.Marshal(map[string]string{
		"out_trade_no":    referenceId,
		"product_code":    alipayProductCode,
		"total_amount":    formatAlipayAmount(payMoney),
		"subject":         getAlipaySubject(),
		"timeout_express": "30m",
	})
	if err != nil {
		return "", fmt.Errorf("序列化支付宝请求失败: %w", err)
	}

	params := map[string]string{
		"app_id":      strings.TrimSpace(setting.AlipayAppID),
		"biz_content": string(bizContentBytes),
		"charset":     alipayCharset,
		"format":      "JSON",
		"method":      "alipay.trade.page.pay",
		"notify_url":  notifyURL,
		"return_url":  returnURL,
		"sign_type":   alipaySignType,
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     alipayVersion,
	}

	sign, err := signAlipayParams(params, privateKey)
	if err != nil {
		return "", fmt.Errorf("支付宝请求签名失败: %w", err)
	}

	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	values.Set("sign", sign)

	payLink := alipayGatewayURL + "?" + values.Encode()
	logger.LogInfo(ctx, fmt.Sprintf("支付宝支付链接生成成功 trade_no=%s notify_url=%q return_url=%q", referenceId, notifyURL, returnURL))
	return payLink, nil
}

func getAlipaySubject() string {
	subject := strings.TrimSpace(setting.AlipayProductName)
	if subject == "" {
		return "new-api Balance Top-up"
	}
	return subject
}

func formatAlipayAmount(amount float64) string {
	return strconv.FormatFloat(amount, 'f', 2, 64)
}

func getAlipayPayMoney(ctx context.Context, amount float64, group string) (float64, service.ExchangeRateQuote, error) {
	exchangeRate, err := service.GetUSDToCNYExchangeRate(ctx)
	if err != nil {
		return 0, service.ExchangeRateQuote{}, err
	}

	originalAmount := amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		amount = amount / common.QuotaPerUnit
	}
	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}
	discount := 1.0
	if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(originalAmount)]; ok && ds > 0 {
		discount = ds
	}
	payMoney := calculateAlipayPayMoney(amount, exchangeRate.Rate, topupGroupRatio, discount)
	return payMoney, exchangeRate, nil
}

func calculateAlipayPayMoney(amount, exchangeRate, topupGroupRatio, discount float64) float64 {
	return decimal.NewFromFloat(amount).
		Mul(decimal.NewFromFloat(exchangeRate)).
		Mul(decimal.NewFromFloat(topupGroupRatio)).
		Mul(decimal.NewFromFloat(discount)).
		Round(2).
		InexactFloat64()
}

func getAlipayMinTopup() int64 {
	minTopup := setting.AlipayMinTopUp
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		minTopup = int(float64(minTopup) * common.QuotaPerUnit)
	}
	return int64(minTopup)
}

func verifyAlipayNotify(params map[string]string) bool {
	sign := strings.TrimSpace(params["sign"])
	if sign == "" {
		return false
	}

	publicKey, err := parseRSAPublicKey(setting.AlipayPublicKey)
	if err != nil {
		return false
	}

	content := buildAlipayNotifySignContent(params)
	signature, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return false
	}

	hashed := sha256.Sum256([]byte(content))
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], signature) == nil
}

func signAlipayParams(params map[string]string, privateKey *rsa.PrivateKey) (string, error) {
	content := buildAlipayRequestSignContent(params)
	hashed := sha256.Sum256([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func buildAlipayRequestSignContent(params map[string]string) string {
	return buildAlipaySignContent(params, false)
}

func buildAlipayNotifySignContent(params map[string]string) string {
	return buildAlipaySignContent(params, true)
}

func buildAlipaySignContent(params map[string]string, excludeSignType bool) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "sign" || (excludeSignType && key == "sign_type") || strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	return strings.Join(parts, "&")
}

func parseRSAPrivateKey(raw string) (*rsa.PrivateKey, error) {
	blockBytes, err := decodePEMOrBase64Key(raw)
	if err != nil {
		return nil, err
	}

	if privateKey, err := x509.ParsePKCS1PrivateKey(blockBytes); err == nil {
		return privateKey, nil
	}

	key, err := x509.ParsePKCS8PrivateKey(blockBytes)
	if err != nil {
		return nil, err
	}

	privateKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not RSA private key")
	}
	return privateKey, nil
}

func parseRSAPublicKey(raw string) (*rsa.PublicKey, error) {
	blockBytes, err := decodePEMOrBase64Key(raw)
	if err != nil {
		return nil, err
	}

	if publicKey, err := x509.ParsePKIXPublicKey(blockBytes); err == nil {
		rsaKey, ok := publicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("not RSA public key")
		}
		return rsaKey, nil
	}

	publicKey, err := x509.ParsePKCS1PublicKey(blockBytes)
	if err != nil {
		return nil, err
	}
	return publicKey, nil
}

func decodePEMOrBase64Key(raw string) ([]byte, error) {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return nil, errors.New("empty key")
	}

	if block, _ := pem.Decode([]byte(normalized)); block != nil {
		return block.Bytes, nil
	}

	normalized = strings.ReplaceAll(normalized, "\r", "")
	normalized = strings.ReplaceAll(normalized, "\n", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	return base64.StdEncoding.DecodeString(normalized)
}
