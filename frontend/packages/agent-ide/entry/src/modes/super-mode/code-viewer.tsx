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

/* 代码语法高亮:用 highlight.js 的隔离核心实例(无全局副作用),按需注册语言。
 * 完全打包进项目产物,运行时不请求任何外网资源。 */
import { useMemo } from 'react';

import hljs from 'highlight.js/lib/core';
import javascript from 'highlight.js/lib/languages/javascript';
import typescript from 'highlight.js/lib/languages/typescript';
import python from 'highlight.js/lib/languages/python';
import go from 'highlight.js/lib/languages/go';
import java from 'highlight.js/lib/languages/java';
import kotlin from 'highlight.js/lib/languages/kotlin';
import c from 'highlight.js/lib/languages/c';
import cpp from 'highlight.js/lib/languages/cpp';
import csharp from 'highlight.js/lib/languages/csharp';
import rust from 'highlight.js/lib/languages/rust';
import ruby from 'highlight.js/lib/languages/ruby';
import php from 'highlight.js/lib/languages/php';
import swift from 'highlight.js/lib/languages/swift';
import bash from 'highlight.js/lib/languages/bash';
import sql from 'highlight.js/lib/languages/sql';
import css from 'highlight.js/lib/languages/css';
import less from 'highlight.js/lib/languages/less';
import scss from 'highlight.js/lib/languages/scss';
import json from 'highlight.js/lib/languages/json';
import yaml from 'highlight.js/lib/languages/yaml';
import ini from 'highlight.js/lib/languages/ini';
import xml from 'highlight.js/lib/languages/xml';
import dockerfile from 'highlight.js/lib/languages/dockerfile';
import makefile from 'highlight.js/lib/languages/makefile';

import './code-viewer.css';

let registered = false;
const ensureRegistered = () => {
  if (registered) {
    return;
  }
  registered = true;
  const langs: Record<string, Parameters<typeof hljs.registerLanguage>[1]> = {
    javascript,
    typescript,
    python,
    go,
    java,
    kotlin,
    c,
    cpp,
    csharp,
    rust,
    ruby,
    php,
    swift,
    bash,
    sql,
    css,
    less,
    scss,
    json,
    yaml,
    ini,
    xml,
    dockerfile,
    makefile,
  };
  Object.entries(langs).forEach(([name, def]) => hljs.registerLanguage(name, def));
};

// 业务语言别名 → highlight.js 语言 id
const LANG_ALIAS: Record<string, string> = {
  tsx: 'typescript',
  jsx: 'javascript',
  html: 'xml',
  vue: 'xml',
  toml: 'ini',
  kt: 'kotlin',
};

const escapeHtml = (s: string): string => {
  const div = document.createElement('div');
  div.textContent = s;
  return div.innerHTML;
};

export const CodeViewer: React.FC<{
  content: string;
  lang?: string;
  withLineNumbers?: boolean;
}> = ({ content, lang, withLineNumbers }) => {
  ensureRegistered();
  const resolved = lang ? (LANG_ALIAS[lang] ?? lang) : '';

  const html = useMemo(() => {
    if (resolved && hljs.getLanguage(resolved)) {
      try {
        return hljs.highlight(content, { language: resolved }).value;
      } catch {
        /* fall back to plain */
      }
    }
    return escapeHtml(content);
  }, [content, resolved]);

  const lineCount = content.split('\n').length;

  return (
    <div className="coz-codeview flex text-[12.5px] font-mono leading-[1.7]">
      {withLineNumbers ? (
        <div className="coz-codeview-gutter shrink-0 select-none text-right px-[10px] py-[12px] coz-fg-dim">
          {Array.from({ length: lineCount }, (_, i) => (
            <div key={i}>{i + 1}</div>
          ))}
        </div>
      ) : null}
      <pre className="flex-1 min-w-0 overflow-auto py-[12px] px-[14px] m-0">
        <code
          className="hljs"
          // eslint-disable-next-line @typescript-eslint/naming-convention
          dangerouslySetInnerHTML={{ __html: html }}
        />
      </pre>
    </div>
  );
};
