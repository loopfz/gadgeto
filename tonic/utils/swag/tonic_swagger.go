package swag

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/loopfz/gadgeto/tonic/utils/bootstrap"
	"github.com/loopfz/gadgeto/tonic/utils/swag/doc"
	"github.com/loopfz/gadgeto/tonic/utils/swag/swagger"
)

var (
	api *swagger.ApiDeclaration // singleton api declaration, generated once
)

func Swagger(e *gin.Engine, godocStr string) gin.HandlerFunc {
	if api == nil {
		bootstrap.Bootstrap(e)

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
