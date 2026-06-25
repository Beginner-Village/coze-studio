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
	"fmt"
	"strings"
)

type TemplateSandbox interface {
	EnsureSandbox(ctx context.Context, key string) error
	SyncSkill(ctx context.Context, key, skillName string, files map[string]string) error
	Exec(ctx context.Context, key, cmd string, timeoutSec int) (stdout, stderr string, exit int, err error)
	Checkpoint(ctx context.Context, key, objectKey string) (contentHash string, err error)
	Destroy(ctx context.Context, key string) error
}

type BuildSkill struct {
	Name    string
	Files   map[string]string
	PipDeps []string
	NpmDeps []string
}

type BuildTemplateRequest struct {
	BuildKey  string
	ObjectKey string
	Skills    []BuildSkill
}

type BuildTemplateResult struct {
	Status      string
	ContentHash string
	Detail      string
}

const buildExecTimeoutSec = 600

// BuildTemplate spins a throwaway sandbox, injects each skill's pinned files,
// installs declared pip/npm dependencies, then checkpoints /workspace to the
// template object key. The build sandbox is always destroyed.
func BuildTemplate(ctx context.Context, sb TemplateSandbox, req BuildTemplateRequest) (BuildTemplateResult, error) {
	defer func() { _ = sb.Destroy(ctx, req.BuildKey) }()

	if err := sb.EnsureSandbox(ctx, req.BuildKey); err != nil {
		return BuildTemplateResult{Status: "failed", Detail: "ensure: " + err.Error()}, nil
	}

	for _, sk := range req.Skills {
		if err := sb.SyncSkill(ctx, req.BuildKey, sk.Name, sk.Files); err != nil {
			return BuildTemplateResult{Status: "failed", Detail: "sync " + sk.Name + ": " + err.Error()}, nil
		}
		if len(sk.PipDeps) > 0 {
			cmd := "pip install " + strings.Join(sk.PipDeps, " ")
			if _, stderr, exit, err := sb.Exec(ctx, req.BuildKey, cmd, buildExecTimeoutSec); err != nil || exit != 0 {
				detail := fmt.Sprintf("pip(%s): exit=%d stderr=%s", sk.Name, exit, stderr)
				if err != nil {
					detail += fmt.Sprintf(" err=%v", err)
				}
				return BuildTemplateResult{Status: "failed", Detail: detail}, nil
			}
		}
		if len(sk.NpmDeps) > 0 {
			cmd := "npm i -g " + strings.Join(sk.NpmDeps, " ")
			if _, stderr, exit, err := sb.Exec(ctx, req.BuildKey, cmd, buildExecTimeoutSec); err != nil || exit != 0 {
				detail := fmt.Sprintf("npm(%s): exit=%d stderr=%s", sk.Name, exit, stderr)
				if err != nil {
					detail += fmt.Sprintf(" err=%v", err)
				}
				return BuildTemplateResult{Status: "failed", Detail: detail}, nil
			}
		}
	}

	hash, err := sb.Checkpoint(ctx, req.BuildKey, req.ObjectKey)
	if err != nil {
		return BuildTemplateResult{Status: "failed", Detail: "checkpoint: " + err.Error()}, nil
	}
	return BuildTemplateResult{Status: "ready", ContentHash: hash}, nil
}
