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
	"context"
	"fmt"
	"strings"

	"github.com/coze-dev/coze-studio/backend/api/model/app/bot_common"
)

// CardOutputMode defines the card output format mode
type CardOutputMode int

const (
	// CardOutputModeJSON uses traditional JSON format (for non-streaming)
	CardOutputModeJSON CardOutputMode = iota
	// CardOutputModeStreaming uses streaming tag format (for real-time rendering)
	CardOutputModeStreaming
)

// boundCardsRender renders prompt text for bound cards
type boundCardsRender struct {
	boundCards []*bot_common.BoundCardInfo
	variables  map[string]string
	outputMode CardOutputMode
}

// RenderBoundCards generates prompt text based on bound cards configuration
func (b *boundCardsRender) RenderBoundCards(ctx context.Context, req *AgentRequest) (string, error) {
	if len(b.boundCards) == 0 {
		return "", nil
	}

	if b.outputMode == CardOutputModeStreaming {
		return b.renderStreamingFormat(ctx, req)
	}
	return b.renderJSONFormat(ctx, req)
}

// renderStreamingFormat generates prompt for streaming card format
func (b *boundCardsRender) renderStreamingFormat(ctx context.Context, req *AgentRequest) (string, error) {
	var sb strings.Builder

	sb.WriteString("**可用卡片**\n")
	sb.WriteString("当需要以结构化卡片形式展示内容时，请使用以下标记格式输出：\n\n")

	for i, card := range b.boundCards {
		cardName := ""
		if card.CardName != nil {
			cardName = *card.CardName
		}

		code := ""
		if card.Code != nil {
			code = *card.Code
		}

		sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, cardName))
		sb.WriteString(fmt.Sprintf("卡片代码: `%s`\n", code))

		// Write parameter description
		if card.ParamList != nil && len(card.ParamList) > 0 {
			sb.WriteString("参数说明：\n")
			for _, param := range card.ParamList {
				paramName := ""
				if param.ParamName != nil {
					paramName = *param.ParamName
				}

				paramType := "string"
				if param.ParamType != nil {
					paramType = *param.ParamType
				}

				desc := ""
				if param.Desc != nil {
					desc = *param.Desc
				}

				requiredMark := ""
				if param.Required != nil && *param.Required {
					requiredMark = " (必填)"
				}

				sb.WriteString(fmt.Sprintf("- `%s` (%s): %s%s\n", paramName, paramType, desc, requiredMark))
			}
		}

		// Generate streaming format example
		sb.WriteString("\n输出格式示例：\n```\n")
		sb.WriteString(fmt.Sprintf("<<CARD:%s:%s>>\n", code, cardName))

		if card.ParamList != nil && len(card.ParamList) > 0 {
			for _, param := range card.ParamList {
				paramName := ""
				if param.ParamName != nil {
					paramName = *param.ParamName
				}

				// Get example value
				exampleValue := b.getMappedVariableValue(card, paramName, req)
				if exampleValue == "" {
					exampleValue = fmt.Sprintf("<%s的值>", paramName)
				}

				sb.WriteString(fmt.Sprintf("<<%s>>%s\n", paramName, exampleValue))
			}
		}

		sb.WriteString("<</CARD>>\n")
		sb.WriteString("```\n\n")
	}

	// Important notes for streaming format
	sb.WriteString("**格式说明**：\n")
	sb.WriteString("1. 卡片开始标记：`<<CARD:卡片代码:卡片名称>>`\n")
	sb.WriteString("2. 字段标记：`<<字段名>>` 后紧跟字段值\n")
	sb.WriteString("3. 卡片结束标记：`<</CARD>>`\n")
	sb.WriteString("4. 字段值可以包含任意文本，直到下一个 `<<` 出现\n")
	sb.WriteString("5. 可以在卡片前后添加普通文字说明\n\n")

	sb.WriteString("**重要提示**：当需要使用卡片展示内容时，请严格按照上述标记格式输出。\n")

	return sb.String(), nil
}

