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
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { SettingsSwitchField } from '../components/settings-form-layout'
import { SettingsPageActionsPortal } from '../components/settings-page-context'
import { useUpdateOption } from '../hooks/use-update-option'

export interface AxoneSettingsValues {
  AxoneEnabled: boolean
  AxoneBaseURL: string
  AxoneAccount: string
  AxonePassword: string
  AxoneAccessToken: string
  AxoneWebhookPublicKey: string
  AxoneCurrencies: string
  AxoneFeePercent: number
}

interface Props {
  defaultValues: AxoneSettingsValues
}

export function AxoneSettingsSection({ defaultValues }: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [loading, setLoading] = useState(false)

  const form = useForm<AxoneSettingsValues>({
    defaultValues,
  })

  useEffect(() => {
    form.reset(defaultValues)
  }, [defaultValues, form])

  const handleSave = async () => {
    setLoading(true)
    try {
      const values = form.getValues()
      const options: { key: string; value: string }[] = [
        { key: 'AxoneEnabled', value: String(values.AxoneEnabled) },
        {
          key: 'AxoneBaseURL',
          value: values.AxoneBaseURL.trim().replace(/\/+$/, ''),
        },
        { key: 'AxoneAccount', value: values.AxoneAccount.trim() },
        { key: 'AxonePassword', value: values.AxonePassword },
        { key: 'AxoneAccessToken', value: values.AxoneAccessToken.trim() },
        { key: 'AxoneWebhookPublicKey', value: values.AxoneWebhookPublicKey },
        { key: 'AxoneCurrencies', value: values.AxoneCurrencies.trim() },
        {
          key: 'AxoneFeePercent',
          value: String(Math.max(0, Number(values.AxoneFeePercent) || 0)),
        },
      ]

      for (const option of options) {
        await updateOption.mutateAsync(option)
      }
      toast.success(t('Updated successfully'))
    } catch {
      toast.error(t('Update failed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className='space-y-4 pt-4'>
      <SettingsPageActionsPortal>
        <Button type='button' size='sm' onClick={handleSave} disabled={loading}>
          {loading ? t('Saving...') : t('Save AXOne settings')}
        </Button>
      </SettingsPageActionsPortal>

      <div>
        <h3 className='text-lg font-medium'>{t('AXOne Stablecoin Wallet')}</h3>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Configure the AXOne payment order API used to create stablecoin top-up orders.'
          )}
        </p>
      </div>

      <Alert>
        <AlertDescription className='text-xs'>
          {t(
            'AXOne sends payment success notifications to /api/axone/webhook. Configure this path as the merchant callback URL in AXOne.'
          )}
        </AlertDescription>
      </Alert>

      <SettingsSwitchField
        checked={form.watch('AxoneEnabled')}
        onCheckedChange={(value) => form.setValue('AxoneEnabled', value)}
        label={t('Enable AXOne stablecoin top-up')}
        className='border-b-0 py-0'
      />

      <div className='grid gap-4 sm:grid-cols-2'>
        <div className='grid gap-1.5 sm:col-span-2'>
          <Label htmlFor='axone-base-url'>{t('Base URL')}</Label>
          <Input
            id='axone-base-url'
            placeholder='https://test-api.alloyx-payment.net'
            {...form.register('AxoneBaseURL')}
          />
        </div>

        <div className='grid gap-1.5 sm:col-span-2'>
          <Label htmlFor='axone-access-token'>{t('Access Token')}</Label>
          <Input
            id='axone-access-token'
            type='password'
            placeholder={t('Bearer token from AXOne')}
            {...form.register('AxoneAccessToken')}
          />
        </div>

        <div className='grid gap-1.5 sm:col-span-2'>
          <Label htmlFor='axone-webhook-public-key'>{t('Webhook public key')}</Label>
          <Textarea
            id='axone-webhook-public-key'
            rows={7}
            placeholder={t('AXOne public key used to verify webhook signatures')}
            {...form.register('AxoneWebhookPublicKey')}
          />
        </div>

        <div className='grid gap-1.5'>
          <Label htmlFor='axone-account'>{t('Account')}</Label>
          <Input
            id='axone-account'
            placeholder='user@example.com'
            {...form.register('AxoneAccount')}
          />
        </div>

        <div className='grid gap-1.5'>
          <Label htmlFor='axone-password'>{t('Password')}</Label>
          <Input
            id='axone-password'
            type='password'
            placeholder='••••••••'
            {...form.register('AxonePassword')}
          />
        </div>

        <div className='grid gap-1.5 sm:col-span-2'>
          <Label htmlFor='axone-currencies'>{t('Supported Currencies')}</Label>
          <Input
            id='axone-currencies'
            placeholder='USDT,USDC'
            {...form.register('AxoneCurrencies')}
          />
          <p className='text-muted-foreground text-xs'>
            {t(
              'Comma-separated currency codes shown in the wallet, for example: USDT,USDC'
            )}
          </p>
        </div>

        <div className='grid gap-1.5 sm:col-span-2'>
          <Label htmlFor='axone-fee-percent'>{t('Top-up Fee Percent')}</Label>
          <Input
            id='axone-fee-percent'
            type='number'
            min='0'
            step='0.01'
            placeholder='0'
            {...form.register('AxoneFeePercent', { valueAsNumber: true })}
          />
          <p className='text-muted-foreground text-xs'>
            {t(
              'Percentage fee added to the stablecoin transfer amount. For example, 1 means 1%.'
            )}
          </p>
        </div>
      </div>
    </div>
  )
}
