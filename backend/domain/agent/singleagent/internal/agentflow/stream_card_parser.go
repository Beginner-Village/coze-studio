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

package agentflow

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// CardEventType defines the type of card streaming event
type CardEventType string

const (
	CardEventCreate     CardEventType = "card_create"      // Create a new card
	CardEventDelta      CardEventType = "card_delta"       // Update card field incrementally
	CardEventDone       CardEventType = "card_done"        // Card is complete
	CardEventGroupStart CardEventType = "card_group_start" // Start a card group
	CardEventGroupEnd   CardEventType = "card_group_end"   // End a card group
)

// StreamEvent represents an event emitted by the stream parser
type StreamEvent interface {
	isStreamEvent()
}

// TextEvent represents a plain text chunk
type TextEvent struct {
	Text string
}

func (TextEvent) isStreamEvent() {}

// CardEvent represents a card-related event
type CardEvent struct {
	Type         CardEventType
	CardID       string
	TemplateID   string // Only for CardEventCreate
	TemplateName string // Only for CardEventCreate
	Field        string // Only for CardEventDelta
	Delta        string // Only for CardEventDelta (incremental content)
}

func (CardEvent) isStreamEvent() {}

// GroupEvent represents a card group event
type GroupEvent struct {
	Type    CardEventType
	GroupID string
	Layout  string // horizontal, vertical, waterfall
	Columns int    // Number of columns
}

func (GroupEvent) isStreamEvent() {}

// CardState tracks the state of a card being parsed
type CardState struct {
	ID           string
	TemplateID   string
	TemplateName string
	Fields       map[string]string
}

// ParserState represents the state of the stream parser
type ParserState int

const (
	StateIdle     ParserState = iota // Normal text mode
	StateMaybeTag                    // Encountered '<', might be a tag
	StateInTag                       // Confirmed '<<', reading tag content
	StateInCard                      // Inside a card, reading field values
)

// GroupState tracks the state of a group being parsed
type GroupState struct {
	ID      string
	Layout  string
	Columns int
}

// StreamCardParser parses streaming LLM output and extracts card events
type StreamCardParser struct {
	mu           sync.Mutex
	state        ParserState
	buffer       strings.Builder
	currentCard  *CardState
	currentGroup *GroupState
	currentField string
	cardIDGen    func() string
	groupIDGen   func() string
}

// NewStreamCardParser creates a new stream card parser
func NewStreamCardParser() *StreamCardParser {
	return &StreamCardParser{
		state:      StateIdle,
		cardIDGen:  defaultCardIDGen,
		groupIDGen: defaultGroupIDGen,
	}
}

// defaultCardIDGen generates a unique card ID
func defaultCardIDGen() string {
	return fmt.Sprintf("card_%d", time.Now().UnixNano())
}

// defaultGroupIDGen generates a unique group ID
func defaultGroupIDGen() string {
	return fmt.Sprintf("group_%d", time.Now().UnixNano())
}

// SetCardIDGenerator sets a custom card ID generator
func (p *StreamCardParser) SetCardIDGenerator(gen func() string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cardIDGen = gen
}

// Feed processes a chunk of streaming input and returns events
func (p *StreamCardParser) Feed(chunk string) []StreamEvent {
	p.mu.Lock()
	defer p.mu.Unlock()

	var events []StreamEvent

	for _, char := range chunk {
		charEvents := p.processChar(char)
		events = append(events, charEvents...)
	}

	return events
}

// processChar processes a single character and returns any resulting events
func (p *StreamCardParser) processChar(char rune) []StreamEvent {
	var events []StreamEvent

	switch p.state {
	case StateIdle:
		events = p.handleIdle(char)

	case StateMaybeTag:
		events = p.handleMaybeTag(char)

	case StateInTag:
		events = p.handleInTag(char)

	case StateInCard:
		events = p.handleInCard(char)
	}

	return events
}

// handleIdle processes characters in idle (normal text) state
func (p *StreamCardParser) handleIdle(char rune) []StreamEvent {
	if char == '<' {
		p.state = StateMaybeTag
		p.buffer.WriteRune(char)
		return nil
	}

	// Normal text, emit directly
	return []StreamEvent{TextEvent{Text: string(char)}}
}

// handleMaybeTag processes characters when we might be starting a tag
func (p *StreamCardParser) handleMaybeTag(char rune) []StreamEvent {
	if char == '<' {
		// Confirmed '<<' - this is a tag start
		p.state = StateInTag
		p.buffer.WriteRune(char)
		return nil
	}

	// Not a tag, emit buffered '<' and current char as text
	text := p.buffer.String() + string(char)
	p.buffer.Reset()
	p.state = StateIdle
	return []StreamEvent{TextEvent{Text: text}}
}

// handleInTag processes characters while reading tag content
func (p *StreamCardParser) handleInTag(char rune) []StreamEvent {
	p.buffer.WriteRune(char)

	// Check if tag is complete (ends with '>>')
	content := p.buffer.String()
	if strings.HasSuffix(content, ">>") {
		events := p.parseTag(content)
		p.buffer.Reset()
		return events
	}

	return nil
}