// renderJSONFormat generates prompt for traditional JSON format (legacy)
func (b *boundCardsRender) renderJSONFormat(ctx context.Context, req *AgentRequest) (string, error) {
	var sb strings.Builder
	sb.WriteString("**可用卡片**\n")
	sb.WriteString("当需要以结构化卡片形式展示内容时，请使用以下JSON格式输出：\n\n")

	for i, card := range b.boundCards {
		cardName := ""
		if card.CardName != nil {
			cardName = *card.CardName
		}

		code := ""
		if card.Code != nil {
			code = *card.Code
		}

		sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, cardName))
		sb.WriteString(fmt.Sprintf("卡片代码: `%s`\n", code))

		// Build dataResponse example
		dataResponseExample := make(map[string]string)
		if card.ParamList != nil && len(card.ParamList) > 0 {
			sb.WriteString("参数说明：\n")
			for _, param := range card.ParamList {
				paramName := ""
				if param.ParamName != nil {
					paramName = *param.ParamName
				}

				paramType := "string"
				if param.ParamType != nil {
					paramType = *param.ParamType
				}

				desc := ""
				if param.Desc != nil {
					desc = *param.Desc
				}

				requiredMark := ""
				if param.Required != nil && *param.Required {
					requiredMark = " (必填)"
				}

				sb.WriteString(fmt.Sprintf("- `%s` (%s): %s%s\n", paramName, paramType, desc, requiredMark))

				// Add to example with placeholder or mapped value
				mappedValue := b.getMappedVariableValue(card, paramName, req)
				if mappedValue != "" {
					dataResponseExample[paramName] = mappedValue
				} else {
					dataResponseExample[paramName] = fmt.Sprintf("<%s的值>", paramName)
				}
			}
		}

		// Generate JSON example
		sb.WriteString("\n输出格式示例：\n```json\n")
		sb.WriteString("{\n")
		sb.WriteString("  \"contentList\": [\n")
		sb.WriteString("    {\n")
		sb.WriteString("      \"displayResponseType\": \"TEMPLATE\",\n")
		sb.WriteString("      \"rawContent\": {},\n")
		sb.WriteString(fmt.Sprintf("      \"templateId\": \"%s\",\n", code))
		sb.WriteString(fmt.Sprintf("      \"templateName\": \"%s\",\n", cardName))
		sb.WriteString("      \"kvMap\": {},\n")
		sb.WriteString("      \"dataResponse\": {\n")

		// Write dataResponse fields
		paramCount := 0
		totalParams := len(dataResponseExample)
		for paramName, paramValue := range dataResponseExample {
			paramCount++
			comma := ","
			if paramCount == totalParams {
				comma = ""
			}
			sb.WriteString(fmt.Sprintf("        \"%s\": \"%s\"%s\n", paramName, paramValue, comma))
		}

		sb.WriteString("      }\n")
		sb.WriteString("    }\n")
		sb.WriteString("  ]\n")
		sb.WriteString("}\n```\n\n")
	}

	sb.WriteString("**重要提示**：当需要使用卡片展示内容时，请严格按照上述JSON格式输出，确保 `displayResponseType` 为 `TEMPLATE`，`templateId` 为对应的卡片代码。\n")

	return sb.String(), nil
}

// getMappedVariableValue gets the value of a mapped variable for a card parameter
func (b *boundCardsRender) getMappedVariableValue(card *bot_common.BoundCardInfo, paramName string, req *AgentRequest) string {
	if card.ParamMapping == nil {
		return ""
	}

	for _, mapping := range card.ParamMapping {
		if mapping.ParamName != nil && *mapping.ParamName == paramName && mapping.VariableName != nil {
			varName := *mapping.VariableName
			// First try to get from request variables
			if val, ok := req.Variables[varName]; ok {
				return val
			}
			// Fall back to bound cards render variables
			if val, ok := b.variables[varName]; ok {
				return val
			}
		}
	}

	return ""
}

// newBoundCardsRender creates a new bound cards renderer
func newBoundCardsRender(boundCards []*bot_common.BoundCardInfo, variables map[string]string) *boundCardsRender {
	return &boundCardsRender{
		boundCards: boundCards,
		variables:  variables,
		outputMode: CardOutputModeStreaming, // Default to streaming mode
	}
}

// newBoundCardsRenderWithMode creates a new bound cards renderer with specified output mode
func newBoundCardsRenderWithMode(boundCards []*bot_common.BoundCardInfo, variables map[string]string, mode CardOutputMode) *boundCardsRender {
	return &boundCardsRender{
		boundCards: boundCards,
		variables:  variables,
		outputMode: mode,
	}
}

// SetOutputMode sets the card output mode
func (b *boundCardsRender) SetOutputMode(mode CardOutputMode) {
	b.outputMode = mode
}
