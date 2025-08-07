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
	"context"
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/schema"

	"github.com/coze-dev/coze-studio/backend/api/model/crossdomain/message"
	"github.com/coze-dev/coze-studio/backend/crossdomain/contract/crossagent"
	"github.com/coze-dev/coze-studio/backend/domain/conversation/message/entity"
	"github.com/coze-dev/coze-studio/backend/infra/contract/imagex"
)

func HistoryPairs(historyMsg []*message.Message) []*message.Message {

	fcMsgPairs := make(map[int64][]*message.Message)
	for _, one := range historyMsg {
		if one.MessageType != message.MessageTypeFunctionCall && one.MessageType != message.MessageTypeToolResponse {
			continue
		}
		if _, ok := fcMsgPairs[one.RunID]; !ok {
			fcMsgPairs[one.RunID] = []*message.Message{one}
		} else {
			fcMsgPairs[one.RunID] = append(fcMsgPairs[one.RunID], one)
		}
	}

	var historyAfterPairs []*message.Message
	for _, value := range historyMsg {
		if value.MessageType == message.MessageTypeFunctionCall {
			if len(fcMsgPairs[value.RunID])%2 == 0 {
				historyAfterPairs = append(historyAfterPairs, value)
			}
		} else {
			historyAfterPairs = append(historyAfterPairs, value)
		}
	}
	return historyAfterPairs

}

func TransMessageToSchemaMessage(ctx context.Context, msgs []*message.Message, imagexClient imagex.ImageX) []*schema.Message {
	schemaMessage := make([]*schema.Message, 0, len(msgs))
	
	// Group messages by RunID for processing function_call -> tool_response sequences
	runMessages := make(map[int64][]*message.Message)
	for _, msg := range msgs {
		runMessages[msg.RunID] = append(runMessages[msg.RunID], msg)
	}
	
	processed := make(map[int64]bool)

	for _, msgOne := range msgs {
		if processed[msgOne.ID] {
			continue
		}
		if msgOne.ModelContent == "" {
			continue
		}
		if msgOne.MessageType == message.MessageTypeVerbose || msgOne.MessageType == message.MessageTypeFlowUp {
			continue
		}
		
		var sm *schema.Message
		err := json.Unmarshal([]byte(msgOne.ModelContent), &sm)
		if err != nil {
			continue
		}
		
		// Special handling for function_call -> tool_response sequences
		if sm.Role == schema.Assistant && sm.ToolCalls != nil && len(sm.ToolCalls) > 0 {
			// This is a function_call message, preserve it
			schemaMessage = append(schemaMessage, parseMessageURI(ctx, sm, imagexClient))
			processed[msgOne.ID] = true
			
			// Now find and merge all intermediate assistant messages into tool_response
			runMsgs := runMessages[msgOne.RunID]
			var toolResponseMsg *message.Message
			intermediateAssistantMsgs := make([]*message.Message, 0)
			
			
			// Find tool_response and intermediate assistant messages in this run
			for _, runMsg := range runMsgs {
				if runMsg.MessageType == message.MessageTypeToolResponse && runMsg.ModelContent != "" {
					toolResponseMsg = runMsg
				} else if runMsg.MessageType == message.MessageTypeAnswer && runMsg.ModelContent != "" && runMsg.ID != msgOne.ID {
					var testSm *schema.Message
					if json.Unmarshal([]byte(runMsg.ModelContent), &testSm) == nil && 
						testSm.Role == schema.Assistant && 
						(testSm.ToolCalls == nil || len(testSm.ToolCalls) == 0) {
						intermediateAssistantMsgs = append(intermediateAssistantMsgs, runMsg)
					}
				}
			}
			
			// Merge intermediate content into tool_response
			if toolResponseMsg != nil {
				var toolSm *schema.Message
				if json.Unmarshal([]byte(toolResponseMsg.ModelContent), &toolSm) == nil {
					// Collect all intermediate content and remove duplicates
					allContent := make([]string, 0)
					contentSet := make(map[string]bool)
					
					// Add original tool response content
					if toolSm.Content != "" && !contentSet[toolSm.Content] {
						allContent = append(allContent, toolSm.Content)
						contentSet[toolSm.Content] = true
					}
					
					// Add intermediate assistant content, skip duplicates only
					for _, intMsg := range intermediateAssistantMsgs {
						content := strings.TrimSpace(intMsg.Content)
						if content != "" && !contentSet[content] {
							// Keep all content including complex JSON templates
							allContent = append(allContent, content)
							contentSet[content] = true
						}
					}
					
					// Merge all unique content
					if len(allContent) > 1 {
						toolSm.Content = strings.Join(allContent, "\n\n")
					} else if len(allContent) == 1 {
						toolSm.Content = allContent[0]
					}
					
					schemaMessage = append(schemaMessage, parseMessageURI(ctx, toolSm, imagexClient))
					processed[toolResponseMsg.ID] = true
					
					// Mark all intermediate assistant messages as processed
					for _, intMsg := range intermediateAssistantMsgs {
						processed[intMsg.ID] = true
					}
				}
			}
			continue
		}
		
		// For non-function_call assistant messages, only add if not already processed by merge logic
		if sm.Role == schema.Assistant {
			// Check if this is part of a function_call sequence that was already processed
			runMsgs := runMessages[msgOne.RunID]
			hasFunctionCall := false
			for _, runMsg := range runMsgs {
				if runMsg.MessageType == message.MessageTypeFunctionCall {
					hasFunctionCall = true
					break
				}
			}
			
			// Skip assistant messages that are part of function_call sequences
			if hasFunctionCall {
				processed[msgOne.ID] = true
				continue
			}
		}
		
		// Add other messages (user, tool not in function_call sequence, etc.)
		schemaMessage = append(schemaMessage, parseMessageURI(ctx, sm, imagexClient))
		processed[msgOne.ID] = true
	}

	return schemaMessage
}

