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

import { useState } from 'react';
import { Button, Checkbox, Modal, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import MarkdownRenderer from '../common/markdown/MarkdownRenderer';
import { acceptKovarTerms, termsContent } from '../../constants/kovarTermsOfUse';

const TermsOfUseModal = ({ visible, onAccept, isMobile }) => {
  const { t } = useTranslation();
  const [agreed, setAgreed] = useState(false);

  const handleAccept = () => {
    acceptKovarTerms();
    onAccept();
  };

  const handleDecline = () => {
    window.location.href = 'https://kovar.ai';
  };

  return (
    <Modal
      title={t('KOVAR 使用条款')}
      visible={visible}
      closable={false}
      maskClosable={false}
      closeOnEsc={false}
      footer={
        <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between w-full'>
          <Checkbox
            checked={agreed}
            onChange={(e) => setAgreed(Boolean(e.target?.checked))}
          >
            {t('我已阅读并同意 KOVAR 使用条款')}
          </Checkbox>
          <div className='flex justify-end gap-2'>
            <Button type='tertiary' onClick={handleDecline}>
              {t('不同意')}
            </Button>
            <Button type='primary' disabled={!agreed} onClick={handleAccept}>
              {t('我同意')}
            </Button>
          </div>
        </div>
      }
      size={isMobile ? 'full-width' : 'large'}
      style={{ maxWidth: isMobile ? undefined : 900 }}
    >
      <Typography.Paragraph type='tertiary' className='mb-4'>
        {t('请阅读以下使用条款。同意后方可继续使用 KOVAR 服务。')}
      </Typography.Paragraph>
      <div className='max-h-[55vh] overflow-y-auto pr-2 border border-semi-color-border rounded-lg p-4 bg-semi-color-bg-0'>
        <div className='prose prose-sm max-w-none dark:prose-invert'>
          <MarkdownRenderer content={termsContent} />
        </div>
      </div>
    </Modal>
  );
};

export default TermsOfUseModal;
