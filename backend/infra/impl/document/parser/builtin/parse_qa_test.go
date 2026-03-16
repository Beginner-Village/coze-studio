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

package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document"
	contract "github.com/ynet-dev/ynet-studio/backend/infra/contract/document/parser"
)

func TestParseQACSV(t *testing.T) {
	tests := []struct {
		name        string
		csvContent  string
		wantDocs    int
		wantQ       []string
		wantA       []string
		wantErr     bool
		errContains string
	}{
		{
			name: "basic Q,A format",
			csvContent: `Q,A
What is AI?,AI stands for Artificial Intelligence
How does machine learning work?,Machine learning uses algorithms to learn from data`,
			wantDocs: 2,
			wantQ:    []string{"What is AI?", "How does machine learning work?"},
			wantA:    []string{"AI stands for Artificial Intelligence", "Machine learning uses algorithms to learn from data"},
		},
		{
			name: "question,answer format",
			csvContent: `question,answer
What is Go?,Go is a programming language
What is Rust?,Rust is a systems programming language`,
			wantDocs: 2,
			wantQ:    []string{"What is Go?", "What is Rust?"},
			wantA:    []string{"Go is a programming language", "Rust is a systems programming language"},
		},
		{
			name: "Chinese column names",
			csvContent: `问题,答案
什么是人工智能,人工智能是模拟人类智能的技术
如何学习编程,可以通过在线课程和实践来学习`,
			wantDocs: 2,
			wantQ:    []string{"什么是人工智能", "如何学习编程"},
			wantA:    []string{"人工智能是模拟人类智能的技术", "可以通过在线课程和实践来学习"},
		},
		{
			name: "with extra columns",
			csvContent: `id,Q,A,category
1,What is AI?,Artificial Intelligence,tech
2,What is Go?,Programming Language,dev`,
			wantDocs: 2,
			wantQ:    []string{"What is AI?", "What is Go?"},
			wantA:    []string{"Artificial Intelligence", "Programming Language"},
		},
		{
			name: "skip empty questions",
			csvContent: `Q,A
What is AI?,AI is cool
,This has no question
Another question?,Another answer`,
			wantDocs: 2,
			wantQ:    []string{"What is AI?", "Another question?"},
			wantA:    []string{"AI is cool", "Another answer"},
		},
		{
			name: "empty answer allowed",
			csvContent: `Q,A
What is AI?,
What is Go?,Go is great`,
			wantDocs: 2,
			wantQ:    []string{"What is AI?", "What is Go?"},
			wantA:    []string{"", "Go is great"},
		},
		{
			name:        "missing Q column",
			csvContent:  `A,B\nanswer1,something`,
			wantErr:     true,
			errContains: "question column not found",
		},
		{
			name:        "missing A column",
			csvContent:  `Q,B\nquestion1,something`,
			wantErr:     true,
			errContains: "answer column not found",
		},
		{
			name:        "empty file",
			csvContent:  ``,
			wantErr:     true,
			errContains: "empty CSV file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &contract.Config{}
			parseFn := ParseQACSV(config)

			reader := strings.NewReader(tt.csvContent)
			docs, err := parseFn(context.Background(), reader)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Len(t, docs, tt.wantDocs)

			for i, doc := range docs {
				assert.Equal(t, tt.wantQ[i], doc.Content, "Question mismatch at index %d", i)
				answer, ok := document.GetDocumentAnswer(doc)
				assert.True(t, ok, "Answer not found in metadata at index %d", i)
				assert.Equal(t, tt.wantA[i], answer, "Answer mismatch at index %d", i)
			}
		})
	}
}

func TestParseQAJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonContent string
		wantDocs    int
		wantQ       []string
		wantA       []string
		wantErr     bool
		errContains string
	}{
		{
			name: "basic q,a format",
			jsonContent: `[
				{"q": "What is AI?", "a": "AI stands for Artificial Intelligence"},
				{"q": "What is Go?", "a": "Go is a programming language"}
			]`,
			wantDocs: 2,
			wantQ:    []string{"What is AI?", "What is Go?"},
			wantA:    []string{"AI stands for Artificial Intelligence", "Go is a programming language"},
		},
		{
			name: "question,answer format",
			jsonContent: `[
				{"question": "What is AI?", "answer": "Artificial Intelligence"},
				{"question": "What is ML?", "answer": "Machine Learning"}
			]`,
			wantDocs: 2,
			wantQ:    []string{"What is AI?", "What is ML?"},
			wantA:    []string{"Artificial Intelligence", "Machine Learning"},
		},
		{
			name: "Chinese field names",
			jsonContent: `[
				{"问题": "什么是AI", "答案": "人工智能"},
				{"问题": "什么是Go", "答案": "编程语言"}
			]`,
			wantDocs: 2,
			wantQ:    []string{"什么是AI", "什么是Go"},
			wantA:    []string{"人工智能", "编程语言"},
		},
		{
			name: "mixed formats",
			jsonContent: `[
				{"q": "Question 1", "a": "Answer 1"},
				{"question": "Question 2", "answer": "Answer 2"},
				{"Q": "Question 3", "A": "Answer 3"}
			]`,
			wantDocs: 3,
			wantQ:    []string{"Question 1", "Question 2", "Question 3"},
			wantA:    []string{"Answer 1", "Answer 2", "Answer 3"},
		},
		{
			name: "with extra fields",
			jsonContent: `[
				{"id": 1, "q": "What is AI?", "a": "AI tech", "category": "tech"},
				{"id": 2, "q": "What is Go?", "a": "Go lang", "tags": ["dev"]}
			]`,
			wantDocs: 2,
			wantQ:    []string{"What is AI?", "What is Go?"},
			wantA:    []string{"AI tech", "Go lang"},
		},
		{
			name: "skip items without question",
			jsonContent: `[
				{"q": "What is AI?", "a": "AI"},
				{"a": "No question here"},
				{"q": "What is Go?", "a": "Go"}
			]`,
			wantDocs: 2,
			wantQ:    []string{"What is AI?", "What is Go?"},
			wantA:    []string{"AI", "Go"},
		},
		{
			name: "empty answer allowed",
			jsonContent: `[
				{"q": "What is AI?", "a": ""},
				{"q": "What is Go?"}
			]`,
			wantDocs: 2,
			wantQ:    []string{"What is AI?", "What is Go?"},
			wantA:    []string{"", ""},
		},
		{
			name:        "empty array",
			jsonContent: `[]`,
			wantErr:     true,
			errContains: "empty JSON array",
		},
		{
			name:        "invalid JSON",
			jsonContent: `{not valid json}`,
			wantErr:     true,
			errContains: "invalid JSON format",
		},
		{
			name:        "no valid QA pairs",
			jsonContent: `[{"x": "1", "y": "2"}]`,
			wantErr:     true,
			errContains: "no valid QA pairs found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &contract.Config{}
			parseFn := ParseQAJSON(config)

			reader := strings.NewReader(tt.jsonContent)
			docs, err := parseFn(context.Background(), reader)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Len(t, docs, tt.wantDocs)

			for i, doc := range docs {
				assert.Equal(t, tt.wantQ[i], doc.Content, "Question mismatch at index %d", i)
				answer, ok := document.GetDocumentAnswer(doc)
				assert.True(t, ok, "Answer not found in metadata at index %d", i)
				assert.Equal(t, tt.wantA[i], answer, "Answer mismatch at index %d", i)
			}
		})
	}
}

func TestFindQAColumnIndices(t *testing.T) {
	tests := []struct {
		name       string
		header     []string
		wantQIndex int
		wantAIndex int
	}{
		{
			name:       "Q,A",
			header:     []string{"Q", "A"},
			wantQIndex: 0,
			wantAIndex: 1,
		},
		{
			name:       "question,answer",
			header:     []string{"question", "answer"},
			wantQIndex: 0,
			wantAIndex: 1,
		},
		{
			name:       "问题,答案",
			header:     []string{"问题", "答案"},
			wantQIndex: 0,
			wantAIndex: 1,
		},
		{
			name:       "reversed order",
			header:     []string{"A", "Q"},
			wantQIndex: 1,
			wantAIndex: 0,
		},
		{
			name:       "with extra columns",
			header:     []string{"id", "Q", "A", "category"},
			wantQIndex: 1,
			wantAIndex: 2,
		},
		{
			name:       "case insensitive",
			header:     []string{"QUESTION", "ANSWER"},
			wantQIndex: 0,
			wantAIndex: 1,
		},
		{
			name:       "missing Q",
			header:     []string{"X", "A"},
			wantQIndex: -1,
			wantAIndex: 1,
		},
		{
			name:       "missing A",
			header:     []string{"Q", "X"},
			wantQIndex: 0,
			wantAIndex: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qIndex, aIndex := findQAColumnIndices(tt.header)
			assert.Equal(t, tt.wantQIndex, qIndex, "Q index mismatch")
			assert.Equal(t, tt.wantAIndex, aIndex, "A index mismatch")
		})
	}
}

func TestAnyToString(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"string", "hello", "hello"},
		{"string with spaces", "  hello  ", "hello"},
		{"nil", nil, ""},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := anyToString(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}