func parseMessageURI(ctx context.Context, mcMsg *schema.Message, imagexClient imagex.ImageX) *schema.Message {
	if mcMsg.MultiContent == nil {
		return mcMsg
	}
	for k, one := range mcMsg.MultiContent {
		switch one.Type {
		case schema.ChatMessagePartTypeImageURL:

			if one.ImageURL.URI != "" {
				url, err := imagexClient.GetResourceURL(ctx, one.ImageURL.URI)
				if err == nil {
					mcMsg.MultiContent[k].ImageURL.URL = url.URL
				}
			}
		case schema.ChatMessagePartTypeFileURL:
			if one.FileURL.URI != "" {
				url, err := imagexClient.GetResourceURL(ctx, one.FileURL.URI)
				if err == nil {
					mcMsg.MultiContent[k].FileURL.URL = url.URL
				}
			}
		case schema.ChatMessagePartTypeAudioURL:
			if one.AudioURL.URI != "" {
				url, err := imagexClient.GetResourceURL(ctx, one.AudioURL.URI)
				if err == nil {
					mcMsg.MultiContent[k].AudioURL.URL = url.URL
				}
			}
		case schema.ChatMessagePartTypeVideoURL:
			if one.VideoURL.URI != "" {
				url, err := imagexClient.GetResourceURL(ctx, one.VideoURL.URI)
				if err == nil {
					mcMsg.MultiContent[k].VideoURL.URL = url.URL
				}
			}
		}
	}
	return mcMsg
}

func ParseResumeInfo(_ context.Context, historyMsg []*message.Message) *crossagent.ResumeInfo {

	var resumeInfo *crossagent.ResumeInfo
	for i := len(historyMsg) - 1; i >= 0; i-- {
		if historyMsg[i].MessageType == message.MessageTypeQuestion {
			break
		}
		if historyMsg[i].MessageType == message.MessageTypeVerbose {
			if historyMsg[i].Ext[string(entity.ExtKeyResumeInfo)] != "" {
				err := json.Unmarshal([]byte(historyMsg[i].Ext[string(entity.ExtKeyResumeInfo)]), &resumeInfo)
				if err != nil {
					return nil
				}
			}
		}
	}
	return resumeInfo
}
