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
import { Banner, Button, Form, Row, Col, Spin } from '@douyinfe/semi-ui';
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';
import { BookOpen, TriangleAlert } from 'lucide-react';

export default function SettingsPaymentGatewayAlipay(props) {
  const { t } = useTranslation();
  const sectionTitle = props.hideSectionTitle ? undefined : t('支付宝设置');
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    AlipayAppID: '',
    AlipayPrivateKey: '',
    AlipayPublicKey: '',
    AlipayMinTopUp: 1,
    AlipayNotifyURL: '',
    AlipayReturnURL: '',
    AlipayProductName: 'new-api Balance Top-up',
  });
  const [originInputs, setOriginInputs] = useState({});
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        AlipayAppID: props.options.AlipayAppID || '',
        AlipayPrivateKey: props.options.AlipayPrivateKey || '',
        AlipayPublicKey: props.options.AlipayPublicKey || '',
        AlipayMinTopUp:
          props.options.AlipayMinTopUp !== undefined
            ? parseFloat(props.options.AlipayMinTopUp)
            : 1,
        AlipayNotifyURL: props.options.AlipayNotifyURL || '',
        AlipayReturnURL: props.options.AlipayReturnURL || '',
        AlipayProductName:
          props.options.AlipayProductName || 'new-api Balance Top-up',
      };
      setInputs(currentInputs);
      setOriginInputs({ ...currentInputs });
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitAlipaySetting = async () => {
    if (props.options.ServerAddress === '') {
      showError(t('请先填写服务器地址'));
      return;
    }

    setLoading(true);
    try {
      const options = [];

      if (inputs.AlipayAppID !== originInputs.AlipayAppID) {
        options.push({ key: 'AlipayAppID', value: inputs.AlipayAppID });
      }
      if (
        inputs.AlipayPrivateKey &&
        inputs.AlipayPrivateKey !== originInputs.AlipayPrivateKey
      ) {
        options.push({
          key: 'AlipayPrivateKey',
          value: inputs.AlipayPrivateKey,
        });
      }
      if (
        inputs.AlipayPublicKey &&
        inputs.AlipayPublicKey !== originInputs.AlipayPublicKey
      ) {
        options.push({
          key: 'AlipayPublicKey',
          value: inputs.AlipayPublicKey,
        });
      }
      if (
        inputs.AlipayMinTopUp !== undefined &&
        inputs.AlipayMinTopUp !== null
      ) {
        options.push({
          key: 'AlipayMinTopUp',
          value: inputs.AlipayMinTopUp.toString(),
        });
      }
      if (inputs.AlipayNotifyURL !== originInputs.AlipayNotifyURL) {
        options.push({
          key: 'AlipayNotifyURL',
          value: removeTrailingSlash(inputs.AlipayNotifyURL || ''),
        });
      }
      if (inputs.AlipayReturnURL !== originInputs.AlipayReturnURL) {
        options.push({
          key: 'AlipayReturnURL',
          value: removeTrailingSlash(inputs.AlipayReturnURL || ''),
        });
      }
      if (inputs.AlipayProductName !== originInputs.AlipayProductName) {
        options.push({
          key: 'AlipayProductName',
          value: inputs.AlipayProductName,
        });
      }

      const requestQueue = options.map((opt) =>
        API.put('/api/option/', {
          key: opt.key,
          value: opt.value,
        }),
      );

      const results = await Promise.all(requestQueue);
      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => {
          showError(res.data.message);
        });
      } else {
        showSuccess(t('更新成功'));
        setOriginInputs({
          ...inputs,
          AlipayNotifyURL: removeTrailingSlash(inputs.AlipayNotifyURL || ''),
          AlipayReturnURL: removeTrailingSlash(inputs.AlipayReturnURL || ''),
        });
        props.refresh?.();
      }
    } catch (error) {
      showError(t('更新失败'));
    }
    setLoading(false);
  };

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={sectionTitle}>
          <Banner
            type='info'
            icon={<BookOpen size={16} />}
            description={
              <>
                {t('请在支付宝开放平台配置电脑网站支付，并确保回调地址可公网访问。')}
                <br />
                {t('回调地址')}：
                {props.options.ServerAddress
                  ? removeTrailingSlash(props.options.ServerAddress)
                  : t('网站地址')}
                /api/alipay/notify
              </>
            }
            style={{ marginBottom: 12 }}
          />
          <Banner
            type='warning'
            icon={<TriangleAlert size={16} />}
            description={t(
              '支持粘贴 PEM 或 Base64 格式的 RSA2 应用私钥和支付宝公钥，留空表示保持当前密钥不变。',
            )}
            style={{ marginBottom: 16 }}
          />

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AlipayAppID'
                label={t('App ID')}
                placeholder={t('例如：2021000xxxxxxxxx')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='AlipayMinTopUp'
                label={t('最低充值美元数量')}
                placeholder={t('例如：2，就是最低充值2$')}
                extraText={t('用户单次最少可充值的美元数量')}
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AlipayPrivateKey'
                label={t('应用私钥')}
                placeholder={t('粘贴 RSA2 应用私钥，留空表示保持当前不变')}
                autosize={{ minRows: 5 }}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AlipayPublicKey'
                label={t('支付宝公钥')}
                placeholder={t('粘贴支付宝 RSA2 公钥，留空表示保持当前不变')}
                autosize={{ minRows: 5 }}
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AlipayNotifyURL'
                label={t('异步回调地址（可选）')}
                placeholder={t('留空默认使用 /api/alipay/notify')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AlipayReturnURL'
                label={t('同步返回地址（可选）')}
                placeholder={t('留空默认返回充值页')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AlipayProductName'
                label={t('订单标题')}
                placeholder={t('例如：new-api Balance Top-up')}
              />
            </Col>
          </Row>

          <Button onClick={submitAlipaySetting}>
            {t('更新支付宝设置')}
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
