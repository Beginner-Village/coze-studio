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

package docker

import (
	"fmt"
	"sort"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/sandbox"
)

const (
	defaultImage   = "python:3.11-slim"
	defaultWorkDir = "/workspace"
)

func containerName(prefix, sandboxID string) string { return prefix + sandboxID }

func buildCreateArgs(req *sandbox.CreateRequest, prefix string) []string {
	image := req.Image
	if image == "" {
		image = defaultImage
	}
	args := []string{"run", "-d", "--name", containerName(prefix, req.SandboxID)}
	if req.MemoryMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", req.MemoryMB))
	}
	if req.CPUs > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%.2f", req.CPUs))
	}
	keys := make([]string, 0, len(req.Env))
	for k := range req.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, req.Env[k]))
	}
	args = append(args, "-w", defaultWorkDir, image, "sleep", "infinity")
	return args
}

func buildExecArgs(req *sandbox.ExecRequest, prefix string) []string {
	wd := req.WorkDir
	if wd == "" {
		wd = defaultWorkDir
	}
	return []string{"exec", "-w", wd, containerName(prefix, req.SandboxID), "bash", "-lc", req.Cmd}
}
