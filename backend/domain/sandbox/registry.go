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

package sandbox

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/cache"
	sbx "github.com/ynet-dev/ynet-studio/backend/infra/contract/sandbox"
)

// Entry 是注册表里一个活沙箱的元数据。
type Entry struct {
	SandboxID      string    `json:"sandbox_id"`
	State          sbx.State `json:"state"`
	LastActiveUnix int64     `json:"last_active_unix"`
}

// Registry 是活沙箱注册表的窄接口；单节点用内存实现，多节点用 Redis 实现。
type Registry interface {
	Put(ctx context.Context, e *Entry) error
	Get(ctx context.Context, id string) (*Entry, bool, error)
	List(ctx context.Context) ([]*Entry, error)
	Delete(ctx context.Context, id string) error
	Touch(ctx context.Context, id string, unix int64) error
	SetState(ctx context.Context, id string, st sbx.State) error
}

// ---- 内存实现（单节点默认，线程安全）----

type memRegistry struct {
	mu sync.Mutex
	m  map[string]*Entry
}

// NewMemRegistry 返回进程内注册表。
func NewMemRegistry() Registry { return &memRegistry{m: map[string]*Entry{}} }

func (r *memRegistry) Put(_ context.Context, e *Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *e
	r.m[e.SandboxID] = &cp
	return nil
}

func (r *memRegistry) Get(_ context.Context, id string) (*Entry, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.m[id]
	if !ok {
		return nil, false, nil
	}
	cp := *e
	return &cp, true, nil
}

func (r *memRegistry) List(_ context.Context) ([]*Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*Entry, 0, len(r.m))
	for _, e := range r.m {
		cp := *e
		out = append(out, &cp)
	}
	return out, nil
}

func (r *memRegistry) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.m, id)
	return nil
}

func (r *memRegistry) Touch(_ context.Context, id string, unix int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.m[id]; ok {
		e.LastActiveUnix = unix
	}
	return nil
}

func (r *memRegistry) SetState(_ context.Context, id string, st sbx.State) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.m[id]; ok {
		e.State = st
	}
	return nil
}

// ---- Redis 实现（整张表存单 key，进程内锁串行化 RMW）----

const redisRegistryKey = "ynet:sandbox:registry"

type redisRegistry struct {
	cli cache.Cmdable
	mu  sync.Mutex
}

// NewRedisRegistry 返回 Redis 注册表（survive 重启、多节点可见）。
func NewRedisRegistry(cli cache.Cmdable) Registry { return &redisRegistry{cli: cli} }

func (r *redisRegistry) load(ctx context.Context) (map[string]*Entry, error) {
	b, err := r.cli.Get(ctx, redisRegistryKey).Bytes()
	if err != nil {
		if cache.Nil != nil && err == cache.Nil {
			return map[string]*Entry{}, nil
		}
		// key 不存在的其它表现统一当空表处理。
		return map[string]*Entry{}, nil
	}
	if len(b) == 0 {
		return map[string]*Entry{}, nil
	}
	m := map[string]*Entry{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (r *redisRegistry) save(ctx context.Context, m map[string]*Entry) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return r.cli.Set(ctx, redisRegistryKey, b, 0).Err()
}

func (r *redisRegistry) Put(ctx context.Context, e *Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, err := r.load(ctx)
	if err != nil {
		return err
	}
	cp := *e
	m[e.SandboxID] = &cp
	return r.save(ctx, m)
}

func (r *redisRegistry) Get(ctx context.Context, id string) (*Entry, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, err := r.load(ctx)
	if err != nil {
		return nil, false, err
	}
	e, ok := m[id]
	return e, ok, nil
}

func (r *redisRegistry) List(ctx context.Context) ([]*Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, err := r.load(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*Entry, 0, len(m))
	for _, e := range m {
		out = append(out, e)
	}
	return out, nil
}

func (r *redisRegistry) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, err := r.load(ctx)
	if err != nil {
		return err
	}
	delete(m, id)
	return r.save(ctx, m)
}

func (r *redisRegistry) Touch(ctx context.Context, id string, unix int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, err := r.load(ctx)
	if err != nil {
		return err
	}
	if e, ok := m[id]; ok {
		e.LastActiveUnix = unix
		return r.save(ctx, m)
	}
	return nil
}

func (r *redisRegistry) SetState(ctx context.Context, id string, st sbx.State) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, err := r.load(ctx)
	if err != nil {
		return err
	}
	if e, ok := m[id]; ok {
		e.State = st
		return r.save(ctx, m)
	}
	return nil
}
