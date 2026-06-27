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

import { describe, expect, it } from 'vitest';
import { ProjectResourceActionKey } from '@coze-arch/bot-api/plugin_develop';

import {
  BizResourceTypeEnum,
  type BizResourceType,
} from '../src/resource-folder-coze/type';
import {
  createFolderedResourceTree,
  getBackendFolderId,
  getResourceFolderId,
  ROOT_FOLDER_ID,
  RESOURCE_FOLDER_TYPE,
} from '../src/resource-folder-coze/folder-tree';

const workflow = (id: string, name: string): BizResourceType => ({
  id,
  name,
  type: BizResourceTypeEnum.Workflow,
  res_id: id,
});

describe('createFolderedResourceTree', () => {
  it('places category folders above unclassified workflows', () => {
    const tree = createFolderedResourceTree({
      folders: [
        {
          id: '20',
          space_id: '1',
          name: 'Beta',
          description: '',
          creator_id: '1',
          created_at: 0,
          updated_at: 0,
          resource_ids: ['2'],
        },
        {
          id: '10',
          space_id: '1',
          name: 'Alpha',
          description: '',
          creator_id: '1',
          created_at: 0,
          updated_at: 0,
          resource_ids: ['1'],
        },
      ],
      resources: [
        workflow('3', 'Unclassified B'),
        workflow('1', 'Classified A'),
        workflow('2', 'Classified B'),
        workflow('4', 'Unclassified A'),
      ],
    });

    expect(tree.map(item => item.name)).toEqual([
      'Alpha',
      'Beta',
      'Unclassified A',
      'Unclassified B',
    ]);
    expect(tree[0]).toMatchObject({
      id: getResourceFolderId('10'),
      type: RESOURCE_FOLDER_TYPE,
      name: 'Alpha',
      folder_id: '10',
    });
    expect(tree[0].children?.map(item => item.id)).toEqual(['1']);
    expect(tree[0].actions).toEqual([
      { key: ProjectResourceActionKey.Rename, enable: true },
      { key: ProjectResourceActionKey.Delete, enable: true },
    ]);
    expect(tree[2].id).toBe('4');
    expect(tree[3].id).toBe('3');
  });

  it('maps tree folder ids back to backend folder ids', () => {
    expect(getBackendFolderId(getResourceFolderId('10'))).toBe('10');
    expect(getBackendFolderId('$-ROOT-$')).toBe(ROOT_FOLDER_ID);
  });
});
