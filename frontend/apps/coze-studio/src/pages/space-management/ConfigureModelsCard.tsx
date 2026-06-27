/*
 * Copyright 2025 ynet-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { useState, type FC } from 'react';

import { I18n, type I18nKeysNoOptionsType } from '@coze-arch/i18n';
import { Button, Input, InputNumber, Toast } from '@coze-arch/coze-design';
import { getLocalizedErrorMessage, SpaceApi } from '@coze-arch/bot-api';

import styles from './DataMaintenanceSection.module.less';

interface ConfigureForm {
  chatBaseURL: string;
  chatAPIKey: string;
  chatModel: string;
  embedderBaseURL: string;
  embedderAPIKey: string;
  embedderModel: string;
  embedderDims: number;
  rerankBaseURL: string;
  rerankAPIKey: string;
  rerankModel: string;
}

const getDisplayErrorMessage = (message?: string) =>
  getLocalizedErrorMessage(message) || message || '未知错误';

const t = (
  key: string,
  options: Record<string, unknown>,
  fallbackText: string,
) => I18n.t(key as I18nKeysNoOptionsType, options, fallbackText);

const emptyForm = (): ConfigureForm => ({
  chatBaseURL: '',
  chatAPIKey: '',
  chatModel: '',
  embedderBaseURL: '',
  embedderAPIKey: '',
  embedderModel: '',
  embedderDims: 1024,
  rerankBaseURL: '',
  rerankAPIKey: '',
  rerankModel: '',
});

interface Props {
  spaceId: string;
}

interface ChatFieldsProps {
  form: ConfigureForm;
  setForm: (f: ConfigureForm) => void;
}

const ChatFields: FC<ChatFieldsProps> = ({ form, setForm }) => (
  <div className={styles.formGroup}>
    <div className={styles.formGroupTitle}>Chat LLM</div>
    <Input
      placeholder="base_url (e.g. http://61.128.212.240:30080/inference/v1)"
      value={form.chatBaseURL}
      onChange={v => setForm({ ...form, chatBaseURL: v ?? '' })}
      className={styles.formInput}
    />
    <Input
      placeholder="api_key"
      value={form.chatAPIKey}
      onChange={v => setForm({ ...form, chatAPIKey: v ?? '' })}
      className={styles.formInput}
    />
    <Input
      placeholder="model (e.g. Qwen3-30B-A3B-Instruct-2507)"
      value={form.chatModel}
      onChange={v => setForm({ ...form, chatModel: v ?? '' })}
      className={styles.formInput}
    />
  </div>
);

const EmbedderFields: FC<ChatFieldsProps> = ({ form, setForm }) => (
  <div className={styles.formGroup}>
    <div className={styles.formGroupTitle}>Embedder</div>
    <Input
      placeholder="base_url"
      value={form.embedderBaseURL}
      onChange={v => setForm({ ...form, embedderBaseURL: v ?? '' })}
      className={styles.formInput}
    />
    <Input
      placeholder="api_key"
      value={form.embedderAPIKey}
      onChange={v => setForm({ ...form, embedderAPIKey: v ?? '' })}
      className={styles.formInput}
    />
    <Input
      placeholder="model (e.g. bge-large-zh-v1.5)"
      value={form.embedderModel}
      onChange={v => setForm({ ...form, embedderModel: v ?? '' })}
      className={styles.formInput}
    />
    <InputNumber
      placeholder="dims"
      value={form.embedderDims}
      min={1}
      onChange={v =>
        setForm({
          ...form,
          embedderDims: typeof v === 'number' ? v : 1024,
        })
      }
      className={styles.formInput}
    />
  </div>
);

const RerankFields: FC<ChatFieldsProps> = ({ form, setForm }) => (
  <div className={styles.formGroup}>
    <div className={styles.formGroupTitle}>Rerank</div>
    <Input
      placeholder="base_url"
      value={form.rerankBaseURL}
      onChange={v => setForm({ ...form, rerankBaseURL: v ?? '' })}
      className={styles.formInput}
    />
    <Input
      placeholder="api_key"
      value={form.rerankAPIKey}
      onChange={v => setForm({ ...form, rerankAPIKey: v ?? '' })}
      className={styles.formInput}
    />
    <Input
      placeholder="model (e.g. bge-reranker-large)"
      value={form.rerankModel}
      onChange={v => setForm({ ...form, rerankModel: v ?? '' })}
      className={styles.formInput}
    />
  </div>
);

export const ConfigureModelsCard: FC<Props> = ({ spaceId }) => {
  const [form, setForm] = useState<ConfigureForm>(emptyForm);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async () => {
    // Strip out empty sections so the backend skips that UPDATE entirely.
    const chat =
      form.chatBaseURL.trim() && form.chatAPIKey.trim() && form.chatModel.trim()
        ? {
            base_url: form.chatBaseURL.trim(),
            api_key: form.chatAPIKey.trim(),
            model: form.chatModel.trim(),
          }
        : undefined;
    const embedder =
      form.embedderBaseURL.trim() &&
      form.embedderAPIKey.trim() &&
      form.embedderModel.trim()
        ? {
            base_url: form.embedderBaseURL.trim(),
            api_key: form.embedderAPIKey.trim(),
            model: form.embedderModel.trim(),
            dims: form.embedderDims,
          }
        : undefined;
    const rerank =
      form.rerankBaseURL.trim() &&
      form.rerankAPIKey.trim() &&
      form.rerankModel.trim()
        ? {
            base_url: form.rerankBaseURL.trim(),
            api_key: form.rerankAPIKey.trim(),
            model: form.rerankModel.trim(),
          }
        : undefined;

    if (!chat && !embedder && !rerank) {
      Toast.warning(
        t(
'space_configure_models_empty',
          {},
          '请至少填写一组完整的模型配置（chat / embedder / rerank 三选一）',
        ),
      );
      return;
    }

    setLoading(true);
    try {
      const resp = await SpaceApi.configureModels({
        space_id: spaceId,
        chat,
        embedder,
        rerank,
      });
      if (resp.code !== 0) {
        throw new Error(getDisplayErrorMessage(resp.msg));
      }
      const d = resp.data;
      const summaryDetail = d
        ? ` (model_meta=${d.model_meta_updated} / space_embedding=${d.space_embedding_updated} / space_rerank=${d.space_rerank_updated} / redis_del=${d.redis_keys_deleted})`
        : '';
      Toast.success(
        t(
'space_configure_models_success',
          {},
          `配置已保存，缓存已清空，下次对话生效${summaryDetail}`,
        ),
      );
      if (d?.warnings?.length) {
        Toast.warning(d.warnings.join('; '));
      }
    } catch (e) {
      const msg =
        e instanceof Error ? getDisplayErrorMessage(e.message) : '未知错误';
      Toast.error(
        t('space_configure_models_failed', { msg }, `配置失败: ${msg}`),
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className={styles.card}>
      <div className={styles.cardTitle}>
        {t('space_configure_models_title', {}, '一键配置模型 + 同步')}
      </div>
      <div className={styles.cardDesc}>
        {t(
'space_configure_models_desc',
          {},
          '一次性写入 chat / embedder / rerank 三组模型配置并清空空间级模型缓存，下次对话立即生效。任意一组留空表示跳过。仅 space owner 可执行。',
        )}
      </div>

      <ChatFields form={form} setForm={setForm} />
      <EmbedderFields form={form} setForm={setForm} />
      <RerankFields form={form} setForm={setForm} />

      <Button
        color="primary"
        loading={loading}
        onClick={handleSubmit}
        data-testid="space-configure-models-button"
      >
        {t('space_configure_models_submit', {}, '保存并应用')}
      </Button>
    </div>
  );
};

export default ConfigureModelsCard;
