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
	"testing"

	sandbox "github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox/contract"
)

// recordingRunner extends fakeRunner-like behaviour but records the
// CreateRequest it was handed and lets a test seed pre-existing files so
// ReadFile (e.g. of the workspace archive) returns deterministic bytes.
type recordingRunner struct {
	*fakeRunner
	lastCreate *sandbox.CreateRequest
}

func newRecordingRunner() *recordingRunner {
	return &recordingRunner{fakeRunner: newFakeRunner()}
}

func (r *recordingRunner) Create(ctx context.Context, req *sandbox.CreateRequest) (*sandbox.CreateResponse, error) {
	cp := *req
	r.lastCreate = &cp
	return r.fakeRunner.Create(ctx, req)
}

// seedArchive places bytes at the workspace archive path so a subsequent
// CheckpointTo can read them back (the fake Exec does not actually tar).
func (r *recordingRunner) seedArchive(id string, data []byte) {
	r.fakeRunner.mu.Lock()
	defer r.fakeRunner.mu.Unlock()
	if r.fakeRunner.files[id] == nil {
		r.fakeRunner.files[id] = map[string][]byte{}
	}
	r.fakeRunner.files[id][workspaceArchivePath] = data
}

func testManagerWithStore(runner sandbox.Runner, store Blob, cfg Config) *Manager {
	m := New(runner, NewMemRegistry(), store, cfg)
	var clk int64 = 1000
	m.now = func() int64 { return clk }
	return m
}

