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
import { useCallback, useEffect, useState } from 'react'
import i18next from 'i18next'
import { toast } from 'sonner'
import { getAxoneChains, isApiSuccess, requestAxoneAddress } from '../api'
import type { AxoneAddressData, AxoneChain } from '../types'

function getErrorMessage(message: string | undefined, data: unknown): string {
  if (typeof data === 'string' && data.trim()) {
    return data
  }
  return message || i18next.t('Request failed')
}

export function useAxoneTopup(enabled: boolean) {
  const [chains, setChains] = useState<AxoneChain[]>([])
  const [chainsLoading, setChainsLoading] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [addressData, setAddressData] = useState<AxoneAddressData | null>(null)

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
    async (amount: number, currency: string, chainID: string) => {
      setGenerating(true)
      try {
        const response = await requestAxoneAddress({
          amount,
          currency,
          chain_id: chainID,
        })
        if (isApiSuccess(response) && response.data) {
          setAddressData(response.data)
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
      return
    }
    void loadChains()
  }, [enabled, loadChains])

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
