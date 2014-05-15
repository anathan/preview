package common

type mockUploader struct {
}

func (uploader *mockUploader) Upload(destination string, path string) error {
	return nil
}

func (uploader *mockUploader) Url(sourceAssetId, templateId, placeholderSize string, page int32) string {
	return "mock://" + sourceAssetId
}

func newMockUploader() Uploader {
	return new(mockUploader)
}
