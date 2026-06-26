package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
)

type axoneAddressRequest struct {
	Currency string `json:"currency"`
	ChainID  string `json:"chain_id"`
}

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

	client := service.GetAxoneClient()
	address, err := client.GetWalletAddress(c.Request.Context(), currency, chainID)
	if err != nil {
		common.ApiErrorMsg(c, "Failed to generate AXOne wallet address")
		return
	}

	common.ApiSuccess(c, gin.H{
		"currency": currency,
		"chain_id": chainID,
		"address":  address,
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
