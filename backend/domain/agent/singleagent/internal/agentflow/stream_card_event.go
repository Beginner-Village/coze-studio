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

package agentflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// StreamCardMetaKey defines metadata keys for streaming card events
type StreamCardMetaKey string

const (
	// MetaKeyYnetType is the message type key
	MetaKeyYnetType StreamCardMetaKey = "ynet_type"
	// MetaKeyCardID is the unique card identifier
	MetaKeyCardID StreamCardMetaKey = "card_id"
	// MetaKeyTemplateID is the card template identifier
	MetaKeyTemplateID StreamCardMetaKey = "template_id"
	// MetaKeyTemplateName is the card template name
	MetaKeyTemplateName StreamCardMetaKey = "template_name"
	// MetaKeyCardField is the current field name being updated
	MetaKeyCardField StreamCardMetaKey = "card_field"
	// MetaKeyCardValue is the field value for card_delta events
	MetaKeyCardValue StreamCardMetaKey = "card_value"
	// MetaKeyCardOp is the operation type for card_delta events: "add" for array elements, "set" for simple values
	MetaKeyCardOp StreamCardMetaKey = "card_op"
	// MetaKeyGroupID is the unique group identifier
	MetaKeyGroupID StreamCardMetaKey = "group_id"
	// MetaKeyLayout is the layout type for card group
	MetaKeyLayout StreamCardMetaKey = "layout"
	// MetaKeyColumns is the number of columns for card group
	MetaKeyColumns StreamCardMetaKey = "columns"
)

// CardOp defines operation types for card_delta events
type CardOp string

const (
	// CardOpAdd appends value to an array field
	CardOpAdd CardOp = "add"
	// CardOpSet sets/replaces a field value
	CardOpSet CardOp = "set"
)

// StreamCardYnetType defines ynet_type values for card events
type StreamCardYnetType string

const (
	// YnetTypeCardGroupStart indicates a card group is starting
	YnetTypeCardGroupStart StreamCardYnetType = "card_group_start"
	// YnetTypeCardGroupEnd indicates a card group has ended
	YnetTypeCardGroupEnd StreamCardYnetType = "card_group_end"
	// YnetTypeCardCreate indicates a new card is being created
	YnetTypeCardCreate StreamCardYnetType = "card_create"
	// YnetTypeCardDelta indicates a card field is being updated incrementally
	YnetTypeCardDelta StreamCardYnetType = "card_delta"
	// YnetTypeCardDone indicates a card has completed
	YnetTypeCardDone StreamCardYnetType = "card_done"
)

// StreamCardOutput represents the output from card stream processing
type StreamCardOutput struct {
	// Type indicates the type of output: "text", "card_create", "card_delta", "card_done", "card_group_start", "card_group_end"
	Type string
	// Text is the text content for text output
	Text string
	// CardID is the unique card identifier
	CardID string
	// TemplateID is the card template ID (for card_create)
	TemplateID string
	// TemplateName is the card template name (for card_create)
	TemplateName string
	// Field is the field name (for card_delta)
	Field string
	// Delta is the incremental content (for card_delta)
	Delta string
	// GroupID is the unique group identifier (for card_group_start/end)
	GroupID string
	// Layout is the layout type for card group (for card_group_start)
	Layout string
	// Columns is the number of columns for card group (for card_group_start)
	Columns int
}

// StreamCardProcessor wraps StreamCardParser for integration with message pipeline
type StreamCardProcessor struct {
	parser  *StreamCardParser
	enabled bool
}

// NewStreamCardProcessor creates a new stream card processor
func NewStreamCardProcessor(enabled bool) *StreamCardProcessor {
	return &StreamCardProcessor{
		parser:  NewStreamCardParser(),
		enabled: enabled,
	}
}

// Process processes a text chunk and returns outputs
func (p *StreamCardProcessor) Process(text string) []StreamCardOutput {
	if !p.enabled {
		// If card parsing is disabled, just return text as-is
		if text != "" {
			return []StreamCardOutput{{Type: "text", Text: text}}
		}
		return nil
	}

	events := p.parser.Feed(text)
	return p.convertEvents(events)
}

// Flush flushes any remaining buffered content
func (p *StreamCardProcessor) Flush() []StreamCardOutput {
	if !p.enabled {
		return nil
	}

	events := p.parser.Flush()
	return p.convertEvents(events)
}

