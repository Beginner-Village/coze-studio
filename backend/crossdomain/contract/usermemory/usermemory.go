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

// Package usermemory is the cross-domain contract for the redesigned super-agent
// long-term memory: per-user, DB-backed, structured (kind + optional stable key),
// with relevance recall — replacing the old flat string-array sandbox file.
// Scope is per user_id ("grows with you"); space_id/agent_id are provenance +
// optional filters, not isolation. The agent runtime (agentflow) calls DefaultSVC();
// the application layer injects a DB-backed implementation via SetDefaultSVC at startup.
// See docs/superpowers/specs/2026-06-20-super-agent-memory-redesign.md.
package usermemory

import "context"

// MemoryKind classifies a memory entry.
type MemoryKind int8

const (
	KindProfile    MemoryKind = 1 // who the user is (durable profile; always recalled)
	KindPreference MemoryKind = 2 // how the user wants the agent to behave
	KindFact       MemoryKind = 3 // a discrete fact/observation (default)
	KindProject    MemoryKind = 4 // project/work-context fact
	KindFeedback   MemoryKind = 5 // a correction/feedback the agent should honor
)

// Status values for a memory entry.
const (
	StatusActive   int8 = 1
	StatusArchived int8 = 2 // superseded / no longer authoritative
)

// ProfileMemKey is the reserved stable key under which the per-user NARRATIVE
// profile document (a Hermes USER.md-style markdown prose about who the user is)
// is stored as a single kind=profile entry, kept current by the review fork and
// auto-injected into the super-agent system prompt. Distinct from discrete facts.
const ProfileMemKey = "user_profile_doc"

// ParseMemoryKind maps a free-form string (tool argument) to a MemoryKind,
// defaulting to KindFact for empty/unknown values.
func ParseMemoryKind(s string) MemoryKind {
	switch s {
	case "profile":
		return KindProfile
	case "preference":
		return KindPreference
	case "project":
		return KindProject
	case "feedback":
		return KindFeedback
	case "fact", "":
		return KindFact
	default:
		return KindFact
	}
}

// String renders a MemoryKind as its tool-facing label.
func (k MemoryKind) String() string {
	switch k {
	case KindProfile:
		return "profile"
	case KindPreference:
		return "preference"
	case KindProject:
		return "project"
	case KindFeedback:
		return "feedback"
	default:
		return "fact"
	}
}

// UserMemory is one structured memory entry owned by a user.
type UserMemory struct {
	ID                   int64
	UserID               int64
	SpaceID              int64
	AgentID              int64
	Kind                 MemoryKind
	MemKey               string // stable key for upsert/supersede; empty = always-insert
	Content              string
	Tags                 string
	SourceConversationID int64
	SourceRunID          int64
	Status               int8
	CreatedAt            int64
	UpdatedAt            int64
}

// RecallQuery parameterizes a relevance recall. Profile entries are always
// returned regardless of Query; other kinds are filtered by Query (Phase A:
// substring match on content/tags) and optional Kind, capped at Limit.
type RecallQuery struct {
	Query string
	Kind  *MemoryKind
	Limit int
}

// Manager persists and recalls per-user super-agent memory.
type Manager interface {
	// Save inserts a new memory, or—when MemKey is non-empty and an active entry
	// already exists for (UserID, MemKey)—updates that entry in place (supersede).
	Save(ctx context.Context, m *UserMemory) (*UserMemory, error)
	// Recall returns the user's profile entries (always) plus entries matching the
	// query/kind filter, most-recent first, capped at q.Limit.
	Recall(ctx context.Context, userID int64, q RecallQuery) ([]*UserMemory, error)
}

var defaultSVC Manager

// DefaultSVC returns the globally injected memory manager (nil if not wired).
func DefaultSVC() Manager { return defaultSVC }

// SetDefaultSVC injects the global memory manager (called at app startup).
func SetDefaultSVC(m Manager) { defaultSVC = m }
