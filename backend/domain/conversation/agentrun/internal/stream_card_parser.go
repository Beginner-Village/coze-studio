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
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CardEventType defines the type of card streaming event
type CardEventType string

const (
	CardEventCreate CardEventType = "card_create" // Create a new card
	CardEventDelta  CardEventType = "card_delta"  // Update card field incrementally
	CardEventDone   CardEventType = "card_done"   // Card is complete
)

// GroupEventType defines the type of card group streaming event
type GroupEventType string

const (
	GroupEventStart GroupEventType = "group_start" // Start a card group
	GroupEventEnd   GroupEventType = "group_end"   // End a card group
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
	GroupID      string // Only for CardEventCreate - ID of the group this card belongs to
	Field        string // Only for CardEventDelta
	Delta        string // Only for CardEventDelta (incremental content)
}

func (CardEvent) isStreamEvent() {}

// GroupEvent represents a card group event for layout control
type GroupEvent struct {
	Type    GroupEventType
	GroupID string
	Layout  string // Only for GroupEventStart: "horizontal", "vertical", "waterfall"
	Columns int    // Only for GroupEventStart: number of columns (e.g., 2)
}

func (GroupEvent) isStreamEvent() {}

// CardState tracks the state of a card being parsed
type CardState struct {
	ID           string
	TemplateID   string
	TemplateName string
	Fields       map[string]string
	GroupID      string // ID of the group this card belongs to (empty if not in a group)
}

// GroupState tracks the state of a card group being parsed
type GroupState struct {
	ID      string
	Layout  string // "horizontal", "vertical", "waterfall"
	Columns int    // Number of columns for layout
	CardIDs []string
}

// ParserState represents the state of the stream parser
type ParserState int

const (
	StateIdle          ParserState = iota // Normal text mode
	StateMaybeTag                         // Encountered '<', might be a tag
	StateInTag                            // Confirmed '<<', reading tag content
	StateInCard                           // Inside a card, reading field values
	StateInSingleTag                      // Inside a card, reading single-bracket tag <field_name>
)

