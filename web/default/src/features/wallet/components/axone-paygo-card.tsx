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
import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { InformationCircleIcon, Wallet02Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { TitledCard } from '@/components/ui/titled-card'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { CopyButton } from '@/components/copy-button'
import {
  closeAxonePaygoSession,
  createAxonePaygoSession,
  getAxonePaygoSessions,
  getAxoneWallets,
  isApiSuccess,
  refreshAxonePaygoSession,
} from '../api'
import type { AxonePaygoSession } from '../types'

const sessionsKey = ['wallet', 'axone-paygo-sessions'] as const

function idempotencyKey(prefix: string) {
  return `${prefix}-${crypto.randomUUID()}`
}

function q8(value: number) {
  return (value / 100_000_000).toFixed(8)
}

function statusVariant(status: string) {
  if (status === 'active') return 'default' as const
  if (status === 'closed') return 'secondary' as const
  return 'outline' as const
}

export function AxonePaygoCard({ enabled }: { enabled: boolean }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [walletId, setWalletId] = useState('')
  const [maxAmount, setMaxAmount] = useState('')
  const [closingSession, setClosingSession] =
    useState<AxonePaygoSession | null>(null)

  const walletsQuery = useQuery({
    queryKey: ['wallet', 'axone-wallets'],
    enabled,
    queryFn: async () => {
      const response = await getAxoneWallets()
      if (!isApiSuccess(response)) throw new Error(response.message)
      return response.data?.list ?? []
    },
  })
  const sessionsQuery = useQuery({
    queryKey: sessionsKey,
    enabled,
    queryFn: async () => {
      const response = await getAxonePaygoSessions()
      if (!isApiSuccess(response)) throw new Error(response.message)
      return response.data ?? []
    },
  })

  const selectedWallet = useMemo(
    () => walletsQuery.data?.find((wallet) => wallet.id === walletId),
    [walletId, walletsQuery.data]
  )
  const amount = Number(maxAmount)
  const validAmount =
    /^\d+(\.\d{1,8})?$/.test(maxAmount) &&
    amount > 0 &&
    (!selectedWallet || amount <= selectedWallet.total_balance)

  const createMutation = useMutation({
    mutationFn: () =>
      createAxonePaygoSession(
        { wallet_id: walletId, max_amount: maxAmount },
        idempotencyKey('create')
      ),
    onSuccess: (response) => {
      if (!isApiSuccess(response)) {
        toast.error(response.message || t('Failed to lock balance'))
        return
      }
      toast.success(t('Balance locked successfully'))
      setMaxAmount('')
      void queryClient.invalidateQueries({ queryKey: sessionsKey })
      void queryClient.invalidateQueries({
        queryKey: ['wallet', 'axone-wallets'],
      })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const refreshMutation = useMutation({
    mutationFn: refreshAxonePaygoSession,
    onSuccess: (response) => {
      if (!isApiSuccess(response)) toast.error(response.message)
      void queryClient.invalidateQueries({ queryKey: sessionsKey })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const closeMutation = useMutation({
    mutationFn: (sessionId: string) =>
      closeAxonePaygoSession(sessionId, idempotencyKey('close')),
    onSuccess: (response) => {
      if (isApiSuccess(response)) toast.success(t('Payment session closed'))
      else toast.error(response.message)
      setClosingSession(null)
      void queryClient.invalidateQueries({ queryKey: sessionsKey })
      void queryClient.invalidateQueries({
        queryKey: ['wallet', 'axone-wallets'],
      })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  if (!enabled) return null

  const loading = walletsQuery.isLoading || sessionsQuery.isLoading
  const error = walletsQuery.error || sessionsQuery.error

  return (
    <>
      <TitledCard
        title={t('AXOne PayGo')}
        description={t('Lock wallet balance and pay as AI requests are used')}
        icon={<HugeiconsIcon icon={Wallet02Icon} strokeWidth={2} />}
        contentClassName='flex flex-col gap-5'
      >
        <Alert>
          <HugeiconsIcon icon={InformationCircleIcon} strokeWidth={2} />
          <AlertTitle>{t('How to use')}</AlertTitle>
          <AlertDescription>
            {t(
              'Create a payment session, then send its ID in the X-Kovar-Payment-Session header with each AI API request.'
            )}
          </AlertDescription>
        </Alert>

        {error && (
          <Alert variant='destructive'>
            <AlertTitle>{t('Failed to load AXOne data')}</AlertTitle>
            <AlertDescription>{error.message}</AlertDescription>
          </Alert>
        )}

        {loading ? (
          <div className='grid gap-3 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]'>
            <Skeleton className='h-8' />
            <Skeleton className='h-8' />
            <Skeleton className='h-8 w-28' />
          </div>
        ) : (
          <div className='grid gap-3 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] sm:items-end'>
            <label className='flex min-w-0 flex-col gap-1.5 text-sm font-medium'>
              {t('AXOne wallet')}
              <Select
                items={(walletsQuery.data ?? []).map((wallet) => ({
                  value: wallet.id,
                  label: `${wallet.currency} · ${wallet.total_balance}`,
                }))}
                value={walletId || null}
                onValueChange={(value) => setWalletId(value ?? '')}
              >
                <SelectTrigger className='w-full'>
                  <SelectValue placeholder={t('Select a wallet')} />
                </SelectTrigger>
                <SelectContent>
                  {(walletsQuery.data ?? []).map((wallet) => (
                    <SelectItem key={wallet.id} value={wallet.id}>
                      {wallet.currency} · {wallet.total_balance}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </label>
            <label className='flex min-w-0 flex-col gap-1.5 text-sm font-medium'>
              {t('Amount to lock')}
              <Input
                inputMode='decimal'
                placeholder={t('Up to 8 decimal places')}
                value={maxAmount}
                onChange={(event) => setMaxAmount(event.target.value.trim())}
              />
            </label>
            <Button
              disabled={!walletId || !validAmount || createMutation.isPending}
              onClick={() => createMutation.mutate()}
            >
              {createMutation.isPending && <Spinner />}
              {t('Lock balance')}
            </Button>
            {selectedWallet &&
              maxAmount &&
              amount > selectedWallet.total_balance && (
                <p className='text-destructive text-xs sm:col-span-3'>
                  {t('The lock amount cannot exceed the wallet balance')}
                </p>
              )}
          </div>
        )}

        <div className='flex flex-col gap-2'>
          <h3 className='text-sm font-semibold'>{t('Payment sessions')}</h3>
          {!sessionsQuery.isLoading &&
          (sessionsQuery.data?.length ?? 0) === 0 ? (
            <div className='text-muted-foreground rounded-lg border border-dashed p-6 text-center text-sm'>
              {t('No payment sessions yet')}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Session')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Locked')}</TableHead>
                  <TableHead>{t('Consumed')}</TableHead>
                  <TableHead>{t('Available')}</TableHead>
                  <TableHead className='text-right'>{t('Actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(sessionsQuery.data ?? []).map((session) => (
                  <TableRow key={session.session_id}>
                    <TableCell>
                      <div className='flex max-w-64 items-center gap-1'>
                        <code className='truncate text-xs'>
                          {session.session_id}
                        </code>
                        <CopyButton value={session.session_id} size='icon' />
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={statusVariant(session.status)}>
                        {session.status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {q8(session.reserved_q8)} {session.currency}
                    </TableCell>
                    <TableCell>
                      {q8(session.accrued_q8)} {session.currency}
                    </TableCell>
                    <TableCell>
                      {q8(
                        Math.max(
                          0,
                          session.reserved_q8 -
                            session.accrued_q8 -
                            session.in_flight_q8
                        )
                      )}{' '}
                      {session.currency}
                    </TableCell>
                    <TableCell>
                      <div className='flex justify-end gap-2'>
                        <Button
                          variant='outline'
                          size='sm'
                          disabled={refreshMutation.isPending}
                          onClick={() =>
                            refreshMutation.mutate(session.session_id)
                          }
                        >
                          {t('Refresh')}
                        </Button>
                        {session.status === 'active' && (
                          <Button
                            variant='destructive'
                            size='sm'
                            onClick={() => setClosingSession(session)}
                          >
                            {t('Close')}
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </div>
      </TitledCard>

      <ConfirmDialog
        open={closingSession != null}
        onOpenChange={(open) => !open && setClosingSession(null)}
        title={t('Close payment session?')}
        desc={t(
          'AXOne will settle consumed funds and release the remaining locked balance.'
        )}
        destructive
        isLoading={closeMutation.isPending}
        confirmText={t('Close session')}
        handleConfirm={() =>
          closingSession && closeMutation.mutate(closingSession.session_id)
        }
      />
    </>
  )
}
