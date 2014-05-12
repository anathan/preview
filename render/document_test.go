package render

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/testutils"
	"log"
	"testing"
	"time"
)

func TestConvertDocxToPdf(t *testing.T) {
	if testing.Short() {
		t.Skip("Short Tests Only: TestConvertDocxToPdf")
		return
	}

	common.DumpErrors()

	dm := testutils.NewDirectoryManager()
	defer dm.Close()

	rm, sasm, gasm, tm := setupTest(dm.Path)
	defer rm.Stop()

	testListener := make(RenderStatusChannel, 25)
	rm.AddListener(testListener)

	sourceAssetId := uuid.New()
	sourceAsset := common.NewSourceAsset(sourceAssetId, common.SourceAssetTypeOrigin)
	sourceAsset.AddAttribute(common.SourceAssetAttributeSize, []string{"12345"})
	sourceAsset.AddAttribute(common.SourceAssetAttributeSource, []string{fileUrl("test-data", "ChefConf2014schedule.docx")})
	sourceAsset.AddAttribute(common.SourceAssetAttributeType, []string{"docx"})

	err := sasm.Store(sourceAsset)
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}

	templates, err := tm.FindByIds([]string{common.DocumentConversionTemplateId})
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}
	for _, template := range templates {
		log.Println("Found template", template.Id, "with service", template.Renderer)
		ga := common.NewGeneratedAssetFromSourceAsset(sourceAsset, template, fmt.Sprintf("local:///%s/pdf", sourceAssetId))
		gasm.Store(ga)
	}
	time.Sleep(10 * time.Second)
	if assertGeneratedAssetCount(sourceAssetId, gasm, common.GeneratedAssetStatusComplete, 5) {
		t.Errorf("Could not verify that %d generated assets had status '%s' for source asset '%s'", 5, common.GeneratedAssetStatusComplete, sourceAssetId)
		return
	}
}
