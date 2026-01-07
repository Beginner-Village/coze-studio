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

package internal

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coze-dev/coze-studio/backend/api/model/app/bot_common"
	"github.com/coze-dev/coze-studio/backend/domain/conversation/agentrun/entity"
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
	// MetaKeyGroupID is the card group identifier
	MetaKeyGroupID StreamCardMetaKey = "group_id"
	// MetaKeyCardLayout is the layout type for card group
	MetaKeyCardLayout StreamCardMetaKey = "card_layout"
	// MetaKeyCardColumns is the number of columns for card group
	MetaKeyCardColumns StreamCardMetaKey = "card_columns"
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

// CardLayout defines the layout types for card groups
type CardLayout string

const (
	// CardLayoutHorizontal arranges cards horizontally
	CardLayoutHorizontal CardLayout = "horizontal"
	// CardLayoutVertical arranges cards vertically (default)
	CardLayoutVertical CardLayout = "vertical"
	// CardLayoutWaterfall arranges cards in waterfall/masonry style
	CardLayoutWaterfall CardLayout = "waterfall"
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
	// GroupID is the unique group identifier (for group events and card_create when card belongs to a group)
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

// ProcessorFlush flushes any remaining buffered content
func (p *StreamCardProcessor) ProcessorFlush() []StreamCardOutput {
	if !p.enabled {
		return nil
	}

	events := p.parser.Flush()
	return p.convertEvents(events)
}

// ProcessorIsInCard returns true if currently parsing a card
func (p *StreamCardProcessor) ProcessorIsInCard() bool {
	return p.enabled && p.parser.IsInCard()
}

// ProcessorReset resets the processor state
func (p *StreamCardProcessor) ProcessorReset() {
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
					GroupID:      e.GroupID,
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
			case GroupEventStart:
				outputs = append(outputs, StreamCardOutput{
					Type:    string(YnetTypeCardGroupStart),
					GroupID: e.GroupID,
					Layout:  e.Layout,
					Columns: e.Columns,
				})
			case GroupEventEnd:
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
		meta[string(MetaKeyCardLayout)] = output.Layout
		meta[string(MetaKeyCardColumns)] = fmt.Sprintf("%d", output.Columns)

	case string(YnetTypeCardGroupEnd):
		meta[string(MetaKeyYnetType)] = string(YnetTypeCardGroupEnd)
		meta[string(MetaKeyGroupID)] = output.GroupID

	case string(YnetTypeCardCreate):
		meta[string(MetaKeyYnetType)] = string(YnetTypeCardCreate)
		meta[string(MetaKeyCardID)] = output.CardID
		meta[string(MetaKeyTemplateID)] = output.TemplateID
		meta[string(MetaKeyTemplateName)] = output.TemplateName
		if output.GroupID != "" {
			meta[string(MetaKeyGroupID)] = output.GroupID
		}

	case string(YnetTypeCardDelta):
		meta[string(MetaKeyYnetType)] = string(YnetTypeCardDelta)
		meta[string(MetaKeyCardID)] = output.CardID
		meta[string(MetaKeyCardField)] = output.Field
		// Put card value in meta_data, remove trailing newlines/whitespace
		meta[string(MetaKeyCardValue)] = strings.TrimRight(output.Delta, "\n\r\t ")

	case string(YnetTypeCardDone):
		meta[string(MetaKeyYnetType)] = string(YnetTypeCardDone)
		meta[string(MetaKeyCardID)] = output.CardID
	}

	return meta
}

// StreamCardHandler handles streaming card events in the message pipeline
// It buffers card_delta events and only emits them when field boundaries are crossed
type StreamCardHandler struct {
	processor *StreamCardProcessor
	enabled   bool

	// Buffering for card_delta events - only emit when field changes or card ends
	pendingDeltaCardID string
	pendingDeltaField  string
	pendingDeltaBuffer string

	// Track raw text content for final output (text outside card tags)
	rawTextContent strings.Builder
}

// NewStreamCardHandler creates a new stream card handler
func NewStreamCardHandler(boundCards []*bot_common.BoundCardInfo) *StreamCardHandler {
	// Enable streaming card processing only if there are bound cards
	enabled := len(boundCards) > 0

	return &StreamCardHandler{
		processor: NewStreamCardProcessor(enabled),
		enabled:   enabled,
	}
}

// IsEnabled returns true if streaming card processing is enabled
func (h *StreamCardHandler) IsEnabled() bool {
	return h.enabled
}

