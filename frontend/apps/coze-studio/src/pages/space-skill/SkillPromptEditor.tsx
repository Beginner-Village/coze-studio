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

import React, { useState, useRef, useCallback, useEffect } from 'react';

import { Spin, Search } from '@coze-arch/coze-design';

import type { ResourceItem, ResourceGroup } from './hooks/use-space-resources';

/** 资源引用正则: {type:name|id:xxx} */
const RESOURCE_REF_REGEX =
  /\{(workflow|plugin|knowledge):([^|}]+)\|id:([^}]+)\}/g;

const TYPE_ICONS: Record<string, string> = {
  workflow: '⚡',
  plugin: '🔌',
  knowledge: '📚',
};

const TYPE_COLORS: Record<string, string> = {
  workflow: '#10b981',
  plugin: '#3b82f6',
  knowledge: '#f59e0b',
};

/** 资源标签的内联样式，匹配 LibraryBlockWidget 视觉效果 */
const TAG_STYLE =
  'display:inline-block;margin:0 2px;padding:1px 6px;color:#4E40E5;background-color:rgba(186,192,255,0.2);border-radius:4px;cursor:default;font-size:14px;line-height:22px;user-select:all;vertical-align:baseline;white-space:nowrap;';

interface SkillPromptEditorProps {
  value: string;
  onChange: (value: string) => void;
  resources: ResourceGroup[];
  resourcesLoading: boolean;
  onRequestResources: () => void;
  placeholder?: string;
}

interface PopoverPosition {
  top: number;
  left: number;
}

// ── HTML 序列化工具 ──

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

/** 将 value 字符串（含 {type:name|id:xxx}）转为编辑器 innerHTML */
function valueToHtml(value: string): string {
  if (!value) {
    return '';
  }

  const regex = new RegExp(RESOURCE_REF_REGEX.source, 'g');
  let html = '';
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  // eslint-disable-next-line no-cond-assign
  while ((match = regex.exec(value)) !== null) {
    const textBefore = value.substring(lastIndex, match.index);
    html += escapeHtml(textBefore).replace(/\n/g, '<br>');

    const [, type, name, id] = match;
    const icon = TYPE_ICONS[type] || '📦';
    const safeName = escapeHtml(name);

    html +=
      '<span class="skill-resource-tag" contenteditable="false"' +
      ` data-type="${type}" data-id="${id}" data-name="${safeName}"` +
      ` style="${TAG_STYLE}">${icon}&nbsp;${safeName}</span>`;

    lastIndex = match.index + match[0].length;
  }

  html += escapeHtml(value.substring(lastIndex)).replace(/\n/g, '<br>');
  return html;
}

/** 将编辑器 DOM 反序列化为 value 字符串 */
function editorToValue(root: HTMLDivElement): string {
  let result = '';

  function walk(node: Node, needNewline: boolean): void {
    if (node.nodeType === Node.TEXT_NODE) {
      result += node.textContent || '';
      return;
    }
    if (!(node instanceof HTMLElement)) {
      return;
    }

    if (node.classList.contains('skill-resource-tag')) {
      const { type, name, id } = node.dataset;
      if (type && name && id) {
        result += `{${type}:${name}|id:${id}}`;
      }
      return;
    }

    if (node.tagName === 'BR') {
      result += '\n';
      return;
    }

    // DIV 等块级元素（浏览器 contenteditable 换行默认用 div）
    if (node.tagName === 'DIV' && needNewline) {
      result += '\n';
    }

    let idx = 0;
    node.childNodes.forEach(child => {
      walk(child, idx > 0);
      idx++;
    });
  }

  let idx = 0;
  root.childNodes.forEach(child => {
    walk(child, idx > 0);
    idx++;
  });

  return result;
}

// ── 组件 ──

