package tonic

import (
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic/swagger"
)

var (
	api *swagger.ApiDeclaration // singleton api declaration, generated once
)

func Swagger(e *gin.Engine) (func(c *gin.Context), error) {
	if api == nil {
		// complete route data: get method and path from gin engine
		ginRoutes := e.Routes()
		for _, r := range ginRoutes {
			if tonicRoute, ok := routes[r.Handler]; ok {
				tonicRoute.RouteInfo = r
			}
		}

		// generate Api Declaration
		gen := NewSchemaGenerator()
		if err := gen.GenerateSwagDeclaration(routes, "", ""); err != nil {
			return nil, err
		}

		// store once
		api = gen.apiDeclaration
	}

	return func(c *gin.Context) {
		c.JSON(200, api)
	}, nil
}

// GENERATOR

// SchemaGenerator is the object users have to manipulate, it internally collects data about packages used by handlers
// and so on
type SchemaGenerator struct {
	apiDeclaration *swagger.ApiDeclaration

	swaggedTypesMap map[reflect.Type]*swagger.Model
	generatedTypes  map[reflect.Type]string

	genesisDefer map[reflect.Type][]nameSetter
	genesis      map[reflect.Type]bool

	docInfos *Infos
}

// NewSchemaGenerator bootstraps a generator, don't instantiate SchemaGenerator yourself
func NewSchemaGenerator() *SchemaGenerator {
	s := &SchemaGenerator{}
	s.swaggedTypesMap = map[reflect.Type]*swagger.Model{}
	s.generatedTypes = map[reflect.Type]string{}
	s.genesisDefer = map[reflect.Type][]nameSetter{}
	s.genesis = map[reflect.Type]bool{}

	return s
}

// GenerateSwagDeclaration parses all routes (handlers, structs) and returns ready to serialize/use ApiDeclaration
func (s *SchemaGenerator) GenerateSwagDeclaration(routes map[string]*Route, basePath, version string) error {

	s.docInfos = LoadDoc(routes)
	s.apiDeclaration = swagger.NewApiDeclaration(version, basePath)

	// create Operation for each route, creating models as we go
	for _, route := range routes {
		if err := s.addOperation(route); err != nil {
			return err
		}
	}

	for t, list := range s.genesisDefer {
		for _, ns := range list {
			if ns == nil {
				if reflect.Ptr != t.Kind() {
					//fmt.Println("incomplete generator: missing defered setter somewhere. FYI type was: " + t.Name() + " / " + t.Kind().String())
				}
			} else {
				ns(s.generatedTypes[t])
			}
		}
	}

	return nil
}

func (s *SchemaGenerator) generateModels(routes map[string]*Route) error {
	for _, route := range routes {
		s.generateSwagModel(route.GetInType(), nil)
		s.generateSwagModel(route.GetOutType(), nil)
	}

	return nil
}

func (s *SchemaGenerator) addOperation(route *Route) error {

	op, err := s.generateOperation(route)
	if err != nil {
		return err
	}

	if _, ok := s.apiDeclaration.Paths[route.GetPath()]; !ok {
		s.apiDeclaration.Paths[route.GetPath()] = make(map[string]swagger.Operation)
	}
	s.apiDeclaration.Paths[route.GetPath()][strings.ToLower(op.HttpMethod)] = *op

	return nil
}

func (s *SchemaGenerator) generateOperation(route *Route) (*swagger.Operation, error) {

	in := route.GetInType()
	out := route.GetOutType()

	op := swagger.NewOperation(
		route.GetVerb(),
		route.GetHandlerName(),
		route.GetDescription(),
		s.generateSwagModel(out, nil),
		s.docInfos.FunctionsDoc[route.GetHandlerNameWithPackage()],
	)

	//Mark `osterone` package routes (monitoring handlers)
	if strings.Index(route.GetHandlerNameWithPackage(), "osterone") >= 0 {
		op.IsMonitoring = true
	}

	if err := s.setOperationParams(&op, in); err != nil {
		return nil, err
	}
	if err := s.setOperationResponse(&op, out); err != nil {
		return nil, err
	}

	op.Tags = route.GetTags()

	return &op, nil
}

// sometimes recursive types can only be fully determined
// after full analysis, we use this interface to do so
type nameSetter func(string)

// ###################################
