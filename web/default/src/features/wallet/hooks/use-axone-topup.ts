/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useCallback, useEffect, useRef, useState } from 'react'
import i18next from 'i18next'
import { toast } from 'sonner'
import {
  getAxoneChains,
  getTopupStatus,
  isApiSuccess,
  requestAxoneAddress,
} from '../api'
import type { AxoneAddressData, AxoneChain } from '../types'

function getErrorMessage(message: string | undefined, data: unknown): string {
  if (typeof data === 'string' && data.trim()) {
    return data
  }
  return message || i18next.t('Request failed')
}

interface UseAxoneTopupOptions {
  onPaymentSuccess?: () => void | Promise<void>
}

export function useAxoneTopup(
  enabled: boolean,
  options: UseAxoneTopupOptions = {}
) {
  const { onPaymentSuccess } = options
  const [chains, setChains] = useState<AxoneChain[]>([])
  const [chainsLoading, setChainsLoading] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [addressData, setAddressData] = useState<AxoneAddressData | null>(null)
  const notifiedTradeNoRef = useRef('')

  const loadChains = useCallback(async () => {
    if (!enabled) return
    setChainsLoading(true)
    try {
      const response = await getAxoneChains()
      if (isApiSuccess(response) && Array.isArray(response.data)) {
        setChains(response.data)
        return
      }
      toast.error(getErrorMessage(response.message, response.data))
    } catch {
      toast.error(i18next.t('Failed to load supported chains'))
    } finally {
      setChainsLoading(false)
    }
  }, [enabled])

  const generateAddress = useCallback(
    async (
      amount: number,
      currency: string,
      chainID: string,
      paymentWalletAddress: string
    ) => {
      setGenerating(true)
      try {
        const response = await requestAxoneAddress({
          amount,
          currency,
          chain_id: chainID,
          payment_wallet_address: paymentWalletAddress,
        })
        if (isApiSuccess(response) && response.data) {
          setAddressData(response.data)
          notifiedTradeNoRef.current = ''
          toast.success(i18next.t('Payment order generated'))
          return response.data
        }
        toast.error(getErrorMessage(response.message, response.data))
        return null
      } catch {
        toast.error(i18next.t('Failed to generate payment order'))
        return null
      } finally {
        setGenerating(false)
      }
    },
    []
  )

  useEffect(() => {
    if (!enabled) {
      setChains([])
      setAddressData(null)
      notifiedTradeNoRef.current = ''
      return
    }
    void loadChains()
  }, [enabled, loadChains])

  useEffect(() => {
    const tradeNo = addressData?.trade_no
    if (!enabled || !tradeNo || addressData.status !== 'pending') {
      return
    }

    const checkStatus = async () => {
      try {
        const response = await getTopupStatus(tradeNo)
        if (!isApiSuccess(response) || !response.data) {
          return
        }

        const nextStatus = response.data.status
        if (nextStatus === 'pending') {
          return
        }

        setAddressData((previous) =>
          previous?.trade_no === tradeNo
            ? { ...previous, status: nextStatus }
            : previous
        )

        if (notifiedTradeNoRef.current === tradeNo) {
          return
        }
        notifiedTradeNoRef.current = tradeNo

        if (nextStatus === 'success') {
          toast.success(i18next.t('Stablecoin payment received'))
          await onPaymentSuccess?.()
          setAddressData(null)
        } else if (nextStatus === 'failed') {
          toast.error(i18next.t('Payment order failed'))
        } else if (nextStatus === 'expired') {
          toast.error(i18next.t('Payment order expired'))
        }
      } catch {
        // Keep polling; transient network errors should not interrupt the user.
      }
    }

    void checkStatus()
    const timer = window.setInterval(() => {
      void checkStatus()
    }, 5000)
    return () => window.clearInterval(timer)
  }, [addressData?.status, addressData?.trade_no, enabled, onPaymentSuccess])

  return {
    chains,
    chainsLoading,
    generating,
    addressData,
    setAddressData,
    loadChains,
    generateAddress,
  }
}
