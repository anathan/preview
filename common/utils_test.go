package common

type mockUploader struct {
}

func (uploader *mockUploader) Upload(destination string, path string) error {
	return nil
}

func newMockUploader() Uploader {
	return new(mockUploader)
}
