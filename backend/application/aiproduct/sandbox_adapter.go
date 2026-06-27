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

package aiproduct

import (
	"context"

	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
)

// sandboxAdapter bridges crosssandbox.Manager to the TemplateSandbox port used
// by BuildTemplate.
//
// Differences to bridge:
//   - EnsureSandbox: not exposed by the crosssandbox.Manager interface; the
//     concrete Manager calls it internally before every Exec/SyncSkill, so a
//     no-op is safe here.
//   - Exec: real returns (*ExecResponse, error); TemplateSandbox needs (stdout,
//     stderr string, exit int, err error).
//   - SyncSkill: real takes map[string][]byte; TemplateSandbox needs map[string]string.
//   - Checkpoint: bridges to crosssandbox.Manager.CheckpointTo(key, objectKey),
//     which tars the build sandbox's /workspace to the passed template object
//     key and returns the archive content hash.
//   - Destroy: not exposed by the crosssandbox.Manager interface; build-sandbox
//     cleanup is handled by the Manager's own idle-reaper (no-op).
type sandboxAdapter struct {
	m crosssandbox.Manager
}

// NewSandboxAdapter wraps the global crosssandbox.Manager as a TemplateSandbox.
// Returns nil if the manager has not been initialised yet (sandbox disabled).
func NewSandboxAdapter() TemplateSandbox {
	m := crosssandbox.DefaultSVC()
	if m == nil {
		return nil
	}
	return &sandboxAdapter{m: m}
}

// EnsureSandbox is a no-op: the crosssandbox.Manager interface does not expose
// EnsureSandbox; the concrete implementation calls it automatically before each
// Exec / SyncSkill invocation.
func (a *sandboxAdapter) EnsureSandbox(_ context.Context, _ string) error {
	return nil
}

func (a *sandboxAdapter) SyncSkill(ctx context.Context, key, skillName string, files map[string]string) error {
	// Convert map[string]string → map[string][]byte.
	b := make(map[string][]byte, len(files))
	for k, v := range files {
		b[k] = []byte(v)
	}
	return a.m.SyncSkill(ctx, key, skillName, b)
}

func (a *sandboxAdapter) Exec(ctx context.Context, key, cmd string, timeoutSec int) (stdout, stderr string, exit int, err error) {
	res, err := a.m.Exec(ctx, key, cmd, timeoutSec)
	if err != nil {
		return "", "", -1, err
	}
	return res.Stdout, res.Stderr, res.ExitCode, nil
}

// Checkpoint tars the build sandbox's /workspace and persists it to the passed
// template object key via the concrete Manager's CheckpointTo, returning the
// archive content hash so the caller can record the template version.
func (a *sandboxAdapter) Checkpoint(ctx context.Context, key, objectKey string) (string, error) {
	return a.m.CheckpointTo(ctx, key, objectKey)
}

// Destroy is a no-op: the crosssandbox.Manager interface does not expose
// Destroy.  Build-sandbox cleanup is handled by the Manager's idle-reaper.
func (a *sandboxAdapter) Destroy(_ context.Context, _ string) error {
	return nil
}
