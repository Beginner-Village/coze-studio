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

package skill

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/assert"
)

func TestMarketplaceInstallRouteRejectsMalformedJSON(t *testing.T) {
	h := server.Default()
	Register(h)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/skill/marketplace/install",
		&ut.Body{Body: bytes.NewBufferString(`{`), Len: len(`{`)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMarketplaceGetRouteIsRegistered(t *testing.T) {
	h := server.Default()
	Register(h)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodGet,
		"/api/skill/marketplace/get",
		nil,
	)

	assert.NotEqual(t, http.StatusNotFound, w.Code)
}
