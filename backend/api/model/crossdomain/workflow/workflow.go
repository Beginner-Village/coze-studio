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

package workflow

import (
	"encoding/json"
	"sync"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/api/model/workflow"
	"github.com/ynet-dev/ynet-studio/backend/domain/workflow/crossdomain/conversation"
)

func init() {
	// Register HiAgentConversationInfo for Eino checkpoint serialization
	// This is required because ExecuteConfig contains HiAgentConversations map[string]*HiAgentConversationInfo
	// which needs to be serialized when using Eino's checkpoint feature
	_ = compose.RegisterSerializableType[HiAgentConversationInfo]("workflow.HiAgentConversationInfo")
	// HiAgentConvStore is referenced by ExecuteConfig via pointer; register it so it
	// round-trips through Eino's checkpoint serialization. It implements
	// json.Marshaler/Unmarshaler to serialize as a plain map[string]*HiAgentConversationInfo.
	_ = compose.RegisterSerializableType[HiAgentConvStore]("workflow.HiAgentConvStore")
}

type Locator uint8

const (
	FromDraft Locator = iota
	FromSpecificVersion
	FromLatestVersion
)

type ExecuteConfig struct {
	ID                                int64
	From                              Locator
	Version                           string
	CommitID                          string
	Operator                          int64
	Mode                              ExecuteMode
	AppID                             *int64
	AgentID                           *int64
	ConnectorID                       int64
	ConnectorUID                      string
	TaskType                          TaskType
	SyncPattern                       SyncPattern
	InputFailFast                     bool // whether to fail fast if input conversion has warnings
	BizType                           BizType
	Cancellable                       bool
	WorkflowMode                      WorkflowMode
	RoundID                           *int64 // if workflow is chat flow, conversation round id is required
	InitRoundID                       *int64 // if workflow is chat flow, init conversation round id is required
	ConversationID                    *int64 // if workflow is chat flow, conversation id is required
	UserMessage                       *schema.Message
	ConversationHistory               []*conversation.Message
	ConversationHistorySchemaMessages []*schema.Message
	SectionID                         *int64
	MaxHistoryRounds                  *int32

	// CustomVariables 会话级自定义变量，用于覆盖智能体/工作流预设变量
	// 在发起会话时通过 custom_variables 参数传入，工作流变量节点读取时优先使用此值
	CustomVariables map[string]string

	// HiAgent conversation mapping: map[agentID]HiAgentConversationInfo
	// Used to maintain HiAgent conversation state across multiple calls in the same ChatFlow session.
	//
	// The mapping (and its lock) live behind a pointer in HiAgentConvStore so that
	// ExecuteConfig itself contains no lock and stays safe to copy by value
	// (it is passed by value throughout the workflow execution chain).
	HiAgentConversations *HiAgentConvStore
}

type ExecuteMode string

const (
	ExecuteModeDebug     ExecuteMode = "debug"
	ExecuteModeRelease   ExecuteMode = "release"
	ExecuteModeNodeDebug ExecuteMode = "node_debug"
)

type WorkflowMode = workflow.WorkflowMode

type TaskType string

const (
	TaskTypeForeground TaskType = "foreground"
	TaskTypeBackground TaskType = "background"
)

type SyncPattern string

const (
	SyncPatternSync   SyncPattern = "sync"
	SyncPatternAsync  SyncPattern = "async"
	SyncPatternStream SyncPattern = "stream"
)

var DebugURLTpl = "http://127.0.0.1:3000/work_flow?execute_id=%d&space_id=%d&workflow_id=%d&execute_mode=2"

type BizType string

const (
	BizTypeAgent    BizType = "agent"
	BizTypeWorkflow BizType = "workflow"
)

// HiAgentConversationInfo stores HiAgent conversation state
type HiAgentConversationInfo struct {
	AppConversationID string `json:"app_conversation_id"`
	LastSectionID     int64  `json:"last_section_id"`
}

