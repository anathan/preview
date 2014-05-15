package api

import (
	"bytes"
	"encoding/json"
	"github.com/codegangsta/martini"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/config"
	"github.com/ngerakines/preview/render"
	"github.com/rcrowley/go-metrics"
	"net/http"
	"strconv"
)

type adminBlueprint struct {
	base                 string
	registry             metrics.Registry
	appConfig            config.AppConfig
	placeholderManager   common.PlaceholderManager
	temporaryFileManager common.TemporaryFileManager
	agentManager         *render.RenderAgentManager
}

type placeholdersView struct {
	Placeholders []*common.Placeholder
}

type temporaryFilesView struct {
	Files map[string]int
}

type renderAgentViewElement struct {
	Count      int      `json:"count"`
	Enabled    bool     `json:"enabled"`
	ActiveWork []string `json:"activeWork"`
}

type renderAgentsView struct {
	RenderAgents map[string]renderAgentViewElement `json:"renderAgents"`
}

type errorViewError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type errorsView struct {
	Errors []errorViewError `json:"errors"`
}

// NewAdminBlueprint creates a new adminBlueprint object.
func NewAdminBlueprint(registry metrics.Registry, appConfig config.AppConfig, placeholderManager common.PlaceholderManager, temporaryFileManager common.TemporaryFileManager, agentManager *render.RenderAgentManager) *adminBlueprint {
	blueprint := new(adminBlueprint)
	blueprint.base = "/admin"
	blueprint.registry = registry
	blueprint.appConfig = appConfig
	blueprint.placeholderManager = placeholderManager
	blueprint.temporaryFileManager = temporaryFileManager
	blueprint.agentManager = agentManager
	return blueprint
}

// ConfigureMartini adds the adminBlueprint handlers/controllers to martini.
func (blueprint *adminBlueprint) ConfigureMartini(m *martini.ClassicMartini) error {
	m.Get(blueprint.base+"/config", blueprint.configHandler)
	m.Get(blueprint.base+"/placeholders", blueprint.placeholdersHandler)
	m.Get(blueprint.base+"/temporaryFiles", blueprint.temporaryFilesHandler)
	m.Get(blueprint.base+"/errors", blueprint.errorsHandler)
	m.Get(blueprint.base+"/renderAgents", blueprint.renderAgentsHandler)
	m.Get(blueprint.base+"/metrics", blueprint.metricsHandler)
	return nil
}

func (blueprint *adminBlueprint) configHandler(res http.ResponseWriter, req *http.Request) {
	content := blueprint.appConfig.Source()
	res.Header().Set("Content-Length", strconv.Itoa(len(content)))
	res.Write([]byte(content))
}

func (blueprint *adminBlueprint) metricsHandler(res http.ResponseWriter, req *http.Request) {
	content := &bytes.Buffer{}
	enc := json.NewEncoder(content)
	enc.Encode(blueprint.registry)
	res.Header().Set("Content-Length", strconv.Itoa(content.Len()))
	res.Write(content.Bytes())
}

func (blueprint *adminBlueprint) placeholdersHandler(res http.ResponseWriter, req *http.Request) {
	view := new(placeholdersView)
	view.Placeholders = make([]*common.Placeholder, 0, 0)
	for _, fileType := range blueprint.placeholderManager.AllFileTypes() {
		for _, placeholderSize := range common.DefaultPlaceholderSizes {
			view.Placeholders = append(view.Placeholders, blueprint.placeholderManager.Url(fileType, placeholderSize))
		}
	}

	body, err := json.Marshal(view)
	if err != nil {
		res.WriteHeader(500)
		return
	}

	res.Header().Set("Content-Length", strconv.Itoa(len(body)))
	res.Write(body)
}

func (blueprint *adminBlueprint) renderAgentsHandler(res http.ResponseWriter, req *http.Request) {
	view := new(renderAgentsView)
	view.RenderAgents = make(map[string]renderAgentViewElement)
	for _, name := range common.RenderAgents {
		view.RenderAgents[name] = blueprint.newRenderAgentViewElement(name)
	}

	body, err := json.Marshal(view)
	if err != nil {
		res.WriteHeader(500)
		return
	}

	res.Header().Set("Content-Length", strconv.Itoa(len(body)))
	res.Write(body)
}

func (blueprint *adminBlueprint) newRenderAgentViewElement(name string) renderAgentViewElement {
	enabled, count, activeWork := blueprint.agentManager.ActiveWorkForRenderAgent(common.RenderAgentDocument)
	return renderAgentViewElement{count, enabled, activeWork}
}

func (blueprint *adminBlueprint) errorsHandler(res http.ResponseWriter, req *http.Request) {
	view := new(errorsView)
	view.Errors = make([]errorViewError, 0, 0)
	for _, err := range common.AllErrors {
		view.Errors = append(view.Errors, errorViewError{err.Error(), err.Description()})
	}
	body, err := json.Marshal(view)
	if err != nil {
		res.WriteHeader(500)
		return
	}

	res.Header().Set("Content-Length", strconv.Itoa(len(body)))
	res.Write(body)
}

func (blueprint *adminBlueprint) temporaryFilesHandler(res http.ResponseWriter, req *http.Request) {
	view := new(temporaryFilesView)
	view.Files = blueprint.temporaryFileManager.List()

	body, err := json.Marshal(view)
	if err != nil {
		res.WriteHeader(500)
		return
	}

	res.Header().Set("Content-Length", strconv.Itoa(len(body)))
	res.Write(body)
}