// TestCheckpointToPutsToPassedKey proves CheckpointTo writes the archive to
// the *passed* objectKey (not the fixed workspaceKey) and returns a non-empty
// content hash.
func TestCheckpointToPutsToPassedKey(t *testing.T) {
	ctx := context.Background()
	fr := newRecordingRunner()
	store := newFakeStorage()
	m := testManagerWithStore(fr, store, DefaultConfig())

	// Bring the sandbox up and seed deterministic archive bytes that the
	// (fake) tar would have produced.
	if err := m.EnsureSandbox(ctx, "u1"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	fr.seedArchive("u1", []byte("TEMPLATE-TGZ"))

	const objectKey = "templates/agent_app/100/1.tgz"
	hash, err := m.CheckpointTo(ctx, "u1", objectKey)
	if err != nil {
		t.Fatalf("CheckpointTo: %v", err)
	}
	if hash == "" {
		t.Fatalf("expected non-empty content hash")
	}
	// Must land at the passed key.
	if got, _ := store.GetObject(ctx, objectKey); string(got) != "TEMPLATE-TGZ" {
		t.Fatalf("object at %q = %q, want TEMPLATE-TGZ", objectKey, got)
	}
	// Must NOT have written the fixed per-instance workspace key.
	if got, _ := store.GetObject(ctx, m.workspaceKey("u1")); len(got) != 0 {
		t.Fatalf("CheckpointTo must not write the fixed workspace key, got %q", got)
	}
}

// TestRestoreFromReadsPassedKey proves RestoreFrom pulls from the passed
// objectKey and untars it into the sandbox (archive written to the sandbox).
func TestRestoreFromReadsPassedKey(t *testing.T) {
	ctx := context.Background()
	fr := newRecordingRunner()
	store := newFakeStorage()
	m := testManagerWithStore(fr, store, DefaultConfig())
	if err := m.EnsureSandbox(ctx, "u1"); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	const objectKey = "templates/agent_app/100/1.tgz"
	_ = store.PutObject(ctx, objectKey, []byte("TPL-BYTES"))

	if err := m.RestoreFrom(ctx, "u1", objectKey); err != nil {
		t.Fatalf("RestoreFrom: %v", err)
	}
	// The archive bytes should have been written into the sandbox for untar.
	got, _ := fr.ReadFile(ctx, &sandbox.ReadFileRequest{SandboxID: "u1", Path: workspaceArchivePath})
	if string(got) != "TPL-BYTES" {
		t.Fatalf("archive in sandbox = %q, want TPL-BYTES", got)
	}
}

// TestColdStartTemplatePrecedence proves the cold-start precedence:
//
//	(a) instance checkpoint present  -> template NOT used;
//	(b) only template present        -> template restored;
//	(c) neither                      -> blank (no restore from either).
func TestColdStartTemplatePrecedence(t *testing.T) {
	const tplKey = "templates/agent_app/100/1.tgz"

	t.Run("instance checkpoint wins over template", func(t *testing.T) {
		ctx := context.Background()
		fr := newRecordingRunner()
		store := newFakeStorage()
		cfg := DefaultConfig()
		cfg.TemplateObjectKey = tplKey
		m := testManagerWithStore(fr, store, cfg)

		// Seed BOTH an instance checkpoint and a template.
		_ = store.PutObject(ctx, m.workspaceKey("u1"), []byte("INSTANCE"))
		_ = store.PutObject(ctx, tplKey, []byte("TEMPLATE"))

		if err := m.EnsureSandbox(ctx, "u1"); err != nil {
			t.Fatalf("ensure: %v", err)
		}
		// The instance checkpoint must be what got written into the sandbox.
		got, _ := fr.ReadFile(ctx, &sandbox.ReadFileRequest{SandboxID: "u1", Path: workspaceArchivePath})
		if string(got) != "INSTANCE" {
			t.Fatalf("restored archive = %q, want INSTANCE (instance precedence)", got)
		}
	})

	t.Run("template used when no instance checkpoint", func(t *testing.T) {
		ctx := context.Background()
		fr := newRecordingRunner()
		store := newFakeStorage()
		cfg := DefaultConfig()
		cfg.TemplateObjectKey = tplKey
		m := testManagerWithStore(fr, store, cfg)

		_ = store.PutObject(ctx, tplKey, []byte("TEMPLATE"))

		if err := m.EnsureSandbox(ctx, "u1"); err != nil {
			t.Fatalf("ensure: %v", err)
		}
		got, _ := fr.ReadFile(ctx, &sandbox.ReadFileRequest{SandboxID: "u1", Path: workspaceArchivePath})
		if string(got) != "TEMPLATE" {
			t.Fatalf("restored archive = %q, want TEMPLATE", got)
		}
	})

	t.Run("blank when neither present", func(t *testing.T) {
		ctx := context.Background()
		fr := newRecordingRunner()
		store := newFakeStorage()
		cfg := DefaultConfig()
		cfg.TemplateObjectKey = tplKey // configured but the object does not exist
		m := testManagerWithStore(fr, store, cfg)

		if err := m.EnsureSandbox(ctx, "u1"); err != nil {
			t.Fatalf("ensure: %v", err)
		}
		got, _ := fr.ReadFile(ctx, &sandbox.ReadFileRequest{SandboxID: "u1", Path: workspaceArchivePath})
		if len(got) != 0 {
			t.Fatalf("expected blank cold-start (no archive written), got %q", got)
		}
	})
}

// TestColdStartReadonlySkillsPropagates proves the ReadonlySkills config flag
// is threaded into the CreateRequest the runner receives (so the docker layer
// can add the :ro mount).
func TestColdStartReadonlySkillsPropagates(t *testing.T) {
	ctx := context.Background()
	fr := newRecordingRunner()
	cfg := DefaultConfig()
	cfg.ReadonlySkills = true
	m := testManagerWithStore(fr, nil, cfg)

	if err := m.EnsureSandbox(ctx, "u1"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if fr.lastCreate == nil || !fr.lastCreate.ReadonlySkills {
		t.Fatalf("CreateRequest.ReadonlySkills not propagated: %+v", fr.lastCreate)
	}
}

// TestEnsureSandboxWithTemplateNoOverwriteOnExistingCheckpoint is the core
// TDD proof for the per-turn-overwrite fix:
//
// An instance whose sandbox ALREADY has an instance checkpoint must NOT have
// its workspace overwritten by the template on a subsequent EnsureSandboxWithTemplate
// call (simulating the 2nd, 3rd, … conversation turn).
//
// Precedence: instance checkpoint (restore) > template > blank.
// For an ALREADY-RUNNING sandbox: no restore at all — workspace stays as-is.
func TestEnsureSandboxWithTemplateNoOverwriteOnExistingCheckpoint(t *testing.T) {
	const (
		sandboxKey = "instance-u1"
		tplKey     = "templates/agent_app/100/1.tgz"
	)

	t.Run("first cold-start: template used when no instance checkpoint", func(t *testing.T) {
		ctx := context.Background()
		fr := newRecordingRunner()
		store := newFakeStorage()
		m := testManagerWithStore(fr, store, DefaultConfig())

		_ = store.PutObject(ctx, tplKey, []byte("TEMPLATE"))

		if err := m.EnsureSandboxWithTemplate(ctx, sandboxKey, tplKey, true); err != nil {
			t.Fatalf("EnsureSandboxWithTemplate (first): %v", err)
		}
		got, _ := fr.ReadFile(ctx, &sandbox.ReadFileRequest{SandboxID: sandboxKey, Path: workspaceArchivePath})
		if string(got) != "TEMPLATE" {
			t.Fatalf("first cold-start: restored archive = %q, want TEMPLATE", got)
		}
		if fr.lastCreate == nil || !fr.lastCreate.ReadonlySkills {
			t.Fatalf("ReadonlySkills not propagated to Create: %+v", fr.lastCreate)
		}
	})

	t.Run("first cold-start: instance checkpoint wins over template", func(t *testing.T) {
		ctx := context.Background()
		fr := newRecordingRunner()
		store := newFakeStorage()
		m := testManagerWithStore(fr, store, DefaultConfig())

		_ = store.PutObject(ctx, m.workspaceKey(sandboxKey), []byte("INSTANCE-CKPT"))
		_ = store.PutObject(ctx, tplKey, []byte("TEMPLATE"))

		if err := m.EnsureSandboxWithTemplate(ctx, sandboxKey, tplKey, true); err != nil {
			t.Fatalf("EnsureSandboxWithTemplate (first, with checkpoint): %v", err)
		}
		got, _ := fr.ReadFile(ctx, &sandbox.ReadFileRequest{SandboxID: sandboxKey, Path: workspaceArchivePath})
		if string(got) != "INSTANCE-CKPT" {
			t.Fatalf("instance checkpoint must take precedence: got %q, want INSTANCE-CKPT", got)
		}
	})

	t.Run("second call on running sandbox: workspace NOT overwritten", func(t *testing.T) {
		ctx := context.Background()
		fr := newRecordingRunner()
		store := newFakeStorage()
		m := testManagerWithStore(fr, store, DefaultConfig())

		_ = store.PutObject(ctx, tplKey, []byte("TEMPLATE"))

		// First call: cold-start → template applied.
		if err := m.EnsureSandboxWithTemplate(ctx, sandboxKey, tplKey, true); err != nil {
			t.Fatalf("first ensure: %v", err)
		}
		// Simulate user work: the archive in the sandbox is now their accumulated work.
		fr.seedArchive(sandboxKey, []byte("USER-WORK"))
		// Simulate that an instance checkpoint now exists (mirroring a Checkpoint() call).
		_ = store.PutObject(ctx, m.workspaceKey(sandboxKey), []byte("USER-WORK"))

		createCountBefore := fr.CreateCount[sandboxKey]

		// Second call: sandbox is already running → must NOT restore template.
		if err := m.EnsureSandboxWithTemplate(ctx, sandboxKey, tplKey, true); err != nil {
			t.Fatalf("second ensure: %v", err)
		}
		// Container must not have been recreated.
		if fr.CreateCount[sandboxKey] != createCountBefore {
			t.Fatalf("Create called on already-running sandbox (count %d → %d); template overwrite detected",
				createCountBefore, fr.CreateCount[sandboxKey])
		}
		// Archive in sandbox must still be USER-WORK, not TEMPLATE.
		got, _ := fr.ReadFile(ctx, &sandbox.ReadFileRequest{SandboxID: sandboxKey, Path: workspaceArchivePath})
		if string(got) != "USER-WORK" {
			t.Fatalf("workspace overwritten on second call: got %q, want USER-WORK", got)
		}
	})
}
