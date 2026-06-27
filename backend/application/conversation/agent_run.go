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

package conversation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/hertz-contrib/sse"

	"github.com/ynet-dev/ynet-studio/backend/api/model/conversation/message"

	"github.com/ynet-dev/ynet-studio/backend/api/model/conversation/run"
	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/agentrun"
	crossDomainMessage "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/message"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	saEntity "github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/entity"
	convEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/conversation/entity"
	msgEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/message/entity"
	cmdEntity "github.com/ynet-dev/ynet-studio/backend/domain/shortcutcmd/entity"
	sseImpl "github.com/ynet-dev/ynet-studio/backend/infra/impl/sse"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/conv"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

func (c *ConversationApplicationService) Run(ctx context.Context, sseSender *sseImpl.SSenderImpl, ar *run.AgentRunRequest) error {
	agentInfo, caErr := c.checkAgent(ctx, ar)
	if caErr != nil {
		logs.CtxErrorf(ctx, "checkAgent err:%v", caErr)
		return caErr
	}

	userID := ctxutil.MustGetUIDFromCtx(ctx)
	conversationData, ccErr := c.checkConversation(ctx, ar, userID)

	if ccErr != nil {
		logs.CtxErrorf(ctx, "checkConversation err:%v", ccErr)
		return ccErr
	}

	if ar.RegenMessageID != nil && ptr.From(ar.RegenMessageID) > 0 {
		msgMeta, err := c.MessageDomainSVC.GetByID(ctx, ptr.From(ar.RegenMessageID))
		if err != nil {
			return err
		}
		if msgMeta != nil {
			if msgMeta.UserID != conv.Int64ToStr(userID) {
				return errorx.New(errno.ErrConversationPermissionCode, errorx.KV("msg", "message not match"))
			}

			err = c.AgentRunDomainSVC.Delete(ctx, []int64{msgMeta.RunID})
			if err != nil {
				return err
			}

			delErr := c.MessageDomainSVC.Delete(ctx, &msgEntity.DeleteMeta{
				RunIDs: []int64{msgMeta.RunID},
			})
			if delErr != nil {
				return delErr
			}
		}

	}
	var shortcutCmd *cmdEntity.ShortcutCmd
	if ar.GetShortcutCmdID() > 0 {
		cmdID := ar.GetShortcutCmdID()
		cmdMeta, err := c.ShortcutDomainSVC.GetByCmdID(ctx, cmdID, 0)
		if err != nil {
			return err
		}
		shortcutCmd = cmdMeta
	}

	arr, err := c.buildAgentRunRequest(ctx, ar, userID, agentInfo.SpaceID, conversationData, shortcutCmd)
	if err != nil {
		logs.CtxErrorf(ctx, "buildAgentRunRequest err:%v", err)
		return err
	}
	// 可取消的执行上下文:当客户端断开(点"停止响应"/关闭页面导致 SSE 写失败)时,
	// 取消该 ctx 会向下传播到 eino 图的模型/工具调用,使后端 run 真正停手并退出,
	// 进而触发会话级 run 锁的释放,避免"停止后再发消息发不出去"。
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	streamer, err := c.AgentRunDomainSVC.AgentRun(runCtx, arr)
	if err != nil {
		return err
	}
	c.pullStream(runCtx, sseSender, streamer, ar, cancel)
	return nil
}