// handleInCard processes characters inside a card (field values)
func (p *StreamCardParser) handleInCard(char rune) []StreamEvent {
	if char == '<' {
		p.state = StateMaybeTag
		p.buffer.WriteRune(char)
		return nil
	}

	// Field value content
	if p.currentField != "" && p.currentCard != nil {
		// Accumulate field value
		p.currentCard.Fields[p.currentField] += string(char)

		// Emit card delta event
		return []StreamEvent{
			CardEvent{
				Type:   CardEventDelta,
				CardID: p.currentCard.ID,
				Field:  p.currentField,
				Delta:  string(char),
			},
		}
	}

	return nil
}

// parseTag parses a complete tag and returns appropriate events
func (p *StreamCardParser) parseTag(tag string) []StreamEvent {
	var events []StreamEvent

	// Remove << and >> delimiters
	inner := strings.TrimPrefix(tag, "<<")
	inner = strings.TrimSuffix(inner, ">>")

	// Check for group start: <<GROUP:layout:columns>> e.g., <<GROUP:horizontal:2>>
	if strings.HasPrefix(inner, "GROUP:") {
		parts := strings.SplitN(inner[6:], ":", 2)
		layout := "horizontal"
		columns := 2
		if len(parts) >= 1 {
			layout = parts[0]
		}
		if len(parts) >= 2 {
			if n, err := fmt.Sscanf(parts[1], "%d", &columns); err != nil || n != 1 {
				columns = 2
			}
		}

		// Create new group state
		p.currentGroup = &GroupState{
			ID:      p.groupIDGen(),
			Layout:  layout,
			Columns: columns,
		}
		p.state = StateIdle // Stay in idle to allow card parsing

		events = append(events, GroupEvent{
			Type:    CardEventGroupStart,
			GroupID: p.currentGroup.ID,
			Layout:  layout,
			Columns: columns,
		})
		return events
	}

	// Check for group end: <</GROUP>>
	if inner == "/GROUP" {
		if p.currentGroup != nil {
			events = append(events, GroupEvent{
				Type:    CardEventGroupEnd,
				GroupID: p.currentGroup.ID,
			})
			p.currentGroup = nil
		}
		p.state = StateIdle
		return events
	}

	// Check for card start: <<CARD:template_id:template_name>>
	if strings.HasPrefix(inner, "CARD:") {
		parts := strings.SplitN(inner[5:], ":", 2)
		if len(parts) >= 1 {
			templateID := parts[0]
			templateName := ""
			if len(parts) >= 2 {
				templateName = parts[1]
			}

			// Create new card state
			p.currentCard = &CardState{
				ID:           p.cardIDGen(),
				TemplateID:   templateID,
				TemplateName: templateName,
				Fields:       make(map[string]string),
			}
			p.currentField = ""
			p.state = StateInCard

			events = append(events, CardEvent{
				Type:         CardEventCreate,
				CardID:       p.currentCard.ID,
				TemplateID:   templateID,
				TemplateName: templateName,
			})
		}
		return events
	}

	// Check for card end: <</CARD>>
	if inner == "/CARD" {
		if p.currentCard != nil {
			events = append(events, CardEvent{
				Type:   CardEventDone,
				CardID: p.currentCard.ID,
			})
			p.currentCard = nil
			p.currentField = ""
		}
		// If in a group, stay in idle to allow next card; otherwise go to idle
		p.state = StateIdle
		return events
	}

	// Field tag: <<field_name>>
	if p.currentCard != nil && !strings.HasPrefix(inner, "/") {
		p.currentField = inner
		// Initialize field if not exists
		if _, ok := p.currentCard.Fields[p.currentField]; !ok {
			p.currentCard.Fields[p.currentField] = ""
		}
		p.state = StateInCard
	}

	return events
}

// Flush returns any remaining buffered content as events
func (p *StreamCardParser) Flush() []StreamEvent {
	p.mu.Lock()
	defer p.mu.Unlock()

	var events []StreamEvent

	// If there's buffered content, emit it as text
	if p.buffer.Len() > 0 {
		events = append(events, TextEvent{Text: p.buffer.String()})
		p.buffer.Reset()
	}

	// If a card is still open, emit done event
	if p.currentCard != nil {
		events = append(events, CardEvent{
			Type:   CardEventDone,
			CardID: p.currentCard.ID,
		})
		p.currentCard = nil
	}

	// If a group is still open, emit group end event
	if p.currentGroup != nil {
		events = append(events, GroupEvent{
			Type:    CardEventGroupEnd,
			GroupID: p.currentGroup.ID,
		})
		p.currentGroup = nil
	}

	p.state = StateIdle
	p.currentField = ""

	return events
}

// Reset resets the parser to its initial state
func (p *StreamCardParser) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.state = StateIdle
	p.buffer.Reset()
	p.currentCard = nil
	p.currentGroup = nil
	p.currentField = ""
}

// GetCurrentCard returns the current card being parsed (if any)
func (p *StreamCardParser) GetCurrentCard() *CardState {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentCard == nil {
		return nil
	}

	// Return a copy
	cardCopy := &CardState{
		ID:           p.currentCard.ID,
		TemplateID:   p.currentCard.TemplateID,
		TemplateName: p.currentCard.TemplateName,
		Fields:       make(map[string]string),
	}
	for k, v := range p.currentCard.Fields {
		cardCopy.Fields[k] = v
	}
	return cardCopy
}

// IsInCard returns true if parser is currently inside a card
func (p *StreamCardParser) IsInCard() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentCard != nil
}
