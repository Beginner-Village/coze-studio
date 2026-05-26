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

package errno

// Space export/import error codes
// Using 112xxxxx range for space operations
const (
	ErrSpaceInvalidParamCode   = 112000000
	ErrSpacePermissionCode     = 112000001
	ErrSpaceNotFoundCode       = 112000002
	ErrSpaceExportFailedCode   = 112000003
	ErrSpaceImportFailedCode   = 112000004
	ErrSpaceImportTokenExpired = 112000005
	ErrSpaceImportTokenInvalid = 112000006

	// Release version management
	ErrSpaceReleaseNotFoundCode   = 112000010
	ErrSpaceReleaseExistsCode     = 112000011
	ErrSpaceReleaseInvalidVersion = 112000012
	ErrSpaceRollbackFailedCode    = 112000013

	// ES per-space resync
	ErrSpaceResyncESCode = 112000020

	// One-shot per-space model + embedder + rerank reconfig
	ErrSpaceConfigureModelsCode = 112000021

	// Read-only per-space health-check / diagnose
	ErrSpaceDiagnoseCode = 112000022
)
