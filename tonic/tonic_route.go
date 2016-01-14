package tonic

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

type Route struct {
	gin.RouteInfo

	defaultStatusCode int

	description string // ???

	handler     reflect.Value
	handlerType reflect.Type
	inputType   reflect.Type
	outputType  reflect.Type
}

func (r *Route) GetVerb() string {
	return r.Method
}

func (r *Route) GetPath() string {
	return r.Path
}

func (r *Route) GetDescription() string {
	return r.description
}

func (r *Route) GetDefaultStatusCode() int {
	return r.defaultStatusCode
}

func (r *Route) GetHandler() reflect.Value {
	return r.handler
}

func (r *Route) GetInType() reflect.Type {
	in := r.inputType
	if in != nil && in.Kind() == reflect.Ptr {
		in = in.Elem()
	}
	return in
}

func (r *Route) GetOutType() reflect.Type {
	out := r.outputType
	if out != nil && out.Kind() == reflect.Ptr { // should always be true
		out = out.Elem()
	}

	return out
}

func (r *Route) GetHandlerName() string {
	p := strings.Split(r.GetHandlerNameWithPackage(), ".")
	return p[len(p)-1]
}

func (r *Route) GetHandlerNameWithPackage() string {
	ptr := r.handler.Pointer()
	f := runtime.FuncForPC(ptr).Name()
	p := strings.Split(f, "/")
	return p[len(p)-1]
}

//GetTags generates a list of tags for the swagger spec
//from one route definition.
//Currently it only takes the first path of the route as the tag.
func (r *Route) GetTags() []string {
	tags := make([]string, 0, 1)
	paths := strings.SplitN(r.GetPath(), "/", 3)
	if len(paths) > 1 {
		tags = append(tags, paths[1])
	}
	return tags
}
