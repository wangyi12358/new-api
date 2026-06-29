package setting

import "strings"

var AxoneEnabled = false
var AxoneBaseURL = ""
var AxoneAccount = ""
var AxonePassword = ""
var AxoneCurrencies = "USDT,USDC"
var AxoneFeePercent = 0.0

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
