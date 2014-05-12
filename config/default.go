package config

import (
	"github.com/ngerakines/preview/util"
	"os"
	"path/filepath"
)

func NewDefaultAppConfig() (AppConfig, error) {
	basePath := func(section string) string {
		cacheDirectory := filepath.Join(util.Cwd(), ".cache", section)
		os.MkdirAll(cacheDirectory, 00777)
		return cacheDirectory
	}
	config := `{
   "common": {
      "placeholderBasePath":"` + basePath("placeholders") + `",
      "placeholderGroups": {
         "image":["jpg", "jpeg", "png", "gif"],
         "document":["pdf", "doc", "docx"]
      },
      "localAssetStoragePath":"` + basePath("assets") + `",
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
      "basePath":"` + basePath("documentRenderAgentTmp") + `"
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
      "basePath":"` + basePath("cache") + `"
   }
}`
	return NewUserAppConfig([]byte(config))
}

func NewDefaultAppConfigWithBaseDirectory(root string) (AppConfig, error) {
	basePath := func(section string) string {
		cacheDirectory := filepath.Join(root, ".cache", section)
		os.MkdirAll(cacheDirectory, 00777)
		return cacheDirectory
	}
	config := `{
   "common": {
      "placeholderBasePath":"` + basePath("placeholders") + `",
      "placeholderGroups": {
         "image":["jpg", "jpeg", "png", "gif"],
         "document":["pdf", "doc", "docx"]
      },
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
      "basePath":"` + basePath("documentRenderAgentTmp") + `"
   },
   "imageMagickRenderer":{
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
      "basePath":"` + basePath("assets") + `"
   },
   "uploader":{
      "engine":"local",
      "localBasePath":"` + basePath("assets") + `"
   },
   "downloader":{
      "basePath":"` + basePath("cache") + `"
   }
}`
	return NewUserAppConfig([]byte(config))
}
