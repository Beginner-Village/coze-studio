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

import { useNavigate } from 'react-router-dom';
import { useCallback, useEffect, useState } from 'react';

import { useBotInfoStore } from '@coze-studio/bot-detail-store/bot-info';
import { Toast } from '@coze-arch/coze-design';
import { axiosInstance } from '@coze-arch/bot-api';

// ─── API helpers ──────────────────────────────────────────────────────────────

interface MarketplaceProduct {
  product_id: string;
  name: string;
  description?: string;
  icon_uri?: string;
}

interface ProductListResult {
  code: number;
  msg?: string;
  data?: { products?: MarketplaceProduct[] };
}

interface RecruitResult {
  code: number;
  msg?: string;
  data?: { shadow_agent_id: string };
}

const postApi = <T,>(url: string, data: Record<string, unknown>): Promise<T> =>
  axiosInstance.request({
    url,
    method: 'POST',
    data,
    withCredentials: true,
  }) as unknown as Promise<T>;

// ─── Avatar ───────────────────────────────────────────────────────────────────

const AVATAR_SIZE = 32;
const HASH_PRIME = 31;
const DIMMED_OPACITY = 0.6;

const AVATAR_COLORS = [
  '#2E73FF',
  '#7C3AED',
  '#0EA5A4',
  '#F97316',
  '#DB2777',
  '#0891B2',
  '#65A30D',
  '#E11D48',
];

const pickColor = (seed: string): string => {
  let hash = 0;
  for (let i = 0; i < seed.length; i++) {
    hash = (hash * HASH_PRIME + seed.charCodeAt(i)) | 0;
  }
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
};

const EmployeeAvatar: React.FC<{ product: MarketplaceProduct }> = ({
  product,
}) => {
  if (product.icon_uri) {
    return (
      <img
        src={product.icon_uri}
        alt={product.name}
        style={{
          width: AVATAR_SIZE,
          height: AVATAR_SIZE,
          borderRadius: 8,
          objectFit: 'cover',
          flex: '0 0 auto',
        }}
      />
    );
  }
  const initial = (product.name || '?').trim().charAt(0).toUpperCase() || '?';
  return (
    <div
      style={{
        width: AVATAR_SIZE,
        height: AVATAR_SIZE,
        borderRadius: 8,
        flex: '0 0 auto',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: '#fff',
        fontSize: 14,
        fontWeight: 600,
        background: pickColor(product.product_id || product.name),
      }}
    >
      {initial}
    </div>
  );
};

// ─── Row ────────────────────────────────────────────────────────────────────────

interface EmployeeRowProps {
  product: MarketplaceProduct;
  active: boolean;
  recruiting: boolean;
  disabled: boolean;
  onSelect: (product: MarketplaceProduct) => void;
}

const EmployeeRow: React.FC<EmployeeRowProps> = ({
  product,
  active,
  recruiting,
  disabled,
  onSelect,
}) => (
  <div
    role="button"
    tabIndex={0}
    onClick={() => onSelect(product)}
    onKeyDown={e => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        onSelect(product);
      }
    }}
    style={{
      display: 'flex',
      alignItems: 'center',
      gap: 10,
      padding: '10px 10px',
      marginBottom: 4,
      borderRadius: 8,
      cursor: disabled ? 'default' : 'pointer',
      opacity: disabled && !recruiting ? DIMMED_OPACITY : 1,
      background: active
        ? 'var(--coz-mg-hglt, rgba(46, 115, 255, 0.08))'
        : 'transparent',
      border: active
        ? '1px solid var(--coz-stroke-hglt, rgba(46, 115, 255, 0.35))'
        : '1px solid transparent',
      transition: 'background 0.15s ease',
    }}
    onMouseEnter={e => {
      if (!active && !disabled) {
        e.currentTarget.style.background =
          'var(--coz-mg-secondary, rgba(82, 100, 154, 0.06))';
      }
    }}
    onMouseLeave={e => {
      if (!active) {
        e.currentTarget.style.background = 'transparent';
      }
    }}
  >
    <EmployeeAvatar product={product} />
    <div style={{ flex: 1, minWidth: 0 }}>
      <div
        style={{
          fontSize: 14,
          fontWeight: 550,
          lineHeight: '18px',
          overflow: 'hidden',
          whiteSpace: 'nowrap',
          textOverflow: 'ellipsis',
          color: 'var(--coz-fg-plus, rgba(15, 21, 40, 0.9))',
        }}
      >
        {product.name || '未命名员工'}
      </div>
      {product.description ? (
        <div
          style={{
            marginTop: 2,
            fontSize: 12,
            lineHeight: '16px',
            overflow: 'hidden',
            whiteSpace: 'nowrap',
            textOverflow: 'ellipsis',
            color: 'var(--coz-fg-secondary, rgba(32, 41, 69, 0.62))',
          }}
        >
          {product.description}
        </div>
      ) : null}
    </div>
    {recruiting ? (
      <span
        style={{
          flex: '0 0 auto',
          fontSize: 12,
          color: 'var(--coz-fg-secondary, rgba(32, 41, 69, 0.62))',
        }}
      >
        进入中…
      </span>
    ) : null}
  </div>
);

// ─── Placeholder ─────────────────────────────────────────────────────────────────

const placeholderStyle: React.CSSProperties = {
  padding: '24px 12px',
  fontSize: 13,
  textAlign: 'center',
  color: 'var(--coz-fg-secondary, rgba(32, 41, 69, 0.62))',
};

