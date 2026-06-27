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

package goutil

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

func GetPythonFilePath(fileName string) string {
	cwd, err := os.Getwd()
	if err != nil {
		logs.Warnf("[GetPythonFilePath] Failed to get current working directory: %v", err)
		return fileName
	}

	return filepath.Join(cwd, fileName)
}

func GetPython3Path() string {
	cwd, err := os.Getwd()
	if err != nil {
		logs.Warnf("[GetPython3Path] Failed to get current working directory: %v", err)
		return pythonExecutableFallback()
	}

	venvPython := filepath.Join(cwd, ".venv/bin/python3")
	if _, statErr := os.Stat(venvPython); statErr == nil {
		return venvPython
	}
	logs.Warnf("[GetPython3Path] venv python missing at %s, fallback to system python3", venvPython)
	return pythonExecutableFallback()
}

// pythonExecutableFallback 在项目内 .venv 缺失时退回系统 python3,
// 避免容器重建后 .venv 丢失导致 code runner / 文档解析报"解释器不存在"。
func pythonExecutableFallback() string {
	if p, err := exec.LookPath("python3"); err == nil {
		return p
	}
	return "python3"
}
