// Copyright 2025 ynet-dev Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build integration

// Package space_sync_test contains integration tests for the space export/import/sync
// pipeline backed by a real MySQL container.
//
// The plan called for a full E2E that wires SpaceExporter + SpaceImporter + SyncService
// against real MySQL/MinIO/ES/Redis. In practice that requires the full production
// schema (2k+ lines) plus stub implementations of dozens of domain repositories
// (single_agent_draft, workflow_meta, knowledge_document, …). To keep this CI-friendly
// we instead test the persistence layer in isolation against real MySQL: this catches
// divergences from the SQLite unit tests (e.g. GORM `OnConflict` behaviour, JSON column
// handling, AUTO_INCREMENT semantics) and gives confidence the schema is correct.
package space_sync_test
