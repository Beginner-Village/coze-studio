package coze

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	crossknowledge "github.com/coze-dev/coze-studio/backend/api/model/crossdomain/knowledge"
	"github.com/coze-dev/coze-studio/backend/application/base/ctxutil"
	knowledgecontract "github.com/coze-dev/coze-studio/backend/crossdomain/contract/knowledge"
)

type RetrieveTestRequest struct {
	DatasetID          string   `json:"dataset_id"`
	Query              string   `json:"query"`
	TopK               *int64   `json:"top_k,omitempty"`
	MinScore           *float64 `json:"min_score,omitempty"`
	SearchType         string   `json:"search_type,omitempty"`
	EnableQueryRewrite *bool    `json:"enable_query_rewrite,omitempty"`
	EnableRerank       *bool    `json:"enable_rerank,omitempty"`
	EnableNL2SQL       *bool    `json:"enable_nl2sql,omitempty"`
}

type RetrieveTestResponse struct {
	Code int32                  `json:"code"`
	Msg  string                 `json:"msg"`
	Data *RetrieveTestResultSet `json:"data,omitempty"`
}

type RetrieveTestResultSet struct {
	Total int32              `json:"total"`
	Hits  []*RetrieveTestHit `json:"hits"`
}

type RetrieveTestHit struct {
	SliceID      string  `json:"slice_id"`
	KnowledgeID  string  `json:"knowledge_id"`
	DocumentID   string  `json:"document_id"`
	DocumentName string  `json:"document_name"`
	Content      string  `json:"content"`
	Answer       string  `json:"answer,omitempty"`
	Score        float64 `json:"score"`
}

// RetrieveTest .
// @router /api/knowledge/retrieve_test [POST]
func RetrieveTest(ctx context.Context, c *app.RequestContext) {
	var req RetrieveTestRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(req.Query) == "" {
		c.JSON(consts.StatusBadRequest, &RetrieveTestResponse{
			Code: 400,
			Msg:  "query is required",
		})
		return
	}
	if strings.TrimSpace(req.DatasetID) == "" {
		c.JSON(consts.StatusBadRequest, &RetrieveTestResponse{
			Code: 400,
			Msg:  "dataset_id is required",
		})
		return
	}
	datasetID, err := strconv.ParseInt(req.DatasetID, 10, 64)
	if err != nil || datasetID <= 0 {
		c.JSON(consts.StatusBadRequest, &RetrieveTestResponse{
			Code: 400,
			Msg:  "dataset_id is invalid",
		})
		return
	}

	uid := ctxutil.GetUIDFromCtx(ctx)
	if uid == nil {
		c.JSON(consts.StatusUnauthorized, &RetrieveTestResponse{
			Code: 401,
			Msg:  "unauthorized",
		})
		return
	}

	searchType, err := parseRetrieveSearchType(req.SearchType)
	if err != nil {
		c.JSON(consts.StatusBadRequest, &RetrieveTestResponse{
			Code: 400,
			Msg:  err.Error(),
		})
		return
	}

	enableRewrite := true
	if req.EnableQueryRewrite != nil {
		enableRewrite = *req.EnableQueryRewrite
	}
	enableRerank := true
	if req.EnableRerank != nil {
		enableRerank = *req.EnableRerank
	}
	enableNL2SQL := false
	if req.EnableNL2SQL != nil {
		enableNL2SQL = *req.EnableNL2SQL
	}

	retrieveReq := &crossknowledge.RetrieveRequest{
		Query:        strings.TrimSpace(req.Query),
		KnowledgeIDs: []int64{datasetID},
		Strategy: &crossknowledge.RetrievalStrategy{
			TopK:               req.TopK,
			MinScore:           req.MinScore,
			SearchType:         searchType,
			EnableQueryRewrite: enableRewrite,
			EnableRerank:       enableRerank,
			EnableNL2SQL:       enableNL2SQL,
		},
	}

	resp, err := knowledgecontract.DefaultSVC().Retrieve(ctx, retrieveReq)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}

	result := &RetrieveTestResultSet{
		Hits: make([]*RetrieveTestHit, 0, len(resp.RetrieveSlices)),
	}
	for _, item := range resp.RetrieveSlices {
		if item == nil || item.Slice == nil {
			continue
		}
		s := item.Slice
		result.Hits = append(result.Hits, &RetrieveTestHit{
			SliceID:      strconv.FormatInt(s.ID, 10),
			KnowledgeID:  strconv.FormatInt(s.KnowledgeID, 10),
			DocumentID:   strconv.FormatInt(s.DocumentID, 10),
			DocumentName: s.DocumentName,
			Content:      s.GetSliceContent(),
			Answer:       s.Answer,
			Score:        item.Score,
		})
	}
	result.Total = int32(len(result.Hits))

	c.JSON(consts.StatusOK, &RetrieveTestResponse{
		Code: 0,
		Msg:  "ok",
		Data: result,
	})
}

func parseRetrieveSearchType(searchType string) (crossknowledge.SearchType, error) {
	switch strings.ToLower(strings.TrimSpace(searchType)) {
	case "", "semantic":
		return crossknowledge.SearchTypeSemantic, nil
	case "fulltext", "full_text":
		return crossknowledge.SearchTypeFullText, nil
	case "hybrid":
		return crossknowledge.SearchTypeHybrid, nil
	default:
		return 0, errors.New("search_type must be one of: semantic, fulltext, hybrid")
	}
}