func (c *ConversationApplicationService) pullStream(ctx context.Context, sseSender *sseImpl.SSenderImpl, arStream *schema.StreamReader[*entity.AgentRunResponse], req *run.AgentRunRequest, cancel context.CancelFunc) {
	var ackMessageInfo *entity.ChunkMessageItem
	// clientGone:客户端已断开。一旦探测到,就取消后端 run 并停止向已死连接写,
	// 但继续 drain 流直到 EOF,让执行 goroutine 收尾(关闭 writer、释放会话 run 锁)。
	clientGone := false
	send := func(ev *sse.Event) {
		if clientGone || ev == nil {
			return
		}
		if err := sseSender.Send(ctx, ev); err != nil {
			logs.CtxWarnf(ctx, "sse send failed, client gone, cancel agent run: %v", err)
			clientGone = true
			if cancel != nil {
				cancel()
			}
		}
	}
	for {
		chunk, recvErr := arStream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				return
			}
			send(buildErrorEvent(errno.ErrConversationAgentRunError, recvErr.Error()))
			return
		}

		switch chunk.Event {
		case entity.RunEventCreated, entity.RunEventInProgress, entity.RunEventCompleted:
		case entity.RunEventError:
			id, err := c.GenID(ctx)
			if err != nil {
				send(buildErrorEvent(errno.ErrConversationAgentRunError, err.Error()))

			} else {
				send(buildMessageChunkEvent(run.RunEventMessage, buildErrMsg(ackMessageInfo, chunk.Error, id)))
			}
		case entity.RunEventStreamDone:
			send(buildDoneEvent(run.RunEventDone))
		case entity.RunEventAck:
			ackMessageInfo = chunk.ChunkMessageItem
			send(buildMessageChunkEvent(run.RunEventMessage, buildARSM2Message(chunk, req)))
		case entity.RunEventMessageDelta, entity.RunEventMessageCompleted:
			send(buildMessageChunkEvent(run.RunEventMessage, buildARSM2Message(chunk, req)))
		default:
			logs.CtxErrorf(ctx, "unknown handler event:%v", chunk.Event)
		}

	}
}

func buildARSM2Message(chunk *entity.AgentRunResponse, req *run.AgentRunRequest) []byte {
	chunkMessageItem := chunk.ChunkMessageItem

	chunkMessage := &run.RunStreamResponse{
		ConversationID: strconv.FormatInt(chunkMessageItem.ConversationID, 10),
		IsFinish:       ptr.Of(chunk.ChunkMessageItem.IsFinish),
		Message: &message.ChatMessage{
			Role:        string(chunkMessageItem.Role),
			ContentType: string(chunkMessageItem.ContentType),
			MessageID:   strconv.FormatInt(chunkMessageItem.ID, 10),
			SectionID:   strconv.FormatInt(chunkMessageItem.SectionID, 10),
			ContentTime: chunkMessageItem.CreatedAt,
			ExtraInfo:   buildExt(chunkMessageItem.Ext),
			ReplyID:     strconv.FormatInt(chunkMessageItem.ReplyID, 10),

			Status:           "",
			Type:             string(chunkMessageItem.MessageType),
			Content:          chunkMessageItem.Content,
			ReasoningContent: chunkMessageItem.ReasoningContent,
			RequiredAction:   chunkMessageItem.RequiredAction,
		},
		Index: int32(chunkMessageItem.Index),
		SeqID: int32(chunkMessageItem.SeqID),
	}
	if chunkMessageItem.MessageType == crossDomainMessage.MessageTypeAck {
		chunkMessage.Message.Content = req.GetQuery()
		chunkMessage.Message.ContentType = req.GetContentType()
		chunkMessage.Message.ExtraInfo = &message.ExtraInfo{
			LocalMessageID: req.GetLocalMessageID(),
		}
	} else {
		chunkMessage.Message.ExtraInfo = buildExt(chunkMessageItem.Ext)
		chunkMessage.Message.SenderID = ptr.Of(strconv.FormatInt(chunkMessageItem.AgentID, 10))
		chunkMessage.Message.Content = chunkMessageItem.Content

		if chunkMessageItem.MessageType == crossDomainMessage.MessageTypeKnowledge {
			chunkMessage.Message.Type = string(crossDomainMessage.MessageTypeVerbose)
		}
	}

	if chunk.ChunkMessageItem.IsFinish && chunkMessageItem.MessageType == crossDomainMessage.MessageTypeAnswer {
		chunkMessage.Message.Content = ""
		chunkMessage.Message.ReasoningContent = ptr.Of("")
	}

	mCM, _ := json.Marshal(chunkMessage)
	return mCM
}

