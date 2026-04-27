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

package entity

import "strings"

const (
	MaxImageFileSize = 10 * 1024 * 1024  // 10 MB
	MaxOtherFileSize = 100 * 1024 * 1024 // 100 MB
)

var imageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".bmp":  true,
	".tiff": true,
}

// IsImageFile 根据文件名扩展名判断是否图片类型
func IsImageFile(filename string) bool {
	idx := strings.LastIndex(filename, ".")
	if idx == -1 {
		return false
	}
	ext := strings.ToLower(filename[idx:])
	return imageExtensions[ext]
}

// MaxFileSizeFor 返回该文件应用的大小上限
func MaxFileSizeFor(filename string) int64 {
	if IsImageFile(filename) {
		return MaxImageFileSize
	}
	return MaxOtherFileSize
}