const RosterHeader: React.FC = () => (
  <div
    style={{
      flex: '0 0 auto',
      padding: '14px 16px 12px',
      borderBottom:
        '1px solid var(--coz-stroke-primary, rgba(82, 100, 154, 0.12))',
    }}
  >
    <div
      style={{
        fontSize: 15,
        fontWeight: 650,
        lineHeight: '20px',
        color: 'var(--coz-fg-plus, rgba(15, 21, 40, 0.9))',
      }}
    >
      我的员工
    </div>
    <div
      style={{
        marginTop: 3,
        fontSize: 12,
        lineHeight: '16px',
        color: 'var(--coz-fg-secondary, rgba(32, 41, 69, 0.62))',
      }}
    >
      选择一位虚拟员工开始对话
    </div>
  </div>
);

interface RosterBodyProps {
  loadState: LoadState;
  products: MarketplaceProduct[];
  currentBotId?: string;
  recruitingId: string | null;
  onReload: () => void;
  onSelect: (product: MarketplaceProduct) => void;
}

const RosterBody: React.FC<RosterBodyProps> = ({
  loadState,
  products,
  currentBotId,
  recruitingId,
  onReload,
  onSelect,
}) => (
  <div style={{ flex: 1, minHeight: 0, overflowY: 'auto', padding: 8 }}>
    {loadState === 'loading' ? (
      <div style={placeholderStyle}>加载中…</div>
    ) : null}

    {loadState === 'error' ? (
      <div style={placeholderStyle}>
        <div>加载失败</div>
        <button
          type="button"
          onClick={onReload}
          style={{
            marginTop: 8,
            padding: '4px 12px',
            fontSize: 13,
            color: 'var(--coz-fg-hglt, #2E73FF)',
            background: 'transparent',
            border: 'none',
            cursor: 'pointer',
          }}
        >
          重试
        </button>
      </div>
    ) : null}

    {loadState === 'ready' && products.length === 0 ? (
      <div style={placeholderStyle}>暂无可用员工</div>
    ) : null}

    {loadState === 'ready'
      ? products.map(product => (
          <EmployeeRow
            key={product.product_id}
            product={product}
            active={!!currentBotId && currentBotId === product.product_id}
            recruiting={recruitingId === product.product_id}
            disabled={!!recruitingId}
            onSelect={onSelect}
          />
        ))
      : null}
  </div>
);

// ─── Component ──────────────────────────────────────────────────────────────────

export interface EmployeeRosterProps {
  /** 当前正在对话的 bot（shadow agent）id，用于高亮。 */
  currentBotId?: string;
}

type LoadState = 'loading' | 'ready' | 'error';

export const EmployeeRoster: React.FC<EmployeeRosterProps> = ({
  currentBotId,
}) => {
  const spaceId = useBotInfoStore(
    (state: { space_id: string }) => state.space_id,
  );
  const navigate = useNavigate();

  const [products, setProducts] = useState<MarketplaceProduct[]>([]);
  const [loadState, setLoadState] = useState<LoadState>('loading');
  const [recruitingId, setRecruitingId] = useState<string | null>(null);

  const loadProducts = useCallback(async () => {
    setLoadState('loading');
    try {
      const res = await postApi<ProductListResult>(
        '/api/super-agent/marketplace/products/list',
        {
          type: 'agent_app',
          page: 1,
          page_size: 100,
          ...(spaceId ? { space_id: spaceId } : {}),
        },
      );
      if (res.code !== 0) {
        setLoadState('error');
        return;
      }
      setProducts(res.data?.products ?? []);
      setLoadState('ready');
    } catch (err: unknown) {
      console.warn('[employee-roster] list error:', err);
      setLoadState('error');
    }
  }, [spaceId]);

  useEffect(() => {
    void loadProducts();
  }, [loadProducts]);

  const handleSelect = useCallback(
    async (product: MarketplaceProduct) => {
      if (recruitingId) {
        return;
      }
      if (!spaceId) {
        Toast.error('空间信息未就绪，请稍后重试');
        return;
      }
      setRecruitingId(product.product_id);
      try {
        const res = await postApi<RecruitResult>(
          '/api/super-agent/agent-app/recruit',
          { product_id: product.product_id, space_id: spaceId },
        );
        if (res.code !== 0 || !res.data?.shadow_agent_id) {
          Toast.error(res.msg || '进入员工失败，请重试');
          return;
        }
        const shadowId = res.data.shadow_agent_id;
        navigate(
          `/space/${spaceId}/bot/${shadowId}/arrange?employeeChat=1&name=${encodeURIComponent(
            product.name,
          )}`,
        );
      } catch (err: unknown) {
        console.warn('[employee-roster] recruit error:', err);
        Toast.error('进入员工失败，请重试');
      } finally {
        setRecruitingId(null);
      }
    },
    [navigate, recruitingId, spaceId],
  );

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        minWidth: 0,
        height: '100%',
        overflow: 'hidden',
        background: 'var(--coz-bg-max, #fff)',
        border: '1px solid var(--coz-stroke-primary, rgba(82, 100, 154, 0.13))',
        borderRadius: 8,
      }}
    >
      <RosterHeader />
      <RosterBody
        loadState={loadState}
        products={products}
        currentBotId={currentBotId}
        recruitingId={recruitingId}
        onReload={() => void loadProducts()}
        onSelect={handleSelect}
      />
    </div>
  );
};
