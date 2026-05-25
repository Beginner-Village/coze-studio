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

package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/rerank"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
)

type Config struct {
	BaseURL string // e.g. "https://api.jina.ai/v1" or "http://localhost:8080/v1"
	APIKey  string // Bearer token
	Model   string // model name
}

func NewReranker(config *Config) rerank.Reranker {
	return &reranker{config: config}
}

type reranker struct {
	config *Config
}

type rerankReq struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      *int64   `json:"top_n,omitempty"`
}

type rerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

type rerankUsage struct {
	TotalTokens int64 `json:"total_tokens"`
}

type rerankResp struct {
	Results []rerankResult `json:"results"`
	Usage   rerankUsage    `json:"usage"`
}

func (r *reranker) Rerank(ctx context.Context, req *rerank.Request) (*rerank.Response, error) {
	var flat []*rerank.Data
	for _, channel := range req.Data {
		flat = append(flat, channel...)
	}

	// Defensive: upstream vLLM rejects requests with empty query OR empty
	// documents (after filtering blank content) with 400 "The decoder prompt
	// cannot be empty". Short-circuit with empty result instead of bubbling up
	// a confusing error.
	if req.Query == "" {
		return &rerank.Response{SortedData: []*rerank.Data{}, TokenUsage: ptr.Of(int64(0))}, nil
	}
	if len(flat) == 0 {
		return &rerank.Response{SortedData: []*rerank.Data{}, TokenUsage: ptr.Of(int64(0))}, nil
	}

	// Filter out docs whose Content is blank — they would surface to vLLM
	// as empty decoder prompts even when len(documents)>0.
	documents := make([]string, 0, len(flat))
	keptFlat := make([]*rerank.Data, 0, len(flat))
	for _, item := range flat {
		if item == nil || item.Document == nil || item.Document.Content == "" {
			continue
		}
		documents = append(documents, item.Document.Content)
		keptFlat = append(keptFlat, item)
	}
	if len(documents) == 0 {
		return &rerank.Response{SortedData: []*rerank.Data{}, TokenUsage: ptr.Of(int64(0))}, nil
	}
	flat = keptFlat

	rReq := &rerankReq{
		Model:     r.config.Model,
		Query:     req.Query,
		Documents: documents,
		TopN:      req.TopN,
	}

	body, err := json.Marshal(rReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.config.BaseURL+"/rerank", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+r.config.APIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[Rerank] request failed, status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var rResp rerankResp
	if err = json.Unmarshal(respBody, &rResp); err != nil {
		return nil, err
	}

	sorted := make([]*rerank.Data, 0, len(rResp.Results))
	for _, result := range rResp.Results {
		if result.Index < 0 || result.Index >= len(flat) {
			continue
		}
		sorted = append(sorted, &rerank.Data{
			Document: flat[result.Index].Document,
			Score:    result.RelevanceScore,
		})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})

	right := len(sorted)
	if req.TopN != nil {
		right = min(right, int(*req.TopN))
	}

	return &rerank.Response{
		SortedData: sorted[:right],
		TokenUsage: ptr.Of(rResp.Usage.TotalTokens),
	}, nil
}
