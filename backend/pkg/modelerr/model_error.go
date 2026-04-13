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

package modelerr

import (
	"errors"
	"strings"

	openai "github.com/meguminnnnnnnnn/go-openai"
)

// ErrorKind classifies model API errors for retry and user-facing decisions.
type ErrorKind int

const (
	ErrorKindUnknown       ErrorKind = iota
	ErrorKindRateLimit               // 429 — retryable after delay
	ErrorKindTokenLimit              // 400 context_length_exceeded — retryable with shorter input
	ErrorKindContentFilter           // 400 content_filter — not retryable
	ErrorKindInvalidImage            // 400 invalid image / unsupported modality — not retryable
	ErrorKindInvalidParam            // 400 other bad request — not retryable
	ErrorKindAuth                    // 401/403 — not retryable
	ErrorKindNotFound                // 404 model not found — not retryable
	ErrorKindServerError             // 500+ — retryable
)

// Retryable returns true if the error kind is worth retrying.
func (k ErrorKind) Retryable() bool {
	switch k {
	case ErrorKindRateLimit, ErrorKindServerError, ErrorKindTokenLimit:
		return true
	default:
		return false
	}
}

// UserMessage returns a human-readable Chinese message for the error kind.
func (k ErrorKind) UserMessage(detail string) string {
	switch k {
	case ErrorKindRateLimit:
		return "模型调用频率超限，正在重试..."
	case ErrorKindTokenLimit:
		return "输入内容过长，超出模型上下文长度限制。请缩短输入或清理对话历史后重试。"
	case ErrorKindContentFilter:
		return "内容触发了安全过滤策略，请修改输入内容后重试。"
	case ErrorKindInvalidImage:
		return "当前模型不支持处理图片/多模态内容，请切换为支持多模态的模型或移除图片后重试。"
	case ErrorKindInvalidParam:
		if detail != "" {
			return "模型调用参数错误: " + detail
		}
		return "模型调用参数错误，请检查配置后重试。"
	case ErrorKindAuth:
		return "模型认证失败，请检查 API Key 配置。"
	case ErrorKindNotFound:
		return "模型不存在或已下线，请切换其他模型。"
	case ErrorKindServerError:
		return "模型服务暂时不可用，正在重试..."
	default:
		if detail != "" {
			return "模型调用失败: " + detail
		}
		return "模型调用失败，请稍后重试。"
	}
}

// Classify inspects an error and returns its ErrorKind.
func Classify(err error) ErrorKind {
	if err == nil {
		return ErrorKindUnknown
	}

	// Check go-openai APIError (used by eino OpenAI/Qwen/Deepseek builders)
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		return classifyByHTTPCode(apiErr.HTTPStatusCode, apiErr.Message)
	}

	// Check go-openai RequestError
	var reqErr *openai.RequestError
	if errors.As(err, &reqErr) {
		return classifyByHTTPCode(reqErr.HTTPStatusCode, "")
	}

	// Fallback: inspect error string for common patterns
	return classifyByMessage(err.Error())
}

func classifyByHTTPCode(code int, message string) ErrorKind {
	switch {
	case code == 429:
		return ErrorKindRateLimit
	case code == 401 || code == 403:
		return ErrorKindAuth
	case code == 404:
		return ErrorKindNotFound
	case code >= 500:
		return ErrorKindServerError
	case code == 400:
		return classifyBadRequest(message)
	default:
		return ErrorKindUnknown
	}
}

func classifyBadRequest(message string) ErrorKind {
	msg := strings.ToLower(message)

	// Token/context length
	if strings.Contains(msg, "context_length") ||
		strings.Contains(msg, "max_tokens") ||
		strings.Contains(msg, "token") && strings.Contains(msg, "exceed") ||
		strings.Contains(msg, "too long") ||
		strings.Contains(msg, "maximum context") {
		return ErrorKindTokenLimit
	}

	// Content filter / safety
	if strings.Contains(msg, "content_filter") ||
		strings.Contains(msg, "content_policy") ||
		strings.Contains(msg, "safety") ||
		strings.Contains(msg, "moderation") ||
		strings.Contains(msg, "blocked") {
		return ErrorKindContentFilter
	}

	// Image / multimodal
	if strings.Contains(msg, "image") ||
		strings.Contains(msg, "vision") ||
		strings.Contains(msg, "multimodal") ||
		strings.Contains(msg, "does not support") && strings.Contains(msg, "image") ||
		strings.Contains(msg, "invalid_image") ||
		strings.Contains(msg, "could not process image") {
		return ErrorKindInvalidImage
	}

	return ErrorKindInvalidParam
}

func classifyByMessage(msg string) ErrorKind {
	lower := strings.ToLower(msg)

	switch {
	case strings.Contains(lower, "rate limit") || strings.Contains(lower, "rate_limit") || strings.Contains(lower, "429"):
		return ErrorKindRateLimit
	case strings.Contains(lower, "context_length") || strings.Contains(lower, "token") && strings.Contains(lower, "exceed"):
		return ErrorKindTokenLimit
	case strings.Contains(lower, "content_filter") || strings.Contains(lower, "content_policy"):
		return ErrorKindContentFilter
	case strings.Contains(lower, "unauthorized") || strings.Contains(lower, "authentication") || strings.Contains(lower, "401"):
		return ErrorKindAuth
	case strings.Contains(lower, "not found") || strings.Contains(lower, "404"):
		return ErrorKindNotFound
	case strings.Contains(lower, "500") || strings.Contains(lower, "internal server error") || strings.Contains(lower, "502") || strings.Contains(lower, "503"):
		return ErrorKindServerError
	default:
		return ErrorKindUnknown
	}
}
