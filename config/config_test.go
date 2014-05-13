package config

import (
	"github.com/ngerakines/testutils"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"testing"
)

type tempFileManager struct {
	path  string
	files map[string]string
}

func (fm *tempFileManager) initFile(name, body string) {
	path := filepath.Join(fm.path, name)
	err := ioutil.WriteFile(path, []byte(body), 00777)
	if err != nil {
		log.Fatal(err)
	}
	fm.files[name] = path
}

func (fm *tempFileManager) get(name string) (string, error) {
	path, hasPath := fm.files[name]
	if hasPath {
		return path, nil
	}
	return "", appConfigError{"No config file exists with that label."}
}

func initTempFileManager(path string) *tempFileManager {
	fm := new(tempFileManager)
	fm.path = path
	fm.files = make(map[string]string)
	fm.initFile("basic", `{
		"http": {"listen": ":8081"},
		"common": {"nodeId": "9D7DB7FC75B4", "placeholderBasePath": "./", "placeholderGroups": {"image": ["jpg"]}, "localAssetStoragePath":"./"},
		"storage": {"engine": "cassandra", "cassandraNodes": ["localhost"], "cassandraKeyspace": "preview"},
		"imageMagickRenderAgent": {"enabled": true, "count": 16, "supportedFileTypes":{"jpg": 123456}},
		"documentRenderAgent": {"enabled": true, "count": 16, "basePath": "./"},
		"simpleApi": {"enabled": true, "edgeBaseUrl": "http://localhost:8080"},
		"assetApi": {"basePath": "./", "enabled": true},
		"uploader": {"engine": "s3", "s3Key": "foo", "s3Secret": "bar", "s3Host": "baz", "s3Buckets": ["previewa", "previewb"]},
		"downloader": {"basePath": "./", "tramEnabled": false}
		}`)
	return fm
}

func TestDefaultConfig(t *testing.T) {
	dm := testutils.NewDirectoryManager()
	defer dm.Close()

	appConfig, err := NewDefaultAppConfig()
	if err != nil {
		t.Error(err.Error())
		return
	}

	if appConfig.Http().Listen() != ":8080" {
		t.Error("Invalid default for appConfig.Http().Listen()", appConfig.Http().Listen())
		return
	}
	if appConfig.Storage().Engine() != "memory" {
		t.Error("Invalid default for appConfig.Storage().Engine()", appConfig.Storage().Engine())
		return
	}
}

func TestBasicConfig(t *testing.T) {
	dm := testutils.NewDirectoryManager()
	defer dm.Close()
	fm := initTempFileManager(dm.Path)

	path, err := fm.get("basic")
	if err != nil {
		t.Error(err.Error())
		return
	}
	appConfig, err := LoadAppConfig(path)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if appConfig.Http().Listen() != ":8081" {
		t.Error("appConfig.Http().Listen()", appConfig.Http().Listen())
	}

	if appConfig.Storage().Engine() != "cassandra" {
		t.Error("appConfig.Storage().Engine()", appConfig.Storage().Engine())
	}
	cassandraNodes, err := appConfig.Storage().CassandraNodes()
	if strings.Join(cassandraNodes, ",") != "localhost" {
		t.Error("appConfig.Storage().CassandraNodes()", cassandraNodes)
	}

	if appConfig.ImageMagickRenderAgent().Enabled() != true {
		t.Error("Invalid default for appConfig.ImageMagickRenderAgent().Enabled()", appConfig.ImageMagickRenderAgent().Enabled())
	}
	if len(appConfig.ImageMagickRenderAgent().SupportedFileTypes()) != 1 {
		t.Error("Invalid count for appConfig.ImageMagickRenderAgent().SupportedFileTypes()", len(appConfig.ImageMagickRenderAgent().SupportedFileTypes()))
	}
	if appConfig.ImageMagickRenderAgent().SupportedFileTypes()["jpg"] != 123456 {
		t.Error("Invalid default for appConfig.ImageMagickRenderAgent().SupportedFileTypes()[\"jpg\"]", appConfig.ImageMagickRenderAgent().SupportedFileTypes()["jpg"])
	}

	if appConfig.SimpleApi().Enabled() != true {
		t.Error("Invalid default for appConfig.SimpleApi().Enabled()", appConfig.SimpleApi().Enabled())
	}
}