// flushPendingDelta flushes any buffered delta content as one or more outputs
// If the buffer contains multiple newline-separated JSON objects (array elements),
// each object is emitted as a separate "add" operation
func (h *StreamCardHandler) flushPendingDelta() []*StreamCardHandlerOutput {
	if h.pendingDeltaBuffer == "" || h.pendingDeltaField == "" || h.pendingDeltaCardID == "" {
		return nil
	}

	trimmedValue := strings.TrimRight(h.pendingDeltaBuffer, "\n\r\t ")

	// Check if value contains multiple newline-separated JSON objects
	// Pattern: {...}\n{...} indicates array elements that should each be "add"
	if strings.Contains(trimmedValue, "}\n{") {
		outputs := h.splitAndEmitArrayElements(trimmedValue)
		if len(outputs) > 0 {
			// Clear the buffer
			h.pendingDeltaCardID = ""
			h.pendingDeltaField = ""
			h.pendingDeltaBuffer = ""
			return outputs
		}
	}

	// Single value - determine card_op based on value format:
	// - If value starts with '{' and is a valid JSON object, use "add" (array element)
	// - Otherwise use "set" (simple value)
	cardOp := determineCardOp(trimmedValue)

	output := &StreamCardHandlerOutput{
		Type:  string(YnetTypeCardDelta),
		Delta: h.pendingDeltaBuffer,
		Meta: map[string]string{
			string(MetaKeyYnetType):  string(YnetTypeCardDelta),
			string(MetaKeyCardID):    h.pendingDeltaCardID,
			string(MetaKeyCardField): h.pendingDeltaField,
			string(MetaKeyCardValue): trimmedValue,
			string(MetaKeyCardOp):    string(cardOp),
		},
	}

	// Clear the buffer
	h.pendingDeltaCardID = ""
	h.pendingDeltaField = ""
	h.pendingDeltaBuffer = ""

	return []*StreamCardHandlerOutput{output}
}

// splitAndEmitArrayElements splits newline-separated JSON objects into multiple "add" outputs
func (h *StreamCardHandler) splitAndEmitArrayElements(value string) []*StreamCardHandlerOutput {
	var outputs []*StreamCardHandlerOutput

	lines := strings.Split(value, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		// Validate that this line is a JSON object
		if strings.HasPrefix(trimmedLine, "{") && strings.HasSuffix(trimmedLine, "}") {
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(trimmedLine), &obj); err == nil {
				// Valid JSON object - emit as "add" operation
				outputs = append(outputs, &StreamCardHandlerOutput{
					Type:  string(YnetTypeCardDelta),
					Delta: trimmedLine,
					Meta: map[string]string{
						string(MetaKeyYnetType):  string(YnetTypeCardDelta),
						string(MetaKeyCardID):    h.pendingDeltaCardID,
						string(MetaKeyCardField): h.pendingDeltaField,
						string(MetaKeyCardValue): trimmedLine,
						string(MetaKeyCardOp):    string(CardOpAdd),
					},
				})
			}
		}
	}

	return outputs
}

// tryFlushCompletedObjects checks the buffer for complete JSON objects ending with "}\n"
// and flushes them immediately for true streaming behavior.
// This allows array elements to be sent as soon as they are complete.
func (h *StreamCardHandler) tryFlushCompletedObjects() []*StreamCardHandlerOutput {
	if h.pendingDeltaBuffer == "" || h.pendingDeltaField == "" || h.pendingDeltaCardID == "" {
		return nil
	}

	var outputs []*StreamCardHandlerOutput

	// Look for complete JSON objects followed by newline: {...}\n
	// Each such object can be emitted immediately as an "add" operation
	for {
		// Find the pattern "}\n" which indicates end of a JSON object in array context
		idx := strings.Index(h.pendingDeltaBuffer, "}\n")
		if idx == -1 {
			break
		}

		// Extract the potential JSON object (everything up to and including "}")
		potentialJSON := strings.TrimSpace(h.pendingDeltaBuffer[:idx+1])

		// Check if this is a valid JSON object
		if strings.HasPrefix(potentialJSON, "{") {
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(potentialJSON), &obj); err == nil {
				// Valid JSON object - emit as "add" operation
				outputs = append(outputs, &StreamCardHandlerOutput{
					Type:  string(YnetTypeCardDelta),
					Delta: potentialJSON,
					Meta: map[string]string{
						string(MetaKeyYnetType):  string(YnetTypeCardDelta),
						string(MetaKeyCardID):    h.pendingDeltaCardID,
						string(MetaKeyCardField): h.pendingDeltaField,
						string(MetaKeyCardValue): potentialJSON,
						string(MetaKeyCardOp):    string(CardOpAdd),
					},
				})

				// Remove the flushed content from buffer (including the newline)
				h.pendingDeltaBuffer = h.pendingDeltaBuffer[idx+2:]
				continue
			}
		}

		// Not a valid JSON object, stop looking
		break
	}

	return outputs
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

