package api

import (
	"github.com/codegangsta/martini"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/util"
	"net/http"
	"path/filepath"
	"strings"
)

type assetBlueprint struct {
	base                         string
	sourceAssetStorageManager    common.SourceAssetStorageManager
	generatedAssetStorageManager common.GeneratedAssetStorageManager
	templateManager              common.TemplateManager
	placeholderManager           common.PlaceholderManager
	localAssetStoragePath        string
	templatesBySize              map[string]string
}

type assetAction int

var (
	assetAction404       = assetAction(0)
	assetActionServeFile = assetAction(1)
	assetActionRedirect  = assetAction(2)
)

// NewAssetBlueprint creates, configures and returns a new blueprint. This structure contains the state and HTTP controllers used to serve assets.
func NewAssetBlueprint(
	localAssetStoragePath string,
	sourceAssetStorageManager common.SourceAssetStorageManager,
	generatedAssetStorageManager common.GeneratedAssetStorageManager,
	templateManager common.TemplateManager,
	placeholderManager common.PlaceholderManager) *assetBlueprint {

	blueprint := new(assetBlueprint)
	blueprint.base = "/asset"
	blueprint.sourceAssetStorageManager = sourceAssetStorageManager
	blueprint.generatedAssetStorageManager = generatedAssetStorageManager
	blueprint.templateManager = templateManager
	blueprint.placeholderManager = placeholderManager
	blueprint.localAssetStoragePath = localAssetStoragePath

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

// ConfigureMartini adds the assetBlueprint handlers/controllers to martini.
func (blueprint *assetBlueprint) ConfigureMartini(m *martini.ClassicMartini) error {
	m.Get(blueprint.base+"/**", blueprint.assetHandler)
	return nil
}

func (blueprint *assetBlueprint) assetHandler(res http.ResponseWriter, req *http.Request) {
	splitIndex := len(blueprint.base + "/")
	parts := strings.Split(req.URL.Path[splitIndex:], "/")

	if len(parts) != 2 {
		res.Header().Set("Content-Length", "0")
		res.WriteHeader(404)
		return
	}

	action, path := blueprint.getAsset(parts[0], parts[1])
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
	}
	res.Header().Set("Content-Length", "0")
	res.WriteHeader(404)
	return
}

func (blueprint *assetBlueprint) getAsset(fileId, placeholderSize string) (assetAction, string) {

	generatedAssets, err := blueprint.generatedAssetStorageManager.FindBySourceAssetId(fileId)
	if err != nil {
		return assetAction404, ""
	}

	templateId, hasTemplateId := blueprint.templatesBySize[placeholderSize]
	if hasTemplateId {
		for _, generatedAsset := range generatedAssets {
			if generatedAsset.TemplateId == templateId {
				if strings.HasPrefix(generatedAsset.Location, "local://") {
					fullPath := filepath.Join(blueprint.localAssetStoragePath, fileId, placeholderSize)
					if util.CanLoadFile(fullPath) {
						return assetActionServeFile, fullPath
					}
					placeholder := blueprint.placeholderManager.Url(fileId, placeholderSize)
					if util.CanLoadFile(placeholder.Path) {
						return assetActionServeFile, placeholder.Path
					}
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
