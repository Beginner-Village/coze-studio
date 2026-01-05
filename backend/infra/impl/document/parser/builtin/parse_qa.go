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

package builtin

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/schema"
	"github.com/dimchansky/utfbom"

	"github.com/coze-dev/coze-studio/backend/infra/contract/document"
	contract "github.com/coze-dev/coze-studio/backend/infra/contract/document/parser"
)

// QA column names (case-insensitive matching)
var (
	questionColumnNames = []string{"q", "question", "问题"}
	answerColumnNames   = []string{"a", "answer", "答案"}
)

// ParseQACSV parses CSV files with Q/A columns.
// Expected format:
//
//	Q,A
//	"What is AI?","AI stands for Artificial Intelligence..."
//	"How does it work?","It works by..."
//
// The Q column content is used for vector embedding (stored in Content),
// while A column is stored in MetaData for retrieval response.
func ParseQACSV(config *contract.Config) ParseFn {
	return func(ctx context.Context, reader io.Reader, opts ...parser.Option) (docs []*schema.Document, err error) {
		csvReader := csv.NewReader(utfbom.SkipOnly(reader))
		options := parser.GetCommonOptions(&parser.Options{}, opts...)

		// Read header row
		header, err := csvReader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("[ParseQACSV] empty CSV file")
			}
			return nil, fmt.Errorf("[ParseQACSV] failed to read header: %w", err)
		}

		// Find Q and A column indices
		qIndex, aIndex := findQAColumnIndices(header)
		if qIndex == -1 {
			return nil, fmt.Errorf("[ParseQACSV] question column not found, expected one of: %v", questionColumnNames)
		}
		if aIndex == -1 {
			return nil, fmt.Errorf("[ParseQACSV] answer column not found, expected one of: %v", answerColumnNames)
		}

		// Parse data rows
		rowNum := 1
		for {
			row, err := csvReader.Read()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, fmt.Errorf("[ParseQACSV] failed to read row %d: %w", rowNum+1, err)
			}
			rowNum++

			// Skip rows with insufficient columns
			if len(row) <= qIndex || len(row) <= aIndex {
				continue
			}

			question := strings.TrimSpace(row[qIndex])
			answer := strings.TrimSpace(row[aIndex])

			// Skip empty questions
			if question == "" {
				continue
			}

			doc := &schema.Document{
				Content:  question, // Q is embedded for vector search
				MetaData: make(map[string]any),
			}

			// Store A in metadata for retrieval
			document.WithDocumentAnswer(doc, answer)

			// Add extra metadata from options
			for k, v := range options.ExtraMeta {
				doc.MetaData[k] = v
			}

			docs = append(docs, doc)
		}

		if len(docs) == 0 {
			return nil, fmt.Errorf("[ParseQACSV] no valid QA pairs found")
		}

		return docs, nil
	}
}

// ParseQAJSON parses JSON files with Q/A fields.
// Expected format:
//
//	[
//	  {"q": "What is AI?", "a": "AI stands for Artificial Intelligence..."},
//	  {"question": "How does it work?", "answer": "It works by..."}
//	]
//
// Supports flexible field names: q/question/问题 for questions, a/answer/答案 for answers.
func ParseQAJSON(config *contract.Config) ParseFn {
	return func(ctx context.Context, reader io.Reader, opts ...parser.Option) (docs []*schema.Document, err error) {
		b, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("[ParseQAJSON] failed to read content: %w", err)
		}

		options := parser.GetCommonOptions(&parser.Options{}, opts...)

		// Parse JSON array
		var rawItems []map[string]any
		if err = json.Unmarshal(b, &rawItems); err != nil {
			return nil, fmt.Errorf("[ParseQAJSON] invalid JSON format: %w", err)
		}

		if len(rawItems) == 0 {
			return nil, fmt.Errorf("[ParseQAJSON] empty JSON array")
		}

		// Process each QA pair
		for i, item := range rawItems {
			question := findQAFieldValue(item, questionColumnNames)
			answer := findQAFieldValue(item, answerColumnNames)

			// Skip items without question
			if question == "" {
				continue
			}

			doc := &schema.Document{
				Content:  question, // Q is embedded for vector search
				MetaData: make(map[string]any),
			}

			// Store A in metadata for retrieval
			document.WithDocumentAnswer(doc, answer)

			// Add row index for debugging
			doc.MetaData["qa_row_index"] = i

			// Add extra metadata from options
			for k, v := range options.ExtraMeta {
				doc.MetaData[k] = v
			}

			docs = append(docs, doc)
		}

		if len(docs) == 0 {
			return nil, fmt.Errorf("[ParseQAJSON] no valid QA pairs found")
		}

		return docs, nil
	}
}

// findQAColumnIndices finds the indices of Q and A columns in the header.
func findQAColumnIndices(header []string) (qIndex, aIndex int) {
	qIndex = -1
	aIndex = -1

	for i, col := range header {
		colLower := strings.ToLower(strings.TrimSpace(col))

		if qIndex == -1 {
			for _, qName := range questionColumnNames {
				if colLower == qName {
					qIndex = i
					break
				}
			}
		}

		if aIndex == -1 {
			for _, aName := range answerColumnNames {
				if colLower == aName {
					aIndex = i
					break
				}
			}
		}

		if qIndex != -1 && aIndex != -1 {
			break
		}
	}

	return qIndex, aIndex
}

// findQAFieldValue finds the value of a field by checking multiple possible field names.
func findQAFieldValue(item map[string]any, fieldNames []string) string {
	for _, name := range fieldNames {
		// Try exact match
		if val, ok := item[name]; ok {
			return anyToString(val)
		}
		// Try case-insensitive match
		for key, val := range item {
			if strings.EqualFold(key, name) {
				return anyToString(val)
			}
		}
	}
	return ""
}

// anyToString converts any value to string.
func anyToString(val any) string {
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}
