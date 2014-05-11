package render

import (
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/util"
	"github.com/ngerakines/testutils"
	"path/filepath"
	"testing"
	"time"
)

func TestRenderJpegPreview(t *testing.T) {
	if testing.Short() {
		t.Skip("Short Tests Only: TestRenderJpegPreview")
		return
	}
	dm := testutils.NewDirectoryManager()
	defer dm.Close()

	sasm := common.NewSourceAssetStorageManager()
	gasm := common.NewGeneratedAssetStorageManager()

	tm := common.NewTemplateManager()
	tfm := common.NewTemporaryFileManager()
	downloader := common.NewDownloader(dm.Path, tfm)
	uploader := newMockUploader()
	rm := NewRendererManager(gasm, tfm)
	defer rm.Stop()

	rm.AddImageMagickRenderer(sasm, tm, downloader, uploader, 5)

	sourceAsset := common.NewSourceAsset("101099BE-AF41-4D2D-A385-BFE44CC94B48", common.SourceAssetTypeOrigin)
	sourceAsset.AddAttribute(common.SourceAssetAttributeSize, []string{"12345"})
	sourceAsset.AddAttribute(common.SourceAssetAttributeSource, []string{fileUrl("test-data", "wallpaper-641916.jpg")})
	sourceAsset.AddAttribute(common.SourceAssetAttributeType, []string{"jpg"})

	err := sasm.Store(sourceAsset)
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}

	templates, err := tm.FindByIds(common.LegacyDefaultTemplates)
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}
	for _, template := range templates {
		ga := common.NewGeneratedAssetFromSourceAsset(sourceAsset, template, "location")
		gasm.Store(ga)
	}

	callback := make(chan bool)
	go func() {
		for {
			generatedAssets, err := gasm.FindBySourceAssetId(sourceAsset.Id)
			if err == nil {
				count := 0
				for _, generatedAsset := range generatedAssets {
					if generatedAsset.Status == common.GeneratedAssetStatusComplete {
						count = count + 1
					}
				}
				if count == 4 {
					callback <- true
				}
			}
		}
	}()

	select {
	case <-callback:
		return
	case <-time.After(10 * time.Second):
		{
			t.Error("Test timed out.")
			return
		}
	}
}

func fileUrl(dir, file string) string {
	return "file://" + filepath.Join(util.Cwd(), "../", dir, file)
}

type mockUploader struct {
}

func (uploader *mockUploader) Upload(destination string, path string) error {
	return nil
}

func newMockUploader() common.Uploader {
	return new(mockUploader)
}
