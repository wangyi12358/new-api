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
import { Input } from '@/components/ui/input'
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
  amount: number
  currencies?: string[]
  onPaymentSuccess?: () => void | Promise<void>
}

export function AxoneTopupSection({
  enabled,
  amount,
  currencies = [],
  onPaymentSuccess,
}: AxoneTopupSectionProps) {
  const { t } = useTranslation()
  const [selectedCurrency, setSelectedCurrency] = useState('')
  const [selectedChainID, setSelectedChainID] = useState('')
  const [paymentWalletAddress, setPaymentWalletAddress] = useState('')
  const [nowMs, setNowMs] = useState(() => Date.now())
  const {
    chains,
    chainsLoading,
    generating,
    addressData,
    setAddressData,
    loadChains,
    generateAddress,
  } = useAxoneTopup(enabled, { onPaymentSuccess })

  const normalizedCurrencies = useMemo(
    () =>
      currencies
        .map((currency) => currency.trim().toUpperCase())
        .filter(Boolean),
    [currencies]
  )

  useEffect(() => {
    setSelectedCurrency((previous) => {
      if (previous && normalizedCurrencies.includes(previous)) {
        return previous
      }
      return normalizedCurrencies[0] || ''
    })
  }, [normalizedCurrencies])

  useEffect(() => {
    if (!selectedCurrency) {
      setSelectedChainID('')
      return
    }
    void loadChains(selectedCurrency)
  }, [selectedCurrency, loadChains])

  useEffect(() => {
    setSelectedChainID((previous) => {
      if (previous && chains.some((chain) => chain.chain_id === previous)) {
        return previous
      }
      return chains[0]?.chain_id || ''
    })
  }, [chains])

  useEffect(() => {
    setAddressData(null)
  }, [selectedCurrency, selectedChainID, paymentWalletAddress, setAddressData])

  useEffect(() => {
    if (!addressData?.expires_at) {
      return
    }
    const timer = window.setInterval(() => {
      setNowMs(Date.now())
    }, 1000)
    return () => window.clearInterval(timer)
  }, [addressData?.expires_at])

  if (!enabled) {
    return null
  }

  const handleGenerateAddress = async () => {
    if (!selectedCurrency || !selectedChainID || !paymentWalletAddress.trim() || amount <= 0) {
      return
    }
    await generateAddress(
      amount,
      selectedCurrency,
      selectedChainID,
      paymentWalletAddress
    )
  }

  const remainingSeconds = Math.max(
    0,
    Math.floor((Number(addressData?.expires_at || 0) * 1000 - nowMs) / 1000)
  )
  const remainingMinutes = Math.floor(remainingSeconds / 60)
  const remainingDisplaySeconds = remainingSeconds % 60
  const hasExpireTime = Number(addressData?.expires_at || 0) > 0

  return (
    <div className='space-y-4'>
      <div className='flex items-center gap-2'>
        <Coins className='text-muted-foreground h-4 w-4' />
        <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
          {t('Stablecoin Top-up')}
        </Label>
      </div>

      <Alert>
        <AlertDescription>
          {t(
            'Choose a currency and chain, enter the wallet address you will transfer from, generate a payment order, and transfer funds to the wallet address.'
          )}
        </AlertDescription>
      </Alert>

      <div className='bg-muted/30 grid gap-3 rounded-lg border p-3 text-sm sm:grid-cols-4'>
        <div>
          <div className='text-muted-foreground text-xs'>{t('Topup Amount')}</div>
          <div className='mt-1 font-semibold'>{amount || '-'}</div>
        </div>
        <div>
          <div className='text-muted-foreground text-xs'>{t('Fee Amount')}</div>
          <div className='mt-1 font-semibold'>
            {addressData?.display_fee || '-'} {selectedCurrency || t('Currency')}
          </div>
        </div>
        <div>
          <div className='text-muted-foreground text-xs'>
            {t('Total')}
          </div>
          <div className='mt-1 flex items-center gap-2 font-bold text-red-600'>
            <span>
              {addressData?.display_payment_money || '-'}{' '}
              {selectedCurrency || t('Currency')}
            </span>
            {addressData?.display_payment_money && (
              <CopyButton
                value={addressData.display_payment_money}
                variant='outline'
                size='sm'
                className='h-7 px-2'
                iconClassName='h-3.5 w-3.5'
              >
                <span className='text-xs'>{t('Copy')}</span>
              </CopyButton>
            )}
          </div>
        </div>
        <div>
          <div className='text-muted-foreground text-xs'>
            {t('Credit Amount')}
          </div>
          <div className='mt-1 font-semibold'>{addressData?.amount || amount || '-'}</div>
        </div>
      </div>

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
          {normalizedCurrencies.length === 0 && (
            <p className='text-muted-foreground text-xs'>
              {t('No stablecoin currencies are configured yet.')}
            </p>
          )}
        </div>

        <div className='space-y-2'>
          <div className='flex items-center justify-between gap-2'>
            <Label>{t('Chain')}</Label>
            <Button
              type='button'
              variant='ghost'
              size='sm'
              className='h-7 px-2'
              onClick={() => void loadChains(selectedCurrency)}
              disabled={chainsLoading || !selectedCurrency}
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
            <SelectTrigger className='h-9' disabled={!selectedCurrency || chainsLoading}>
              <SelectValue
                placeholder={
                  !selectedCurrency
                    ? t('Select currency')
                    : chainsLoading
                      ? t('Loading chains...')
                      : t('Select chain')
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
          {selectedCurrency && !chainsLoading && chains.length === 0 && (
            <p className='text-muted-foreground text-xs'>
              {t(
                'No supported chains were loaded. Please refresh or check the AXOne configuration.'
              )}
            </p>
          )}
        </div>

        <div className='space-y-2 sm:col-span-2'>
          <Label>{t('Payment Wallet Address')}</Label>
          <Input
            value={paymentWalletAddress}
            onChange={(event) => setPaymentWalletAddress(event.target.value)}
            placeholder={t('Enter the wallet address you will transfer from')}
          />
          <p className='text-muted-foreground text-xs'>
            {t(
              'AXOne matches the source wallet address on-chain, so please enter the exact wallet address you will use to transfer.'
            )}
          </p>
        </div>
      </div>

      <Button
        type='button'
        onClick={() => void handleGenerateAddress()}
        disabled={
          generating ||
          chainsLoading ||
          amount <= 0 ||
          !selectedCurrency ||
          !selectedChainID ||
          !paymentWalletAddress.trim() ||
          normalizedCurrencies.length === 0
        }
        className='w-full sm:w-auto'
      >
        {generating && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
        {t('Generate Payment Order')}
      </Button>

      {addressData?.address && (
        <div className='bg-muted/30 space-y-2 rounded-lg border p-3'>
          <Alert variant='destructive'>
            <AlertDescription>
              {t(
                'Please transfer the exact amount shown on this page. Otherwise the system may not be able to identify your payment order.'
              )}
            </AlertDescription>
          </Alert>

          <div className='flex items-center justify-between gap-3'>
            <div>
              <div className='text-sm font-medium'>{t('Wallet Address')}</div>
              <div className='text-muted-foreground text-xs'>
                {addressData.currency} · {addressData.chain_id}
              </div>
            </div>
            <CopyButton value={addressData.address} variant='outline' />
          </div>
          <div className='grid gap-2 text-sm sm:grid-cols-2'>
            <div>
              <span className='text-muted-foreground'>{t('Order No')}: </span>
              <span className='font-mono'>{addressData.trade_no}</span>
            </div>
            {addressData.axone_order_no && (
              <div>
                <span className='text-muted-foreground'>{t('AXOne Order No')}: </span>
                <span className='font-mono'>{addressData.axone_order_no}</span>
              </div>
            )}
            <div>
              <span className='text-muted-foreground'>{t('Payment')}: </span>
              <span>{addressData.base_payment_money || '-'}</span>
            </div>
            <div>
              <span className='text-muted-foreground'>{t('Fee Amount')}: </span>
              <span>{addressData.display_fee || '0.00'}</span>
            </div>
            <div>
              <span className='text-muted-foreground'>{t('Transfer Amount')}: </span>
              <span className='inline-flex items-center gap-2'>
                <span className='font-semibold text-red-600'>
                  {addressData.display_payment_money || addressData.payment_money}
                </span>
                <CopyButton
                  value={addressData.display_payment_money || addressData.payment_money}
                  variant='outline'
                  size='sm'
                  className='h-7 px-2'
                  iconClassName='h-3.5 w-3.5'
                >
                  <span className='text-xs'>{t('Copy')}</span>
                </CopyButton>
              </span>
            </div>
            <div>
              <span className='text-muted-foreground'>{t('Credit Amount')}: </span>
              <span>{addressData.amount}</span>
            </div>
            {hasExpireTime && (
              <div>
                <span className='text-muted-foreground'>{t('Expires In')}: </span>
                <span>
                  {remainingMinutes}:{remainingDisplaySeconds
                    .toString()
                    .padStart(2, '0')}
                </span>
              </div>
            )}
          </div>
          <div className='break-all font-mono text-sm'>{addressData.address}</div>
        </div>
      )}
    </div>
  )
}