// eslint-disable-next-line @coze-arch/max-line-per-function
export const SkillPromptEditor: React.FC<SkillPromptEditorProps> = ({
  value,
  onChange,
  resources,
  resourcesLoading,
  onRequestResources,
  placeholder,
}) => {
  const editorRef = useRef<HTMLDivElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);
  const isInternalChange = useRef(false);
  const lastValueRef = useRef(value);
  // 关键：保存光标位置，防止点击弹窗后丢失
  const savedRangeRef = useRef<Range | null>(null);

  const [showPopover, setShowPopover] = useState(false);
  const [popoverPosition, setPopoverPosition] = useState<PopoverPosition>({
    top: 0,
    left: 0,
  });
  const [searchText, setSearchText] = useState('');
  const [isEmpty, setIsEmpty] = useState(!value);

  // ── 初始渲染 ──
  useEffect(() => {
    if (editorRef.current) {
      editorRef.current.innerHTML = valueToHtml(value);
      lastValueRef.current = value;
      setIsEmpty(!value);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // ── 外部 value 变更同步到 DOM（跳过内部触发的变更） ──
  useEffect(() => {
    if (isInternalChange.current) {
      isInternalChange.current = false;
      return;
    }
    if (editorRef.current && value !== lastValueRef.current) {
      editorRef.current.innerHTML = valueToHtml(value);
      lastValueRef.current = value;
      setIsEmpty(!value);
    }
  }, [value]);

  // ── DOM → value 同步 ──
  const syncValue = useCallback(() => {
    if (!editorRef.current) {
      return;
    }
    const newValue = editorToValue(editorRef.current);
    isInternalChange.current = true;
    lastValueRef.current = newValue;
    setIsEmpty(!newValue);
    onChange(newValue);
  }, [onChange]);

  const handleInput = useCallback(() => {
    syncValue();
  }, [syncValue]);

  // ── 保存当前光标位置 ──
  const saveSelection = useCallback(() => {
    const sel = window.getSelection();
    if (sel && sel.rangeCount > 0 && editorRef.current) {
      const range = sel.getRangeAt(0);
      // 确保 range 在编辑器内
      if (editorRef.current.contains(range.startContainer)) {
        savedRangeRef.current = range.cloneRange();
      }
    }
  }, []);

  // ── 恢复光标位置 ──
  const restoreSelection = useCallback(() => {
    if (!savedRangeRef.current || !editorRef.current) {
      return false;
    }
    const sel = window.getSelection();
    if (!sel) {
      return false;
    }

    try {
      editorRef.current.focus();
      sel.removeAllRanges();
      sel.addRange(savedRangeRef.current);
      return true;
    } catch {
      return false;
    }
  }, []);

  // ── 获取光标像素位置（用于定位弹窗） ──
  const getCursorPixelPosition = useCallback((): PopoverPosition => {
    const sel = window.getSelection();
    if (!sel || sel.rangeCount === 0) {
      return { top: 0, left: 0 };
    }

    const range = sel.getRangeAt(0).cloneRange();
    range.collapse(false);

    // 插入一个零宽标记来测量位置
    const marker = document.createElement('span');
    marker.textContent = '\u200B';
    range.insertNode(marker);

    const rect = marker.getBoundingClientRect();
    const pos = { top: rect.bottom + 4, left: rect.left };

    marker.parentNode?.removeChild(marker);

    // 恢复 selection
    sel.removeAllRanges();
    sel.addRange(range);

    return pos;
  }, []);

  // ── { 键触发资源选择器 ──
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLDivElement>) => {
      if (e.key === '{') {
        requestAnimationFrame(() => {
          const pos = getCursorPixelPosition();
          // 先保存光标位置（在弹窗打开前）
          saveSelection();
          setPopoverPosition(pos);
          setSearchText('');
          setShowPopover(true);
          onRequestResources();
        });
      } else if (e.key === 'Escape' && showPopover) {
        e.preventDefault();
        setShowPopover(false);
      }
    },
    [getCursorPixelPosition, onRequestResources, showPopover, saveSelection],
  );

  // ── 粘贴：只粘贴纯文本 ──
  const handlePaste = useCallback((e: React.ClipboardEvent<HTMLDivElement>) => {
    e.preventDefault();
    const text = e.clipboardData.getData('text/plain');
    document.execCommand('insertText', false, text);
  }, []);

  // ── 选择资源并插入可视标签 ──
  const handleSelectResource = useCallback(
    (item: ResourceItem) => {
      const editor = editorRef.current;
      if (!editor) {
        return;
      }

      // 恢复之前保存的光标位置
      const restored = restoreSelection();

      if (restored) {
        const sel = window.getSelection();
        if (sel && sel.rangeCount > 0) {
          const range = sel.getRangeAt(0);
          const node = range.startContainer;
          const offset = range.startOffset;

          // 删除触发的 '{' 字符
          if (node.nodeType === Node.TEXT_NODE && offset > 0) {
            const text = node.textContent || '';
            const braceIdx = text.lastIndexOf('{', offset - 1);
            if (braceIdx >= 0) {
              const deleteRange = document.createRange();
              deleteRange.setStart(node, braceIdx);
              deleteRange.setEnd(node, offset);
              deleteRange.deleteContents();
            }
          }

          // 创建可视标签
          const icon = TYPE_ICONS[item.type] || '📦';
          const tag = document.createElement('span');
          tag.className = 'skill-resource-tag';
          tag.contentEditable = 'false';
          tag.dataset.type = item.type;
          tag.dataset.id = item.id;
          tag.dataset.name = item.name;
          tag.style.cssText = TAG_STYLE;
          tag.textContent = `${icon}\u00A0${item.name}`;

          // 在光标处插入
          const insertRange = sel.getRangeAt(0);
          insertRange.collapse(false);
          insertRange.insertNode(tag);

          // 光标移到标签后，添加一个空格确保可以继续输入
          const space = document.createTextNode(' ');
          const after = document.createRange();
          after.setStartAfter(tag);
          after.collapse(true);
          after.insertNode(space);

          const cursor = document.createRange();
          cursor.setStartAfter(space);
          cursor.collapse(true);
          sel.removeAllRanges();
          sel.addRange(cursor);
        }
      } else {
        // 无法恢复光标，追加到编辑器末尾
        const icon = TYPE_ICONS[item.type] || '📦';
        const tag = document.createElement('span');
        tag.className = 'skill-resource-tag';
        tag.contentEditable = 'false';
        tag.dataset.type = item.type;
        tag.dataset.id = item.id;
        tag.dataset.name = item.name;
        tag.style.cssText = TAG_STYLE;
        tag.textContent = `${icon}\u00A0${item.name}`;
        editor.appendChild(tag);
        editor.appendChild(document.createTextNode(' '));
      }

      setShowPopover(false);
      savedRangeRef.current = null;
      syncValue();
      editor.focus();
    },
    [restoreSelection, syncValue],
  );

  // ── 点击外部关闭弹窗 ──
  useEffect(() => {
    if (!showPopover) {
      return;
    }

    const handleClickOutside = (e: MouseEvent) => {
      if (
        popoverRef.current &&
        !popoverRef.current.contains(e.target as Node) &&
        editorRef.current &&
        !editorRef.current.contains(e.target as Node)
      ) {
        setShowPopover(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [showPopover]);

  // ── 过滤资源 ──
  const filteredResources = resources
    .map(group => ({
      ...group,
      items: group.items.filter(
        item =>
          !searchText ||
          item.name.toLowerCase().includes(searchText.toLowerCase()) ||
          item.description.toLowerCase().includes(searchText.toLowerCase()),
      ),
    }))
    .filter(group => group.items.length > 0);

  return (
    <div className="relative">
      {/* 富文本编辑区域 - 匹配 Agent IDE Prompt 编辑器样式 */}
      <div
        ref={editorRef}
        contentEditable
        suppressContentEditableWarning
        onInput={handleInput}
        onKeyDown={handleKeyDown}
        onPaste={handlePaste}
        className="w-full border-[1px] border-solid rounded-[8px] coz-stroke-primary focus:outline-none whitespace-pre-wrap break-words coz-fg-primary overflow-y-auto"
        style={{
          background: 'var(--coz-bg-body, #fff)',
          minHeight: '280px',
          maxHeight: '400px',
          padding: '12px',
          fontSize: '14px',
          lineHeight: '22px',
          outline: 'none',
        }}
      />

      {/* 占位符 */}
      {isEmpty ? (
        <div
          className="absolute top-[12px] left-[12px] right-[12px] pointer-events-none whitespace-pre-wrap"
          style={{
            fontSize: '14px',
            lineHeight: '22px',
            color: 'var(--coz-fg-quaternary, #c0c1c3)',
          }}
        >
          {placeholder}
        </div>
      ) : null}

      {/* 资源选择弹窗 */}
      {showPopover ? (
        <div
          ref={popoverRef}
          className="fixed z-[1000] rounded-[8px] border-[1px] border-solid shadow-[0_8px_24px_rgba(0,0,0,0.12)] overflow-hidden"
          style={{
            top: `${popoverPosition.top}px`,
            left: `${popoverPosition.left}px`,
            width: '320px',
            maxHeight: '360px',
            background: 'var(--coz-bg-card, #fff)',
            borderColor: 'var(--coz-stroke-primary, #e5e6e8)',
          }}
        >
          {/* 搜索框 */}
          <div
            className="px-[8px] py-[8px] border-b border-solid"
            style={{ borderColor: 'var(--coz-stroke-secondary, #f0f1f2)' }}
          >
            <Search
              placeholder="搜索资源..."
              value={searchText}
              onChange={val => setSearchText(val as string)}
              size="small"
              showClear
              autoFocus
            />
          </div>

          {/* 资源列表 */}
          <div className="overflow-y-auto" style={{ maxHeight: '300px' }}>
            {resourcesLoading ? (
              <div className="flex justify-center items-center py-[24px]">
                <Spin size="small" />
                <span className="ml-[8px] text-[12px] coz-fg-tertiary">
                  加载中...
                </span>
              </div>
            ) : filteredResources.length === 0 ? (
              <div className="text-center py-[24px] text-[12px] coz-fg-tertiary">
                {searchText ? '未找到匹配的资源' : '暂无可用资源'}
              </div>
            ) : (
              filteredResources.map(group => (
                <div key={group.type}>
                  {/* 分组标题 */}
                  <div
                    className="px-[12px] py-[6px] text-[11px] font-medium coz-fg-tertiary"
                    style={{ background: 'var(--coz-bg-body, #fafafa)' }}
                  >
                    {group.label}
                  </div>
                  {/* 资源项 */}
                  {group.items.map(item => (
                    <div
                      key={`${item.type}-${item.id}`}
                      className="flex items-center gap-[8px] px-[12px] py-[8px] cursor-pointer transition-colors hover:bg-blue-50"
                      onClick={() => handleSelectResource(item)}
                    >
                      <div
                        className="w-[24px] h-[24px] flex items-center justify-center rounded-[4px] flex-shrink-0 text-[12px]"
                        style={{
                          background: `${TYPE_COLORS[item.type]}15`,
                          color: TYPE_COLORS[item.type],
                        }}
                      >
                        {TYPE_ICONS[item.type]}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="text-[13px] coz-fg-primary truncate">
                          {item.name}
                        </div>
                        {item.description ? (
                          <div className="text-[11px] coz-fg-tertiary truncate">
                            {item.description}
                          </div>
                        ) : null}
                      </div>
                      <span className="text-[11px] coz-fg-tertiary flex-shrink-0">
                        添加
                      </span>
                    </div>
                  ))}
                </div>
              ))
            )}
          </div>
        </div>
      ) : null}
    </div>
  );
};

export default SkillPromptEditor;
