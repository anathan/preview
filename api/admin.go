package api

import (
	"encoding/json"
	"github.com/codegangsta/martini"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/config"
	"net/http"
	"strconv"
)

type adminBlueprint struct {
	base                 string
	appConfig            config.AppConfig
	placeholderManager   common.PlaceholderManager
	temporaryFileManager common.TemporaryFileManager
}

type placeholdersView struct {
	Placeholders []*common.Placeholder
}

type temporaryFilesView struct {
	Files map[string]int
}

type errorViewError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type errorsView struct {
	Errors []errorViewError `json:"errors"`
}

// NewAdminBlueprint creates a new adminBlueprint object.
func NewAdminBlueprint(appConfig config.AppConfig, placeholderManager common.PlaceholderManager, temporaryFileManager common.TemporaryFileManager) *adminBlueprint {
	blueprint := new(adminBlueprint)
	blueprint.base = "/admin"
	blueprint.appConfig = appConfig
	blueprint.placeholderManager = placeholderManager
	blueprint.temporaryFileManager = temporaryFileManager
	return blueprint
}

// ConfigureMartini adds the adminBlueprint handlers/controllers to martini.
func (blueprint *adminBlueprint) ConfigureMartini(m *martini.ClassicMartini) error {
	m.Get(blueprint.base+"/config", blueprint.configHandler)
	m.Get(blueprint.base+"/placeholders", blueprint.placeholdersHandler)
	m.Get(blueprint.base+"/temporaryFiles", blueprint.temporaryFilesHandler)
	m.Get(blueprint.base+"/errors", blueprint.errorsHandler)
	return nil
}

// ConfigHandler is an http controller the exposes the daemon configuration as JSON.
func (blueprint *adminBlueprint) configHandler(res http.ResponseWriter, req *http.Request) {
	content := blueprint.appConfig.Source()
	res.Header().Set("Content-Length", strconv.Itoa(len(content)))
	res.Write([]byte(content))
}

// PlaceholdersHandler is an http controller that exposes all of the placeholders in the common.PlaceholderManager as JSON.
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

// errorsHandler is an http controller that exposes all of the errors in the common.AllErrors as JSON.
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

// TemporaryFilesHandler is an http controller that exposes all of the temporary files tracked by a common.TemporaryFileManager as JSON.
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
