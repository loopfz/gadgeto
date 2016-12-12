package genswagger

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/loopfz/gadgeto/tonic/genswagger/doc"
	"github.com/loopfz/gadgeto/tonic/genswagger/swagger"
)

var (
	api *swagger.ApiDeclaration // singleton api declaration, generated once
)

func Swagger(e *gin.Engine, godocStr string) gin.HandlerFunc {
	if api == nil {
		defer tonic.SetExecHook(tonic.GetExecHook())
		tonic.SetExecHook(swaggerHook)

		for _, r := range e.Routes() {
			callRoute(r, e)
		}

		godoc := &doc.Infos{}
		if godocStr != "" {
			err := json.Unmarshal([]byte(godocStr), &godoc)
			if err != nil {
				panic(err)
			}
		}

		// generate Api Declaration
		gen := NewSchemaGenerator()
		if err := gen.GenerateSwagDeclaration(tonic.GetRoutes(), "", "", godoc); err != nil {
			panic(err)
		}

		// store once
		api = gen.apiDeclaration
	}

	return func(c *gin.Context) {
		c.JSON(200, api)
	}
}

func callRoute(r gin.RouteInfo, h http.Handler) {
	req, err := http.NewRequest(r.Method, r.Path, nil)
	if err != nil {
		panic(err)
	}
	h.ServeHTTP(newDummyResponseWriter(), req)
}

func swaggerHook(c *gin.Context, h gin.HandlerFunc, fname string) {
	if r, ok := tonic.GetRoutes()[fname]; ok {
		r.Path = c.Request.URL.Path
		r.Method = c.Request.Method
	}
}

/////

type DummyResponseWriter struct{}

func (d *DummyResponseWriter) Header() http.Header {
	h := make(map[string][]string)
	return h
}
func (d *DummyResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}
func (d *DummyResponseWriter) WriteHeader(int) {}
func newDummyResponseWriter() *DummyResponseWriter {
	return &DummyResponseWriter{}
}
