package common

import (
	"log"
	"time"
)

type SourceAssetStorageManager interface {
	Store(sourceAsset *SourceAsset) error
	FindBySourceAssetId(id string) ([]*SourceAsset, error)
}

type GeneratedAssetStorageManager interface {
	Store(generatedAsset *GeneratedAsset) error
	Update(generatedAsset *GeneratedAsset) error
	FindById(id string) (*GeneratedAsset, error)
	FindByIds(ids []string) ([]*GeneratedAsset, error)
	FindBySourceAssetId(id string) ([]*GeneratedAsset, error)
	FindWorkForService(serviceName string, workCount int) ([]*GeneratedAsset, error)
}

type TemplateManager interface {
	Store(template *Template) error
	FindByIds(id []string) ([]*Template, error)
	FindByRenderService(renderService string) ([]*Template, error)
}

type inMemorySourceAssetStorageManager struct {
	sourceAssets []*SourceAsset
}

type inMemoryGeneratedAssetStorageManager struct {
	generatedAssets []*GeneratedAsset
	templateManager TemplateManager
}

type inMemoryTemplateManager struct {
	templates []*Template
}

func NewSourceAssetStorageManager() SourceAssetStorageManager {
	return &inMemorySourceAssetStorageManager{make([]*SourceAsset, 0, 0)}
}

func NewGeneratedAssetStorageManager(templateManager TemplateManager) GeneratedAssetStorageManager {
	return &inMemoryGeneratedAssetStorageManager{make([]*GeneratedAsset, 0, 0), templateManager}
}

func NewTemplateManager() TemplateManager {
	tm := new(inMemoryTemplateManager)
	tm.templates = make([]*Template, 0, 0)
	tm.Store(DefaultTemplateJumbo)
	tm.Store(DefaultTemplateLarge)
	tm.Store(DefaultTemplateMedium)
	tm.Store(DefaultTemplateSmall)
	tm.Store(DocumentConversionTemplate)
	return tm
}

func (sasm *inMemorySourceAssetStorageManager) Store(sourceAsset *SourceAsset) error {
	sasm.sourceAssets = append(sasm.sourceAssets, sourceAsset)
	return nil
}

func (sasm *inMemorySourceAssetStorageManager) FindBySourceAssetId(id string) ([]*SourceAsset, error) {
	results := make([]*SourceAsset, 0, 0)
	for _, sourceAsset := range sasm.sourceAssets {
		if sourceAsset.Id == id {
			results = append(results, sourceAsset)
		}
	}
	return results, nil
}

func (gasm *inMemoryGeneratedAssetStorageManager) Store(generatedAsset *GeneratedAsset) error {
	gasm.generatedAssets = append(gasm.generatedAssets, generatedAsset)
	return nil
}

func (gasm *inMemoryGeneratedAssetStorageManager) FindById(id string) (*GeneratedAsset, error) {
	for _, generatedAsset := range gasm.generatedAssets {
		if generatedAsset.Id == id {
			return generatedAsset, nil
		}
	}
	return nil, ErrorNoGeneratedAssetsFoundForId
}

func (gasm *inMemoryGeneratedAssetStorageManager) FindByIds(ids []string) ([]*GeneratedAsset, error) {
	results := make([]*GeneratedAsset, 0, 0)
	for _, generatedAsset := range gasm.generatedAssets {
		for _, id := range ids {
			if generatedAsset.Id == id {
				results = append(results, generatedAsset)
			}
		}
	}
	return results, nil
}

func (gasm *inMemoryGeneratedAssetStorageManager) FindBySourceAssetId(id string) ([]*GeneratedAsset, error) {
	results := make([]*GeneratedAsset, 0, 0)
	for _, generatedAsset := range gasm.generatedAssets {
		if generatedAsset.SourceAssetId == id {
			results = append(results, generatedAsset)
		}
	}
	return results, nil
}

func (gasm *inMemoryGeneratedAssetStorageManager) FindWorkForService(serviceName string, workCount int) ([]*GeneratedAsset, error) {
	templates, _ := gasm.templateManager.FindByRenderService(serviceName)
	log.Println("templates for", serviceName, ":", templates)
	results := make([]*GeneratedAsset, 0, 0)
	for _, generatedAsset := range gasm.generatedAssets {
		for _, template := range templates {
			if generatedAsset.TemplateId == template.Id {
				if generatedAsset.Status == GeneratedAssetStatusWaiting {
					generatedAsset.Status = GeneratedAssetStatusScheduled
					generatedAsset.UpdatedAt = time.Now().UnixNano()
					results = append(results, generatedAsset)
				}
				if len(results) >= workCount {
					return results, nil
				}
			}
		}
	}
	log.Println("generated assets for service", serviceName, ":", buildGeneratedAssetIds(results))
	return results, nil
}

func buildGeneratedAssetIds(generatedAssets []*GeneratedAsset) []string {
	results := make([]string, len(generatedAssets))
	for index, generatedAsset := range generatedAssets {
		results[index] = generatedAsset.Id
	}
	return results
}

func (gasm *inMemoryGeneratedAssetStorageManager) Update(givenGeneratedAsset *GeneratedAsset) error {
	for _, generatedAsset := range gasm.generatedAssets {
		if generatedAsset.Id == givenGeneratedAsset.Id {
			generatedAsset.Status = givenGeneratedAsset.Status
			generatedAsset.Attributes = givenGeneratedAsset.Attributes
			generatedAsset.UpdatedAt = time.Now().UnixNano()
			return nil
		}

	}
	return ErrorGeneratedAssetCouldNotBeUpdated
}

func (tm *inMemoryTemplateManager) Store(template *Template) error {
	tm.templates = append(tm.templates, template)
	return nil

}

func (tm *inMemoryTemplateManager) FindByIds(ids []string) ([]*Template, error) {
	results := make([]*Template, 0, 0)
	for _, template := range tm.templates {
		for _, id := range ids {
			if template.Id == id {
				results = append(results, template)
			}
		}
	}
	return results, nil
}

func (tm *inMemoryTemplateManager) FindByRenderService(renderService string) ([]*Template, error) {
	results := make([]*Template, 0, 0)
	for _, template := range tm.templates {
		if template.Renderer == renderService {
			results = append(results, template)
		}
	}
	return results, nil
}
