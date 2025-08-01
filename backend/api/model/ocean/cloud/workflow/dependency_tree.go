/*
 * Copyright 2025 coze-dev Authors
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

package workflow

import (
	"github.com/coze-dev/coze-studio/backend/api/model/base"
)

// WorkflowStorageType represents the storage type of workflow
type WorkflowStorageType string

const (
	WorkflowStorageTypeLibrary WorkflowStorageType = "Library"
	WorkflowStorageTypeProject WorkflowStorageType = "Project"
)

// LibraryWorkflowInfo represents library workflow information
type LibraryWorkflowInfo struct {
	ID      *string `json:"id,omitempty"`
	Version *string `json:"version,omitempty"`
}

// ProjectWorkflowInfo represents project workflow information  
type ProjectWorkflowInfo struct {
	WorkflowID *string `json:"workflow_id,omitempty"`
	ProjectID  *string `json:"project_id,omitempty"`
}

// DependencyTreeRequest represents the request structure for dependency tree
type DependencyTreeRequest struct {
	Type        WorkflowStorageType  `json:"type"`
	LibraryInfo *LibraryWorkflowInfo `json:"library_info,omitempty"`
	ProjectInfo *ProjectWorkflowInfo `json:"project_info,omitempty"`
	Base        *base.Base           `json:"Base,omitempty"`
}

// DependencyTreeResponse represents the response structure for dependency tree
type DependencyTreeResponse struct {
	Data     *DependencyTree `json:"data,omitempty"`
	Code     int64           `json:"code"`
	Msg      string          `json:"msg"`
	BaseResp *base.BaseResp  `json:"BaseResp,omitempty"`
}

// DependencyTree represents the dependency tree structure
type DependencyTree struct {
	RootId   *string                  `json:"root_id,omitempty"`
	Version  *string                  `json:"version,omitempty"`
	NodeList []*DependencyTreeNode    `json:"node_list,omitempty"`
	EdgeList []*DependencyTreeEdge    `json:"edge_list,omitempty"`
}

// DependencyTreeNode represents a node in the dependency tree
type DependencyTreeNode struct {
	Name            *string     `json:"name,omitempty"`
	ID              *string     `json:"id,omitempty"`
	Icon            *string     `json:"icon,omitempty"`
	IsProduct       *bool       `json:"is_product,omitempty"`
	IsRoot          *bool       `json:"is_root,omitempty"`
	IsLibrary       *bool       `json:"is_library,omitempty"`
	WithVersion     *bool       `json:"with_version,omitempty"`
	WorkflowVersion *string     `json:"workflow_version,omitempty"`
	Dependency      *Dependency `json:"dependency,omitempty"`
	CommitID        *string     `json:"commit_id,omitempty"`
}

// DependencyTreeEdge represents an edge in the dependency tree
type DependencyTreeEdge struct {
	From         *string `json:"from,omitempty"`
	FromVersion  *string `json:"from_version,omitempty"`
	FromCommitID *string `json:"from_commit_id,omitempty"`
	To           *string `json:"to,omitempty"`
	ToVersion    *string `json:"to_version,omitempty"`
}

// Dependency represents dependency information
type Dependency struct {
	Type    *string `json:"type,omitempty"`
	ID      *string `json:"id,omitempty"`
	Version *string `json:"version,omitempty"`
}