package setting

import "strings"

var AxoneEnabled = false
var AxoneBaseURL = ""
var AxoneAccount = ""
var AxonePassword = ""
var AxoneWebhookSecret = ""
var AxoneWebhookPublicKey = ""
var AxoneCurrencies = "USDT,USDC"
var AxoneFeePercent = 0.0

const (
	AxonePaygoChargeModePerRequest = "per_request"
	AxonePaygoChargeModeThreshold  = "threshold"
)

// AXOne PayGo is independent from the stablecoin top-up switch. Keeping the
// switches separate prevents enabling metered billing before its session
// workflow is configured and verified.
var AxonePaygoEnabled = false
var AxonePaygoChargeMode = AxonePaygoChargeModePerRequest
var AxonePaygoChargeThreshold = "1.00000000"

func GetAxoneFeePercent() float64 {
	if AxoneFeePercent < 0 {
		return 0
	}
	return AxoneFeePercent
}

func GetAxoneCurrencies() []string {
	rawItems := strings.Split(AxoneCurrencies, ",")
	seen := make(map[string]struct{}, len(rawItems))
	currencies := make([]string, 0, len(rawItems))
	for _, item := range rawItems {
		normalized := strings.ToUpper(strings.TrimSpace(item))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		currencies = append(currencies, normalized)
	}
	return currencies
}

func GetAxonePaygoChargeMode() string {
	if strings.EqualFold(strings.TrimSpace(AxonePaygoChargeMode), AxonePaygoChargeModeThreshold) {
		return AxonePaygoChargeModeThreshold
	}
	return AxonePaygoChargeModePerRequest
}
