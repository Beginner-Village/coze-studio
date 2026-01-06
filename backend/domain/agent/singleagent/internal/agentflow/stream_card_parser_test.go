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
	"strings"
	"testing"
)

func TestStreamCardParser_NormalText(t *testing.T) {
	parser := NewStreamCardParser()

	events := parser.Feed("Hello, world!")

	// Should emit one text event per character
	var text strings.Builder
	for _, event := range events {
		if te, ok := event.(TextEvent); ok {
			text.WriteString(te.Text)
		}
	}

	if text.String() != "Hello, world!" {
		t.Errorf("Expected 'Hello, world!', got '%s'", text.String())
	}
}

func TestStreamCardParser_SingleCard(t *testing.T) {
	parser := NewStreamCardParser()
	parser.SetCardIDGenerator(func() string { return "test_card_1" })

	input := "<<CARD:weather:天气卡片>><<city>>北京<<temp>>25°C<</CARD>>"
	events := parser.Feed(input)

	// Verify events
	var createCount, deltaCount, doneCount int
	var cardID string
	fields := make(map[string]string)

	for _, event := range events {
		switch e := event.(type) {
		case CardEvent:
			switch e.Type {
			case CardEventCreate:
				createCount++
				cardID = e.CardID
				if e.TemplateID != "weather" {
					t.Errorf("Expected template_id 'weather', got '%s'", e.TemplateID)
				}
				if e.TemplateName != "天气卡片" {
					t.Errorf("Expected template_name '天气卡片', got '%s'", e.TemplateName)
				}
			case CardEventDelta:
				deltaCount++
				fields[e.Field] += e.Delta
			case CardEventDone:
				doneCount++
				if e.CardID != cardID {
					t.Errorf("Done event card ID mismatch")
				}
			}
		}
	}

	if createCount != 1 {
		t.Errorf("Expected 1 create event, got %d", createCount)
	}
	if doneCount != 1 {
		t.Errorf("Expected 1 done event, got %d", doneCount)
	}
	if fields["city"] != "北京" {
		t.Errorf("Expected city='北京', got '%s'", fields["city"])
	}
	if fields["temp"] != "25°C" {
		t.Errorf("Expected temp='25°C', got '%s'", fields["temp"])
	}
}

func TestStreamCardParser_MixedContent(t *testing.T) {
	parser := NewStreamCardParser()
	parser.SetCardIDGenerator(func() string { return "test_card" })

	input := "今天天气：<<CARD:weather:天气>><<city>>北京<</CARD>>很不错！"
	events := parser.Feed(input)

	var textParts []string
	var hasCard bool

	for _, event := range events {
		switch e := event.(type) {
		case TextEvent:
			textParts = append(textParts, e.Text)
		case CardEvent:
			if e.Type == CardEventCreate {
				hasCard = true
			}
		}
	}

	combinedText := strings.Join(textParts, "")
	if !strings.Contains(combinedText, "今天天气：") {
		t.Errorf("Missing prefix text")
	}
	if !strings.Contains(combinedText, "很不错！") {
		t.Errorf("Missing suffix text")
	}
	if !hasCard {
		t.Errorf("Card not detected")
	}
}

func TestStreamCardParser_StreamingSimulation(t *testing.T) {
	parser := NewStreamCardParser()
	parser.SetCardIDGenerator(func() string { return "streaming_card" })

	// Simulate streaming: feed character by character
	input := "<<CARD:news:新闻>><<title>>今日头条<</CARD>>"
	var allEvents []StreamEvent

	for _, char := range input {
		events := parser.Feed(string(char))
		allEvents = append(allEvents, events...)
	}

	// Verify we got the expected events
	var createCount, doneCount int
	fields := make(map[string]string)

	for _, event := range allEvents {
		if ce, ok := event.(CardEvent); ok {
			switch ce.Type {
			case CardEventCreate:
				createCount++
			case CardEventDelta:
				fields[ce.Field] += ce.Delta
			case CardEventDone:
				doneCount++
			}
		}
	}

	if createCount != 1 {
		t.Errorf("Expected 1 create event, got %d", createCount)
	}
	if doneCount != 1 {
		t.Errorf("Expected 1 done event, got %d", doneCount)
	}
	if fields["title"] != "今日头条" {
		t.Errorf("Expected title='今日头条', got '%s'", fields["title"])
	}
}

func TestStreamCardParser_MultipleCards(t *testing.T) {
	parser := NewStreamCardParser()
	cardIDCounter := 0
	parser.SetCardIDGenerator(func() string {
		cardIDCounter++
		return string(rune('A' + cardIDCounter - 1))
	})

	input := "<<CARD:card1:卡片1>><<f1>>v1<</CARD>>中间文字<<CARD:card2:卡片2>><<f2>>v2<</CARD>>"
	events := parser.Feed(input)

	var createCount, doneCount int
	for _, event := range events {
		if ce, ok := event.(CardEvent); ok {
			switch ce.Type {
			case CardEventCreate:
				createCount++
			case CardEventDone:
				doneCount++
			}
		}
	}

	if createCount != 2 {
		t.Errorf("Expected 2 create events, got %d", createCount)
	}
	if doneCount != 2 {
		t.Errorf("Expected 2 done events, got %d", doneCount)
	}
}