// ProcessContent processes streaming content and returns outputs with metadata
// Returns: (outputs, shouldSkipOriginal)
// - outputs: list of processed outputs (text or card events)
// - shouldSkipOriginal: if true, caller should not send original content
//
// For card_delta events, this method buffers content and only emits when:
// - A different field starts
// - Card ends (card_done)
// - Non-card content arrives
func (h *StreamCardHandler) ProcessContent(content string) ([]StreamCardHandlerOutput, bool) {
	if !h.enabled {
		return nil, false // Let caller handle normally
	}

	outputs := h.processor.Process(content)
	if len(outputs) == 0 {
		return nil, true // Content is buffered in parser, skip original
	}

	var result []StreamCardHandlerOutput
	for _, out := range outputs {
		switch out.Type {
		case string(YnetTypeCardGroupStart):
			// Group start - flush any pending delta first, then emit group_start
			for _, flushed := range h.flushPendingDelta() {
				result = append(result, *flushed)
			}
			result = append(result, StreamCardHandlerOutput{
				Type:    out.Type,
				Content: "",
				Delta:   "",
				Meta:    BuildCardMetaData(out),
			})

		case string(YnetTypeCardGroupEnd):
			// Group end - flush any pending delta first, then emit group_end
			for _, flushed := range h.flushPendingDelta() {
				result = append(result, *flushed)
			}
			result = append(result, StreamCardHandlerOutput{
				Type:    out.Type,
				Content: "",
				Delta:   "",
				Meta:    BuildCardMetaData(out),
			})

		case string(YnetTypeCardDelta):
			// Check if this is a different field - if so, flush previous buffer first
			if h.pendingDeltaField != "" && h.pendingDeltaField != out.Field {
				for _, flushed := range h.flushPendingDelta() {
					result = append(result, *flushed)
				}
			}
			// Accumulate delta content
			h.pendingDeltaCardID = out.CardID
			h.pendingDeltaField = out.Field
			h.pendingDeltaBuffer += out.Delta

			// Check if we have a complete JSON object that can be flushed immediately
			// This enables true streaming for array elements: emit each object as soon as it's complete
			if completedObjects := h.tryFlushCompletedObjects(); len(completedObjects) > 0 {
				for _, flushed := range completedObjects {
					result = append(result, *flushed)
				}
			}

		case string(YnetTypeCardCreate):
			// Card create - emit directly
			result = append(result, StreamCardHandlerOutput{
				Type:    out.Type,
				Content: out.Text,
				Delta:   out.Delta,
				Meta:    BuildCardMetaData(out),
			})

		case string(YnetTypeCardDone):
			// Card done - flush any pending delta first, then emit card_done
			for _, flushed := range h.flushPendingDelta() {
				result = append(result, *flushed)
			}
			result = append(result, StreamCardHandlerOutput{
				Type:    out.Type,
				Content: out.Text,
				Delta:   out.Delta,
				Meta:    BuildCardMetaData(out),
			})

		case "text":
			// Text content - flush any pending delta first, then emit text
			for _, flushed := range h.flushPendingDelta() {
				result = append(result, *flushed)
			}
			// Accumulate raw text content for final output
			h.rawTextContent.WriteString(out.Text)
			result = append(result, StreamCardHandlerOutput{
				Type:    out.Type,
				Content: out.Text,
				Delta:   out.Delta,
				Meta:    BuildCardMetaData(out),
			})
		}
	}

	return result, true
}

// Flush flushes any remaining buffered content
func (h *StreamCardHandler) Flush() []StreamCardHandlerOutput {
	if !h.enabled {
		return nil
	}

	var result []StreamCardHandlerOutput

	// Flush pending delta buffer first
	for _, flushed := range h.flushPendingDelta() {
		result = append(result, *flushed)
	}

	// Then flush the processor
	outputs := h.processor.ProcessorFlush()
	for _, out := range outputs {
		result = append(result, StreamCardHandlerOutput{
			Type:    out.Type,
			Content: out.Text,
			Delta:   out.Delta,
			Meta:    BuildCardMetaData(out),
		})
	}

	return result
}

// IsInCard returns true if currently parsing a card
func (h *StreamCardHandler) IsInCard() bool {
	return h.processor.ProcessorIsInCard()
}

// StreamCardHandlerOutput represents processed output from the handler
type StreamCardHandlerOutput struct {
	Type    string            // "text", "card_create", "card_delta", "card_done"
	Content string            // Text content (for "text" type)
	Delta   string            // Delta content (for "card_delta" type)
	Meta    map[string]string // Metadata to add to the message
}

// ApplyToMessage applies the output to a ChunkMessageItem
func (o *StreamCardHandlerOutput) ApplyToMessage(msg *entity.ChunkMessageItem) {
	if msg.Ext == nil {
		msg.Ext = make(map[string]string)
	}

	switch o.Type {
	case "text":
		msg.Content = o.Content
	case string(YnetTypeCardGroupStart),
		string(YnetTypeCardGroupEnd),
		string(YnetTypeCardCreate),
		string(YnetTypeCardDelta),
		string(YnetTypeCardDone):
		// For card and group events, set content to empty and merge metadata
		// card_delta value is now in meta_data.card_value, not in content/answer
		msg.Content = ""
		for k, v := range o.Meta {
			msg.Ext[k] = v
		}
	}
}

