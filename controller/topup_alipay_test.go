package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildAlipayRequestSignContentIncludesSignType(t *testing.T) {
	params := map[string]string{
		"method":    "alipay.trade.page.pay",
		"app_id":    "2021000111111111",
		"sign_type": "RSA2",
		"sign":      "ignored",
		"empty":     "",
	}

	content := buildAlipayRequestSignContent(params)

	require.Equal(t, "app_id=2021000111111111&method=alipay.trade.page.pay&sign_type=RSA2", content)
}

func TestBuildAlipayNotifySignContentExcludesSignAndSignType(t *testing.T) {
	params := map[string]string{
		"trade_no":  "2026072200001",
		"app_id":    "2021000111111111",
		"sign_type": "RSA2",
		"sign":      "ignored",
	}

	content := buildAlipayNotifySignContent(params)

	require.Equal(t, "app_id=2021000111111111&trade_no=2026072200001", content)
}
