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
import { Banner, Button, Col, Form, Row, Spin, Typography } from '@douyinfe/semi-ui';
import { API, removeTrailingSlash, showError, showSuccess } from '../../../helpers';
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
    AxoneCurrencies: 'USDT,USDC',
  });
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        AxoneEnabled: toBoolean(props.options.AxoneEnabled),
        AxoneBaseURL: props.options.AxoneBaseURL || '',
        AxoneAccount: props.options.AxoneAccount || '',
        AxonePassword: props.options.AxonePassword || '',
        AxoneCurrencies: props.options.AxoneCurrencies || 'USDT,USDC',
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
          key: 'AxoneEnabled',
          value: inputs.AxoneEnabled ? 'true' : 'false',
        },
        {
          key: 'AxoneBaseURL',
          value: removeTrailingSlash(inputs.AxoneBaseURL || ''),
        },
        {
          key: 'AxoneAccount',
          value: (inputs.AxoneAccount || '').trim(),
        },
        {
          key: 'AxonePassword',
          value: inputs.AxonePassword || '',
        },
        {
          key: 'AxoneCurrencies',
          value: (inputs.AxoneCurrencies || '').trim(),
        },
      ];

      const results = await Promise.all(
        options.map((opt) =>
          API.put('/api/option/', {
            key: opt.key,
            value: opt.value,
          }),
        ),
      );

      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => showError(res.data.message));
      } else {
        showSuccess(t('更新成功'));
        props.refresh?.();
      }
    } catch (error) {
      showError(t('更新失败'));
    }
    setLoading(false);
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
            '配置 AXOne 钱包账号后，用户可在 classic 钱包页选择币种与链，并生成托管钱包地址进行稳定币充值。',
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
                extraText={t(
                  '用英文逗号分隔用户可选的币种，例如：USDT,USDC',
                )}
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
