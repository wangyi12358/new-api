package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type createAxonePaygoSessionRequest struct {
	WalletID  string `json:"wallet_id"`
	MaxAmount string `json:"max_amount"`
}

func CreateAxonePaygoSession(c *gin.Context) {
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Idempotency-Key is required"})
		return
	}
	var req createAxonePaygoSessionRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request body"})
		return
	}
	session, err := service.CreateAxonePaygoSession(c.Request.Context(), c.GetInt("id"), req.WalletID, req.MaxAmount, idempotencyKey)
	if err != nil {
		respondAxonePaygoError(c, err)
		return
	}
	common.ApiSuccess(c, session)
}

func ListAxoneWallets(c *gin.Context) {
	if !service.IsAxonePaygoReady() {
		respondAxonePaygoError(c, service.ErrAxonePaygoDisabled)
		return
	}
	wallets, err := service.GetAxoneClient().ListWallets(c.Request.Context())
	if err != nil {
		respondAxonePaygoError(c, err)
		return
	}
	common.ApiSuccess(c, wallets)
}

func ListAxonePaygoSessions(c *gin.Context) {
	sessions, err := service.ListAxonePaygoSessions(c.GetInt("id"), 50)
	if err != nil {
		respondAxonePaygoError(c, err)
		return
	}
	common.ApiSuccess(c, sessions)
}

func GetAxonePaygoSession(c *gin.Context) {
	session, err := service.GetAxonePaygoSession(c.Request.Context(), c.GetInt("id"), c.Param("id"), true)
	if err != nil {
		respondAxonePaygoError(c, err)
		return
	}
	common.ApiSuccess(c, session)
}

func CloseAxonePaygoSession(c *gin.Context) {
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Idempotency-Key is required"})
		return
	}
	session, err := service.CloseAxonePaygoSession(c.Request.Context(), c.GetInt("id"), c.Param("id"), idempotencyKey)
	if err != nil {
		respondAxonePaygoError(c, err)
		return
	}
	common.ApiSuccess(c, session)
}

func respondAxonePaygoError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, service.ErrAxonePaygoDisabled):
		status = http.StatusServiceUnavailable
	case errors.Is(err, service.ErrAxonePaygoSessionNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrAxonePaygoSessionUnavailable),
		errors.Is(err, service.ErrAxonePaygoInsufficientReserved):
		status = http.StatusPaymentRequired
	case errors.Is(err, service.ErrAxonePaygoIdempotencyConflict):
		status = http.StatusConflict
	}
	c.JSON(status, gin.H{"success": false, "message": err.Error()})
}