// GetOutputContent returns the content to use for the output
func (o *StreamCardHandlerOutput) GetOutputContent() string {
	if o.Type == "text" {
		return o.Content
	}
	// card_delta value is now in meta_data.card_value, not in content
	return ""
}

// IsCardEvent returns true if this is a card or group event
func (o *StreamCardHandlerOutput) IsCardEvent() bool {
	return o.Type == string(YnetTypeCardGroupStart) ||
		o.Type == string(YnetTypeCardGroupEnd) ||
		o.Type == string(YnetTypeCardCreate) ||
		o.Type == string(YnetTypeCardDelta) ||
		o.Type == string(YnetTypeCardDone)
}

// ShouldSend returns true if this output should be sent to the client
func (o *StreamCardHandlerOutput) ShouldSend() bool {
	// Always send card events
	if o.IsCardEvent() {
		return true
	}
	// Only send text if it has content
	return o.Type == "text" && len(o.Content) > 0
}

// CardContentItem represents a single card content item for JSON output
type CardContentItem struct {
	DisplayResponseType string            `json:"displayResponseType"`
	TemplateID          string            `json:"templateId"`
	CardID              string            `json:"cardId,omitempty"`  // Unique card identifier for group association
	KvMap               map[string]string `json:"kvMap"`
	GroupID             string            `json:"groupId,omitempty"` // ID of the group this card belongs to
}

// CardGroupItem represents a card group for JSON output
type CardGroupItem struct {
	GroupID string   `json:"groupId"`
	Layout  string   `json:"layout"`  // "horizontal", "vertical", "waterfall"
	Columns int      `json:"columns"` // Number of columns
	CardIDs []string `json:"cardIds"` // IDs of cards in this group
}

// CardContentWrapper represents the wrapper structure for card content
type CardContentWrapper struct {
	ContentList []CardContentItem `json:"contentList"`
	Groups      []CardGroupItem   `json:"groups,omitempty"`     // Optional group layout information
	RawContent  string            `json:"rawContent,omitempty"` // Original text content (non-card content)
}

// HasCompletedCards returns true if there are any completed cards
func (h *StreamCardHandler) HasCompletedCards() bool {
	if !h.enabled {
		return false
	}
	return h.processor.parser.HasCompletedCards()
}

// GetCompletedCards returns all completed cards
func (h *StreamCardHandler) GetCompletedCards() []*CardState {
	if !h.enabled {
		return nil
	}
	return h.processor.parser.GetCompletedCards()
}

// HasCompletedGroups returns true if there are any completed groups
func (h *StreamCardHandler) HasCompletedGroups() bool {
	if !h.enabled {
		return false
	}
	return h.processor.parser.HasCompletedGroups()
}

// GetCompletedGroups returns all completed groups
func (h *StreamCardHandler) GetCompletedGroups() []*GroupState {
	if !h.enabled {
		return nil
	}
	return h.processor.parser.GetCompletedGroups()
}

// BuildFinalContent builds the final JSON content for storage
// This converts the parsed card data into SpecialAnswerContent-compatible format
func (h *StreamCardHandler) BuildFinalContent() (string, error) {
	if !h.enabled {
		return "", nil
	}

	cards := h.processor.parser.GetCompletedCards()
	if len(cards) == 0 {
		return "", nil
	}

	// Build content list from completed cards
	contentList := make([]CardContentItem, len(cards))
	for i, card := range cards {
		contentList[i] = CardContentItem{
			DisplayResponseType: "TEMPLATE",
			TemplateID:          card.TemplateID,
			CardID:              card.ID, // Include card ID for group association
			KvMap:               card.Fields,
			GroupID:             card.GroupID,
		}
	}

	// Build groups list from completed groups
	groups := h.processor.parser.GetCompletedGroups()
	var groupList []CardGroupItem
	if len(groups) > 0 {
		groupList = make([]CardGroupItem, len(groups))
		for i, group := range groups {
			groupList[i] = CardGroupItem{
				GroupID: group.ID,
				Layout:  group.Layout,
				Columns: group.Columns,
				CardIDs: group.CardIDs,
			}
		}
	}

	// Get accumulated raw text content (text outside card tags)
	rawContent := strings.TrimSpace(h.rawTextContent.String())

	// Create wrapper structure
	wrapper := CardContentWrapper{
		ContentList: contentList,
		Groups:      groupList,
		RawContent:  rawContent,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(wrapper)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}
