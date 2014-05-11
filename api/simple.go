package api

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/martini"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/config"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type simpleBlueprint struct {
	base                         string
	edgeContentHost              string
	sourceAssetStorageManager    common.SourceAssetStorageManager
	generatedAssetStorageManager common.GeneratedAssetStorageManager
	templateManager              common.TemplateManager
	placeholderManager           common.PlaceholderManager
	supportedFileTypes           map[string]int64
}

// NewSimpleBlueprint creates a new simpleBlueprint object.
func NewSimpleBlueprint(
	appConfig config.AppConfig,
	sourceAssetStorageManager common.SourceAssetStorageManager,
	generatedAssetStorageManager common.GeneratedAssetStorageManager,
	templateManager common.TemplateManager,
	placeholderManager common.PlaceholderManager,
	supportedFileTypes map[string]int64) (*simpleBlueprint, error) {
	blueprint := new(simpleBlueprint)
	blueprint.base = "/api"
	blueprint.edgeContentHost = appConfig.SimpleApi().EdgeBaseUrl()
	blueprint.sourceAssetStorageManager = sourceAssetStorageManager
	blueprint.generatedAssetStorageManager = generatedAssetStorageManager
	blueprint.templateManager = templateManager
	blueprint.placeholderManager = placeholderManager
	blueprint.supportedFileTypes = supportedFileTypes
	return blueprint, nil
}

func (blueprint *simpleBlueprint) ConfigureMartini(m *martini.ClassicMartini) error {
	m.Put(blueprint.buildUrl("/v1/preview"), blueprint.GeneratePreviewHandler)
	m.Put(blueprint.buildUrl("/v1/preview/:fileid"), blueprint.GeneratePreviewHandler)
	m.Get(blueprint.buildUrl("/v1/preview"), blueprint.PreviewInfoHandler)
	m.Get(blueprint.buildUrl("/v1/preview/:fileid"), blueprint.PreviewInfoHandler)
	return nil
}

func (blueprint *simpleBlueprint) buildUrl(path string) string {
	return blueprint.base + path
}

func (blueprint *simpleBlueprint) GeneratePreviewHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != "PUT" {
		// TODO: Make sure this is the correct status code being returned.
		res.Header().Set("Content-Length", "0")
		res.WriteHeader(500)
		return
	}

	if !strings.HasPrefix(req.URL.Path, blueprint.buildUrl("/v1/preview")) {
		// TODO: Make sure this is the correct status code being returned.
		res.Header().Set("Content-Length", "0")
		res.WriteHeader(500)
		return
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		res.Header().Set("Content-Length", "0")
		res.WriteHeader(500)
		return
	}
	defer req.Body.Close()

	id, hasId := blueprint.urlHasFileId(req.URL.Path)
	if hasId {
		gprs, err := newGeneratePreviewRequestFromText(id, string(body))
		if err != nil {
			res.Header().Set("Content-Length", "0")
			res.WriteHeader(500)
			return
		}
		blueprint.handleGeneratePreviewRequest(gprs)
	} else {
		gprs, err := newGeneratePreviewRequestFromJson(string(body))
		if err != nil {
			res.Header().Set("Content-Length", "0")
			res.WriteHeader(500)
			return
		}
		blueprint.handleGeneratePreviewRequest(gprs)
	}

	res.Header().Set("Content-Length", "0")
	res.WriteHeader(200)
}

func (blueprint *simpleBlueprint) PreviewInfoHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		// TODO: Make sure this is the correct status code being returned.
		res.Header().Set("Content-Length", "0")
		res.WriteHeader(500)
		return
	}

	if !strings.HasPrefix(req.URL.Path, blueprint.buildUrl("/v1/preview")) {
		// TODO: Make sure this is the correct status code being returned.
		res.Header().Set("Content-Length", "0")
		res.WriteHeader(500)
		return
	}
	fileIds := blueprint.parseFileIds(req)
	previewInfo, err := blueprint.handlePreviewInfoRequest(fileIds)
	if err != nil {
		res.Header().Set("Content-Length", "0")
		res.WriteHeader(500)
		return
	}

	res.Header().Set("Content-Length", strconv.Itoa(len(previewInfo)))
	res.Write(previewInfo)
}

