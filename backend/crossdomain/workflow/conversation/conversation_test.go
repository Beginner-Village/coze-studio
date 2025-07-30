package conversation

import (
	"testing"

	apimessage "github.com/coze-dev/coze-studio/backend/api/model/crossdomain/message"
	"github.com/coze-dev/coze-studio/backend/domain/conversation/message/entity"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/crossdomain/conversation"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/stretchr/testify/assert"
)

func Test_convertMessage(t *testing.T) {
	type args struct {
		lr *entity.ListResult
	}
	tests := []struct {
		name    string
		args    args
		want    *conversation.MessageListResponse
		wantErr bool
	}{
		{
			name: "pure text",
			args: args{
				lr: &entity.ListResult{
					Messages: []*entity.Message{
						{
							ID:          1,
							Role:        "user",
							ContentType: "text",
							MultiContent: []*apimessage.InputMetaData{
								{
									Type: "text",
									Text: "hello",
								},
							},
						},
					},
				},
			},
			want: &conversation.MessageListResponse{
				Messages: []*conversation.Message{
					{
						ID:          1,
						Role:        "user",
						ContentType: "text",
						MultiContent: []*conversation.Content{
							{Type: "text", Text: ptr.Of("hello")},
						},
					},
				},
			},
		},
		{
			name: "pure file",
			args: args{
				lr: &entity.ListResult{
					Messages: []*entity.Message{
						{
							ID:          2,
							Role:        "user",
							ContentType: "file",
							MultiContent: []*apimessage.InputMetaData{
								{
									Type: "file",
									FileData: []*apimessage.FileData{
										{
											Url: "f_uri_1",
										},
									},
								},
								{
									Type: "text",
									Text: "",
								},
							},
						},
					},
				},
			},
			want: &conversation.MessageListResponse{
				Messages: []*conversation.Message{
					{
						ID:          2,
						Role:        "user",
						ContentType: "file",
						MultiContent: []*conversation.Content{
							{Type: "file", Uri: ptr.Of("f_uri_1")},
							{Type: "text", Text: ptr.Of("")},
						},
					},
				},
			},
		},
		{
			name: "text and file",
			args: args{
				lr: &entity.ListResult{
					Messages: []*entity.Message{
						{
							ID:          3,
							Role:        "user",
							ContentType: "text_file",
							MultiContent: []*apimessage.InputMetaData{
								{
									Type: "text",
									Text: "hello",
								},
								{
									Type: "file",
									FileData: []*apimessage.FileData{
										{
											Url: "f_uri_2",
										},
									},
								},
							},
						},
					},
				},
			},
			want: &conversation.MessageListResponse{
				Messages: []*conversation.Message{
					{
						ID:          3,
						Role:        "user",
						ContentType: "text_file",
						MultiContent: []*conversation.Content{
							{Type: "text", Text: ptr.Of("hello")},
							{Type: "file", Uri: ptr.Of("f_uri_2")},
						},
					},
				},
			},
		},
		{
			name: "multiple files",
			args: args{
				lr: &entity.ListResult{
					Messages: []*entity.Message{
						{
							ID:          4,
							Role:        "user",
							ContentType: "file",
							MultiContent: []*apimessage.InputMetaData{
								{
									Type: "file",
									FileData: []*apimessage.FileData{
										{
											Url: "f_uri_3",
										},
										{
											Url: "f_uri_4",
										},
									},
								},
								{
									Type: "text",
									Text: "",
								},
							},
						},
					},
				},
			},
			want: &conversation.MessageListResponse{
				Messages: []*conversation.Message{
					{
						ID:          4,
						Role:        "user",
						ContentType: "file",
						MultiContent: []*conversation.Content{
							{Type: "file", Uri: ptr.Of("f_uri_3")},
							{Type: "file", Uri: ptr.Of("f_uri_4")},
							{Type: "text", Text: ptr.Of("")},
						},
					},
				},
			},
		},
		{
			name: "empty text",
			args: args{
				lr: &entity.ListResult{
					Messages: []*entity.Message{
						{
							ID:          5,
							Role:        "user",
							ContentType: "text",
							MultiContent: []*apimessage.InputMetaData{
								{
									Type: "text",
									Text: "",
								},
							},
						},
					},
				},
			},
			want: &conversation.MessageListResponse{
				Messages: []*conversation.Message{
					{
						ID:          5,
						Role:        "user",
						ContentType: "text",
						MultiContent: []*conversation.Content{
							{Type: "text", Text: ptr.Of("")},
						},
					},
				},
			},
		},
		{
			name: "pure image",
			args: args{
				lr: &entity.ListResult{
					Messages: []*entity.Message{
						{
							ID:          6,
							Role:        "user",
							ContentType: "image",
							MultiContent: []*apimessage.InputMetaData{
								{
									Type: "image",
									FileData: []*apimessage.FileData{
										{
											Url: "image_uri_5",
										},
									},
								},
								{
									Type: "text",
									Text: "",
								},
							},
						},
					},
				},
			},
			want: &conversation.MessageListResponse{
				Messages: []*conversation.Message{
					{
						ID:          6,
						Role:        "user",
						ContentType: "image",
						MultiContent: []*conversation.Content{
							{Type: "image", Uri: ptr.Of("image_uri_5")},
							{Type: "text", Text: ptr.Of("")},
						},
					},
				},
			},
		},
		{
			name: "multiple images",
			args: args{
				lr: &entity.ListResult{
					Messages: []*entity.Message{
						{
							ID:          7,
							Role:        "user",
							ContentType: "image",
							MultiContent: []*apimessage.InputMetaData{
								{
									Type: "image",
									FileData: []*apimessage.FileData{
										{
											Url: "file_id_6",
										},
										{
											Url: "file_id_7",
										},
									},
								},
								{
									Type: "text",
									Text: "",
								},
							},
						},
					},
				},
			},
			want: &conversation.MessageListResponse{
				Messages: []*conversation.Message{
					{
						ID:          7,
						Role:        "user",
						ContentType: "image",
						MultiContent: []*conversation.Content{
							{Type: "image", Uri: ptr.Of("file_id_6")},
							{Type: "image", Uri: ptr.Of("file_id_7")},
							{Type: "text", Text: ptr.Of("")},
						},
					},
				},
			},
		},
		{
			name: "mixed content",
			args: args{
				lr: &entity.ListResult{
					Messages: []*entity.Message{
						{
							ID:          8,
							Role:        "user",
							ContentType: "mix",
							MultiContent: []*apimessage.InputMetaData{
								{
									Type: "text",
									Text: "hello",
								},
								{
									Type: "image",
									FileData: []*apimessage.FileData{
										{
											Url: "file_id_8",
										},
									},
								},
								{
									Type: "file",
									FileData: []*apimessage.FileData{
										{
											Url: "file_id_9",
										},
									},
								},
							},
						},
					},
				},
			},
			want: &conversation.MessageListResponse{
				Messages: []*conversation.Message{
					{
						ID:          8,
						Role:        "user",
						ContentType: "mix",
						MultiContent: []*conversation.Content{
							{Type: "text", Text: ptr.Of("hello")},
							{Type: "image", Uri: ptr.Of("file_id_8")},
							{Type: "file", Uri: ptr.Of("file_id_9")},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs, err := convertMessage(tt.args.lr.Messages)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for i, msg := range msgs {
				assert.Equal(t, msg.MultiContent, tt.want.Messages[i].MultiContent)
			}

		})
	}
}