func buildExt(extra map[string]string) *message.ExtraInfo {
	if extra == nil {
		return nil
	}

	return &message.ExtraInfo{
		InputTokens:    extra["input_tokens"],
		OutputTokens:   extra["output_tokens"],
		Token:          extra["token"],
		PluginStatus:   extra["plugin_status"],
		TimeCost:       extra["time_cost"],
		WorkflowTokens: extra["workflow_tokens"],
		BotState:       extra["bot_state"],
		PluginRequest:  extra["plugin_request"],
		ToolName:       extra["tool_name"],
		Plugin:         extra["plugin"],
		// 透传 call_id:前端按 call_id 配对 function_call 与 tool_response。
		// 缺失时并行工具调用(fc,fc,fc,fc,tr,tr,tr,tr)会退化为索引相邻配对,只有边界一个能收尾,
		// 其余永远停在「正在调用」。ExtraInfo.CallID 的 json tag 即为 "call_id",前端直接消费。
		CallID:              extra["call_id"],
		MockHitInfo:         extra["mock_hit_info"],
		MessageTitle:        extra["message_title"],
		StreamPluginRunning: extra["stream_plugin_running"],
		ExecuteDisplayName:  extra["execute_display_name"],
		TaskType:            extra["task_type"],
		ReferFormat:         extra["refer_format"],
		// Streaming card metadata fields
		YnetType:     extra["ynet_type"],
		CardID:       extra["card_id"],
		TemplateID:   extra["template_id"],
		TemplateName: extra["template_name"],
		CardField:    extra["card_field"],
		// card_delta specific fields
		CardValue: extra["card_value"],
		CardOp:    extra["card_op"],
		// Card group metadata fields
		GroupID:     extra["group_id"],
		CardLayout:  extra["card_layout"],
		CardColumns: extra["card_columns"],
	}
}
func buildErrMsg(ackChunk *entity.ChunkMessageItem, err *entity.RunError, id int64) []byte {
	// 防御:若 run 在收到 Ack 之前就出错(重启打断、模型/多模态立即报错等),
	// ackChunk 为 nil,直接解引用会 panic → 整个请求 500、前端永远转圈。
	// 用零值占位,改为优雅返回错误消息。
	if ackChunk == nil {
		ackChunk = &entity.ChunkMessageItem{}
	}

	chunkMessage := &run.RunStreamResponse{
		IsFinish:       ptr.Of(true),
		ConversationID: strconv.FormatInt(ackChunk.ConversationID, 10),
		Message: &message.ChatMessage{
			Role:        string(schema.Assistant),
			ContentType: string(crossDomainMessage.ContentTypeText),
			Type:        string(crossDomainMessage.MessageTypeAnswer),
			MessageID:   strconv.FormatInt(id, 10),
			SectionID:   strconv.FormatInt(ackChunk.SectionID, 10),
			ReplyID:     strconv.FormatInt(ackChunk.ReplyID, 10),
			Content:     "Something error:" + err.Msg,
			ExtraInfo:   &message.ExtraInfo{},
		},
	}

	mCM, _ := json.Marshal(chunkMessage)
	return mCM
}

func (c *ConversationApplicationService) GenID(ctx context.Context) (int64, error) {
	id, err := c.appContext.IDGen.GenID(ctx)
	return id, err
}