func TestStreamCardParser_FalsePositive(t *testing.T) {
	parser := NewStreamCardParser()

	// Single '<' should not trigger card parsing
	events := parser.Feed("a < b < c")

	var text strings.Builder
	for _, event := range events {
		if te, ok := event.(TextEvent); ok {
			text.WriteString(te.Text)
		}
	}

	if text.String() != "a < b < c" {
		t.Errorf("Expected 'a < b < c', got '%s'", text.String())
	}
}

func TestStreamCardParser_IncompleteTag(t *testing.T) {
	parser := NewStreamCardParser()

	// Feed incomplete tag then flush
	parser.Feed("<<CARD:test")
	events := parser.Flush()

	// Should emit the incomplete content as text
	var text strings.Builder
	for _, event := range events {
		if te, ok := event.(TextEvent); ok {
			text.WriteString(te.Text)
		}
	}

	if text.String() != "<<CARD:test" {
		t.Errorf("Expected '<<CARD:test', got '%s'", text.String())
	}
}

func TestStreamCardParser_MultipleFields(t *testing.T) {
	parser := NewStreamCardParser()
	parser.SetCardIDGenerator(func() string { return "multi_field_card" })

	input := "<<CARD:profile:用户资料>><<name>>张三<<age>>25<<city>>北京<<job>>工程师<</CARD>>"
	events := parser.Feed(input)

	fields := make(map[string]string)
	for _, event := range events {
		if ce, ok := event.(CardEvent); ok && ce.Type == CardEventDelta {
			fields[ce.Field] += ce.Delta
		}
	}

	expected := map[string]string{
		"name": "张三",
		"age":  "25",
		"city": "北京",
		"job":  "工程师",
	}

	for k, v := range expected {
		if fields[k] != v {
			t.Errorf("Field %s: expected '%s', got '%s'", k, v, fields[k])
		}
	}
}

func TestStreamCardParser_LongFieldValue(t *testing.T) {
	parser := NewStreamCardParser()
	parser.SetCardIDGenerator(func() string { return "long_value_card" })

	longValue := strings.Repeat("这是一段很长的文字。", 100)
	input := "<<CARD:article:文章>><<content>>" + longValue + "<</CARD>>"

	events := parser.Feed(input)

	var content strings.Builder
	for _, event := range events {
		if ce, ok := event.(CardEvent); ok && ce.Type == CardEventDelta && ce.Field == "content" {
			content.WriteString(ce.Delta)
		}
	}

	if content.String() != longValue {
		t.Errorf("Long value mismatch, lengths: expected %d, got %d", len(longValue), content.Len())
	}
}

func TestStreamCardParser_Reset(t *testing.T) {
	parser := NewStreamCardParser()

	// Start parsing a card
	parser.Feed("<<CARD:test:测试>><<field>>value")

	if !parser.IsInCard() {
		t.Error("Should be in card after partial input")
	}

	// Reset
	parser.Reset()

	if parser.IsInCard() {
		t.Error("Should not be in card after reset")
	}

	// Should be able to parse new content
	parser.SetCardIDGenerator(func() string { return "new_card" })
	events := parser.Feed("<<CARD:new:新卡片>><<f>>v<</CARD>>")

	var createCount int
	for _, event := range events {
		if ce, ok := event.(CardEvent); ok && ce.Type == CardEventCreate {
			createCount++
		}
	}

	if createCount != 1 {
		t.Errorf("Expected 1 create event after reset, got %d", createCount)
	}
}

func TestStreamCardParser_GetCurrentCard(t *testing.T) {
	parser := NewStreamCardParser()
	parser.SetCardIDGenerator(func() string { return "current_card" })

	// Before card
	if parser.GetCurrentCard() != nil {
		t.Error("Should be nil before card")
	}

	// During card
	parser.Feed("<<CARD:test:测试>><<city>>北")
	card := parser.GetCurrentCard()
	if card == nil {
		t.Fatal("Should have current card")
	}
	if card.TemplateID != "test" {
		t.Errorf("Expected template_id 'test', got '%s'", card.TemplateID)
	}
	if card.Fields["city"] != "北" {
		t.Errorf("Expected city='北', got '%s'", card.Fields["city"])
	}

	// After more input
	parser.Feed("京")
	card = parser.GetCurrentCard()
	if card.Fields["city"] != "北京" {
		t.Errorf("Expected city='北京', got '%s'", card.Fields["city"])
	}

	// After card done
	parser.Feed("<</CARD>>")
	if parser.GetCurrentCard() != nil {
		t.Error("Should be nil after card done")
	}
}
