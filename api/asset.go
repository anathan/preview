package api

import (
	"github.com/bmizerany/pat"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/util"
	"github.com/rcrowley/go-metrics"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

type staticBlueprint struct {
	base               string
	placeholderManager common.PlaceholderManager
}

type assetBlueprint struct {
	base                         string
	sourceAssetStorageManager    common.SourceAssetStorageManager
	generatedAssetStorageManager common.GeneratedAssetStorageManager
	templateManager              common.TemplateManager
	placeholderManager           common.PlaceholderManager
	s3Client                     common.S3Client
	signatureManager             SignatureManager
	localAssetStoragePath        string
	templatesBySize              map[string]string

	requestsMeter               metrics.Meter
	malformedRequestsMeter      metrics.Meter
	emptyRequestsMeter          metrics.Meter
	unknownGeneratedAssetsMeter metrics.Meter
}

type assetAction int

var (
	assetAction404       = assetAction(0)
	assetActionServeFile = assetAction(1)
	assetActionRedirect  = assetAction(2)
	assetActionS3Proxy   = assetAction(3)
)

// NewAssetBlueprint creates, configures and returns a new blueprint. This structure contains the state and HTTP controllers used to serve assets.
func NewAssetBlueprint(
	registry metrics.Registry,
	localAssetStoragePath string,
	sourceAssetStorageManager common.SourceAssetStorageManager,
	generatedAssetStorageManager common.GeneratedAssetStorageManager,
	templateManager common.TemplateManager,
	placeholderManager common.PlaceholderManager,
	s3Client common.S3Client,
	signatureManager SignatureManager) *assetBlueprint {

	blueprint := new(assetBlueprint)
	blueprint.base = "/asset"
	blueprint.sourceAssetStorageManager = sourceAssetStorageManager
	blueprint.generatedAssetStorageManager = generatedAssetStorageManager
	blueprint.templateManager = templateManager
	blueprint.placeholderManager = placeholderManager
	blueprint.localAssetStoragePath = localAssetStoragePath
	blueprint.s3Client = s3Client
	blueprint.signatureManager = signatureManager

	blueprint.requestsMeter = metrics.NewMeter()
	blueprint.malformedRequestsMeter = metrics.NewMeter()
	blueprint.emptyRequestsMeter = metrics.NewMeter()
	blueprint.unknownGeneratedAssetsMeter = metrics.NewMeter()
	registry.Register("assetApi.requests", blueprint.requestsMeter)
	registry.Register("assetApi.malformedRequests", blueprint.malformedRequestsMeter)
	registry.Register("assetApi.emptyRequests", blueprint.emptyRequestsMeter)
	registry.Register("assetApi.unknownGeneratedAssets", blueprint.unknownGeneratedAssetsMeter)

	var err error
	if err != nil {
		panic(err)
	}

	blueprint.templatesBySize = make(map[string]string)

	legacyTemplates, err := blueprint.templateManager.FindByIds(common.LegacyDefaultTemplates)
	if err == nil {
		for _, legacyTemplate := range legacyTemplates {
			placeholderSize, err := common.GetFirstAttribute(legacyTemplate, common.TemplateAttributePlaceholderSize)
			if err == nil {
				blueprint.templatesBySize[placeholderSize] = legacyTemplate.Id
			}
		}
	}

	return blueprint
}

func (blueprint *assetBlueprint) AddRoutes(p *pat.PatternServeMux) {
	p.Get(blueprint.base+"/:id/:template/:page", http.HandlerFunc(blueprint.assetHandler))
}

func (blueprint *assetBlueprint) assetHandler(res http.ResponseWriter, req *http.Request) {
	blueprint.requestsMeter.Mark(1)

	assetId := req.URL.Query().Get(":id")
	templateAlias := req.URL.Query().Get(":template")
	page := req.URL.Query().Get(":page")

	action, path := blueprint.getAsset(assetId, templateAlias, page)
	switch action {
	case assetActionServeFile:
		{
			http.ServeFile(res, req, path)
			return
		}
	case assetActionRedirect:
		{
			http.Redirect(res, req, path, 302)
			return
		}
	case assetActionS3Proxy:
		{
			bucket, file := blueprint.splitS3Url(path)
			err := blueprint.s3Client.Proxy(bucket, file, res)
			if err != nil {
				return
			}
		}
	}
	blueprint.emptyRequestsMeter.Mark(1)
	http.NotFound(res, req)
}

func (blueprint *assetBlueprint) splitS3Url(url string) (string, string) {
	usableData := url[5:]
	// NKG: The url will have the following format: `s3://[bucket][path]`
	// where path will begin with a `/` character.
	parts := strings.SplitN(usableData, "/", 2)
	return parts[0], parts[1]
}

func (blueprint *assetBlueprint) getAsset(fileId, placeholderSize, page string) (assetAction, string) {

	generatedAssets, err := blueprint.generatedAssetStorageManager.FindBySourceAssetId(fileId)
	if err != nil {
		blueprint.unknownGeneratedAssetsMeter.Mark(1)
		return assetAction404, ""
	}
	if len(generatedAssets) == 0 {
		blueprint.unknownGeneratedAssetsMeter.Mark(1)
	}

	templateId, hasTemplateId := blueprint.templatesBySize[placeholderSize]
	if hasTemplateId {
		for _, generatedAsset := range generatedAssets {
			pageVal, _ := common.GetFirstAttribute(generatedAsset, common.GeneratedAssetAttributePage)
			if len(pageVal) == 0 {
				pageVal = "0"
			}
			pageMatch := pageVal == page
			if generatedAsset.TemplateId == templateId && pageMatch {
				if strings.HasPrefix(generatedAsset.Location, "local://") {
					fullPath := filepath.Join(blueprint.localAssetStoragePath, fileId, placeholderSize, page)
					if util.CanLoadFile(fullPath) {
						return assetActionServeFile, fullPath
					}
					placeholder := blueprint.placeholderManager.Url(fileId, placeholderSize)
					if util.CanLoadFile(placeholder.Path) {
						return assetActionServeFile, placeholder.Path
					}
				}
				if strings.HasPrefix(generatedAsset.Location, "s3://") {
					return assetActionS3Proxy, generatedAsset.Location
				}
			}
		}
	}
	placeholder := blueprint.placeholderManager.Url(fileId, placeholderSize)
	if util.CanLoadFile(placeholder.Path) {
		return assetActionServeFile, placeholder.Path
	}

	return assetAction404, ""
}

func NewStaticBlueprint(placeholderManager common.PlaceholderManager) *staticBlueprint {
	blueprint := new(staticBlueprint)
	blueprint.base = "/static"
	blueprint.placeholderManager = placeholderManager
	return blueprint
}

func (blueprint *staticBlueprint) AddRoutes(p *pat.PatternServeMux) {
	p.Get(blueprint.base+"/:fileType/:placeholderSize", http.HandlerFunc(blueprint.RequestHandler))
}

func (blueprint *staticBlueprint) RequestHandler(res http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.URL.Path[len(blueprint.base+"/"):], "/")
	log.Println("parts", parts)

	if len(parts) == 2 {
		placeholder := blueprint.placeholderManager.Url(parts[0], parts[1])
		if placeholder.Path != "" {
			http.ServeFile(res, req, placeholder.Path)
			return
		}
	}

	res.Header().Set("Content-Length", "0")
	res.WriteHeader(500)
}