func (c *ConversationApplicationService) checkConversation(ctx context.Context, ar *run.AgentRunRequest, userID int64) (*convEntity.Conversation, error) {
	var conversationData *convEntity.Conversation
	if ar.ConversationID > 0 {
		convByID, err := c.ConversationDomainSVC.GetByID(ctx, ar.ConversationID)
		if err != nil {
			return nil, err
		}
		if convByID != nil && convByID.AgentID == ar.BotID && convByID.ConnectorID == consts.CozeConnectorID && convByID.Scene == ptr.From(ar.Scene) {
			conversationData = convByID
		} else if convByID != nil {
			logs.CtxWarnf(ctx, "conversation %d mismatch. agent:%d expected:%d connector:%d scene:%d", ar.ConversationID, convByID.AgentID, ar.BotID, convByID.ConnectorID, convByID.Scene)
		}
	}

	if conversationData == nil {
		realCurrCon, err := c.ConversationDomainSVC.GetCurrentConversation(ctx, &convEntity.GetCurrent{
			UserID:      userID,
			AgentID:     ar.BotID,
			Scene:       ptr.From(ar.Scene),
			ConnectorID: consts.CozeConnectorID,
		})
		if err != nil {
			return nil, err
		}
		if realCurrCon != nil {
			logs.CtxInfof(ctx, "conversation data loaded, id=%d", realCurrCon.ID)
			conversationData = realCurrCon
		}
	}

	if ar.ConversationID == 0 || conversationData == nil {

		conData, err := c.ConversationDomainSVC.Create(ctx, &convEntity.CreateMeta{
			AgentID:     ar.BotID,
			UserID:      userID,
			Scene:       ptr.From(ar.Scene),
			ConnectorID: consts.CozeConnectorID,
		})
		if err != nil {
			return nil, err
		}
		logs.CtxInfof(ctx, "conversation created, id=%d", conData.ID)
		conversationData = conData

		ar.ConversationID = conversationData.ID
	}

	if conversationData.CreatorID != userID {
		return nil, errorx.New(errno.ErrConversationPermissionCode, errorx.KV("msg", "conversation not match"))
	}

	return conversationData, nil
}

func (c *ConversationApplicationService) checkAgent(ctx context.Context, ar *run.AgentRunRequest) (*saEntity.SingleAgent, error) {
	agentInfo, err := c.appContext.SingleAgentDomainSVC.GetSingleAgent(ctx, ar.BotID, "")
	if err != nil {
		return nil, err
	}

	if agentInfo == nil {
		return nil, errorx.New(errno.ErrAgentNotExists)
	}
	return agentInfo, nil
}

func (c *ConversationApplicationService) buildAgentRunRequest(ctx context.Context, ar *run.AgentRunRequest, userID int64, spaceID int64, conversationData *convEntity.Conversation, shortcutCMD *cmdEntity.ShortcutCmd) (*entity.AgentRunMeta, error) {
	var contentType crossDomainMessage.ContentType
	contentType = crossDomainMessage.ContentTypeText

	if ptr.From(ar.ContentType) != string(crossDomainMessage.ContentTypeText) {
		contentType = crossDomainMessage.ContentTypeMix
	}

	shortcutCMDData, err := c.buildTools(ctx, ar.ToolList, shortcutCMD)

	if err != nil {
		return nil, err
	}

	arm := &entity.AgentRunMeta{
		ConversationID:   conversationData.ID,
		AgentID:          ar.BotID,
		Content:          c.buildMultiContent(ctx, ar),
		DisplayContent:   c.buildDisplayContent(ctx, ar),
		SpaceID:          spaceID,
		UserID:           conv.Int64ToStr(userID),
		SectionID:        conversationData.SectionID,
		PreRetrieveTools: shortcutCMDData,
		IsDraft:          ptr.From(ar.DraftMode),
		ConnectorID:      consts.CozeConnectorID,
		ContentType:      contentType,
		Ext:              ar.Extra,
		CustomVariables:  ar.CustomVariables, // 传递会话自定义变量，用于覆盖智能体预设变量
	}
	return arm, nil
}

func (c *ConversationApplicationService) buildDisplayContent(ctx context.Context, ar *run.AgentRunRequest) string {
	if *ar.ContentType == run.ContentTypeText {
		return ""
	}
	return ar.Query
}

