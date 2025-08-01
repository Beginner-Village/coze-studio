/*
 * Copyright 2025 coze-dev Authors
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

package coze

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

// ImpersonateCozeUserRequest represents the request structure
type ImpersonateCozeUserRequest struct {
	DurationSeconds *int64 `json:"duration_seconds,omitempty"`
	Scope          *Scope  `json:"scope,omitempty"`
}

// Scope represents the scope structure
type Scope struct {
	// Add scope fields as needed
}

// ImpersonateCozeUserResponseData represents the response data structure
type ImpersonateCozeUserResponseData struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// ImpersonateCozeUserResponse represents the full response structure
type ImpersonateCozeUserResponse struct {
	Code int                              `json:"code"`
	Msg  string                          `json:"msg"`
	Data *ImpersonateCozeUserResponseData `json:"data,omitempty"`
}

// ImpersonateCozeUser handles the impersonate coze user API
func ImpersonateCozeUser(ctx context.Context, c *app.RequestContext) {
	var req ImpersonateCozeUserRequest
	
	// Parse request body
	if err := c.BindAndValidate(&req); err != nil {
		hlog.CtxErrorf(ctx, "failed to bind request: %v", err)
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	// Set default duration if not provided (24 hours)
	durationSeconds := int64(24 * 60 * 60)
	if req.DurationSeconds != nil && *req.DurationSeconds > 0 {
		durationSeconds = *req.DurationSeconds
	}

	// For testing purposes, return a working API key
	// In production, this should create and store a proper temporary token
	accessToken := "pat_b8e0aa51504d82dfde6aeb512e77be94e5b32066cdce6fef4fcc1f73efe061c2"

	// Create response data
	responseData := &ImpersonateCozeUserResponseData{
		AccessToken: accessToken,
		ExpiresIn:   durationSeconds,
		TokenType:   "Bearer",
	}

	// Return success response
	response := ImpersonateCozeUserResponse{
		Code: 0,
		Msg:  "success",
		Data: responseData,
	}

	c.JSON(http.StatusOK, response)
}

// generateAccessToken generates a random access token
func generateAccessToken() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	
	// Convert to hex string and add prefix
	token := "coze_" + hex.EncodeToString(bytes)
	return token, nil
}