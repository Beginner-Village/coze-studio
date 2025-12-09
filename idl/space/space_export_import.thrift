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

namespace go space

include "../base.thrift"

// ==================== Export Structures ====================

// Export statistics information
struct ExportStatistics {
    1: required i32 agents    // Number of agents exported
    2: required i32 plugins   // Number of plugins exported
    3: required i32 workflows // Number of workflows exported
    4: required i32 variables // Number of variables exported
}

// Export result data
struct ExportData {
    1: required string download_url   // Presigned URL to download the export package
    2: required i64    expires_at     // URL expiration timestamp (Unix milliseconds)
    3: required string file_name      // Suggested file name for download
    4: required i64    file_size      // Size of the export file in bytes
    5: required ExportStatistics statistics // Export statistics
}

// Export space request
struct ExportSpaceRequest {
    1: required i64 space_id (api.path="space_id", api.js_conv='true', agw.js_conv="str")

    255: base.Base Base (api.none="true")
}

// Export space response
struct ExportSpaceResponse {
    253: required i32 code
    254: required string msg
    1: required ExportData data
    255: required base.BaseResp BaseResp (api.none="true")
}

// ==================== Import Preview Structures ====================

// Resource reference information
struct ResourceReference {
    1: required i64 id (api.js_conv='true', agw.js_conv="str")
    2: required string name
    3: required string resource_type  // agent, plugin, workflow, variable
}

// Import manifest information
struct ImportManifest {
    1: required string version           // Manifest version
    2: required string source_space_name // Original space name
    3: required i64    exported_at       // Export timestamp
    4: required ExportStatistics statistics // Resource statistics
}

// Import preview result data
struct ImportPreviewData {
    1: required string import_token         // Token for confirming import
    2: required ImportManifest manifest     // Manifest information
    3: required list<string> warnings       // Warning messages
    4: required i64 token_expires_at        // Token expiration timestamp
}

// Import preview request
struct ImportPreviewRequest {
    1: required i64 space_id (api.path="space_id", api.js_conv='true', agw.js_conv="str")
    2: required binary file_content (api.body="file_content") // ZIP file content

    255: base.Base Base (api.none="true")
}

// Import preview response
struct ImportPreviewResponse {
    253: required i32 code
    254: required string msg
    1: required ImportPreviewData data
    255: required base.BaseResp BaseResp (api.none="true")
}

// ==================== Import Confirm Structures ====================

// Import result statistics
struct ImportResultStatistics {
    1: required i32 agents_created    // Number of agents created
    2: required i32 plugins_created   // Number of plugins created
    3: required i32 workflows_created // Number of workflows created
    4: required i32 variables_created // Number of variables created
}

// Import confirm result data
struct ImportConfirmData {
    1: required ImportResultStatistics statistics // Import result statistics
    2: required list<ResourceReference> created_resources // List of created resources
}

// Import confirm request
struct ImportConfirmRequest {
    1: required i64 space_id (api.path="space_id", api.js_conv='true', agw.js_conv="str")
    2: required string import_token (api.body="import_token") // Token from preview

    255: base.Base Base (api.none="true")
}

// Import confirm response
struct ImportConfirmResponse {
    253: required i32 code
    254: required string msg
    1: required ImportConfirmData data
    255: required base.BaseResp BaseResp (api.none="true")
}

// ==================== Service Definition ====================

service SpaceExportImportService {
    // Export a space to a ZIP package
    ExportSpaceResponse ExportSpace(1: ExportSpaceRequest req) (api.post="/api/space/{space_id}/export")

    // Preview import operation and get import token
    ImportPreviewResponse ImportPreview(1: ImportPreviewRequest req) (api.post="/api/space/{space_id}/import/preview")

    // Confirm and execute the import operation
    ImportConfirmResponse ImportConfirm(1: ImportConfirmRequest req) (api.post="/api/space/{space_id}/import/confirm")
}