func (blueprint *simpleBlueprint) urlHasFileId(url string) (string, bool) {
	index := len(blueprint.buildUrl("/v1/preview/"))
	if len(url) > index {
		return url[index:], true
	}
	return "", false
}

func (blueprint *simpleBlueprint) handleGeneratePreviewRequest(gprs []*generatePreviewRequest) {
	defaultTemplates, err := blueprint.templateManager.FindByIds(common.LegacyDefaultTemplates)
	if err != nil {
		return
	}

	for _, gpr := range gprs {
		sourceAsset := common.NewSourceAsset(gpr.id, common.SourceAssetTypeOrigin)
		sourceAsset.AddAttribute(common.SourceAssetAttributeSize, []string{strconv.FormatInt(gpr.size, 10)})
		sourceAsset.AddAttribute(common.SourceAssetAttributeSource, []string{gpr.url})
		sourceAsset.AddAttribute(common.SourceAssetAttributeType, []string{gpr.requestType})
		// TODO: Add support for the expiration attribute.

		blueprint.sourceAssetStorageManager.Store(sourceAsset)

		status := blueprint.generatedAssetStatus(gpr.requestType, gpr.size)
		for _, template := range defaultTemplates {
			placeholderSize, _ := blueprint.templatePlaceholderSize(template)
			location := fmt.Sprintf("local:///%s/%s", gpr.id, placeholderSize)
			ga := common.NewGeneratedAssetFromSourceAsset(sourceAsset, template, location)
			ga.Status = status
			ga.AddAttribute(common.GeneratedAssetAttributePage, []string{"0"})
			blueprint.generatedAssetStorageManager.Store(ga)
		}
	}
}

func (blueprint *simpleBlueprint) generatedAssetStatus(fileType string, fileSize int64) string {
	// TODO: Check the expiration of the file.
	maxSize, hasEntry := blueprint.supportedFileTypes[fileType]
	if !hasEntry {
		return common.NewGeneratedAssetError(common.ErrorNoRenderersSupportFileType)
	}
	if fileSize > maxSize {
		return common.NewGeneratedAssetError(common.ErrorFileTooLarge)
	}
	return common.GeneratedAssetStatusWaiting
}

func (blueprint *simpleBlueprint) parseFileIds(req *http.Request) []string {
	results := make([]string, 0, 0)

	// NKG: See if the url contains a file id
	url := req.URL.Path
	index := len(blueprint.buildUrl("/v1/preview/"))
	if len(url) > index {
		results = append(results, url[index:])
	}

	// NKG: Pull any file ids from the query string parameters.
	queryValues := req.URL.Query()
	for key, values := range queryValues {
		if key == "file_id" {
			for _, value := range values {
				fileIds := strings.Split(value, ",")
				for _, fileId := range fileIds {
					results = append(results, fileId)
				}
			}
		}
	}
	return results
}

type templateTuple struct {
	placeholderSize string
	template        *common.Template
}