// IsInCard returns true if currently parsing a card
func (p *StreamCardProcessor) IsInCard() bool {
	return p.enabled && p.parser.IsInCard()
}

// Reset resets the processor state
func (p *StreamCardProcessor) Reset() {
	if p.enabled {
		p.parser.Reset()
	}
}

// convertEvents converts StreamEvent to StreamCardOutput
func (p *StreamCardProcessor) convertEvents(events []StreamEvent) []StreamCardOutput {
	var outputs []StreamCardOutput

	for _, event := range events {
		switch e := event.(type) {
		case TextEvent:
			outputs = append(outputs, StreamCardOutput{
				Type: "text",
				Text: e.Text,
			})
		case CardEvent:
			switch e.Type {
			case CardEventCreate:
				outputs = append(outputs, StreamCardOutput{
					Type:         string(YnetTypeCardCreate),
					CardID:       e.CardID,
					TemplateID:   e.TemplateID,
					TemplateName: e.TemplateName,
				})
			case CardEventDelta:
				outputs = append(outputs, StreamCardOutput{
					Type:   string(YnetTypeCardDelta),
					CardID: e.CardID,
					Field:  e.Field,
					Delta:  e.Delta,
				})
			case CardEventDone:
				outputs = append(outputs, StreamCardOutput{
					Type:   string(YnetTypeCardDone),
					CardID: e.CardID,
				})
			}
		case GroupEvent:
			switch e.Type {
			case CardEventGroupStart:
				outputs = append(outputs, StreamCardOutput{
					Type:    string(YnetTypeCardGroupStart),
					GroupID: e.GroupID,
					Layout:  e.Layout,
					Columns: e.Columns,
				})
			case CardEventGroupEnd:
				outputs = append(outputs, StreamCardOutput{
					Type:    string(YnetTypeCardGroupEnd),
					GroupID: e.GroupID,
				})
			}
		}
	}

	return outputs
}

// BuildCardMetaData builds metadata map for a card output
func BuildCardMetaData(output StreamCardOutput) map[string]string {
	meta := make(map[string]string)

	switch output.Type {
	case string(YnetTypeCardGroupStart):
		meta[string(MetaKeyYnetType)] = string(YnetTypeCardGroupStart)
		meta[string(MetaKeyGroupID)] = output.GroupID
		meta[string(MetaKeyLayout)] = output.Layout
		meta[string(MetaKeyColumns)] = fmt.Sprintf("%d", output.Columns)

	case string(YnetTypeCardGroupEnd):
		meta[string(MetaKeyYnetType)] = string(YnetTypeCardGroupEnd)
		meta[string(MetaKeyGroupID)] = output.GroupID

	case string(YnetTypeCardCreate):
		meta[string(MetaKeyYnetType)] = string(YnetTypeCardCreate)
		meta[string(MetaKeyCardID)] = output.CardID
		meta[string(MetaKeyTemplateID)] = output.TemplateID
		meta[string(MetaKeyTemplateName)] = output.TemplateName

	case string(YnetTypeCardDelta):
		meta[string(MetaKeyYnetType)] = string(YnetTypeCardDelta)
		meta[string(MetaKeyCardID)] = output.CardID
		meta[string(MetaKeyCardField)] = output.Field
		// Put card value in meta_data, remove trailing newlines/whitespace
		trimmedValue := strings.TrimRight(output.Delta, "\n\r\t ")
		meta[string(MetaKeyCardValue)] = trimmedValue
		// Determine card_op based on value format
		meta[string(MetaKeyCardOp)] = string(determineCardOp(trimmedValue))

	case string(YnetTypeCardDone):
		meta[string(MetaKeyYnetType)] = string(YnetTypeCardDone)
		meta[string(MetaKeyCardID)] = output.CardID
	}

	return meta
}

// determineCardOp determines the operation type based on value format
func determineCardOp(value string) CardOp {
	trimmed := strings.TrimSpace(value)

	// If value starts with '{' and is valid JSON object, it's an array element (add)
	if strings.HasPrefix(trimmed, "{") {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &obj); err == nil {
			return CardOpAdd
		}
	}

	// Otherwise it's a simple value (set)
	return CardOpSet
}
