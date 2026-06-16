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

// Package agentsandbox 是一个自包含、可独立抽出的会话级沙箱模块。
//
// 它对下只依赖本文件定义的注入接口（Cache / Blob）和 contract 子包的 Runner，
// 不依赖 coze-studio 的 domain / application / infra-impl 层。将来可整体抽成独立项目：
// 把 backend/pkg/agentsandbox/ 拷出去、给 Cache/Blob/Runner 各提供一份实现即可。
package agentsandbox

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss 表示 Cache 中不存在该 key。
var ErrCacheMiss = errors.New("agentsandbox: cache miss")

// Cache 是注册表持久化所需的最小 KV 接口（多节点共享活沙箱状态）。
// 由宿主用 Redis 等实现；Get 未命中返回 ErrCacheMiss。
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// Blob 是 workspace 持久化（会话恢复）所需的最小对象存储接口。
// 由宿主用 MinIO / TOS / S3 等实现；GetObject 不存在时返回 (nil, nil) 或错误均可。
type Blob interface {
	PutObject(ctx context.Context, key string, content []byte) error
	GetObject(ctx context.Context, key string) ([]byte, error)
}
