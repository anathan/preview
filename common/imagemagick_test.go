package common

import (
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

	sasm := NewSourceAssetStorageManager()
	gasm := NewGeneratedAssetStorageManager()

	tm := NewTemplateManager()
	tfm := NewTemporaryFileManager()
	downloader := NewDownloader(dm.Path, tfm)
	uploader := newMockUploader()
	rm := NewRendererManager(gasm, tfm)
	defer rm.Stop()

	rm.AddImageMagickRenderer(sasm, tm, downloader, uploader, 5)

	sourceAsset := NewSourceAsset("101099BE-AF41-4D2D-A385-BFE44CC94B48", SourceAssetTypeOrigin)
	sourceAsset.AddAttribute(SourceAssetAttributeSize, []string{"12345"})
	sourceAsset.AddAttribute(SourceAssetAttributeSource, []string{fileUrl("test-data", "wallpaper-641916.jpg")})
	sourceAsset.AddAttribute(SourceAssetAttributeType, []string{"jpg"})

	err := sasm.Store(sourceAsset)
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}

	templates, err := tm.FindByIds(LegacyDefaultTemplates)
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}
	for _, template := range templates {
		ga := NewGeneratedAssetFromSourceAsset(sourceAsset, template, "location")
		gasm.Store(ga)
	}

	callback := make(chan bool)
	go func() {
		for {
			generatedAssets, err := gasm.FindBySourceAssetId(sourceAsset.Id)
			if err == nil {
				count := 0
				for _, generatedAsset := range generatedAssets {
					if generatedAsset.Status == GeneratedAssetStatusComplete {
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
	return "file://" + filepath.Join(util.Cwd(), dir, file)
}
