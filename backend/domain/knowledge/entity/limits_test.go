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

import "testing"

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"photo.jpg", true},
		{"photo.JPG", true},
		{"photo.JPEG", true},
		{"image.png", true},
		{"animation.gif", true},
		{"document.pdf", false},
		{"text.txt", false},
		{"noext", false},
	}
	for _, tt := range tests {
		if got := IsImageFile(tt.filename); got != tt.want {
			t.Errorf("IsImageFile(%q) = %v, want %v", tt.filename, got, tt.want)
		}
	}
}

func TestMaxFileSizeFor(t *testing.T) {
	if MaxFileSizeFor("photo.jpg") != MaxImageFileSize {
		t.Error("image should use MaxImageFileSize")
	}
	if MaxFileSizeFor("doc.pdf") != MaxOtherFileSize {
		t.Error("non-image should use MaxOtherFileSize")
	}
}
