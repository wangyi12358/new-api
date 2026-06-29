package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const axoneTopUpExpireTickInterval = time.Minute

var (
	axoneTopUpExpireOnce    sync.Once
	axoneTopUpExpireRunning atomic.Bool
)

func StartAxoneTopUpExpireTask() {
	axoneTopUpExpireOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}

		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("AXOne topup expire task started: tick=%s", axoneTopUpExpireTickInterval))

			ticker := time.NewTicker(axoneTopUpExpireTickInterval)
			defer ticker.Stop()

			runAxoneTopUpExpireOnce()
			for range ticker.C {
				runAxoneTopUpExpireOnce()
			}
		})
	})
}

func runAxoneTopUpExpireOnce() {
	if !axoneTopUpExpireRunning.CompareAndSwap(false, true) {
		return
	}
	defer axoneTopUpExpireRunning.Store(false)

	if err := model.ExpirePendingAxoneTopUps(common.GetTimestamp()); err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("AXOne topup expire task failed: %v", err))
	}
}