// StreamCardParser parses streaming LLM output and extracts card events
type StreamCardParser struct {
	mu              sync.Mutex
	state           ParserState
	buffer          strings.Builder
	currentCard     *CardState
	currentField    string
	cardIDGen       func() string
	groupIDGen      func() string
	completedCards  []*CardState  // Store completed cards for final content generation
	currentGroup    *GroupState   // Current group being parsed
	completedGroups []*GroupState // Store completed groups for final content generation
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
// This method merges consecutive card_delta events for the same field within a single chunk
func (p *StreamCardParser) Feed(chunk string) []StreamEvent {
	p.mu.Lock()
	defer p.mu.Unlock()

	var events []StreamEvent
	var pendingDelta strings.Builder
	var pendingField string
	var pendingCardID string

	// Helper to flush accumulated delta events
	flushPendingDelta := func() {
		if pendingDelta.Len() > 0 && pendingField != "" && pendingCardID != "" {
			events = append(events, CardEvent{
				Type:   CardEventDelta,
				CardID: pendingCardID,
				Field:  pendingField,
				Delta:  pendingDelta.String(),
			})
			pendingDelta.Reset()
			pendingField = ""
			pendingCardID = ""
		}
	}

	for _, char := range chunk {
		charEvents := p.processChar(char)

		// Merge consecutive card_delta events for the same field
		for _, event := range charEvents {
			if cardEvent, ok := event.(CardEvent); ok && cardEvent.Type == CardEventDelta {
				// If same field and card, accumulate delta
				if pendingField == cardEvent.Field && pendingCardID == cardEvent.CardID {
					pendingDelta.WriteString(cardEvent.Delta)
				} else {
					// Different field or card, flush previous and start new accumulation
					flushPendingDelta()
					pendingField = cardEvent.Field
					pendingCardID = cardEvent.CardID
					pendingDelta.WriteString(cardEvent.Delta)
				}
			} else {
				// Non-delta event, flush any pending delta first
				flushPendingDelta()
				events = append(events, event)
			}
		}
	}

	// Flush any remaining delta
	flushPendingDelta()

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

	case StateInSingleTag:
		events = p.handleInSingleTag(char)
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
		// Confirmed '<<' - this is a double-bracket tag start
		p.state = StateInTag
		p.buffer.WriteRune(char)
		return nil
	}

	// Not a double-bracket tag
	// If we're inside a card, try to parse single-bracket field tag <field_name>
	if p.currentCard != nil {
		// Start collecting potential single-bracket field tag
		// Buffer already has '<', now add the current char
		p.buffer.WriteRune(char)
		p.state = StateInSingleTag
		return nil
	}

	// Outside card, emit buffered '<' and current char as text
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

// handleInSingleTag processes characters while reading single-bracket tag <field_name>
// This is used inside a card to support LLM outputs that use single brackets for field names
func (p *StreamCardParser) handleInSingleTag(char rune) []StreamEvent {
	// Check for tag completion
	if char == '>' {
		// Tag is complete, extract the content
		content := p.buffer.String()
		p.buffer.Reset()

		// Remove '<' prefix to get the tag name
		// Buffer contains "<NAME" after '<' + 'N' + 'A' + 'M' + 'E'
		if len(content) > 1 && content[0] == '<' {
			tagName := content[1:]

			// Check if this is a closing tag </CARD> - handle it specially
			if tagName == "/CARD" {
				// This is the end of the card
				var events []StreamEvent
				if p.currentCard != nil {
					// Save completed card
					completedCard := &CardState{
						ID:           p.currentCard.ID,
						TemplateID:   p.currentCard.TemplateID,
						TemplateName: p.currentCard.TemplateName,
						Fields:       make(map[string]string),
						GroupID:      p.currentCard.GroupID,
					}
					for k, v := range p.currentCard.Fields {
						completedCard.Fields[k] = v
					}
					p.completedCards = append(p.completedCards, completedCard)

					events = append(events, CardEvent{
						Type:   CardEventDone,
						CardID: p.currentCard.ID,
					})
					p.currentCard = nil
					p.currentField = ""
				}
				p.state = StateIdle
				return events
			}

			// Validate tag name (alphanumeric, underscore, allowed for field names)
			if isValidFieldName(tagName) {
				// This is a valid field tag, set currentField
				p.currentField = tagName
				if _, ok := p.currentCard.Fields[p.currentField]; !ok {
					p.currentCard.Fields[p.currentField] = ""
				}
				p.state = StateInCard
				return nil
			}
		}

		// Not a valid field tag, output as card delta content
		// Include the full tag text as field content
		text := content + string(char) // content + '>'
		p.state = StateInCard

		if p.currentField != "" && p.currentCard != nil {
			p.currentCard.Fields[p.currentField] += text
			return []StreamEvent{
				CardEvent{
					Type:   CardEventDelta,
					CardID: p.currentCard.ID,
					Field:  p.currentField,
					Delta:  text,
				},
			}
		}
		return nil
	}

	// Check for nested '<' which indicates this might be a double-bracket tag
	if char == '<' {
		// This could be the start of <<TAG>>
		// Flush buffer as field content and handle the new '<'
		content := p.buffer.String()
		p.buffer.Reset()
		p.buffer.WriteRune(char)
		p.state = StateMaybeTag

		// Output buffered content as field delta
		if p.currentField != "" && p.currentCard != nil && len(content) > 0 {
			p.currentCard.Fields[p.currentField] += content
			return []StreamEvent{
				CardEvent{
					Type:   CardEventDelta,
					CardID: p.currentCard.ID,
					Field:  p.currentField,
					Delta:  content,
				},
			}
		}
		return nil
	}

	// Continue collecting tag content
	p.buffer.WriteRune(char)
	return nil
}

// isValidFieldName checks if a string is a valid field name
// Valid field names contain only letters, digits, and underscores
func isValidFieldName(name string) bool {
	if len(name) == 0 {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

// parseTag parses a complete tag and returns appropriate events
func (p *StreamCardParser) parseTag(tag string) []StreamEvent {
	var events []StreamEvent

	// Remove << and >> delimiters
	inner := strings.TrimPrefix(tag, "<<")
	inner = strings.TrimSuffix(inner, ">>")

	// Check for group start: <<GROUP:layout:columns>>
	// Example: <<GROUP:horizontal:2>> or <<GROUP:waterfall:3>>
	if strings.HasPrefix(inner, "GROUP:") {
		parts := strings.SplitN(inner[6:], ":", 2)
		layout := "vertical" // Default layout
		columns := 1         // Default columns
		if len(parts) >= 1 {
			layout = parts[0]
		}
		if len(parts) >= 2 {
			if cols, err := strconv.Atoi(parts[1]); err == nil && cols > 0 {
				columns = cols
			}
		}

		// Create new group state
		p.currentGroup = &GroupState{
			ID:      p.groupIDGen(),
			Layout:  layout,
			Columns: columns,
			CardIDs: make([]string, 0),
		}

		events = append(events, GroupEvent{
			Type:    GroupEventStart,
			GroupID: p.currentGroup.ID,
			Layout:  layout,
			Columns: columns,
		})
		// CRITICAL: Must return to StateIdle so subsequent CARD tags can be parsed correctly
		p.state = StateIdle
		return events
	}

	// Check for group end: <</GROUP>>
	if inner == "/GROUP" {
		if p.currentGroup != nil {
			// Save completed group
			completedGroup := &GroupState{
				ID:      p.currentGroup.ID,
				Layout:  p.currentGroup.Layout,
				Columns: p.currentGroup.Columns,
				CardIDs: make([]string, len(p.currentGroup.CardIDs)),
			}
			copy(completedGroup.CardIDs, p.currentGroup.CardIDs)
			p.completedGroups = append(p.completedGroups, completedGroup)

			events = append(events, GroupEvent{
				Type:    GroupEventEnd,
				GroupID: p.currentGroup.ID,
			})
			p.currentGroup = nil
		}
		// Return to StateIdle after GROUP ends
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

			// If we're in a group, associate the card with the group
			if p.currentGroup != nil {
				p.currentCard.GroupID = p.currentGroup.ID
				p.currentGroup.CardIDs = append(p.currentGroup.CardIDs, p.currentCard.ID)
			}

			p.currentField = ""
			p.state = StateInCard

			events = append(events, CardEvent{
				Type:         CardEventCreate,
				CardID:       p.currentCard.ID,
				TemplateID:   templateID,
				TemplateName: templateName,
				GroupID:      p.currentCard.GroupID,
			})
		}
		return events
	}

	// Check for card end: <</CARD>>
	if inner == "/CARD" {
		if p.currentCard != nil {
			// Save completed card for final content generation
			completedCard := &CardState{
				ID:           p.currentCard.ID,
				TemplateID:   p.currentCard.TemplateID,
				TemplateName: p.currentCard.TemplateName,
				Fields:       make(map[string]string),
				GroupID:      p.currentCard.GroupID,
			}
			for k, v := range p.currentCard.Fields {
				completedCard.Fields[k] = v
			}
			p.completedCards = append(p.completedCards, completedCard)

			events = append(events, CardEvent{
				Type:   CardEventDone,
				CardID: p.currentCard.ID,
			})
			p.currentCard = nil
			p.currentField = ""
		}
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

	// If a group is still open, emit end event
	if p.currentGroup != nil {
		events = append(events, GroupEvent{
			Type:    GroupEventEnd,
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
	p.currentField = ""
	p.currentGroup = nil
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
		GroupID:      p.currentCard.GroupID,
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

// GetCompletedCards returns all completed cards
func (p *StreamCardParser) GetCompletedCards() []*CardState {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Return a copy of completed cards
	cards := make([]*CardState, len(p.completedCards))
	for i, card := range p.completedCards {
		cardCopy := &CardState{
			ID:           card.ID,
			TemplateID:   card.TemplateID,
			TemplateName: card.TemplateName,
			Fields:       make(map[string]string),
			GroupID:      card.GroupID,
		}
		for k, v := range card.Fields {
			cardCopy.Fields[k] = v
		}
		cards[i] = cardCopy
	}
	return cards
}

// HasCompletedCards returns true if there are any completed cards
func (p *StreamCardParser) HasCompletedCards() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.completedCards) > 0
}

// GetCurrentGroup returns the current group being parsed (if any)
func (p *StreamCardParser) GetCurrentGroup() *GroupState {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentGroup == nil {
		return nil
	}

	// Return a copy
	groupCopy := &GroupState{
		ID:      p.currentGroup.ID,
		Layout:  p.currentGroup.Layout,
		Columns: p.currentGroup.Columns,
		CardIDs: make([]string, len(p.currentGroup.CardIDs)),
	}
	copy(groupCopy.CardIDs, p.currentGroup.CardIDs)
	return groupCopy
}

// IsInGroup returns true if parser is currently inside a group
func (p *StreamCardParser) IsInGroup() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentGroup != nil
}

// GetCompletedGroups returns all completed groups
func (p *StreamCardParser) GetCompletedGroups() []*GroupState {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Return a copy of completed groups
	groups := make([]*GroupState, len(p.completedGroups))
	for i, group := range p.completedGroups {
		groupCopy := &GroupState{
			ID:      group.ID,
			Layout:  group.Layout,
			Columns: group.Columns,
			CardIDs: make([]string, len(group.CardIDs)),
		}
		copy(groupCopy.CardIDs, group.CardIDs)
		groups[i] = groupCopy
	}
	return groups
}

// HasCompletedGroups returns true if there are any completed groups
func (p *StreamCardParser) HasCompletedGroups() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.completedGroups) > 0
}
