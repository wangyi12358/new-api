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
import { useEffect, useMemo, useState } from 'react'
import { Coins, Loader2, RefreshCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { CopyButton } from '@/components/copy-button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useAxoneTopup } from '../hooks'

interface AxoneTopupSectionProps {
  enabled: boolean
  currencies?: string[]
}

export function AxoneTopupSection({
  enabled,
  currencies = [],
}: AxoneTopupSectionProps) {
  const { t } = useTranslation()
  const [selectedCurrency, setSelectedCurrency] = useState('')
  const [selectedChainID, setSelectedChainID] = useState('')
  const {
    chains,
    chainsLoading,
    generating,
    addressData,
    setAddressData,
    loadChains,
    generateAddress,
  } = useAxoneTopup(enabled)

  const normalizedCurrencies = useMemo(
    () =>
      currencies
        .map((currency) => currency.trim().toUpperCase())
        .filter(Boolean),
    [currencies]
  )

  useEffect(() => {
    setSelectedCurrency((previous) => previous || normalizedCurrencies[0] || '')
  }, [normalizedCurrencies])

  useEffect(() => {
    setAddressData(null)
  }, [selectedCurrency, selectedChainID, setAddressData])

  if (!enabled) {
    return null
  }

  const handleGenerateAddress = async () => {
    if (!selectedCurrency || !selectedChainID) {
      return
    }
    await generateAddress(selectedCurrency, selectedChainID)
  }

  return (
    <div className='space-y-2.5 border-t pt-4 sm:space-y-3 sm:pt-6'>
      <div className='flex items-center gap-2'>
        <Coins className='text-muted-foreground h-4 w-4' />
        <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
          {t('Stablecoin Top-up')}
        </Label>
      </div>

      <Alert>
        <AlertDescription>
          {t(
            'Choose a currency and chain, generate the hosted wallet address, and transfer funds to that address.'
          )}
        </AlertDescription>
      </Alert>

      <div className='grid gap-3 sm:grid-cols-2'>
        <div className='space-y-2'>
          <Label>{t('Currency')}</Label>
          <Select value={selectedCurrency} onValueChange={setSelectedCurrency}>
            <SelectTrigger className='h-9'>
              <SelectValue placeholder={t('Select currency')} />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              {normalizedCurrencies.map((currency) => (
                <SelectItem key={currency} value={currency}>
                  {currency}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className='space-y-2'>
          <div className='flex items-center justify-between gap-2'>
            <Label>{t('Chain')}</Label>
            <Button
              type='button'
              variant='ghost'
              size='sm'
              className='h-7 px-2'
              onClick={() => void loadChains()}
              disabled={chainsLoading}
            >
              {chainsLoading ? (
                <Loader2 className='h-3.5 w-3.5 animate-spin' />
              ) : (
                <RefreshCcw className='h-3.5 w-3.5' />
              )}
              <span className='ml-1'>{t('Refresh')}</span>
            </Button>
          </div>
          <Select value={selectedChainID} onValueChange={setSelectedChainID}>
            <SelectTrigger className='h-9'>
              <SelectValue
                placeholder={
                  chainsLoading ? t('Loading chains...') : t('Select chain')
                }
              />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              {chains.map((chain) => (
                <SelectItem key={chain.chain_id} value={chain.chain_id}>
                  {chain.chain_name} ({chain.symbol})
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <Button
        type='button'
        onClick={() => void handleGenerateAddress()}
        disabled={
          generating ||
          chainsLoading ||
          !selectedCurrency ||
          !selectedChainID ||
          normalizedCurrencies.length === 0
        }
        className='w-full sm:w-auto'
      >
        {generating && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
        {t('Generate Wallet Address')}
      </Button>

      {addressData?.address && (
        <div className='bg-muted/30 space-y-2 rounded-lg border p-3'>
          <div className='flex items-center justify-between gap-3'>
            <div>
              <div className='text-sm font-medium'>{t('Wallet Address')}</div>
              <div className='text-muted-foreground text-xs'>
                {addressData.currency} · {addressData.chain_id}
              </div>
            </div>
            <CopyButton value={addressData.address} variant='outline' />
          </div>
          <div className='break-all font-mono text-sm'>{addressData.address}</div>
        </div>
      )}
    </div>
  )
}
