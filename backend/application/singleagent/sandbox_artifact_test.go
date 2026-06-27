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

package singleagent

import (
	"context"
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	"github.com/ynet-dev/ynet-studio/backend/domain/user/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/ctxcache"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
)

func TestArtifactObjectKeyIsUserSpaceScoped(t *testing.T) {
	got := artifactObjectKey(1, 9, 100, 50, "report.pdf")
	want := "artifacts/s1/u9/p100/c50/report.pdf"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

// ctxWithUID returns a context that resolves to the given userID via GetUIDFromCtx.
func ctxWithUID(uid int64) context.Context {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &entity.Session{UserID: uid})
	return ctx
}

// ctxWithNoAuth returns a context with no session and no API auth (callerID == 0).
func ctxWithNoAuth() context.Context {
	return ctxcache.Init(context.Background())
}

// verifyCallerID is a small sanity-check: ctxutil must resolve what we stored.
func verifyCallerID(t *testing.T, ctx context.Context, wantUID int64) {
	t.Helper()
	got := ctxutil.GetUIDFromCtx(ctx)
	if wantUID == 0 {
		if got != nil {
			t.Fatalf("expected no UID in ctx, got %d", *got)
		}
		return
	}
	if got == nil || *got != wantUID {
		var actual int64
		if got != nil {
			actual = *got
		}
		t.Fatalf("expected UID %d in ctx, got %d", wantUID, actual)
	}
}

func TestCheckArtifactObjectKeyOwner(t *testing.T) {
	isolatedKey := "artifacts/s1/u9/p1/c1/x.pdf"

	t.Run("non-artifact key is always allowed", func(t *testing.T) {
		ctx := ctxWithNoAuth()
		if err := checkArtifactObjectKeyOwner(ctx, "outputs/x.txt"); err != nil {
			t.Fatalf("expected nil for non-artifact key, got %v", err)
		}
	})

	t.Run("unknown caller (callerID==0) denied - fail closed", func(t *testing.T) {
		ctx := ctxWithNoAuth()
		verifyCallerID(t, ctx, 0)
		err := checkArtifactObjectKeyOwner(ctx, isolatedKey)
		if err == nil {
			t.Fatal("expected error for unknown caller on isolated artifact key (fail-closed), got nil")
		}
	})

	t.Run("matching caller allowed", func(t *testing.T) {
		ctx := ctxWithUID(9)
		verifyCallerID(t, ctx, 9)
		if err := checkArtifactObjectKeyOwner(ctx, isolatedKey); err != nil {
			t.Fatalf("expected nil for owner caller, got %v", err)
		}
	})

	t.Run("mismatched caller denied", func(t *testing.T) {
		ctx := ctxWithUID(7)
		verifyCallerID(t, ctx, 7)
		err := checkArtifactObjectKeyOwner(ctx, isolatedKey)
		if err == nil {
			t.Fatal("expected error for non-owner caller, got nil")
		}
	})
}
