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
		fmt.Printf("[BoundCards] Using STREAMING format, %d cards\n", len(b.boundCards))
		return b.renderStreamingFormat(ctx, req)
	}
	fmt.Printf("[BoundCards] Using JSON format, %d cards\n", len(b.boundCards))
	return b.renderJSONFormat(ctx, req)
}

// renderStreamingFormat generates prompt for streaming card format
func (b *boundCardsRender) renderStreamingFormat(ctx context.Context, req *AgentRequest) (string, error) {
	var sb strings.Builder

	sb.WriteString("**可用卡片**\n")
	sb.WriteString("当需要以结构化卡片形式展示内容时，请**直接输出**以下标记格式（不要用代码块包裹）：\n\n")

	// List all available cards with their parameters and example
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

		// Write parameter description with nested structure support
		if card.ParamList != nil && len(card.ParamList) > 0 {
			sb.WriteString("参数说明：\n")
			b.writeParamDescriptionRecursive(&sb, card.ParamList, 0)
		}

		// Generate streaming format example for each card with nested structure
		sb.WriteString("\n输出格式：\n")
		sb.WriteString(fmt.Sprintf("<<CARD:%s:%s>>\n", code, cardName))
		if card.ParamList != nil && len(card.ParamList) > 0 {
			b.writeParamOutputExample(&sb, card, card.ParamList, req)
		}
		sb.WriteString("<</CARD>>\n\n")
	}

	// Add GROUP layout documentation (always show, supports single or multiple cards)
	sb.WriteString("**卡片组布局**（控制卡片显示布局）：\n")
	sb.WriteString("当需要控制卡片布局时，可以使用 GROUP 标签包裹卡片：\n")
	sb.WriteString("```\n")
	sb.WriteString("<<GROUP:布局类型:列数>>\n")
	sb.WriteString("<<CARD:卡片代码:卡片名称>>\n")
	sb.WriteString("<<字段1>>值1\n")
	sb.WriteString("<<字段2>>值2\n")
	sb.WriteString("<</CARD>>\n")
	if len(b.boundCards) >= 2 {
		sb.WriteString("<<CARD:卡片代码:卡片名称>>\n")
		sb.WriteString("...\n")
		sb.WriteString("<</CARD>>\n")
	}
	sb.WriteString("<</GROUP>>\n")
	sb.WriteString("```\n\n")
	sb.WriteString("**布局类型**：`horizontal`（水平排列）、`vertical`（垂直排列）、`waterfall`（瀑布流）\n")
	sb.WriteString("**列数**：1、2、3、4（控制每行显示的卡片数量）\n\n")
	sb.WriteString("示例：`<<GROUP:horizontal:2>>` 表示水平排列，每行2张卡片\n\n")

	sb.WriteString("**⚠️ 重要规则**：\n")
	sb.WriteString("1. 直接输出标记，**绝对不要**用```代码块包裹！\n")
	sb.WriteString("2. GROUP和CARD标签之间、多个CARD标签之间，**禁止输出任何纯文本、换行或空格**！\n")
	sb.WriteString("3. 纯文本说明必须放在所有卡片/分组的**前面或后面**，不能穿插在中间。\n")
	sb.WriteString("4. **数组字段**：每个数组元素单独用一行`<<字段名>>{完整JSON对象}`输出，可连续输出多个元素。\n")
	sb.WriteString("5. **嵌套对象**：对象类型字段用`<<字段名>>{完整JSON对象}`格式输出。\n")

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

// writeParamDescriptionRecursive recursively writes parameter descriptions with nested structure
func (b *boundCardsRender) writeParamDescriptionRecursive(sb *strings.Builder, params []*bot_common.CardParam, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, param := range params {
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

		// Write current parameter
		sb.WriteString(fmt.Sprintf("%s- `%s` (%s): %s%s\n", indent, paramName, paramType, desc, requiredMark))

		// Recursively process children for array/object types
		if param.Children != nil && len(param.Children) > 0 {
			if paramType == "array" {
				sb.WriteString(fmt.Sprintf("%s  数组元素结构：\n", indent))
			} else if paramType == "object" {
				sb.WriteString(fmt.Sprintf("%s  对象属性：\n", indent))
			}
			b.writeParamDescriptionRecursive(sb, param.Children, depth+1)
		}
	}
}

// writeParamOutputExample generates output format example with nested structure support
func (b *boundCardsRender) writeParamOutputExample(sb *strings.Builder, card *bot_common.BoundCardInfo, params []*bot_common.CardParam, req *AgentRequest) {
	for _, param := range params {
		paramName := ""
		if param.ParamName != nil {
			paramName = *param.ParamName
		}

		paramType := "string"
		if param.ParamType != nil {
			paramType = *param.ParamType
		}

		// Handle different parameter types
		if paramType == "array" && param.Children != nil && len(param.Children) > 0 {
			// For array with nested structure, generate complete JSON object example
			exampleJSON := b.generateNestedJSONExample(param.Children)
			sb.WriteString(fmt.Sprintf("<<%s>>{%s}\n", paramName, exampleJSON))
			sb.WriteString(fmt.Sprintf("<<%s>>{%s}\n", paramName, exampleJSON))
			sb.WriteString(fmt.Sprintf("... (可继续添加更多%s元素)\n", paramName))
		} else if paramType == "object" && param.Children != nil && len(param.Children) > 0 {
			// For object with nested structure
			exampleJSON := b.generateNestedJSONExample(param.Children)
			sb.WriteString(fmt.Sprintf("<<%s>>{%s}\n", paramName, exampleJSON))
		} else {
			// Simple field
			exampleValue := b.getMappedVariableValue(card, paramName, req)
			if exampleValue == "" {
				exampleValue = fmt.Sprintf("<%s的值>", paramName)
			}
			sb.WriteString(fmt.Sprintf("<<%s>>%s\n", paramName, exampleValue))
		}
	}
}

// generateNestedJSONExample generates a JSON example string for nested parameters
func (b *boundCardsRender) generateNestedJSONExample(params []*bot_common.CardParam) string {
	var parts []string
	for _, param := range params {
		paramName := ""
		if param.ParamName != nil {
			paramName = *param.ParamName
		}

		paramType := "string"
		if param.ParamType != nil {
			paramType = *param.ParamType
		}

		var value string
		if paramType == "object" && param.Children != nil && len(param.Children) > 0 {
			// Nested object
			nestedJSON := b.generateNestedJSONExample(param.Children)
			value = fmt.Sprintf("{%s}", nestedJSON)
		} else if paramType == "array" && param.Children != nil && len(param.Children) > 0 {
			// Nested array
			nestedJSON := b.generateNestedJSONExample(param.Children)
			value = fmt.Sprintf("[{%s}]", nestedJSON)
		} else {
			// Simple value with placeholder
			value = fmt.Sprintf("\"<%s值>\"", paramName)
		}

		parts = append(parts, fmt.Sprintf("\"%s\":%s", paramName, value))
	}
	return strings.Join(parts, ",")
}