func (c *ConversationApplicationService) buildTools(ctx context.Context, tools []*run.Tool, shortcutCMD *cmdEntity.ShortcutCmd) ([]*entity.Tool, error) {
	var ts []*entity.Tool
	for _, tool := range tools {
		if shortcutCMD != nil {

			arguments := make(map[string]string)
			for key, parametersStruct := range tool.Parameters {
				if parametersStruct == nil {
					continue
				}

				arguments[key] = parametersStruct.Value
				// URI needs to be converted to url.
				if parametersStruct.ResourceType == consts.ShortcutCommandResourceType {

					resourceInfo, err := c.appContext.ImageX.GetResourceURL(ctx, parametersStruct.Value)

					if err != nil {
						return nil, err
					}
					arguments[key] = resourceInfo.URL
				}
			}

			argBytes, err := json.Marshal(arguments)
			if err == nil {
				ts = append(ts, &entity.Tool{
					PluginID:  shortcutCMD.PluginID,
					Arguments: string(argBytes),
					ToolName:  shortcutCMD.PluginToolName,
					ToolID:    shortcutCMD.PluginToolID,
					Type:      agentrun.ToolType(shortcutCMD.ToolType),
				})
			}

		}
	}

	return ts, nil
}

func (c *ConversationApplicationService) buildMultiContent(ctx context.Context, ar *run.AgentRunRequest) []*crossDomainMessage.InputMetaData {
	var multiContents []*crossDomainMessage.InputMetaData

	switch *ar.ContentType {
	case run.ContentTypeText:
		multiContents = append(multiContents, &crossDomainMessage.InputMetaData{
			Type: crossDomainMessage.InputTypeText,
			Text: ar.Query,
		})
	case run.ContentTypeImage, run.ContentTypeFile, run.ContentTypeMix, run.ContentTypeVideo, run.ContentTypeAudio:
		var mc *run.MixContentModel

		err := json.Unmarshal([]byte(ar.Query), &mc)
		if err != nil {
			multiContents = append(multiContents, &crossDomainMessage.InputMetaData{
				Type: crossDomainMessage.InputTypeText,
				Text: ar.Query,
			})
			return multiContents
		}

		mcContent, newItemList := c.parseMultiContent(ctx, mc.ItemList)

		multiContents = append(multiContents, mcContent...)

		mc.ItemList = newItemList
		mcByte, err := json.Marshal(mc)
		if err == nil {
			ar.Query = string(mcByte)
		}
	}

	return multiContents
}

func (c *ConversationApplicationService) parseMultiContent(ctx context.Context, mc []*run.Item) (multiContents []*crossDomainMessage.InputMetaData, mcNew []*run.Item) {
	for index, item := range mc {
		switch item.Type {
		case run.ContentTypeText:
			multiContents = append(multiContents, &crossDomainMessage.InputMetaData{
				Type: crossDomainMessage.InputTypeText,
				Text: item.Text,
			})
		case run.ContentTypeImage:
			if item.Image == nil {
				continue
			}

			resourceUrl := getImageItemExistingURL(item.Image)
			if item.Image.Key != "" {
				signedURL, err := c.getUrlByUri(ctx, item.Image.Key)
				if err != nil {
					logs.CtxErrorf(ctx, "failed to generate resource url, err is %v, will try to use existing URL", err)
				} else if signedURL != "" {
					resourceUrl = signedURL
				}
			}

			if resourceUrl == "" {
				logs.CtxErrorf(ctx, "failed to get resource url, uri is %v", item.Image.Key)
				continue
			}
			if dataURL, ok := normalizeImageDataURL(resourceUrl); ok {
				resourceUrl = dataURL
			}

			if mc[index].Image.ImageThumb == nil {
				mc[index].Image.ImageThumb = &run.ImageDetail{}
			}
			if mc[index].Image.ImageOri == nil {
				mc[index].Image.ImageOri = &run.ImageDetail{}
			}
			mc[index].Image.ImageThumb.URL = resourceUrl
			mc[index].Image.ImageOri.URL = resourceUrl

			// 入库只存对象存储 key（不内联 base64），避免 model_content 被几百KB的 base64 撑爆 DB；
			// 调用模型前会在 parseMessageURI 处把 key 展开成 base64。
			multiContents = append(multiContents, &crossDomainMessage.InputMetaData{
				Type: crossDomainMessage.InputTypeImage,
				FileData: []*crossDomainMessage.FileData{
					{
						Url: item.Image.Key,
						URI: item.Image.Key,
					},
				},
			})
		case run.ContentTypeFile, run.ContentTypeAudio, run.ContentTypeVideo:

			resourceUrl, err := c.getUrlByUri(ctx, item.File.FileKey)
			if err != nil {
				continue
			}

			mc[index].File.FileURL = resourceUrl

			multiContents = append(multiContents, &crossDomainMessage.InputMetaData{
				Type: c.getType(item.File.FileType),
				FileData: []*crossDomainMessage.FileData{
					{
						Url: resourceUrl,
						URI: item.File.FileKey,
					},
				},
			})
		}
	}

	return multiContents, mc
}
func (c *ConversationApplicationService) getType(fileType string) crossDomainMessage.InputType {
	switch fileType {
	case string(crossDomainMessage.InputTypeAudio):
		return crossDomainMessage.InputTypeAudio
	case string(crossDomainMessage.InputTypeVideo):
		return crossDomainMessage.InputTypeVideo
	default:
		return crossDomainMessage.InputTypeFile
	}
}

