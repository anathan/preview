package api

import (
	"fmt"
	"github.com/codegangsta/martini"
	"github.com/ngerakines/preview/common"
	_ "github.com/ngerakines/testutils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeneratePreviewHandlerBasic(t *testing.T) {
	blueprint := testSimpleBlueprint()
	m := martini.Classic()
	blueprint.ConfigureMartini(m)

	func() {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/v1/preview/1234", strings.NewReader("type: html\nurl: http://github.com/\nsize: 4123"))
		m.ServeHTTP(res, req)

		if res.Code != 200 {
			t.Errorf("Invalid response: %d", res.Code)
		}
	}()
}

/*
Scenario: A valid request is sent with a text payload that describes a file that can be rendered.
Expects: Source asset for the payload, Generated assets that are waiting.
*/
func TestGeneratePreviewHandlerSuccess(t *testing.T) {

	blueprint := testSimpleBlueprint()
	m := martini.Classic()
	blueprint.ConfigureMartini(m)

	func() {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/v1/preview/85DC8226CCC1", strings.NewReader(composeTextPayload("jpg", "http://hightail.com/logo.jpg", "2231")))
		m.ServeHTTP(res, req)

		if res.Code != 200 {
			t.Errorf("Invalid response: %d", res.Code)
			return
		}
		sourceAssets, err := blueprint.sourceAssetStorageManager.FindBySourceAssetId("85DC8226CCC1")
		if err != nil {
			t.Error("No error expected when looking up source asset.")
			return
		}
		if len(sourceAssets) != 1 {
			t.Error("One or more results expected.")
			return
		}
		if sourceAssets[0].Id != "85DC8226CCC1" {
			t.Errorf("Invalid result returned: %s", sourceAssets[0])
			return
		}
		if sourceAssets[0].IdType != "origin" {
			t.Errorf("Invalid result returned: %s", sourceAssets[0])
			return
		}

		generatedAssets, err := blueprint.generatedAssetStorageManager.FindBySourceAssetId("85DC8226CCC1")
		if err != nil {
			t.Error("No error expected when looking up generated assets.")
			return
		}
		if len(generatedAssets) != 4 {
			t.Error("Four generated assets expected.")
			return
		}
		for _, generatedAsset := range generatedAssets {
			if generatedAsset.Status != "waiting" {
				t.Errorf("Invalid result returned: %s", generatedAsset)
				return
			}
		}
	}()
}

/*
Scenario: A valid request is sent with a text payload that describes a file that has expired.
Expects: Source asset for the payload, Generated assets that have failed.
*/

/*
Scenario: A valid request is sent with a text payload that describes a file that has no render support.
Expects: Source asset for the payload, Generated assets that have failed.
*/

/*
Scenario: A valid request is sent with a text payload that describes a file that is too large.
Expects: Source asset for the payload, Generated assets that have failed.
*/

func testSimpleBlueprint() *simpleBlueprint {
	blueprint := new(simpleBlueprint)
	blueprint.base = ""
	blueprint.templateManager = common.NewTemplateManager()
	blueprint.sourceAssetStorageManager = common.NewSourceAssetStorageManager()
	blueprint.generatedAssetStorageManager = common.NewGeneratedAssetStorageManager(blueprint.templateManager)
	blueprint.supportedFileTypes = map[string]int64{"jpg": 9999999}
	return blueprint
}

func composeTextPayload(fileType, url, size string) string {
	return fmt.Sprintf("type: %s\nurl: %s\nsize: %s\nbatch_id: 4431", fileType, url, size)
}
