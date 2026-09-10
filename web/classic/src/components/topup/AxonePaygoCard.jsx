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
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  InputNumber,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import { IconCopy, IconRefresh } from '@douyinfe/semi-icons';
import { Wallet } from 'lucide-react';
import { API, copy } from '../../helpers';

const q8 = (value) => (Number(value || 0) / 100000000).toFixed(8);
const newKey = (prefix) => `${prefix}-${crypto.randomUUID()}`;

const AxonePaygoCard = ({ t, enabled }) => {
  const [wallets, setWallets] = useState([]);
  const [sessions, setSessions] = useState([]);
  const [walletId, setWalletId] = useState('');
  const [maxAmount, setMaxAmount] = useState();
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [actionSession, setActionSession] = useState('');

  const load = useCallback(async () => {
    if (!enabled) return;
    setLoading(true);
    try {
      const [walletResponse, sessionResponse] = await Promise.all([
        API.get('/api/user/axone/wallets'),
        API.get('/api/user/axone/paygo/sessions'),
      ]);
      if (!walletResponse.data?.success) {
        throw new Error(
          walletResponse.data?.message || t('加载 AXOne 钱包失败'),
        );
      }
      if (!sessionResponse.data?.success) {
        throw new Error(sessionResponse.data?.message || t('加载支付会话失败'));
      }
      setWallets(walletResponse.data.data?.list || []);
      setSessions(sessionResponse.data.data || []);
    } catch (error) {
      Toast.error(error.message || t('加载 AXOne 数据失败'));
    } finally {
      setLoading(false);
    }
  }, [enabled, t]);

  useEffect(() => {
    load();
  }, [load]);

  const selectedWallet = useMemo(
    () => wallets.find((wallet) => wallet.id === walletId),
    [walletId, wallets],
  );
  const amountValid =
    Number(maxAmount) > 0 &&
    (!selectedWallet ||
      Number(maxAmount) <= Number(selectedWallet.total_balance));

  const createSession = async () => {
    setCreating(true);
    try {
      const response = await API.post(
        '/api/user/axone/paygo/sessions',
        { wallet_id: walletId, max_amount: Number(maxAmount).toFixed(8) },
        { headers: { 'Idempotency-Key': newKey('create') } },
      );
      if (!response.data?.success) {
        throw new Error(response.data?.message || t('锁定余额失败'));
      }
      Toast.success(t('余额锁定成功'));
      setMaxAmount(undefined);
      await load();
    } catch (error) {
      Toast.error(error.message || t('锁定余额失败'));
    } finally {
      setCreating(false);
    }
  };

  const refreshSession = async (sessionId) => {
    setActionSession(sessionId);
    try {
      const response = await API.get(
        `/api/user/axone/paygo/sessions/${encodeURIComponent(sessionId)}`,
      );
      if (!response.data?.success) throw new Error(response.data?.message);
      await load();
    } catch (error) {
      Toast.error(error.message || t('刷新支付会话失败'));
    } finally {
      setActionSession('');
    }
  };

  const closeSession = (sessionId) => {
    Modal.confirm({
      title: t('关闭支付会话？'),
      content: t('AXOne 将结算已消费金额，并释放剩余的锁定余额。'),
      okType: 'danger',
      onOk: async () => {
        setActionSession(sessionId);
        try {
          const response = await API.post(
            `/api/user/axone/paygo/sessions/${encodeURIComponent(sessionId)}/close`,
            null,
            { headers: { 'Idempotency-Key': newKey('close') } },
          );
          if (!response.data?.success) throw new Error(response.data?.message);
          Toast.success(t('支付会话已关闭'));
          await load();
        } catch (error) {
          Toast.error(error.message || t('关闭支付会话失败'));
          throw error;
        } finally {
          setActionSession('');
        }
      },
    });
  };

  if (!enabled) return null;

  const columns = [
    {
      title: t('会话'),
      dataIndex: 'session_id',
      render: (value) => (
        <Space spacing={4}>
          <Typography.Text
            code
            ellipsis={{ showTooltip: true }}
            style={{ maxWidth: 180 }}
          >
            {value}
          </Typography.Text>
          <Button
            icon={<IconCopy />}
            theme='borderless'
            size='small'
            aria-label={t('复制会话 ID')}
            onClick={async () => {
              await copy(value);
              Toast.success(t('复制成功'));
            }}
          />
        </Space>
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      render: (value) => (
        <Tag color={value === 'active' ? 'green' : 'grey'}>{value}</Tag>
      ),
    },
    {
      title: t('已锁定'),
      render: (_, row) => `${q8(row.reserved_q8)} ${row.currency}`,
    },
    {
      title: t('已消费'),
      render: (_, row) => `${q8(row.accrued_q8)} ${row.currency}`,
    },
    {
      title: t('可用'),
      render: (_, row) =>
        `${q8(Math.max(0, row.reserved_q8 - row.accrued_q8 - row.in_flight_q8))} ${row.currency}`,
    },
    {
      title: t('操作'),
      fixed: 'right',
      render: (_, row) => (
        <Space>
          <Button
            icon={<IconRefresh />}
            size='small'
            loading={actionSession === row.session_id}
            onClick={() => refreshSession(row.session_id)}
          >
            {t('刷新')}
          </Button>
          {row.status === 'active' && (
            <Button
              type='danger'
              size='small'
              onClick={() => closeSession(row.session_id)}
            >
              {t('关闭')}
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <Card className='!rounded-2xl shadow-sm border-0 lg:col-span-2'>
      <div className='flex items-center mb-4'>
        <div className='mr-3 flex h-8 w-8 items-center justify-center rounded-lg bg-blue-50 text-blue-600'>
          <Wallet size={17} />
        </div>
        <div>
          <Typography.Text className='text-lg font-medium'>
            AXOne PayGo
          </Typography.Text>
          <div className='text-xs'>
            {t('锁定钱包余额，按 AI API 实际使用量扣款')}
          </div>
        </div>
      </div>

      <Banner
        type='info'
        closeIcon={null}
        description={t(
          '最新的有效支付会话会自动用于 Playground 和 AI API 请求；也可以通过 X-Kovar-Payment-Session 请求头指定其他会话。',
        )}
        className='!rounded-xl mb-4'
      />

      <div className='grid grid-cols-1 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] gap-3 items-end mb-5'>
        <div>
          <div className='text-sm font-medium mb-1'>{t('AXOne 钱包')}</div>
          <Select
            value={walletId || undefined}
            placeholder={t('选择钱包')}
            className='w-full'
            loading={loading}
            onChange={setWalletId}
            optionList={wallets.map((wallet) => ({
              value: wallet.id,
              label: `${wallet.currency} · ${wallet.total_balance}`,
            }))}
          />
        </div>
        <div>
          <div className='text-sm font-medium mb-1'>{t('锁定金额')}</div>
          <InputNumber
            value={maxAmount}
            min={0}
            precision={8}
            className='w-full'
            placeholder={t('最多 8 位小数')}
            onChange={setMaxAmount}
          />
        </div>
        <Button
          type='primary'
          theme='solid'
          loading={creating}
          disabled={!walletId || !amountValid}
          onClick={createSession}
        >
          {t('锁定余额')}
        </Button>
      </div>
      {selectedWallet &&
        Number(maxAmount) > Number(selectedWallet.total_balance) && (
          <Typography.Text type='danger' size='small'>
            {t('锁定金额不能超过钱包余额')}
          </Typography.Text>
        )}

      <Table
        columns={columns}
        dataSource={sessions}
        rowKey='session_id'
        loading={loading}
        pagination={false}
        scroll={{ x: 900 }}
        empty={t('暂无支付会话')}
        size='small'
      />
    </Card>
  );
};

export default AxonePaygoCard;
