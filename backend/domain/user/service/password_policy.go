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

package service

import (
	"strings"
	"unicode"

	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

const (
	// 电子银行级密码强度要求：长度 8~64，且必须同时包含大写、小写、数字、特殊字符。
	passwordMinLength = 8
	passwordMaxLength = 64
)

// 常见弱密码片段字典：密码（小写后）包含其中任一片段即判定为弱密码。
var weakPasswordTokens = []string{
	"password", "passwd", "admin", "root", "qwerty", "abc123",
	"123456", "111111", "000000", "654321", "iloveyou",
	"welcome", "letmein", "monkey", "dragon", "master",
	"login", "huawei", "ynet", "coze", "guard",
}

// validatePasswordStrength 校验密码是否满足电子银行级安全策略，不满足返回 ErrUserWeakPasswordCode。
func validatePasswordStrength(password string) error {
	weak := func(msg string) error {
		return errorx.New(errno.ErrUserWeakPasswordCode, errorx.KV("msg", msg))
	}

	if len(password) < passwordMinLength {
		return weak("length must be at least 8 characters")
	}
	if len(password) > passwordMaxLength {
		return weak("length must not exceed 64 characters")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r) || r == ' ':
			hasSpecial = true
		}
	}
	if !hasUpper {
		return weak("must contain at least one uppercase letter")
	}
	if !hasLower {
		return weak("must contain at least one lowercase letter")
	}
	if !hasDigit {
		return weak("must contain at least one digit")
	}
	if !hasSpecial {
		return weak("must contain at least one special character")
	}

	lower := strings.ToLower(password)
	for _, token := range weakPasswordTokens {
		if strings.Contains(lower, token) {
			return weak("must not contain common weak patterns")
		}
	}

	return nil
}
