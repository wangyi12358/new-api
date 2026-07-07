/*
Copyright (C) 2025 QuantumNous

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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import {
  Avatar,
  Typography,
  Card,
  Button,
  Banner,
  Skeleton,
  Form,
  Space,
  Row,
  Col,
  Spin,
  Tooltip,
  Select,
  Tag,
  Tabs,
  TabPane,
  Modal,
  Input,
} from '@douyinfe/semi-ui';
import { SiAlipay, SiWechat, SiStripe, SiTether } from 'react-icons/si';
import {
  CreditCard,
  Coins,
  Wallet,
  BarChart2,
  TrendingUp,
  Receipt,
  Sparkles,
} from 'lucide-react';
import { IconCopy, IconGift } from '@douyinfe/semi-icons';
import { useMinimumLoadingTime } from '../../hooks/common/useMinimumLoadingTime';
import { useActualTheme } from '../../context/Theme';
import { getCurrencyConfig } from '../../helpers/render';
import SubscriptionPlansCard from './SubscriptionPlansCard';

const { Text } = Typography;

const RechargeCard = ({
  t,
  enableOnlineTopUp,
  enableAlipayTopUp,
  enableStripeTopUp,
  enableCreemTopUp,
  creemProducts,
  creemPreTopUp,
  enableAxoneTopUp,
  axoneCurrencies,
  selectedAxoneCurrency,
  setSelectedAxoneCurrency,
  axoneChains,
  selectedAxoneChain,
  setSelectedAxoneChain,
  axonePaymentWalletAddress,
  setAxonePaymentWalletAddress,
  axoneAddress,
  axoneTradeNo,
  axoneOrderNo,
  axoneBasePaymentMoney,
  axoneFee,
  axonePaymentMoney,
  axoneExpireAt,
  axoneChainLoading,
  axoneAddressLoading,
  getAxoneChains,
  generateAxoneAddress,
  handleCopyAxoneAddress,
  handleCopyAxonePaymentMoney,
  presetAmounts,
  selectedPreset,
  selectPresetAmount,
  formatLargeNumber,
  priceRatio,
  topUpCount,
  minTopUp,
  renderQuotaWithAmount,
  getAmount,
  setTopUpCount,
  setSelectedPreset,
  renderAmount,
  amountLoading,
  amountNumber = 0,
  payMethods,
  preTopUp,
  paymentLoading,
  payWay,
  redemptionCode,
  setRedemptionCode,
  topUp,
  isSubmitting,
  topUpLink,
  openTopUpLink,
  userState,
  renderQuota,
  statusLoading,
  topupInfo,
  onOpenHistory,
  enableWaffoTopUp,
  enableWaffoPancakeTopUp,
  subscriptionLoading = false,
  subscriptionPlans = [],
  billingPreference,
  onChangeBillingPreference,
  activeSubscriptions = [],
  allSubscriptions = [],
  reloadSubscriptionSelf,
  enableRedemption = true,
}) => {
  const onlineFormApiRef = useRef(null);
  const redeemFormApiRef = useRef(null);
  const initialTabSetRef = useRef(false);
  const showAmountSkeleton = useMinimumLoadingTime(amountLoading);
  const actualTheme = useActualTheme();
  const [activeTab, setActiveTab] = useState('topup');
  const [axoneModalOpen, setAxoneModalOpen] = useState(false);
  const shouldShowSubscription =
    !subscriptionLoading && subscriptionPlans.length > 0;
  const regularPayMethods = payMethods || [];

  console.log('presetAmounts', presetAmounts);

  const axoneTransferAmount = useMemo(() => {
    if (!amountNumber || amountNumber <= 0) {
      return '0.00';
    }
    return Number(amountNumber).toFixed(2);
  }, [amountNumber]);

  const selectedAxoneChainInfo = useMemo(
    () =>
      (axoneChains || []).find((chain) => chain.chain_id === selectedAxoneChain),
    [axoneChains, selectedAxoneChain],
  );

  const selectedAxoneChainLabel = selectedAxoneChainInfo
    ? `${selectedAxoneChainInfo.chain_name} (${selectedAxoneChainInfo.symbol})`
    : selectedAxoneChain;

  const [axoneNowMs, setAxoneNowMs] = useState(() => Date.now());

  useEffect(() => {
    if (!axoneExpireAt || axoneExpireAt <= 0) {
      return;
    }
    setAxoneNowMs(Date.now());
    const timer = window.setInterval(() => {
      setAxoneNowMs(Date.now());
    }, 1000);
    return () => window.clearInterval(timer);
  }, [axoneExpireAt]);

  const axoneRemainingSeconds = Math.max(
    0,
    Math.floor((Number(axoneExpireAt || 0) * 1000 - axoneNowMs) / 1000),
  );
  const axoneRemainingHours = Math.floor(axoneRemainingSeconds / 3600);
  const axoneRemainingMinutes = Math.floor((axoneRemainingSeconds % 3600) / 60);
  const axoneRemainingDisplaySeconds = axoneRemainingSeconds % 60;
  const getAxoneCountdownText = () => {
    if (axoneRemainingSeconds <= 0) {
      return t('已过期');
    }
    const secondsText = String(axoneRemainingDisplaySeconds).padStart(2, '0');
    if (axoneRemainingHours > 0) {
      const minutesText = String(axoneRemainingMinutes).padStart(2, '0');
      return `${axoneRemainingHours}:${minutesText}:${secondsText}`;
    }
    return `${axoneRemainingMinutes}:${secondsText}`;
  };
  const axoneCountdownText = getAxoneCountdownText();

  useEffect(() => {
    if (initialTabSetRef.current) return;
    if (subscriptionLoading) return;
    setActiveTab(shouldShowSubscription ? 'subscription' : 'topup');
    initialTabSetRef.current = true;
  }, [shouldShowSubscription, subscriptionLoading]);

  useEffect(() => {
    if (!shouldShowSubscription && activeTab !== 'topup') {
      setActiveTab('topup');
    }
  }, [shouldShowSubscription, activeTab]);
  const topupContent = (
    <Space vertical style={{ width: '100%' }}>
      {/* 统计数据 */}
      <Card
        className='!rounded-xl w-full'
        cover={
          <div
            className='relative h-30'
            style={{
              '--palette-primary-darkerChannel': '37 99 235',
              backgroundImage: `linear-gradient(0deg, rgba(var(--palette-primary-darkerChannel) / 80%), rgba(var(--palette-primary-darkerChannel) / 80%)), url('/cover-4.webp')`,
              backgroundSize: 'cover',
              backgroundPosition: 'center',
              backgroundRepeat: 'no-repeat',
            }}
          >
            <div className='relative z-10 h-full flex flex-col justify-between p-4'>
              <div className='flex justify-between items-center'>
                <Text strong style={{ color: 'white', fontSize: '16px' }}>
                  {t('账户统计')}
                </Text>
              </div>

              {/* 统计数据 */}
              <div className='grid grid-cols-3 gap-6 mt-4'>
                {/* 当前余额 */}
                <div className='text-center'>
                  <div
                    className='text-base sm:text-2xl font-bold mb-2'
                    style={{ color: 'white' }}
                  >
                    {renderQuota(userState?.user?.quota)}
                  </div>
                  <div className='flex items-center justify-center text-sm'>
                    <Wallet
                      size={14}
                      className='mr-1'
                      style={{ color: 'rgba(255,255,255,0.8)' }}
                    />
                    <Text
                      style={{
                        color: 'rgba(255,255,255,0.8)',
                        fontSize: '12px',
                      }}
                    >
                      {t('当前余额')}
                    </Text>
                  </div>
                </div>

                {/* 历史消耗 */}
                <div className='text-center'>
                  <div
                    className='text-base sm:text-2xl font-bold mb-2'
                    style={{ color: 'white' }}
                  >
                    {renderQuota(userState?.user?.used_quota)}
                  </div>
                  <div className='flex items-center justify-center text-sm'>
                    <TrendingUp
                      size={14}
                      className='mr-1'
                      style={{ color: 'rgba(255,255,255,0.8)' }}
                    />
                    <Text
                      style={{
                        color: 'rgba(255,255,255,0.8)',
                        fontSize: '12px',
                      }}
                    >
                      {t('历史消耗')}
                    </Text>
                  </div>
                </div>

                {/* 请求次数 */}
                <div className='text-center'>
                  <div
                    className='text-base sm:text-2xl font-bold mb-2'
                    style={{ color: 'white' }}
                  >
                    {userState?.user?.request_count || 0}
                  </div>
                  <div className='flex items-center justify-center text-sm'>
                    <BarChart2
                      size={14}
                      className='mr-1'
                      style={{ color: 'rgba(255,255,255,0.8)' }}
                    />
                    <Text
                      style={{
                        color: 'rgba(255,255,255,0.8)',
                        fontSize: '12px',
                      }}
                    >
                      {t('请求次数')}
                    </Text>
                  </div>
                </div>
              </div>
            </div>
          </div>
        }
      >
        {/* 在线充值表单 */}
        {statusLoading ? (
          <div className='py-8 flex justify-center'>
            <Spin size='large' />
          </div>
        ) : enableOnlineTopUp ||
          enableAlipayTopUp ||
          enableStripeTopUp ||
          enableAxoneTopUp ||
          enableCreemTopUp ||
          enableWaffoTopUp ||
          enableWaffoPancakeTopUp ||
          enableAxoneTopUp ? (
          <Form
            getFormApi={(api) => (onlineFormApiRef.current = api)}
            initValues={{ topUpCount: topUpCount }}
          >
            <div className='space-y-6'>
              {(enableOnlineTopUp ||
                enableAlipayTopUp ||
                enableStripeTopUp ||
                enableWaffoTopUp ||
                enableWaffoPancakeTopUp ||
                enableAxoneTopUp) && (
                <Row gutter={12}>
                  <Col xs={24} sm={24} md={24} lg={10} xl={10}>
                    <Form.InputNumber
                      field='topUpCount'
                      label={t('充值数量')}
                      disabled={
                        !enableOnlineTopUp &&
                        !enableAlipayTopUp &&
                        !enableStripeTopUp &&
                        !enableWaffoTopUp &&
                        !enableWaffoPancakeTopUp &&
                        !enableAxoneTopUp
                      }
                      placeholder={
                        t('充值数量，最低 ') + renderQuotaWithAmount(minTopUp)
                      }
                      value={topUpCount}
                      min={minTopUp}
                      max={999999999}
                      step={1}
                      precision={0}
                      onChange={async (value) => {
                        if (value && value >= 1) {
                          setTopUpCount(value);
                          setSelectedPreset(null);
                          await getAmount(value);
                        }
                      }}
                      onBlur={(e) => {
                        const value = parseInt(e.target.value);
                        if (!value || value < 1) {
                          setTopUpCount(1);
                          getAmount(1);
                        }
                      }}
                      formatter={(value) => (value ? `${value}` : '')}
                      parser={(value) =>
                        value ? parseInt(value.replace(/[^\d]/g, '')) : 0
                      }
                      // extraText={
                      //   <Skeleton
                      //     loading={showAmountSkeleton}
                      //     active
                      //     placeholder={
                      //       <Skeleton.Title
                      //         style={{
                      //           width: 120,
                      //           height: 20,
                      //           borderRadius: 6,
                      //         }}
                      //       />
                      //     }
                      //   >
                      //     <Text type='secondary' className='text-red-600'>
                      //       {t('实付金额：')}
                      //       <span style={{ color: 'red' }}>
                      //         {renderAmount()}
                      //       </span>
                      //     </Text>
                      //   </Skeleton>
                      // }
                      style={{ width: '100%' }}
                    />
                  </Col>
                  {regularPayMethods.length > 0 && (
                    <Col xs={24} sm={24} md={24} lg={14} xl={14}>
                      <Form.Slot label={t('选择支付方式')}>
                        <Space vertical align='start'>
                          <Space wrap>
                            {regularPayMethods.map((payMethod) => {
                              const minTopupVal =
                                Number(payMethod.min_topup) || 0;
                              const isAlipayGateway =
                                payMethod.type === 'alipay_gateway';
                              const isStripe = payMethod.type === 'stripe';
                              const isWaffo =
                                typeof payMethod.type === 'string' &&
                                payMethod.type.startsWith('waffo:');
                              const isWaffoPancake =
                                payMethod.type === 'waffo_pancake';
                              const disabled =
                                (!enableOnlineTopUp &&
                                  !isAlipayGateway &&
                                  !isStripe &&
                                  !isWaffo &&
                                  !isWaffoPancake) ||
                                (!enableAlipayTopUp && isAlipayGateway) ||
                                (!enableStripeTopUp && isStripe) ||
                                (!enableWaffoTopUp && isWaffo) ||
                                (!enableWaffoPancakeTopUp && isWaffoPancake) ||
                                minTopupVal > Number(topUpCount || 0);

                              const buttonEl = (
                                <Button
                                  key={payMethod.type}
                                  theme='outline'
                                  type='tertiary'
                                  onClick={() => preTopUp(payMethod.type)}
                                  disabled={disabled}
                                  loading={
                                    paymentLoading && payWay === payMethod.type
                                  }
                                  icon={
                                    payMethod.type === 'alipay' ||
                                    payMethod.type === 'alipay_gateway' ? (
                                      <SiAlipay size={18} color='#1677FF' />
                                    ) : payMethod.type === 'wxpay' ? (
                                      <SiWechat size={18} color='#07C160' />
                                    ) : payMethod.type === 'stripe' ? (
                                      <SiStripe size={18} color='#635BFF' />
                                    ) : payMethod.icon ? (
                                      <img
                                        src={payMethod.icon}
                                        alt={payMethod.name}
                                        style={{
                                          width: 18,
                                          height: 18,
                                          objectFit: 'contain',
                                        }}
                                      />
                                    ) : payMethod.type === 'waffo_pancake' ? (
                                      <img
                                        src={
                                          actualTheme === 'dark'
                                            ? '/waffo-logo-dark.svg'
                                            : '/waffo-logo-light.svg'
                                        }
                                        alt='Waffo'
                                        style={{
                                          width: 18,
                                          height: 18,
                                          objectFit: 'contain',
                                        }}
                                      />
                                    ) : (
                                      <CreditCard
                                        size={18}
                                        color={
                                          payMethod.color ||
                                          'var(--semi-color-text-2)'
                                        }
                                      />
                                    )
                                  }
                                  className='!rounded-lg !px-4 !py-2'
                                >
                                  {payMethod.name}
                                </Button>
                              );

                              return disabled &&
                                minTopupVal > Number(topUpCount || 0) ? (
                                <Tooltip
                                  content={
                                    t('此支付方式最低充值金额为') +
                                    ' ' +
                                    minTopupVal
                                  }
                                  key={payMethod.type}
                                >
                                  {buttonEl}
                                </Tooltip>
                              ) : (
                                <React.Fragment key={payMethod.type}>
                                  {buttonEl}
                                </React.Fragment>
                              );
                            })}
                          </Space>
                          {enableAxoneTopUp && (
                            <Button
                              theme='outline'
                              type='tertiary'
                              icon={<SiTether size={18} color='#26A17B' />}
                              onClick={() => setAxoneModalOpen(true)}
                              className='!rounded-lg !px-4 !py-2'
                            >
                              {t('稳定币支付')}
                            </Button>
                          )}
                        </Space>
                      </Form.Slot>
                    </Col>
                  )}
                  {regularPayMethods.length === 0 && enableAxoneTopUp && (
                    <Col xs={24} sm={24} md={24} lg={14} xl={14}>
                      <Form.Slot label={t('选择支付方式')}>
                        <Button
                          theme='outline'
                          type='tertiary'
                          icon={<SiTether size={18} color='#26A17B' />}
                          onClick={() => setAxoneModalOpen(true)}
                          className='!rounded-lg !px-4 !py-2'
                        >
                          {t('稳定币支付')}
                        </Button>
                      </Form.Slot>
                    </Col>
                  )}
                </Row>
              )}

              {(enableOnlineTopUp ||
                enableAlipayTopUp ||
                enableStripeTopUp ||
                enableWaffoTopUp ||
                enableWaffoPancakeTopUp ||
                enableAxoneTopUp) && (
                <Form.Slot
                  label={
                    <div className='flex items-center gap-2'>
                      <span>{t('选择充值额度')}</span>
                      {(() => {
                        const { symbol, rate, type } = getCurrencyConfig();
                        if (type === 'USD') return null;

                        return (
                          <span
                            style={{
                              color: 'var(--semi-color-text-2)',
                              fontSize: '12px',
                              fontWeight: 'normal',
                            }}
                          >
                            (1 $ = {rate.toFixed(2)} {symbol})
                          </span>
                        );
                      })()}
                    </div>
                  }
                >
                  <div className='grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2'>
                    {presetAmounts.map((preset, index) => {
                      const discount =
                        preset.discount ||
                        topupInfo?.discount?.[preset.value] ||
                        1.0;
                      const originalPrice = preset.value * priceRatio;
                      const discountedPrice = originalPrice * discount;
                      const hasDiscount = discount < 1.0;
                      const actualPay = discountedPrice;
                      const save = originalPrice - discountedPrice;

                      // 根据当前货币类型换算显示金额和数量
                      const { symbol, rate, type } = getCurrencyConfig();
                      const statusStr = localStorage.getItem('status');
                      let usdRate = 7; // 默认CNY汇率
                      try {
                        if (statusStr) {
                          const s = JSON.parse(statusStr);
                          usdRate = s?.usd_exchange_rate || 7;
                        }
                      } catch (e) {}

                      let displayValue = preset.value; // 显示的数量
                      let displayActualPay = actualPay;
                      let displaySave = save;

                      if (type === 'USD') {
                        // 数量保持USD，价格从CNY转USD
                        displayActualPay = actualPay / usdRate;
                        displaySave = save / usdRate;
                      } else if (type === 'CNY') {
                        // 数量转CNY，价格已是CNY
                        displayValue = preset.value * usdRate;
                      } else if (type === 'CUSTOM') {
                        // 数量和价格都转自定义货币
                        displayValue = preset.value * rate;
                        displayActualPay = (actualPay / usdRate) * rate;
                        displaySave = (save / usdRate) * rate;
                      }

                      return (
                        <Card
                          key={index}
                          style={{
                            cursor: 'pointer',
                            border:
                              selectedPreset === preset.value
                                ? '2px solid var(--semi-color-primary)'
                                : '1px solid var(--semi-color-border)',
                            height: '100%',
                            width: '100%',
                          }}
                          bodyStyle={{ padding: '12px' }}
                          onClick={() => {
                            selectPresetAmount(preset);
                            onlineFormApiRef.current?.setValue(
                              'topUpCount',
                              preset.value,
                            );
                          }}
                        >
                          <div style={{ textAlign: 'center' }}>
                            <Typography.Title
                              heading={6}
                              style={{ margin: '0 0 8px 0' }}
                            >
                              <Coins size={18} />
                              {formatLargeNumber(displayValue)} {symbol}
                              {hasDiscount && (
                                <Tag style={{ marginLeft: 4 }} color='green'>
                                  {t('折').includes('off')
                                    ? (
                                        (1 - parseFloat(discount)) *
                                        100
                                      ).toFixed(1)
                                    : (discount * 10).toFixed(1)}
                                  {t('折')}
                                </Tag>
                              )}
                            </Typography.Title>
                            <div
                              style={{
                                color: 'var(--semi-color-text-2)',
                                fontSize: '12px',
                                margin: '4px 0',
                              }}
                            >
                              {/* {t('实付')} {symbol}
                              {displayActualPay.toFixed(2)}，
                              {hasDiscount
                                ? `${t('节省')} ${symbol}${displaySave.toFixed(2)}`
                                : `${t('节省')} ${symbol}0.00`} */}
                              {t('实付')} {symbol}
                              {displayActualPay.toFixed(2)}，
                              {hasDiscount
                                ? `${t('节省')} ${symbol}${displaySave.toFixed(2)}`
                                : `${t('节省')} ${symbol}0.00`}
                            </div>
                          </div>
                        </Card>
                      );
                    })}
                  </div>
                </Form.Slot>
              )}

              {/* Creem 充值区域 */}
              {enableCreemTopUp && creemProducts.length > 0 && (
                <Form.Slot label={t('Creem 充值')}>
                  <div className='grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3'>
                    {creemProducts.map((product, index) => (
                      <Card
                        key={index}
                        onClick={() => creemPreTopUp(product)}
                        className='cursor-pointer !rounded-2xl transition-all hover:shadow-md border-gray-200 hover:border-gray-300'
                        bodyStyle={{ textAlign: 'center', padding: '16px' }}
                      >
                        <div className='font-medium text-lg mb-2'>
                          {product.name}
                        </div>
                        <div className='text-sm text-gray-600 mb-2'>
                          {t('充值额度')}: {product.quota}
                        </div>
                        <div className='text-lg font-semibold text-blue-600'>
                          {product.currency === 'EUR' ? '€' : '$'}
                          {product.price}
                        </div>
                      </Card>
                    ))}
                  </div>
                </Form.Slot>
              )}

            </div>
          </Form>
        ) : (
          <Banner
            type='info'
            description={t(
              '管理员未开启在线充值功能，请联系管理员开启或使用兑换码充值。',
            )}
            className='!rounded-xl'
            closeIcon={null}
          />
        )}
      </Card>

      <Modal
        title={
          <div className='flex items-center'>
            <Coins className='mr-2' size={18} />
            {t('稳定币支付')}
          </div>
        }
        visible={axoneModalOpen}
        footer={null}
        onCancel={() => setAxoneModalOpen(false)}
        maskClosable={false}
        centered
        width={520}
        bodyStyle={{ paddingBottom: 16 }}
      >
        <div className='space-y-4'>
          <Card
            className='!rounded-xl w-full'
            bodyStyle={{ padding: '14px 16px' }}
          >
            <div className='grid grid-cols-1 gap-4 sm:grid-cols-4'>
              <div>
                <Text type='tertiary' size='small'>
                  {t('充值额度')}
                </Text>
                <div className='mt-1 text-base font-semibold'>
                  {renderQuotaWithAmount(topUpCount)}
                </div>
              </div>
              <div className='sm:text-center'>
                <Text type='tertiary' size='small'>
                  {t('手续费')}
                </Text>
                <div className='mt-1 text-base font-semibold'>
                  {axoneFee || '-'} {selectedAxoneCurrency || t('稳定币')}
                </div>
              </div>
              <div className='text-right'>
                <Text type='tertiary' size='small'>
                  {t('总计')}
                </Text>
                <Skeleton
                  loading={showAmountSkeleton}
                  active
                  placeholder={
                    <Skeleton.Title
                      style={{ width: 96, height: 24, borderRadius: 6 }}
                    />
                  }
                >
                  <div className='mt-1 flex items-center justify-end gap-2 text-base font-semibold text-red-600'>
                    <span>
                      {axonePaymentMoney || '-'}{' '}
                      {selectedAxoneCurrency || t('稳定币')}
                    </span>
                  </div>
                </Skeleton>
              </div>
              <div className='text-right'>
                <Text type='tertiary' size='small'>
                  {t('到账金额')}
                </Text>
                <div className='mt-1 text-base font-semibold'>
                  {renderQuotaWithAmount(topUpCount)}
                </div>
              </div>
            </div>
          </Card>

          <div className='grid grid-cols-1 sm:grid-cols-2 gap-3'>
            <div className='space-y-1.5'>
              <Text strong>{t('选择币种')}</Text>
              <Select
                placeholder={t('请选择币种')}
                value={selectedAxoneCurrency}
                onChange={(value) => setSelectedAxoneCurrency(value)}
                optionList={(axoneCurrencies || []).map((currency) => ({
                  label: currency,
                  value: currency,
                }))}
                style={{ width: '100%' }}
              />
              {(!axoneCurrencies || axoneCurrencies.length === 0) && (
                <Text type='tertiary' size='small'>
                  {t('暂未配置稳定币币种')}
                </Text>
              )}
            </div>
            <div className='space-y-1.5'>
              <div className='flex items-center justify-between gap-2'>
                <Text strong>{t('选择链')}</Text>
                <Button
                  theme='borderless'
                  type='tertiary'
                  size='small'
                  icon={<Receipt size={14} />}
                  loading={axoneChainLoading}
                  onClick={getAxoneChains}
                >
                  {t('刷新')}
                </Button>
              </div>
              <Select
                placeholder={
                  axoneChainLoading ? t('加载链列表中...') : t('请选择链')
                }
                value={selectedAxoneChain}
                onChange={(value) => setSelectedAxoneChain(value)}
                optionList={(axoneChains || []).map((chain) => ({
                  label: `${chain.chain_name} (${chain.symbol})`,
                  value: chain.chain_id,
                }))}
                style={{ width: '100%' }}
              />
              {!axoneChainLoading &&
                (!axoneChains || axoneChains.length === 0) && (
                  <Text type='tertiary' size='small'>
                    {t('未加载到可用链，请刷新链列表或检查稳定币配置')}
                  </Text>
                )}
            </div>
            <div className='space-y-1.5 sm:col-span-2'>
              <Text strong>{t('付款钱包地址')}</Text>
              <Input
                placeholder={t('请输入实际转出的付款钱包地址')}
                value={axonePaymentWalletAddress}
                onChange={(value) => setAxonePaymentWalletAddress(value)}
                style={{ width: '100%' }}
              />
              <Text type='tertiary' size='small'>
                {t('AXOne 会按链上付款地址匹配订单，请填写你实际转账使用的钱包地址。')}
              </Text>
            </div>
          </div>

          <Button
            theme='solid'
            type='primary'
            style={{ width: '100%' }}
            loading={axoneAddressLoading}
            disabled={
              !selectedAxoneCurrency ||
              !selectedAxoneChain ||
              axoneChainLoading ||
              !axonePaymentWalletAddress?.trim()
            }
            onClick={generateAxoneAddress}
          >
            {axoneAddress ? t('重新生成支付订单') : t('生成支付订单')}
          </Button>

          {axoneAddress ? (
            <Card
              className='!rounded-xl w-full border-gray-200'
              bodyStyle={{ padding: '16px' }}
            >
              <Banner
                type='danger'
                closeIcon={null}
                fullMode={false}
                description={t(
                  '请务必按页面展示的转账金额转账，否则系统可能无法识别支付订单。',
                )}
                style={{ marginBottom: 16 }}
              />
              <div className='flex flex-col gap-5'>
                <div className='w-full space-y-4'>
                  <div className='rounded-xl bg-gray-50 px-4 py-3 dark:bg-gray-800'>
                    <Text type='tertiary' size='small'>
                      {t('请向以下地址转账')}
                    </Text>
                    <div className='mt-1 flex flex-wrap items-baseline gap-x-2 gap-y-1'>
                      <span className='text-2xl font-bold text-red-600'>
                        {axonePaymentMoney || axoneTransferAmount}
                      </span>
                      <span className='text-base font-medium'>
                        {selectedAxoneCurrency}
                      </span>
                    </div>
                    <Text type='tertiary' size='small' className='mt-1 block'>
                      {selectedAxoneChainLabel}
                    </Text>
                    <Text type='tertiary' size='small' className='mt-1 block'>
                      {t('付款钱包地址')}：{axonePaymentWalletAddress}
                    </Text>
                  </div>

                  {axoneTradeNo ? (
                    <div className='space-y-2 rounded-lg border border-gray-200 px-3 py-2.5'>
                      <div className='flex items-start justify-between gap-3'>
                        <Text type='tertiary' size='small' className='shrink-0'>
                          {t('订单号')}
                        </Text>
                        <Text
                          size='small'
                          style={{ wordBreak: 'break-all', textAlign: 'right' }}
                        >
                          {axoneTradeNo}
                        </Text>
                      </div>
                      {axoneOrderNo ? (
                        <div className='flex items-start justify-between gap-3'>
                          <Text type='tertiary' size='small' className='shrink-0'>
                            {t('AXOne 订单号')}
                          </Text>
                          <Text
                            size='small'
                            style={{ wordBreak: 'break-all', textAlign: 'right' }}
                          >
                            {axoneOrderNo}
                          </Text>
                        </div>
                      ) : null}
                      {axoneExpireAt > 0 ? (
                        <div className='flex items-start justify-between gap-3'>
                          <Text type='tertiary' size='small' className='shrink-0'>
                            {t('剩余时间')}
                          </Text>
                          <Text
                            size='small'
                            type={axoneRemainingSeconds <= 0 ? 'danger' : 'warning'}
                            strong={axoneRemainingSeconds <= 60}
                            style={{ textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}
                          >
                            {axoneCountdownText}
                          </Text>
                        </div>
                      ) : null}
                    </div>
                  ) : null}

                  <div className='space-y-2 rounded-lg border border-gray-200 px-3 py-2.5'>
                    <div className='flex items-center justify-between gap-3'>
                      <Text type='tertiary' size='small'>
                        {t('支付金额')}
                      </Text>
                      <Text size='small'>{axoneBasePaymentMoney || '-'}</Text>
                    </div>
                    <div className='flex items-center justify-between gap-3'>
                      <Text type='tertiary' size='small'>
                        {t('手续费')}
                      </Text>
                      <Text size='small'>{axoneFee || '0.0000'}</Text>
                    </div>
                    <div className='flex items-center justify-between gap-3 border-t border-gray-200 pt-2'>
                      <Text type='danger' size='small' strong>
                        {t('总支付金额')}
                      </Text>
                      <div className='flex items-center gap-2'>
                        <Text type='danger' size='small' strong>
                          {axonePaymentMoney || axoneTransferAmount}
                        </Text>
                        <Button
                          size='small'
                          theme='outline'
                          type='tertiary'
                          icon={<IconCopy />}
                          onClick={handleCopyAxonePaymentMoney}
                        >
                          {t('复制')}
                        </Button>
                      </div>
                    </div>
                  </div>

                  <div className='space-y-2'>
                    <Text type='tertiary' size='small'>
                      {t('钱包地址')}
                    </Text>
                    <div className='rounded-lg border border-gray-200 bg-gray-50 px-3 py-2.5 dark:bg-gray-800'>
                      <Text
                        copyable={false}
                        style={{
                          wordBreak: 'break-all',
                          fontFamily: 'monospace',
                          fontSize: '13px',
                        }}
                      >
                        {axoneAddress}
                      </Text>
                    </div>
                    <Button
                      style={{ width: '100%' }}
                      onClick={handleCopyAxoneAddress}
                    >
                      {t('复制地址')}
                    </Button>
                  </div>
                </div>

                <div className='flex flex-col items-center rounded-lg border border-dashed border-gray-200 bg-gray-50 px-4 py-4 dark:bg-gray-800'>
                  <div className='rounded-lg bg-white p-2'>
                    <QRCodeSVG value={axoneAddress} size={180} />
                  </div>
                  <Text type='tertiary' size='small' className='mt-2'>
                    {t('扫码转账')}
                  </Text>
                </div>
              </div>
            </Card>
          ) : null}
        </div>
      </Modal>

      {/* 兑换码充值 */}
      {enableRedemption ? (
        <Card
          className='!rounded-xl w-full'
          title={
            <Text type='tertiary' strong>
              {t('兑换码充值')}
            </Text>
          }
        >
          <Form
            getFormApi={(api) => (redeemFormApiRef.current = api)}
            initValues={{ redemptionCode: redemptionCode }}
          >
            <Form.Input
              field='redemptionCode'
              noLabel={true}
              placeholder={t('请输入兑换码')}
              value={redemptionCode}
              onChange={(value) => setRedemptionCode(value)}
              prefix={<IconGift />}
              suffix={
                <div className='flex items-center gap-2'>
                  <Button
                    type='primary'
                    theme='solid'
                    onClick={topUp}
                    loading={isSubmitting}
                  >
                    {t('兑换额度')}
                  </Button>
                </div>
              }
              showClear
              style={{ width: '100%' }}
              extraText={
                topUpLink && (
                  <Text type='tertiary'>
                    {t('在找兑换码？')}
                    <Text
                      type='secondary'
                      underline
                      className='cursor-pointer'
                      onClick={openTopUpLink}
                    >
                      {t('购买兑换码')}
                    </Text>
                  </Text>
                )
              }
            />
          </Form>
        </Card>
      ) : (
        <Banner
          type='warning'
          description={t('兑换码功能已禁用，管理员需先确认合规声明。')}
          closeIcon={null}
          className='!rounded-xl'
        />
      )}
    </Space>
  );

  return (
    <Card className='!rounded-2xl shadow-sm border-0'>
      {/* 卡片头部 */}
      <div className='flex items-center justify-between mb-4'>
        <div className='flex items-center'>
          <Avatar size='small' color='blue' className='mr-3 shadow-md'>
            <CreditCard size={16} />
          </Avatar>
          <div>
            <Typography.Text className='text-lg font-medium'>
              {t('账户充值')}
            </Typography.Text>
            <div className='text-xs'>{t('多种充值方式，安全便捷')}</div>
          </div>
        </div>
        <Button
          icon={<Receipt size={16} />}
          theme='solid'
          onClick={onOpenHistory}
        >
          {t('账单')}
        </Button>
      </div>

      {shouldShowSubscription ? (
        <Tabs type='card' activeKey={activeTab} onChange={setActiveTab}>
          <TabPane
            tab={
              <div className='flex items-center gap-2'>
                <Sparkles size={16} />
                {t('订阅套餐')}
              </div>
            }
            itemKey='subscription'
          >
            <div className='py-2'>
              <SubscriptionPlansCard
                t={t}
                loading={subscriptionLoading}
                plans={subscriptionPlans}
                payMethods={payMethods}
                enableOnlineTopUp={enableOnlineTopUp}
                enableStripeTopUp={enableStripeTopUp}
                enableCreemTopUp={enableCreemTopUp}
                billingPreference={billingPreference}
                onChangeBillingPreference={onChangeBillingPreference}
                activeSubscriptions={activeSubscriptions}
                allSubscriptions={allSubscriptions}
                reloadSubscriptionSelf={reloadSubscriptionSelf}
                withCard={false}
              />
            </div>
          </TabPane>
          <TabPane
            tab={
              <div className='flex items-center gap-2'>
                <Wallet size={16} />
                {t('额度充值')}
              </div>
            }
            itemKey='topup'
          >
            <div className='py-2'>{topupContent}</div>
          </TabPane>
        </Tabs>
      ) : (
        topupContent
      )}
    </Card>
  );
};

export default RechargeCard;
