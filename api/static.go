package api

import (
	"github.com/codegangsta/martini"
	"github.com/ngerakines/preview/common"
	"log"
	"net/http"
	"strings"
)

type staticBlueprint struct {
	base               string
	placeholderManager common.PlaceholderManager
}

func NewStaticBlueprint(placeholderManager common.PlaceholderManager) *staticBlueprint {
	blueprint := new(staticBlueprint)
	blueprint.base = "/static"
	blueprint.placeholderManager = placeholderManager
	return blueprint
}

func (blueprint *staticBlueprint) ConfigureMartini(m *martini.ClassicMartini) error {
	m.Get(blueprint.base+"/:fileType/:placeholderSize", blueprint.RequestHandler)
	return nil
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
