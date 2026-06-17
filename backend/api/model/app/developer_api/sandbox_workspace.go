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

package developer_api

// 超级智能体「沙箱空间管理」接口的请求/响应模型(手写，Hertz 按 json tag 绑定)。

type SandboxFileInfo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

// ---- 列目录 ----

type ListSandboxFilesRequest struct {
	SpaceID     int64   `json:"space_id,string" query:"space_id"`
	BotID       int64   `json:"bot_id,string" query:"bot_id"`
	Path        string  `json:"path" query:"path"`
	ConnectorID *string `json:"connector_id,omitempty" query:"connector_id"`
}

type ListSandboxFilesData struct {
	Path  string             `json:"path"`
	Files []*SandboxFileInfo `json:"files"`
}

type ListSandboxFilesResponse struct {
	Code int64                 `json:"code"`
	Msg  string                `json:"msg"`
	Data *ListSandboxFilesData `json:"data"`
}

// ---- 读文件 ----

type ReadSandboxFileRequest struct {
	SpaceID     int64   `json:"space_id,string" query:"space_id"`
	BotID       int64   `json:"bot_id,string" query:"bot_id"`
	Path        string  `json:"path" query:"path"`
	ConnectorID *string `json:"connector_id,omitempty" query:"connector_id"`
}

type ReadSandboxFileData struct {
	Path string `json:"path"`
	// Content 为 UTF-8 文本;二进制内容时 IsBinary=true 且 Content 为 base64。
	Content  string `json:"content"`
	IsBinary bool   `json:"is_binary"`
	Size     int64  `json:"size"`
}

type ReadSandboxFileResponse struct {
	Code int64                `json:"code"`
	Msg  string               `json:"msg"`
	Data *ReadSandboxFileData `json:"data"`
}

// ---- 写/上传文件 ----

type UploadSandboxFileRequest struct {
	SpaceID int64  `json:"space_id,string"`
	BotID   int64  `json:"bot_id,string"`
	Path    string `json:"path"`
	// Content 为文本内容;二进制请传 base64 并置 IsBase64=true。
	Content     string  `json:"content"`
	IsBase64    bool    `json:"is_base64"`
	ConnectorID *string `json:"connector_id,omitempty"`
}

type UploadSandboxFileData struct {
	Path string `json:"path"`
}

type UploadSandboxFileResponse struct {
	Code int64                  `json:"code"`
	Msg  string                 `json:"msg"`
	Data *UploadSandboxFileData `json:"data"`
}

// ---- 删除文件 ----

type DeleteSandboxFileRequest struct {
	SpaceID     int64   `json:"space_id,string"`
	BotID       int64   `json:"bot_id,string"`
	Path        string  `json:"path"`
	ConnectorID *string `json:"connector_id,omitempty"`
}

type DeleteSandboxFileResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}
