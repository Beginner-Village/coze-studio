const CONTENT_TYPE_TEXT = 'text';
const CONTENT_TYPE_IMAGE = 'image';
const CONTENT_TYPE_FILE = 'file';
const CONTENT_TYPE_MIX = 'mix';

const isRecord = (v: unknown): v is Record<string, unknown> =>
  !!v && typeof v === 'object' && !Array.isArray(v);

const stringifyBrief = (v: unknown, max = 6000): string => {
  try {
    const s = JSON.stringify(v);
    return s.length > max ? `${s.slice(0, max)}...` : s;
  } catch {
    return '';
  }
};

const parseJsonString = (content: string): unknown => {
  try {
    return JSON.parse(content);
  } catch {
    return undefined;
  }
};

const extractTextFromContentObject = (content: unknown): string => {
  if (typeof content === 'string') {
    return content;
  }

  if (!isRecord(content)) {
    return '';
  }

  if (typeof content.text === 'string') {
    return content.text;
  }

  if (Array.isArray(content.item_list)) {
    return content.item_list
      .map(item =>
        isRecord(item) &&
        item.type === CONTENT_TYPE_TEXT &&
        typeof item.text === 'string'
          ? item.text
          : '',
      )
      .filter(Boolean)
      .join('\n');
  }

  return '';
};

export const getWorkflowAgentMessageText = (content: unknown): string => {
  if (typeof content === 'string') {
    const parsed = parseJsonString(content);
    return parsed === undefined
      ? content
      : extractTextFromContentObject(parsed);
  }

  const text = extractTextFromContentObject(content);
  return text || stringifyBrief(content, 1200);
};

const serializeContent = (content: unknown): string =>
  typeof content === 'string' ? content : stringifyBrief(content);

export const buildWorkflowAgentRequestQuery = ({
  content,
  contentType,
  query,
}: {
  content: unknown;
  contentType?: string;
  query: string;
}): string => {
  const serialized = serializeContent(content);

  if (contentType === CONTENT_TYPE_MIX) {
    const parsed =
      typeof content === 'string' ? parseJsonString(content) : content;
    if (!isRecord(parsed) || !Array.isArray(parsed.item_list)) {
      return serialized;
    }

    if (!query) {
      return JSON.stringify(parsed);
    }

    const itemList = parsed.item_list.map(item =>
      isRecord(item) ? { ...item } : item,
    );
    const textIndex = itemList.findIndex(
      item => isRecord(item) && item.type === CONTENT_TYPE_TEXT,
    );

    if (textIndex >= 0) {
      itemList[textIndex] = {
        ...(isRecord(itemList[textIndex]) ? itemList[textIndex] : {}),
        type: CONTENT_TYPE_TEXT,
        text: query,
      };
    } else {
      itemList.unshift({ type: CONTENT_TYPE_TEXT, text: query });
    }

    return JSON.stringify({
      ...parsed,
      item_list: itemList,
    });
  }

  if (contentType === CONTENT_TYPE_IMAGE || contentType === CONTENT_TYPE_FILE) {
    return serialized;
  }

  return query;
};
