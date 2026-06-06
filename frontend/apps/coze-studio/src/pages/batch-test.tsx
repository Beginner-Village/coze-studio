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

// The standalone "猎鹰批量测试" wizard has been merged into the observability
// experiments flow (create + detail). This module now only redirects any stale
// links to the experiment-create page so bookmarks keep working.
import { Navigate, useParams } from 'react-router-dom';
import React from 'react';

const BatchTestRedirect: React.FC = () => {
  const { space_id: spaceId } = useParams<{ space_id: string }>();
  return (
    <Navigate
      to={`/space/${spaceId}/observability?tab=experiments&action=create`}
      replace
    />
  );
};

export { BatchTestRedirect as Component };
export default BatchTestRedirect;
