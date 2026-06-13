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

package errno

import "github.com/ynet-dev/ynet-studio/backend/pkg/errorx/code"

// Space-level operation audit log error codes.
// Using 112100xxx range to stay within the space-operations 112xxxxxx band
// while not colliding with the space export/import codes (112000xxx).
const (
	ErrOperationLogPermissionCode   = 112100001
	ErrOperationLogInvalidParamCode = 112100002
)

func init() {
	reg := func(c int32, msg string) {
		code.Register(c, msg, code.WithAffectStability(false))
	}
	reg(ErrOperationLogPermissionCode, "无权限: {msg}")
	reg(ErrOperationLogInvalidParamCode, "参数错误: {msg}")
}
