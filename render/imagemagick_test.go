package render

import (
	"code.google.com/p/go-uuid/uuid"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/util"
	"github.com/ngerakines/testutils"
	"path/filepath"
	"testing"
	"time"
)

// TODO: Write tests for source assets. (i.e. missing, missing/invalid attributes)
// TODO: Write tests for generated assets. (i.e. missing, missing/invalid attributes)
// TODO: Write test for different supported file types. (i.e. jpg, png, gif, pdf)
// TODO: Write test for PDF with 0 pages.
// TODO: Write test for PDF with more than 1 page.
// TODO: Write test for animated gif.

func TestRenderJpegPreview(t *testing.T) {
	if testing.Short() {
		t.Skip("Short Tests Only: TestRenderPdfPreview")
		return
	}

	dm := testutils.NewDirectoryManager()
	defer dm.Close()

	rm, sasm, gasm, tm := setupTest(dm.Path)
	defer rm.Stop()

	sourceAssetId := uuid.New()
	sourceAsset := common.NewSourceAsset(sourceAssetId, common.SourceAssetTypeOrigin)
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
	if assertGeneratedAssetCount(sourceAssetId, gasm, common.GeneratedAssetStatusComplete, 4) {
		t.Errorf("Could not verify that %d generated assets had status '%s' for source asset '%s'", 4, common.GeneratedAssetStatusComplete, sourceAssetId)
		return
	}
}

func TestRenderPdfPreview(t *testing.T) {
	if testing.Short() {
		t.Skip("Short Tests Only: TestRenderPdfPreview")
		return
	}

	dm := testutils.NewDirectoryManager()
	defer dm.Close()

	rm, sasm, gasm, tm := setupTest(dm.Path)
	defer rm.Stop()

	sourceAssetId := uuid.New()
	sourceAsset := common.NewSourceAsset(sourceAssetId, common.SourceAssetTypeOrigin)
	sourceAsset.AddAttribute(common.SourceAssetAttributeSize, []string{"12345"})
	sourceAsset.AddAttribute(common.SourceAssetAttributeSource, []string{fileUrl("test-data", "ChefConf2014schedule.pdf")})
	sourceAsset.AddAttribute(common.SourceAssetAttributeType, []string{"pdf"})

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
		ga.AddAttribute(common.GeneratedAssetAttributePage, []string{"0"})
		gasm.Store(ga)
	}
	if assertGeneratedAssetCount(sourceAssetId, gasm, common.GeneratedAssetStatusComplete, 4) {
		t.Errorf("Could not verify that %d generated assets had status '%s' for source asset '%s'", 4, common.GeneratedAssetStatusComplete, sourceAssetId)
		return
	}
}

func assertGeneratedAssetCount(id string, generatedAssetStorageManager common.GeneratedAssetStorageManager, status string, expectedCount int) bool {
	callback := make(chan bool)
	go func() {
		for {
			generatedAssets, err := generatedAssetStorageManager.FindBySourceAssetId(id)
			if err == nil {
				count := 0
				for _, generatedAsset := range generatedAssets {
					if generatedAsset.Status == status {
						count = count + 1
					}
				}
				if count == expectedCount {
					callback <- true
				}
			}
		}
	}()

	select {
	case <-callback:
		return false
	case <-time.After(10 * time.Second):
		return true
	}
}

func setupTest(path string) (*RendererManager, common.SourceAssetStorageManager, common.GeneratedAssetStorageManager, common.TemplateManager) {
	sasm := common.NewSourceAssetStorageManager()
	gasm := common.NewGeneratedAssetStorageManager()

	tm := common.NewTemplateManager()
	tfm := common.NewTemporaryFileManager()
	downloader := common.NewDownloader(path, tfm)
	uploader := newMockUploader()
	rm := NewRendererManager(gasm, tfm)

	rm.AddImageMagickRenderer(sasm, tm, downloader, uploader, 5)

	return rm, sasm, gasm, tm
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
