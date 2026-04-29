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

package logs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWriter_NoFileWhenLogFileEmpty(t *testing.T) {
	t.Setenv("LOG_FILE", "")
	w := NewWriter()
	if w == nil {
		t.Fatal("expected writer, got nil")
	}
	tmp := t.TempDir()
	t.Setenv("LOG_FILE", "")
	_, _ = w.Write([]byte("hello\n"))

	entries, _ := os.ReadDir(tmp)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".log") {
			t.Fatalf("unexpected log file %s", e.Name())
		}
	}
}

func TestNewWriter_CreatesFileWhenLogFileSet(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "app.log")
	t.Setenv("LOG_FILE", logPath)
	t.Setenv("LOG_MAX_SIZE_MB", "1")
	t.Setenv("LOG_MAX_BACKUPS", "2")

	w := NewWriter()
	if _, err := w.Write([]byte("hello\n")); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("expected log file %s, got error: %v", logPath, err)
	}
}

func TestNewWriter_RotatesAtMaxSize(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "app.log")
	t.Setenv("LOG_FILE", logPath)
	t.Setenv("LOG_MAX_SIZE_MB", "1") // 1 MB
	t.Setenv("LOG_MAX_BACKUPS", "3")

	w := NewWriter()
	chunk := make([]byte, 4096)
	for i := range chunk {
		chunk[i] = 'A'
	}
	for i := 0; i < 400; i++ { // 400 * 4KB = 1.6 MB
		if _, err := w.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}

	entries, _ := os.ReadDir(tmp)
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "app") {
			count++
		}
	}
	if count < 2 {
		t.Fatalf("expected >=2 log files after rotation, got %d", count)
	}
}