// HiAgentConvStore holds the HiAgent conversation mapping together with its lock.
//
// It is always used through a pointer (*HiAgentConvStore), so embedding the
// sync.RWMutex by value here is safe — the value is never copied. Keeping the
// lock in this dedicated struct keeps ExecuteConfig lock-free and copy-safe.
type HiAgentConvStore struct {
	mu   sync.RWMutex
	data map[string]*HiAgentConversationInfo
}

// MarshalJSON serializes the store as a plain map so it round-trips through
// Eino's checkpoint serialization identically to the previous map field.
func (s *HiAgentConvStore) MarshalJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.data == nil {
		return []byte("null"), nil
	}
	return json.Marshal(s.data)
}

// UnmarshalJSON restores the store from the plain-map wire format.
func (s *HiAgentConvStore) UnmarshalJSON(b []byte) error {
	var data map[string]*HiAgentConversationInfo
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = data
	return nil
}

// Get retrieves the full HiAgent conversation info for a specific agent.
func (s *HiAgentConvStore) Get(agentID string) *HiAgentConversationInfo {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.data == nil {
		return nil
	}
	return s.data[agentID]
}

// Set stores the full HiAgent conversation info for a specific agent.
func (s *HiAgentConvStore) Set(agentID string, info *HiAgentConversationInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = make(map[string]*HiAgentConversationInfo)
	}
	s.data[agentID] = info
}

// Delete removes the HiAgent conversation info for a specific agent.
func (s *HiAgentConvStore) Delete(agentID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data != nil {
		delete(s.data, agentID)
	}
}

// Reset clears all HiAgent conversation mappings.
func (s *HiAgentConvStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]*HiAgentConversationInfo)
}

// Snapshot returns a shallow copy of the underlying mapping for read-only use
// (e.g. logging) without exposing the internal map or lock.
func (s *HiAgentConvStore) Snapshot() map[string]*HiAgentConversationInfo {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.data == nil {
		return nil
	}
	out := make(map[string]*HiAgentConversationInfo, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	return out
}

// hiAgentStore lazily initializes and returns the conversation store.
// ExecuteConfig is used by pointer wherever these helpers are called, so it is
// safe to assign the new store back onto the receiver.
func (c *ExecuteConfig) hiAgentStore() *HiAgentConvStore {
	if c.HiAgentConversations == nil {
		c.HiAgentConversations = &HiAgentConvStore{}
	}
	return c.HiAgentConversations
}

// GetHiAgentConversationID retrieves the HiAgent conversation ID for a specific agent (backward compatible)
func (c *ExecuteConfig) GetHiAgentConversationID(agentID string) string {
	info := c.GetHiAgentConversationInfo(agentID)
	if info == nil {
		return ""
	}
	return info.AppConversationID
}

// GetHiAgentConversationInfo retrieves the full HiAgent conversation info for a specific agent
func (c *ExecuteConfig) GetHiAgentConversationInfo(agentID string) *HiAgentConversationInfo {
	return c.HiAgentConversations.Get(agentID)
}

// SetHiAgentConversationID sets the HiAgent conversation ID for a specific agent (backward compatible)
func (c *ExecuteConfig) SetHiAgentConversationID(agentID, appConvID string) {
	c.SetHiAgentConversationInfo(agentID, &HiAgentConversationInfo{
		AppConversationID: appConvID,
		LastSectionID:     0, // Will be updated when section info is available
	})
}

// SetHiAgentConversationInfo sets the full HiAgent conversation info for a specific agent
func (c *ExecuteConfig) SetHiAgentConversationInfo(agentID string, info *HiAgentConversationInfo) {
	c.hiAgentStore().Set(agentID, info)
}

// ClearHiAgentConversationID clears the HiAgent conversation ID for a specific agent
func (c *ExecuteConfig) ClearHiAgentConversationID(agentID string) {
	c.HiAgentConversations.Delete(agentID)
}

// ClearAllHiAgentConversations clears all HiAgent conversation mappings
func (c *ExecuteConfig) ClearAllHiAgentConversations() {
	c.hiAgentStore().Reset()
}
