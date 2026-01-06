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
	"testing"
)

func TestStreamCardProcessor_Disabled(t *testing.T) {
	processor := NewStreamCardProcessor(false)

	outputs := processor.Process("Hello, world!")

	if len(outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(outputs))
	}

	if outputs[0].Type != "text" {
		t.Errorf("Expected type 'text', got '%s'", outputs[0].Type)
	}

	if outputs[0].Text != "Hello, world!" {
		t.Errorf("Expected 'Hello, world!', got '%s'", outputs[0].Text)
	}
}

func TestStreamCardProcessor_Enabled_PlainText(t *testing.T) {
	processor := NewStreamCardProcessor(true)

	outputs := processor.Process("Hello, world!")

	// Should emit text character by character
	var combinedText string
	for _, out := range outputs {
		if out.Type == "text" {
			combinedText += out.Text
		}
	}

	if combinedText != "Hello, world!" {
		t.Errorf("Expected 'Hello, world!', got '%s'", combinedText)
	}
}

func TestStreamCardProcessor_Enabled_CardOutput(t *testing.T) {
	processor := NewStreamCardProcessor(true)
	processor.parser.SetCardIDGenerator(func() string { return "test_card_123" })

	input := "<<CARD:weather:天气卡片>><<city>>北京<</CARD>>"
	outputs := processor.Process(input)

	// Verify we get card events
	var createCount, deltaCount, doneCount int
	var city string
	var cardID string

	for _, out := range outputs {
		switch out.Type {
		case string(YnetTypeCardCreate):
			createCount++
			cardID = out.CardID
			if out.TemplateID != "weather" {
				t.Errorf("Expected template_id 'weather', got '%s'", out.TemplateID)
			}
			if out.TemplateName != "天气卡片" {
				t.Errorf("Expected template_name '天气卡片', got '%s'", out.TemplateName)
			}
		case string(YnetTypeCardDelta):
			deltaCount++
			if out.Field == "city" {
				city += out.Delta
			}
		case string(YnetTypeCardDone):
			doneCount++
			if out.CardID != cardID {
				t.Errorf("CardID mismatch in done event")
			}
		}
	}

	if createCount != 1 {
		t.Errorf("Expected 1 create event, got %d", createCount)
	}
	if doneCount != 1 {
		t.Errorf("Expected 1 done event, got %d", doneCount)
	}
	if city != "北京" {
		t.Errorf("Expected city '北京', got '%s'", city)
	}
}

func TestStreamCardProcessor_MixedContent(t *testing.T) {
	processor := NewStreamCardProcessor(true)
	processor.parser.SetCardIDGenerator(func() string { return "mixed_card" })

	input := "前缀文字<<CARD:test:测试>><<field>>value<</CARD>>后缀文字"
	outputs := processor.Process(input)

	var prefixText, suffixText string
	var hasCard bool
	var inCardSuffix bool

	for _, out := range outputs {
		switch out.Type {
		case "text":
			if !hasCard {
				prefixText += out.Text
			} else {
				inCardSuffix = true
				suffixText += out.Text
			}
		case string(YnetTypeCardDone):
			hasCard = true
		}
	}

	if prefixText != "前缀文字" {
		t.Errorf("Expected prefix '前缀文字', got '%s'", prefixText)
	}
	if !inCardSuffix {
		t.Errorf("Expected suffix text after card")
	}
	if suffixText != "后缀文字" {
		t.Errorf("Expected suffix '后缀文字', got '%s'", suffixText)
	}
}

func TestStreamCardProcessor_Flush(t *testing.T) {
	processor := NewStreamCardProcessor(true)

	// Feed incomplete tag
	processor.Process("<<CARD:incomplete")
	outputs := processor.Flush()

	// Should emit buffered content as text
	var text string
	for _, out := range outputs {
		if out.Type == "text" {
			text += out.Text
		}
	}

	if text != "<<CARD:incomplete" {
		t.Errorf("Expected '<<CARD:incomplete', got '%s'", text)
	}
}

func TestStreamCardProcessor_IsInCard(t *testing.T) {
	processor := NewStreamCardProcessor(true)
	processor.parser.SetCardIDGenerator(func() string { return "card_123" })

	if processor.IsInCard() {
		t.Error("Should not be in card initially")
	}

	processor.Process("<<CARD:test:测试>><<field>>partial")

	if !processor.IsInCard() {
		t.Error("Should be in card after partial input")
	}

	processor.Process("<</CARD>>")

	if processor.IsInCard() {
		t.Error("Should not be in card after close tag")
	}
}

func TestBuildCardMetaData_Create(t *testing.T) {
	output := StreamCardOutput{
		Type:         string(YnetTypeCardCreate),
		CardID:       "card_123",
		TemplateID:   "weather",
		TemplateName: "天气卡片",
	}

	meta := BuildCardMetaData(output)

	if meta[string(MetaKeyYnetType)] != string(YnetTypeCardCreate) {
		t.Error("ynet_type mismatch")
	}
	if meta[string(MetaKeyCardID)] != "card_123" {
		t.Error("card_id mismatch")
	}
	if meta[string(MetaKeyTemplateID)] != "weather" {
		t.Error("template_id mismatch")
	}
	if meta[string(MetaKeyTemplateName)] != "天气卡片" {
		t.Error("template_name mismatch")
	}
}

func TestBuildCardMetaData_Delta(t *testing.T) {
	output := StreamCardOutput{
		Type:   string(YnetTypeCardDelta),
		CardID: "card_123",
		Field:  "city",
		Delta:  "北",
	}

	meta := BuildCardMetaData(output)

	if meta[string(MetaKeyYnetType)] != string(YnetTypeCardDelta) {
		t.Error("ynet_type mismatch")
	}
	if meta[string(MetaKeyCardID)] != "card_123" {
		t.Error("card_id mismatch")
	}
	if meta[string(MetaKeyCardField)] != "city" {
		t.Error("card_field mismatch")
	}
}

func TestBuildCardMetaData_Done(t *testing.T) {
	output := StreamCardOutput{
		Type:   string(YnetTypeCardDone),
		CardID: "card_123",
	}

	meta := BuildCardMetaData(output)

	if meta[string(MetaKeyYnetType)] != string(YnetTypeCardDone) {
		t.Error("ynet_type mismatch")
	}
	if meta[string(MetaKeyCardID)] != "card_123" {
		t.Error("card_id mismatch")
	}
}