func (c *ConversationApplicationService) getUrlByUri(ctx context.Context, uri string) (string, error) {
	if uri == "" {
		return "", fmt.Errorf("empty uri")
	}

	if c != nil && c.appContext != nil && c.appContext.ImageX != nil {
		url, err := c.appContext.ImageX.GetResourceURL(ctx, uri)
		if err == nil && url != nil && url.URL != "" {
			return url.URL, nil
		}
		if err != nil {
			logs.CtxWarnf(ctx, "failed to get resource url from ImageX, uri=%s, err=%v", uri, err)
		}
	}

	if c != nil && c.appContext != nil && c.appContext.TosClient != nil {
		return c.appContext.TosClient.GetObjectUrl(ctx, uri)
	}

	return "", fmt.Errorf("resource url service not available")
}

func (c *ConversationApplicationService) toImageModelURL(ctx context.Context, uri, fallbackURL string) string {
	if dataURL, ok := normalizeImageDataURL(fallbackURL); ok {
		return dataURL
	}
	if c == nil || c.appContext == nil || c.appContext.TosClient == nil || uri == "" {
		return fallbackURL
	}

	content, err := c.appContext.TosClient.GetObject(ctx, uri)
	if err != nil {
		logs.CtxWarnf(ctx, "failed to read image object for model input, uri=%s, err=%v", uri, err)
		return fallbackURL
	}
	if len(content) == 0 {
		return fallbackURL
	}

	contentType := mime.TypeByExtension(filepath.Ext(uri))
	if contentType == "" {
		contentType = http.DetectContentType(content)
	}
	if contentType == "" {
		contentType = "image/png"
	}

	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(content)
}

func getImageItemExistingURL(image *run.Image) string {
	if image == nil {
		return ""
	}
	if image.ImageOri != nil && image.ImageOri.URL != "" {
		return image.ImageOri.URL
	}
	if image.ImageThumb != nil && image.ImageThumb.URL != "" {
		return image.ImageThumb.URL
	}
	return ""
}

func isImageDataURL(value string) bool {
	return strings.HasPrefix(value, "data:image/") && strings.Contains(value, ";base64,")
}

func normalizeImageDataURL(value string) (string, bool) {
	if isImageDataURL(value) {
		return value, true
	}
	if looksLikeRawImageBase64(value) {
		return "data:image/png;base64," + value, true
	}
	return value, false
}

func looksLikeRawImageBase64(value string) bool {
	return strings.HasPrefix(value, "/9j/") ||
		strings.HasPrefix(value, "iVBORw0KG") ||
		strings.HasPrefix(value, "R0lGODlh") ||
		strings.HasPrefix(value, "UklGR")
}
