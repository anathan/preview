package config

import (
	"github.com/ngerakines/preview/util"
	"os"
	"path/filepath"
)

func NewDefaultAppConfig() (AppConfig, error) {
	return buildDefaultConfig(defaultBasePath)
}

func NewDefaultAppConfigWithBaseDirectory(root string) (AppConfig, error) {
	return buildDefaultConfig(func(section string) string {
		cacheDirectory := filepath.Join(root, ".cache", section)
		os.MkdirAll(cacheDirectory, 00777)
		return cacheDirectory
	})
}

func buildDefaultConfig(basePathFunc basePath) (AppConfig, error) {
	config := `{
   "common": {
      "placeholderBasePath":"` + basePathFunc("placeholders") + `",
      "placeholderGroups": {
         "image":["jpg", "jpeg", "png", "gif"],
         "document":["pdf", "doc", "docx"]
      },
      "localAssetStoragePath":"` + basePathFunc("assets") + `",
      "nodeId":"E876F147E331"
   },
   "http":{
      "listen":":8080"
   },
   "storage":{
      "engine":"memory"
   },
   "documentRenderAgent":{
      "enabled":true,
      "count":16,
      "basePath":"` + basePathFunc("documentRenderAgentTmp") + `"
   },
   "imageMagickRenderAgent":{
      "enabled":true,
      "count":16,
      "supportedFileTypes":{
         "jpg":33554432,
         "jpeg":33554432,
         "png":33554432,
         "gif":33554432,
         "pdf":33554432
      }
   },
   "simpleApi":{
      "enabled":true,
      "edgeBaseUrl":"http://localhost:8080"
   },
   "assetApi":{
      "enabled":true
   },
   "uploader":{
      "engine":"local"
   },
   "downloader":{
      "basePath":"` + basePathFunc("cache") + `",
      "tramEnabled": false
   }
}`
	return NewUserAppConfig([]byte(config))
}

type basePath func(string) string

func defaultBasePath(section string) string {
	cacheDirectory := filepath.Join(util.Cwd(), ".cache", section)
	os.MkdirAll(cacheDirectory, 00777)
	return cacheDirectory
}
