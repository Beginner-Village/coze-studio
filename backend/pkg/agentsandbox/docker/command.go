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
	"os"
	"sort"
	"strings"

	"github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox/contract"
)

const (
	// 富沙箱镜像:自带 python/node/npm/curl/wget/git/jq,且 apt/pip/npm 均为国内源。
	defaultImage   = "ynet-sandbox:rich"
	defaultWorkDir = "/workspace"
)

func containerName(prefix, sandboxID string) string { return prefix + sandboxID }

// 沙箱契约目录用持久化 named volume 承载,容器被 reaper 回收/重建后文件仍在
// (named volume 不随 docker rm 删除)。键名按 sandboxID 唯一。
// /skills 也持久化:技能文件夹(~180 文件)同步代价高,持久化后容器重建无需重灌,
// SyncSkill 的 .skillhash 去重直接命中,避免每次进入重新同步拖慢首条消息响应。
var sandboxVolumeDirs = []string{"/workspace", "/uploads", "/outputs", "/skills"}

// sandboxDataRoot 是宿主机上持久化沙箱数据的根目录(bind mount 到客户可见/可备份的
// 固定路径,而非藏在 Docker 内部的 named volume)。可用 SANDBOX_DATA_DIR 覆盖。
// 注意:该路径由「宿主机 docker 守护进程」解析(后端容器经 docker.sock 操作宿主机)。
const defaultSandboxDataRoot = "/var/lib/ynet-sandboxes"

func sandboxDataRoot() string {
	if v := strings.TrimSpace(os.Getenv("SANDBOX_DATA_DIR")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return defaultSandboxDataRoot
}

// hostBindPath 返回某沙箱某契约目录在宿主机上的持久化路径。
// docker run 时若该路径不存在,守护进程会自动创建(root)。
func hostBindPath(sandboxID, dir string) string {
	return sandboxDataRoot() + "/" + sandboxID + dir
}

// resolveImage：优先 req.Image，其次环境变量 SANDBOX_IMAGE，最后内置默认富镜像。
func resolveImage(reqImage string) string {
	if reqImage != "" {
		return reqImage
	}
	if env := strings.TrimSpace(os.Getenv("SANDBOX_IMAGE")); env != "" {
		return env
	}
	return defaultImage
}

func buildCreateArgs(req *sandbox.CreateRequest, prefix string) []string {
	image := resolveImage(req.Image)
	args := []string{"run", "-d", "--name", containerName(prefix, req.SandboxID)}
	// 持久化挂载契约目录到宿主机固定路径(bind mount),保证空闲回收/容器重建后文件不丢,
	// 且数据落在客户可见、可直接备份的目录(SANDBOX_DATA_DIR/<id>/<dir>)。
	for _, dir := range sandboxVolumeDirs {
		mount := hostBindPath(req.SandboxID, dir) + ":" + dir
		// 模板化超级体可选只读挂载 /skills,防止运行时改写技能模板。默认读写。
		if req.ReadonlySkills && dir == "/skills" {
			mount += ":ro"
		}
		args = append(args, "-v", mount)
	}
	if req.MemoryMB > 0 {
		// --memory 限内存;--memory-swap 设为与内存相等 = 禁用容器额外 swap,
		// 防止沙箱里跑飞的进程占满宿主机 swap 把整台机器拖死。
		args = append(args,
			"--memory", fmt.Sprintf("%dm", req.MemoryMB),
			"--memory-swap", fmt.Sprintf("%dm", req.MemoryMB),
		)
	}
	if req.CPUs > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%.2f", req.CPUs))
	}
	// 进程数上限:防 fork 炸弹撑爆宿主机 PID(否则连 sshd 都 fork 不出来,整机失联)。
	args = append(args, "--pids-limit", "512")
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
