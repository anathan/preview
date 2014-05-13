package app

import (
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/config"
	"github.com/ngerakines/preview/render"
	"github.com/ngerakines/preview/util"
	"github.com/ngerakines/testutils"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBasicIntegration(t *testing.T) {
	if !testutils.Integration() {
		t.Skip("Skipping integration test")
		return
	}

	common.DumpErrors()

	dm := testutils.NewDirectoryManager()
	defer dm.Close()

	testConfig, err := cassandraConfig(dm.Path)
	if err != nil {
		t.Error("No error expected when creating app:", err)
		return
	}

	previewApp, err := NewApp(testConfig)
	if err != nil {
		t.Error("No error expected when creating app:", err)
		return
	}
	defer previewApp.Stop()

	testListener := make(render.RenderStatusChannel)
	previewApp.agentManager.AddListener(testListener)

	sourceAssetId := uuid.New()

	func() {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/v1/preview/"+sourceAssetId, strings.NewReader(composeTextPayload("jpg", fileUrl("../test-data", "wallpaper-641916.jpg"), "252990")))
		previewApp.martiniClassic.ServeHTTP(res, req)

		if res.Code != 202 {
			t.Errorf("Invalid response: %d", res.Code)
			return
		}
	}()

	generatedAssets := make([]string, 0, 0)
	statusCount := 0
	for statusCount < 4 {
		select {
		case statusEvent := <-testListener:
			{
				generatedAssets = append(generatedAssets, statusEvent.GeneratedAssetId)
				statusCount++
			}
		case <-time.After(10 * time.Second):
			{
				t.Error("Test timed out.")
				return
			}
		}
	}

	callback := make(chan bool)
	verifyGeneratedAssets(4, sourceAssetId, callback, previewApp)

	select {
	case <-callback:
		return
	case <-time.After(10 * time.Second):
		{
			t.Error("Test timed out.")
			return
		}
	}

	func() {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/preview/"+sourceAssetId, nil)
		previewApp.martiniClassic.ServeHTTP(res, req)

		if res.Code != 200 {
			t.Errorf("Invalid response: %d", res.Code)
			return
		}

		body := res.Body.Bytes()
		log.Println(string(body))
		if !strings.Contains(string(body), sourceAssetId) {
			t.Errorf("Response body does not contain '%s': %s", sourceAssetId, string(body))
		}
		for _, placeholderSize := range common.DefaultPlaceholderSizes {
			url := "http://localhost:8080/local/" + sourceAssetId + "/" + placeholderSize
			if !strings.Contains(string(body), url) {
				t.Errorf("Response body does not contain '%s': %s", url, string(body))
			}
		}
	}()

	for _, placeholderSize := range common.DefaultPlaceholderSizes {
		func() {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/local/"+sourceAssetId+"/"+placeholderSize, nil)
			previewApp.martiniClassic.ServeHTTP(res, req)

			if res.Code != 200 {
				t.Errorf("Invalid response: %d", res.Code)
				return
			}
			expectedBytes, err := ioutil.ReadFile(filepath.Join(dm.Path, sourceAssetId, placeholderSize))
			if err != nil {
				t.Error("No error expected when reading file", err)
				return
			}
			if !bytes.Equal(expectedBytes, res.Body.Bytes()) {
				t.Error("image different than expected.", len(res.Body.Bytes()), len(expectedBytes))
			}
		}()
	}

}

func cassandraConfig(tmpFilePath string) (config.AppConfig, error) {
	return config.NewUserAppConfig([]byte(`{
		"common": {"nodeId": "foo", "localAssetStoragePath": "` + tmpFilePath + `", "placeholderBasePath": "` + filepath.Join(util.Cwd(), "../test-data/placeholders/") + `", "placeholderGroups": {"image": ["jpg"]} },
		"http": {"listen": ":8081"},
		"storage": {"engine": "cassandra", "cassandraNodes": ["localhost"], "cassandraKeyspace": "preview"},
		"imageMagickRenderAgent": {"enabled": true, "count": 16, "supportedFileTypes":{"jpg": 33554432}},
		"documentRenderAgent": {"enabled": true, "count": 16,"basePath": "` + tmpFilePath + `"},
		"simpleApi": {"enabled": true, "edgeBaseUrl": "http://localhost:8080"},
		"assetApi": {"enabled": true},
		"uploader": {"engine": "local", "localBasePath": "` + tmpFilePath + `"},
		"downloader": {"basePath": "` + tmpFilePath + `"}
		}`))
}

func composeTextPayload(fileType, url, size string) string {
	return fmt.Sprintf("type: %s\nurl: %s\nsize: %s\nbatch_id: 4431", fileType, url, size)
}

func verifyGeneratedAssets(count int, sourceAssetId string, callback chan bool, previewApp *AppContext) {
	go func() {
		for {
			generatedAssets, err := previewApp.generatedAssetStorageManager.FindBySourceAssetId(sourceAssetId)
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
}

func fileUrl(dir, file string) string {
	return "file://" + filepath.Join(util.Cwd(), dir, file)
}
