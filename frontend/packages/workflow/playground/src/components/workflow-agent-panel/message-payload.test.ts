import { describe, expect, it } from 'vitest';

import {
  buildWorkflowAgentRequestQuery,
  getWorkflowAgentMessageText,
} from './message-payload';

const ContentType = {
  Text: 'text',
  Image: 'image',
  Mix: 'mix',
};

describe('workflow-agent-panel message payload', () => {
  it('extracts user text from mixed image messages', () => {
    const content = JSON.stringify({
      item_list: [
        {
          type: ContentType.Image,
          image: {
            key: 'image-uri',
            image_thumb: { url: 'https://example.com/thumb.png' },
            image_ori: { url: 'https://example.com/origin.png' },
          },
        },
        {
          type: ContentType.Text,
          text: '根据这张截图生成卡片。',
        },
      ],
    });

    expect(getWorkflowAgentMessageText(content)).toBe(
      '根据这张截图生成卡片。',
    );
  });

  it('preserves image items when workflow context rewrites mixed query text', () => {
    const content = JSON.stringify({
      item_list: [
        {
          type: ContentType.Image,
          image: {
            key: 'image-uri',
            image_thumb: { url: 'https://example.com/thumb.png' },
            image_ori: { url: 'https://example.com/origin.png' },
          },
        },
        {
          type: ContentType.Text,
          text: 'old text',
        },
      ],
    });

    const query = buildWorkflowAgentRequestQuery({
      content,
      contentType: ContentType.Mix,
      query: 'new text',
    });
    const parsed = JSON.parse(query);

    expect(parsed.item_list).toHaveLength(2);
    expect(parsed.item_list[0].type).toBe(ContentType.Image);
    expect(parsed.item_list[0].image.key).toBe('image-uri');
    expect(parsed.item_list[1]).toEqual({
      type: ContentType.Text,
      text: 'new text',
    });
  });

  it('does not replace image-only messages with text', () => {
    const content = JSON.stringify({
      image_list: [
        {
          key: 'image-uri',
          image_thumb: { url: 'https://example.com/thumb.png' },
          image_ori: { url: 'https://example.com/origin.png' },
        },
      ],
    });

    expect(
      buildWorkflowAgentRequestQuery({
        content,
        contentType: ContentType.Image,
        query: 'new text',
      }),
    ).toBe(content);
  });
});
