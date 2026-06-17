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

/* 内联 SVG 图标(线性风格,currentColor),替代 emoji,做出专业 IDE 观感。 */
import { type CSSProperties } from 'react';

interface IconProps {
  size?: number;
  className?: string;
  style?: CSSProperties;
  color?: string;
}

const wrap = (paths: React.ReactNode, opts?: { fill?: boolean }) => {
  const Comp: React.FC<IconProps> = ({ size = 16, className, style, color }) => (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill={opts?.fill ? 'currentColor' : 'none'}
      stroke={opts?.fill ? 'none' : 'currentColor'}
      strokeWidth={1.8}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      style={{ color, flexShrink: 0, ...style }}
    >
      {paths}
    </svg>
  );
  return Comp;
};

export const IcFolder = wrap(
  <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z" />,
);
export const IcFolderOpen = wrap(
  <>
    <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v1" />
    <path d="M3 9h17l-2 8a2 2 0 0 1-2 1.6H5A2 2 0 0 1 3 16V9z" />
  </>,
);
export const IcFile = wrap(
  <>
    <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8l-5-5z" />
    <path d="M14 3v5h5" />
  </>,
);
export const IcImage = wrap(
  <>
    <rect x="3" y="4" width="18" height="16" rx="2" />
    <circle cx="8.5" cy="9" r="1.5" />
    <path d="M21 16l-5-5L5 20" />
  </>,
);
export const IcCode = wrap(
  <>
    <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8l-5-5z" />
    <path d="M14 3v5h5" />
    <path d="M9.5 12.5 8 14l1.5 1.5M14.5 12.5 16 14l-1.5 1.5" />
  </>,
);
export const IcPdf = wrap(
  <>
    <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8l-5-5z" />
    <path d="M14 3v5h5" />
    <path d="M8.5 17v-3.5h1a1 1 0 0 1 0 2h-1M13 17v-3.5h1.4M13 15.4h1.2M16.5 13.5v3.5" />
  </>,
);
export const IcTable = wrap(
  <>
    <rect x="3" y="4" width="18" height="16" rx="2" />
    <path d="M3 9h18M3 14h18M9 4v16M15 4v16" />
  </>,
);
export const IcText = wrap(
  <>
    <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8l-5-5z" />
    <path d="M14 3v5h5M8.5 13h7M8.5 16h5" />
  </>,
);
export const IcConfig = wrap(
  <>
    <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8l-5-5z" />
    <path d="M14 3v5h5" />
    <circle cx="12" cy="14" r="1.4" />
  </>,
);
export const IcDownload = wrap(
  <>
    <path d="M12 4v11" />
    <path d="M8 11l4 4 4-4" />
    <path d="M5 19h14" />
  </>,
);
export const IcUpload = wrap(
  <>
    <path d="M12 16V5" />
    <path d="M8 9l4-4 4 4" />
    <path d="M5 19h14" />
  </>,
);
export const IcRefresh = wrap(
  <>
    <path d="M20 11a8 8 0 1 0-2.3 5.7" />
    <path d="M20 5v6h-6" />
  </>,
);
export const IcTrash = wrap(
  <>
    <path d="M4 7h16" />
    <path d="M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
    <path d="M6 7l1 12a2 2 0 0 0 2 1.9h6a2 2 0 0 0 2-1.9L18 7" />
  </>,
);
export const IcSettings = wrap(
  <>
    <circle cx="12" cy="12" r="3" />
    <path d="M19.4 15a1.6 1.6 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.6 1.6 0 0 0-2.7 1.1V21a2 2 0 0 1-4 0v-.1a1.6 1.6 0 0 0-2.7-1.1l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1A1.6 1.6 0 0 0 4.6 15H4.5a2 2 0 0 1 0-4h.1a1.6 1.6 0 0 0 1.1-2.7l-.1-.1A2 2 0 1 1 8.4 5.4l.1.1A1.6 1.6 0 0 0 11 4.6V4.5a2 2 0 0 1 4 0v.1a1.6 1.6 0 0 0 2.7 1.1l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.6 1.6 0 0 0 1.1 2.7h.1a2 2 0 0 1 0 4h-.1a1.6 1.6 0 0 0-1.2 1z" />
  </>,
);
export const IcChevronRight = wrap(<path d="M9 6l6 6-6 6" />);
export const IcChevronDown = wrap(<path d="M6 9l6 6 6-6" />);
export const IcPlus = wrap(<path d="M12 5v14M5 12h14" />);
export const IcArchive = wrap(
  <>
    <rect x="3" y="4" width="18" height="4" rx="1" />
    <path d="M5 8v11a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V8M10 12h4" />
  </>,
);

// Hero 能力图标
export const IcPlan = wrap(
  <>
    <path d="M9 5H7a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2h-2" />
    <rect x="9" y="3" width="6" height="4" rx="1" />
    <path d="M8 12l2 2 4-4" />
  </>,
);
export const IcSandbox = wrap(
  <>
    <path d="M12 2 4 6v6c0 5 3.4 8.5 8 10 4.6-1.5 8-5 8-10V6l-8-4z" />
  </>,
);
export const IcTools = wrap(
  <path d="M14.7 6.3a4 4 0 0 0-5.1 5.1L4 17v3h3l5.6-5.6a4 4 0 0 0 5.1-5.1l-2.5 2.5-2-2 2.5-2.5z" />,
);
export const IcMemory = wrap(
  <>
    <ellipse cx="12" cy="6" rx="7" ry="3" />
    <path d="M5 6v6c0 1.7 3.1 3 7 3s7-1.3 7-3V6M5 12v6c0 1.7 3.1 3 7 3s7-1.3 7-3v-6" />
  </>,
);
export const IcLightning = wrap(<path d="M13 2 4 14h6l-1 8 9-12h-6l1-8z" />, {
  fill: true,
});

export const iconForFile = (name: string, isDir: boolean, open?: boolean) => {
  if (isDir) {
    return open ? IcFolderOpen : IcFolder;
  }
  const ext = name.split('.').pop()?.toLowerCase() ?? '';
  if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'ico'].includes(ext)) {
    return IcImage;
  }
  if (ext === 'pdf') {
    return IcPdf;
  }
  if (['xlsx', 'xls', 'csv'].includes(ext)) {
    return IcTable;
  }
  if (
    ['py', 'js', 'ts', 'tsx', 'jsx', 'go', 'java', 'c', 'cpp', 'rs', 'sh', 'rb', 'php', 'vue', 'css'].includes(ext)
  ) {
    return IcCode;
  }
  if (['json', 'yaml', 'yml', 'toml', 'xml'].includes(ext)) {
    return IcConfig;
  }
  if (['zip', 'tar', 'gz', 'rar', '7z'].includes(ext)) {
    return IcArchive;
  }
  if (['md', 'txt', 'log', 'docx', 'doc'].includes(ext)) {
    return IcText;
  }
  return IcFile;
};
