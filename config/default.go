package config

import (
	"os"
	"path/filepath"
)

func NewDefaultAppConfig() (AppConfig, error) {
	basePath := func() string {
		pwd, err := os.Getwd()
		if err != nil {
			panic(err.Error())
		}
		cacheDirectory := filepath.Join(pwd, ".cache")
		os.MkdirAll(cacheDirectory, 00777)
		return cacheDirectory
	}()
	config := `{
   "common": {
      "placeholderBasePath":"` + basePath + `",
      "placeholderGroups": {
         "image": ["jpg", "jpeg", "png", "gif"]
      },
      "nodeId": "E876F147E331"
   },
   "http":{
      "listen":":8080"
   },
   "storage":{
      "engine":"memory"
   },
   "imageMagickRenderer":{
      "enabled":true,
      "count":16,
      "supportedFileTypes":{
         "jpg":33554432,
         "jpeg":33554432,
         "png":33554432
      }
   },
   "simpleApi":{
      "enabled":true,
      "edgeBaseUrl":"http://localhost:8080"
   },
   "assetApi":{
      "basePath":"` + basePath + `"
   },
   "uploader":{
      "engine":"local",
      "localBasePath":"` + basePath + `"
   },
   "downloader":{
      "basePath":"` + basePath + `"
   }
}`
	return NewUserAppConfig([]byte(config))
}
