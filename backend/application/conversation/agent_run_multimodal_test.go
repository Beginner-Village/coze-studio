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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ynet-dev/ynet-studio/backend/api/model/conversation/run"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/storage"
)

type conversationImageStorageFake struct {
	urls    map[string]string
	objects map[string][]byte
}

func (f *conversationImageStorageFake) PutObject(context.Context, string, []byte, ...storage.PutOptFn) error {
	return nil
}

func (f *conversationImageStorageFake) GetObject(_ context.Context, objectKey string) ([]byte, error) {
	return f.objects[objectKey], nil
}

func (f *conversationImageStorageFake) DeleteObject(context.Context, string) error {
	return nil
}

func (f *conversationImageStorageFake) GetObjectUrl(_ context.Context, objectKey string, _ ...storage.GetOptFn) (string, error) {
	return f.urls[objectKey], nil
}

func TestParseMultiContentUsesDataURLForImageModelInput(t *testing.T) {
	const (
		imageURI  = "images/test.png"
		signedURL = "http://10.10.10.226:9000/openynet/images/test.png"
	)

	svc := &ConversationApplicationService{
		appContext: &ServiceComponents{
			TosClient: &conversationImageStorageFake{
				urls: map[string]string{
					imageURI: signedURL,
				},
				objects: map[string][]byte{
					imageURI: []byte{0x89, 0x50, 0x4e, 0x47},
				},
			},
		},
	}

	items := []*run.Item{
		{
			Type: run.ContentTypeImage,
			Image: &run.Image{
				Key:        imageURI,
				ImageThumb: &run.ImageDetail{},
				ImageOri:   &run.ImageDetail{},
			},
		},
	}

	got, updated := svc.parseMultiContent(context.Background(), items)

	require.Len(t, got, 1)
	require.Len(t, got[0].FileData, 1)
	require.Equal(t, imageURI, got[0].FileData[0].URI)
	require.Equal(t, "data:image/png;base64,iVBORw==", got[0].FileData[0].Url)
	require.Equal(t, signedURL, updated[0].Image.ImageThumb.URL)
	require.Equal(t, signedURL, updated[0].Image.ImageOri.URL)
}

func TestParseMultiContentAcceptsDataURLImageWithoutObjectKey(t *testing.T) {
	const dataURL = "data:image/png;base64,iVBORw=="

	svc := &ConversationApplicationService{}

	items := []*run.Item{
		{
			Type: run.ContentTypeImage,
			Image: &run.Image{
				ImageOri: &run.ImageDetail{URL: dataURL},
			},
		},
	}

	got, updated := svc.parseMultiContent(context.Background(), items)

	require.Len(t, got, 1)
	require.Len(t, got[0].FileData, 1)
	require.Empty(t, got[0].FileData[0].URI)
	require.Equal(t, dataURL, got[0].FileData[0].Url)
	require.Equal(t, dataURL, updated[0].Image.ImageThumb.URL)
	require.Equal(t, dataURL, updated[0].Image.ImageOri.URL)
}
