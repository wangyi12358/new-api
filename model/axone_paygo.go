package model

// AxonePaygoSession mirrors the provider-side balance hold while keeping the
// Kovar ownership and in-flight accounting needed to safely bill concurrent AI
// requests. Monetary values are stored as q8 integers (1 unit = 0.00000001).
type AxonePaygoSession struct {
	Id                   int64  `json:"id"`
	SessionID            string `json:"session_id" gorm:"type:varchar(255);uniqueIndex"`
	UserID               int    `json:"user_id" gorm:"index;uniqueIndex:idx_axone_paygo_create_user_key"`
	WalletID             string `json:"wallet_id" gorm:"type:varchar(255);index"`
	Currency             string `json:"currency" gorm:"type:varchar(20)"`
	Status               string `json:"status" gorm:"type:varchar(32);index"`
	ReservedQ8           int64  `json:"reserved_q8"`
	AccruedQ8            int64  `json:"accrued_q8"`
	ProviderConsumedQ8   int64  `json:"provider_consumed_q8"`
	InFlightQ8           int64  `json:"in_flight_q8"`
	PendingQ8            int64  `json:"pending_q8"`
	LastEventSeq         int64  `json:"last_event_seq"`
	CreateIdempotencyKey string `json:"-" gorm:"type:varchar(255);uniqueIndex:idx_axone_paygo_create_user_key"`
	CreateRequestHash    string `json:"-" gorm:"type:varchar(64)"`
	ProviderPayload      string `json:"-" gorm:"type:text"`
	ExpiresAt            int64  `json:"expires_at" gorm:"index"`
	ClosedAt             int64  `json:"closed_at"`
	CreatedAt            int64  `json:"created_at"`
	UpdatedAt            int64  `json:"updated_at"`
}

type AxonePaygoUsage struct {
	Id            int64  `json:"id"`
	SessionID     string `json:"session_id" gorm:"type:varchar(255);index"`
	UserID        int    `json:"user_id" gorm:"index"`
	RequestID     string `json:"request_id" gorm:"type:varchar(255);uniqueIndex"`
	EstimatedQ8   int64  `json:"estimated_q8"`
	ActualQ8      int64  `json:"actual_q8"`
	Status        string `json:"status" gorm:"type:varchar(32);index"`
	ChargeEventID string `json:"charge_event_id" gorm:"type:varchar(255);index"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

type AxonePaygoCharge struct {
	Id              int64  `json:"id"`
	SessionID       string `json:"session_id" gorm:"type:varchar(255);uniqueIndex:idx_axone_paygo_charge_seq;index"`
	EventID         string `json:"event_id" gorm:"type:varchar(255);uniqueIndex"`
	EventSeq        int64  `json:"event_seq" gorm:"uniqueIndex:idx_axone_paygo_charge_seq"`
	AmountQ8        int64  `json:"amount_q8"`
	Status          string `json:"status" gorm:"type:varchar(32);index"`
	IdempotencyKey  string `json:"-" gorm:"type:varchar(255);uniqueIndex"`
	Attempts        int    `json:"attempts"`
	LastError       string `json:"last_error" gorm:"type:text"`
	ProviderPayload string `json:"-" gorm:"type:text"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}
