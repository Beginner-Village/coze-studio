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

package agentsandbox

import (
	"context"
	"strings"
	"testing"
)

// 写一个文件到 fakeRunner 的 /workspace 里供 EditFile 用。
func seedFile(t *testing.T, m *Manager, key, path, content string) {
	t.Helper()
	if err := m.WriteFile(context.Background(), key, path, []byte(content)); err != nil {
		t.Fatalf("seed write: %v", err)
	}
}

func readBack(t *testing.T, m *Manager, key, path string) string {
	t.Helper()
	b, err := m.ReadFile(context.Background(), key, path)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	return string(b)
}

func TestEditFile_UniqueReplace(t *testing.T) {
	ctx := context.Background()
	m := testManager(newFakeRunner())
	seedFile(t, m, "u1", "/workspace/a.txt", "hello world\nbye world\n")

	n, err := m.EditFile(ctx, "u1", "/workspace/a.txt", "hello", "hi", false)
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if n != 1 {
		t.Fatalf("replacements = %d, want 1", n)
	}
	if got := readBack(t, m, "u1", "/workspace/a.txt"); got != "hi world\nbye world\n" {
		t.Fatalf("content = %q", got)
	}
}

func TestEditFile_NotFound(t *testing.T) {
	ctx := context.Background()
	m := testManager(newFakeRunner())
	seedFile(t, m, "u1", "/workspace/a.txt", "abc")

	_, err := m.EditFile(ctx, "u1", "/workspace/a.txt", "xyz", "zzz", false)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

func TestEditFile_AmbiguousRequiresReplaceAll(t *testing.T) {
	ctx := context.Background()
	m := testManager(newFakeRunner())
	seedFile(t, m, "u1", "/workspace/a.txt", "x x x")

	// 多处匹配且 replace_all=false → 报错，文件不变。
	_, err := m.EditFile(ctx, "u1", "/workspace/a.txt", "x", "y", false)
	if err == nil || !strings.Contains(err.Error(), "found 3 times") {
		t.Fatalf("want ambiguous error, got %v", err)
	}
	if got := readBack(t, m, "u1", "/workspace/a.txt"); got != "x x x" {
		t.Fatalf("file should be unchanged, got %q", got)
	}
}

func TestEditFile_ReplaceAll(t *testing.T) {
	ctx := context.Background()
	m := testManager(newFakeRunner())
	seedFile(t, m, "u1", "/workspace/a.txt", "x x x")

	n, err := m.EditFile(ctx, "u1", "/workspace/a.txt", "x", "y", true)
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if n != 3 {
		t.Fatalf("replacements = %d, want 3", n)
	}
	if got := readBack(t, m, "u1", "/workspace/a.txt"); got != "y y y" {
		t.Fatalf("content = %q", got)
	}
}

func TestEditFile_Validation(t *testing.T) {
	ctx := context.Background()
	m := testManager(newFakeRunner())
	seedFile(t, m, "u1", "/workspace/a.txt", "abc")

	if _, err := m.EditFile(ctx, "u1", "/workspace/a.txt", "", "z", false); err == nil {
		t.Fatal("want error on empty old_string")
	}
	if _, err := m.EditFile(ctx, "u1", "/workspace/a.txt", "abc", "abc", false); err == nil {
		t.Fatal("want error on identical old/new")
	}
	if _, err := m.EditFile(ctx, "u1", "", "a", "b", false); err == nil {
		t.Fatal("want error on empty path")
	}
}

func TestGrepGlob_Validation(t *testing.T) {
	ctx := context.Background()
	m := testManager(newFakeRunner())
	if _, err := m.Grep(ctx, "u1", "", ""); err == nil {
		t.Fatal("want error on empty grep pattern")
	}
	if _, err := m.Glob(ctx, "u1", ""); err == nil {
		t.Fatal("want error on empty glob pattern")
	}
}
