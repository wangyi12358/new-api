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

import React, { useEffect, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Col,
  Form,
  Row,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';
import { Coins } from 'lucide-react';

const { Text } = Typography;
const toBoolean = (value) => value === true || value === 'true';

export default function SettingsPaymentGatewayAxone(props) {
  const { t } = useTranslation();
  const sectionTitle = props.hideSectionTitle ? undefined : t('AXOne 设置');
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    AxoneEnabled: false,
    AxoneBaseURL: '',
    AxoneAccount: '',
    AxonePassword: '',
    AxoneWebhookPublicKey: '',
    AxoneCurrencies: 'USDT,USDC',
    AxoneFeePercent: 0,
    AxonePaygoEnabled: false,
    AxonePaygoChargeMode: 'per_request',
    AxonePaygoChargeThreshold: '1.00000000',
  });
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        AxoneEnabled: toBoolean(props.options.AxoneEnabled),
        AxoneBaseURL: props.options.AxoneBaseURL || '',
        AxoneAccount: props.options.AxoneAccount || '',
        AxonePassword: props.options.AxonePassword || '',
        AxoneWebhookPublicKey: props.options.AxoneWebhookPublicKey || '',
        AxoneCurrencies: props.options.AxoneCurrencies || 'USDT,USDC',
        AxoneFeePercent: Number(props.options.AxoneFeePercent) || 0,
        AxonePaygoEnabled: toBoolean(props.options.AxonePaygoEnabled),
        AxonePaygoChargeMode:
          props.options.AxonePaygoChargeMode || 'per_request',
        AxonePaygoChargeThreshold:
          props.options.AxonePaygoChargeThreshold || '1.00000000',
      };
      setInputs(currentInputs);
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitAxoneSetting = async () => {
    setLoading(true);
    try {
      const options = [
        {
          key: 'AxoneBaseURL',
          value: removeTrailingSlash(inputs.AxoneBaseURL || ''),
        },
        {
          key: 'AxoneAccount',
          value: (inputs.AxoneAccount || '').trim(),
        },
        {
          key: 'AxoneCurrencies',
          value: (inputs.AxoneCurrencies || '').trim(),
        },
        {
          key: 'AxoneFeePercent',
          value: String(Math.max(0, Number(inputs.AxoneFeePercent) || 0)),
        },
        {
          key: 'AxonePaygoChargeMode',
          value: inputs.AxonePaygoChargeMode || 'per_request',
        },
        {
          key: 'AxonePaygoChargeThreshold',
          value: (inputs.AxonePaygoChargeThreshold || '').trim(),
        },
      ];

      if (inputs.AxonePassword) {
        options.push({ key: 'AxonePassword', value: inputs.AxonePassword });
      }
      if (inputs.AxoneWebhookPublicKey) {
        options.push({
          key: 'AxoneWebhookPublicKey',
          value: inputs.AxoneWebhookPublicKey,
        });
      }
      options.push(
        {
          key: 'AxoneEnabled',
          value: inputs.AxoneEnabled ? 'true' : 'false',
        },
        {
          key: 'AxonePaygoEnabled',
          value: inputs.AxonePaygoEnabled ? 'true' : 'false',
        },
      );

      for (const option of options) {
        const response = await API.put('/api/option/', {
          key: option.key,
          value: option.value,
        });
        if (!response.data.success) {
          showError(response.data.message);
          return;
        }
      }

      showSuccess(t('更新成功'));
      props.refresh?.();
    } catch (error) {
      showError(t('更新失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Spin spinning={loading}>
      <div style={{ paddingTop: 4 }}>
        {sectionTitle && (
          <div style={{ marginBottom: 16 }}>
            <Text strong>{sectionTitle}</Text>
          </div>
        )}

        <Banner
          type='info'
          icon={<Coins size={16} />}
          title={t('AXOne 稳定币钱包')}
          description={t(
            '配置 AXOne 支付订单 API 后，用户可在 classic 钱包页选择币种、填写付款钱包地址，并生成稳定币支付订单。',
          )}
          closeIcon={null}
          style={{ marginBottom: 16 }}
          fullMode={false}
        />

        <Form
          getFormApi={(api) => (formApiRef.current = api)}
          initValues={inputs}
          onValueChange={handleFormChange}
        >
          <Row gutter={16}>
            <Col span={24}>
              <Form.Switch
                field='AxoneEnabled'
                label={t('启用 AXOne 稳定币充值')}
                checkedText={t('已启用')}
                uncheckedText={t('已禁用')}
              />
            </Col>

            <Col span={24}>
              <Form.Input
                field='AxoneBaseURL'
                label={t('Base URL')}
                placeholder='https://test-api.alloyx-payment.net'
              />
            </Col>

            <Col span={24}>
              <Form.TextArea
                field='AxoneWebhookPublicKey'
                label={t('Webhook 公钥')}
                placeholder={t('用于校验 AXOne 回调签名的公钥')}
                autosize={{ minRows: 4, maxRows: 8 }}
              />
            </Col>

            <Col xs={24} md={12}>
              <Form.Input
                field='AxoneAccount'
                label={t('账号')}
                placeholder='user@example.com'
              />
            </Col>

            <Col xs={24} md={12}>
              <Form.Input
                mode='password'
                field='AxonePassword'
                label={t('密码')}
                placeholder={t('请输入 AXOne 登录密码')}
              />
            </Col>

            <Col span={24}>
              <Form.Input
                field='AxoneCurrencies'
                label={t('可选币种')}
                placeholder='USDT,USDC'
                extraText={t('用英文逗号分隔用户可选的币种，例如：USDT,USDC')}
              />
            </Col>

            <Col span={24}>
              <Form.InputNumber
                field='AxoneFeePercent'
                label={t('手续费百分比')}
                min={0}
                step={0.01}
                precision={2}
                placeholder='0'
                extraText={t(
                  '用户稳定币转账金额会按该百分比增加，例如 1 表示 1% 手续费',
                )}
              />
            </Col>

            <Col span={24}>
              <Banner
                type='info'
                title={t('AXOne 流式支付')}
                description={t(
                  '携带 Kovar 支付会话的 AI 请求将使用 AXOne 锁定余额，不再扣除平台内部余额。',
                )}
                closeIcon={null}
                fullMode={false}
                style={{ marginBottom: 8 }}
              />
            </Col>

            <Col span={24}>
              <Form.Switch
                field='AxonePaygoEnabled'
                label={t('启用 AXOne 流式支付')}
                checkedText={t('已启用')}
                uncheckedText={t('已禁用')}
              />
            </Col>

            <Col xs={24} md={12}>
              <Form.Select
                field='AxonePaygoChargeMode'
                label={t('AXOne 扣款模式')}
                optionList={[
                  {
                    value: 'per_request',
                    label: t('每次 AI 请求后扣款'),
                  },
                  {
                    value: 'threshold',
                    label: t('达到金额阈值后扣款'),
                  },
                ]}
              />
            </Col>

            <Col xs={24} md={12}>
              <Form.Input
                field='AxonePaygoChargeThreshold'
                label={t('AXOne 扣款阈值')}
                placeholder='1.00000000'
                disabled={inputs.AxonePaygoChargeMode !== 'threshold'}
                extraText={t('支持最多 8 位小数的 USDC 或 USDT 金额。')}
              />
            </Col>

            <Col span={24}>
              <Button onClick={submitAxoneSetting}>
                {t('更新 AXOne 设置')}
              </Button>
            </Col>
          </Row>
        </Form>
      </div>
    </Spin>
  );
}
