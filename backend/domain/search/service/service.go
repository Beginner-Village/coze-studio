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

package service

import (
	"context"

	space "github.com/ynet-dev/ynet-studio/backend/api/model/data/space"
	"github.com/ynet-dev/ynet-studio/backend/domain/search/entity"
)

type ProjectEventBus interface {
	PublishProject(ctx context.Context, event *entity.ProjectDomainEvent) error
}

type ResourceEventBus interface {
	PublishResources(ctx context.Context, event *entity.ResourceDomainEvent) error
}

type Search interface {
	SearchProjects(ctx context.Context, req *entity.SearchProjectsRequest) (resp *entity.SearchProjectsResponse, err error)
	SearchResources(ctx context.Context, req *entity.SearchResourcesRequest) (resp *entity.SearchResourcesResponse, err error)
	// ResyncSpace clears and rebuilds the per-space ES list indices
	// (project_draft / coze_resource / kb_entries). Knowledge chunk indices
	// (openynet_<kb_id>) are handled separately by the knowledge domain.
	// Resync deps must have been wired in via SetResyncDeps; otherwise
	// returns an error.
	ResyncSpace(ctx context.Context, spaceID int64) (*space.ResyncESCounts, error)
	// SetResyncDeps wires in the per-space resync dependencies. Called by
	// the application layer once during init after the relevant domain
	// repos exist. Must be invoked before ResyncSpace can run.
	SetResyncDeps(agentRepo AgentLister, appRepo AppLister, kbRepo KbLister)
}