func (blueprint *simpleBlueprint) handlePreviewInfoRequest(fileIds []string) ([]byte, error) {
	collections := make([]*previewInfoCollection, 0, 0)

	legacyTemplates, err := blueprint.templateManager.FindByIds(common.LegacyDefaultTemplates)
	if err != nil {
		return nil, err
	}

	templates := make(map[string]templateTuple)
	for _, legacyTemplate := range legacyTemplates {
		placeholderSize, err := blueprint.templatePlaceholderSize(legacyTemplate)
		if err != nil {
			return nil, err
		}
		templates[legacyTemplate.Id] = templateTuple{placeholderSize, legacyTemplate}
	}

	for _, fileId := range fileIds {
		log.Println("Creating collection")

		collection := &previewInfoCollection{}
		collection.FileId = fileId

		sourceAssets, err := blueprint.sourceAssetStorageManager.FindBySourceAssetId(fileId)
		if err != nil {
			return nil, err
		}
		fileType := blueprint.getFileType(sourceAssets)

		generatedAssets, err := blueprint.generatedAssetStorageManager.FindBySourceAssetId(fileId)
		if err != nil {
			return nil, err
		}

		for _, generatedAsset := range generatedAssets {
			templateTuple, hasTemplateTuple := templates[generatedAsset.TemplateId]
			if hasTemplateTuple {
				switch templateTuple.placeholderSize {
				case common.PlaceholderSizeSmall:
					collection.Small = blueprint.getPreviewImage(generatedAsset, fileType, templateTuple.placeholderSize)
				case common.PlaceholderSizeMedium:
					collection.Medium = blueprint.getPreviewImage(generatedAsset, fileType, templateTuple.placeholderSize)
				case common.PlaceholderSizeLarge:
					collection.Large = blueprint.getPreviewImage(generatedAsset, fileType, templateTuple.placeholderSize)
				case common.PlaceholderSizeJumbo:
					collection.Jumbo = blueprint.getPreviewImage(generatedAsset, fileType, templateTuple.placeholderSize)
				}
			}
		}

		if collection.Small == nil {
			collection.Small = blueprint.getPlaceholder(fileType, common.PlaceholderSizeSmall)
		}
		if collection.Medium == nil {
			collection.Medium = blueprint.getPlaceholder(fileType, common.PlaceholderSizeMedium)
		}
		if collection.Large == nil {
			collection.Large = blueprint.getPlaceholder(fileType, common.PlaceholderSizeLarge)
		}
		if collection.Jumbo == nil {
			collection.Jumbo = blueprint.getPlaceholder(fileType, common.PlaceholderSizeJumbo)
		}

		collections = append(collections, collection)
	}

	response := &previewInfoResponse{"1", collections}

	return json.Marshal(response)
}

func (blueprint *simpleBlueprint) scrubUrl(location string) string {
	if strings.HasPrefix(location, "s3://") {
		parts := strings.SplitN(location[5:], "/", 2)
		if len(parts) == 2 {
			return "http://s3-host/" + parts[0] + "/" + parts[1]
		}
	}
	if strings.HasPrefix(location, "local://") {
		log.Println("about to split location", location)
		parts := strings.Split(location[9:], "/")
		log.Println("location split into", parts)
		return blueprint.edgeContentHost + "/local/" + parts[0] + "/" + parts[1]
	}
	return location
}

func (blueprint *simpleBlueprint) signUrl(url string) string {
	return url
}

func (blueprint *simpleBlueprint) templatePlaceholderSize(template *common.Template) (string, error) {
	if !template.HasAttribute(common.TemplateAttributePlaceholderSize) {
		// TODO: write this code
		return "", common.ErrorNotImplemented
	}
	placeholderSizes := template.GetAttribute(common.TemplateAttributePlaceholderSize)
	if len(placeholderSizes) < 1 {
		// TODO: write this code
		return "", common.ErrorNotImplemented
	}
	placeholderSize := placeholderSizes[0]
	return placeholderSize, nil
}

func (blueprint *simpleBlueprint) getPreviewImage(generatedAsset *common.GeneratedAsset, fileType, placeholderSize string) *imageInfo {
	log.Println("Building preview image for", generatedAsset)
	if generatedAsset.Status == common.GeneratedAssetStatusComplete {
		return &imageInfo{blueprint.scrubUrl(generatedAsset.Location), 200, 200, 0, true, true}
	}
	if strings.HasPrefix(generatedAsset.Status, common.GeneratedAssetStatusFailed) {
		// NKG: If the job failed, then before we return the placeholder, we set the "isFinal" field.
		placeholder := blueprint.getPlaceholder(fileType, placeholderSize)
		placeholder.IsFinal = true
		return placeholder
	}
	return blueprint.getPlaceholder(fileType, placeholderSize)
}

func (blueprint *simpleBlueprint) getPlaceholder(fileType, placeholderSize string) *imageInfo {
	placeholder := blueprint.placeholderManager.Url(fileType, placeholderSize)
	return &imageInfo{blueprint.edgeContentHost + "/static" + placeholder.Url, 200, 200, 0, false, false}
}

func (blueprint *simpleBlueprint) getFileType(sourceAssets []*common.SourceAsset) string {
	if len(sourceAssets) > 0 {
		sourceAsset := sourceAssets[0]
		if sourceAsset.HasAttribute(common.SourceAssetAttributeType) {
			fileTypes := sourceAsset.GetAttribute(common.SourceAssetAttributeType)
			if len(fileTypes) > 0 {
				return fileTypes[0]
			}
		}
	}
	return common.DefaultPlaceholderType
}
