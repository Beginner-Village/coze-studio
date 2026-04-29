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

package observability

import "strings"

// ClassifyUploadKind maps a file extension or MIME-ish string to one of
// {"image", "doc", "other"} for the studio_file_upload_size_bytes histogram.
// The classification is intentionally coarse - high-cardinality file types
// would blow up the metric's label space.
func ClassifyUploadKind(fileType string) string {
	t := strings.ToLower(strings.TrimSpace(fileType))
	if t == "" {
		return "other"
	}
	if strings.HasPrefix(t, "image") ||
		t == "png" || t == "jpg" || t == "jpeg" ||
		t == "gif" || t == "webp" || t == "bmp" || t == "svg" {
		return "image"
	}
	switch t {
	case "pdf", "doc", "docx", "txt", "md", "markdown",
		"xls", "xlsx", "ppt", "pptx", "csv", "json", "html", "htm":
		return "doc"
	}
	return "other"
}
