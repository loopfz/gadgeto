package tonic_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/iffy"
	"github.com/loopfz/gadgeto/tonic"
)

var r http.Handler

func errorHook(c *gin.Context, e error) (int, interface{}) {
	if _, ok := e.(tonic.BindError); ok {
		return 400, e.Error()
	}
	return 500, e.Error()
}

func TestMain(m *testing.M) {

	tonic.SetErrorHook(errorHook)

	g := gin.Default()
	g.GET("/simple", tonic.Handler(simpleHandler, 200))
	g.GET("/scalar", tonic.Handler(scalarHandler, 200))
	g.GET("/error", tonic.Handler(errorHandler, 200))
	g.GET("/path/:param", tonic.Handler(pathHandler, 200))
	g.GET("/query", tonic.Handler(queryHandler, 200))
	g.GET("/query-old", tonic.Handler(queryHandlerOld, 200))
	g.POST("/body", tonic.Handler(bodyHandler, 200))

	r = g

	m.Run()
}

func TestSimple(t *testing.T) {

	tester := iffy.NewTester(t, r)

	tester.AddCall("simple", "GET", "/simple", "").Checkers(iffy.ExpectStatus(200), expectEmptyBody)
	tester.AddCall("simple", "GET", "/simple/", "").Checkers(iffy.ExpectStatus(301))
	tester.AddCall("simple", "GET", "/simple?", "").Checkers(iffy.ExpectStatus(200))
	tester.AddCall("simple", "GET", "/simple", "{}").Checkers(iffy.ExpectStatus(200))
	tester.AddCall("simple", "GET", "/simple?param=useless", "{}").Checkers(iffy.ExpectStatus(200))

	tester.AddCall("scalar", "GET", "/scalar", "").Checkers(iffy.ExpectStatus(200))

	tester.Run()
}

func TestError(t *testing.T) {

	tester := iffy.NewTester(t, r)

	tester.AddCall("error", "GET", "/error", "").Checkers(iffy.ExpectStatus(500))

	tester.Run()
}

func TestPathQuery(t *testing.T) {

	tester := iffy.NewTester(t, r)

	tester.AddCall("path", "GET", "/path/foo", "").Checkers(iffy.ExpectStatus(200), expectString("param", "foo"))

	tester.AddCall("query-normal", "GET", "/query?param=foo", "").Checkers(iffy.ExpectStatus(200), expectString("param", "foo"))
	tester.AddCall("query-extra-vals", "GET", "/query?param=foo&param=bar", "").Checkers(iffy.ExpectStatus(400))
	tester.AddCall("query-missing-required1", "GET", "/query?param=", "").Checkers(iffy.ExpectStatus(400))
	tester.AddCall("query-missing-required2", "GET", "/query", "").Checkers(iffy.ExpectStatus(400))
	tester.AddCall("query-optional", "GET", "/query?param=foo&param-optional=bar", "").Checkers(iffy.ExpectStatus(200), expectString("param-optional", "bar"))
	tester.AddCall("query-int", "GET", "/query?param=foo&param-int=42", "").Checkers(iffy.ExpectStatus(200), expectInt("param-int", 42))
	tester.AddCall("query-multiple", "GET", "/query?param=foo&params=foo&params=bar", "").Checkers(iffy.ExpectStatus(200), expectStringArr("params", "foo", "bar"))
	tester.AddCall("query-bool", "GET", "/query?param=foo&param-bool=true", "").Checkers(iffy.ExpectStatus(200), expectBool("param-bool", true))
	tester.AddCall("query-override-default", "GET", "/query?param=foo&param-default=bla", "").Checkers(iffy.ExpectStatus(200), expectString("param-default", "bla"))
	tester.AddCall("query-ptr", "GET", "/query?param=foo&param-ptr=bar", "").Checkers(iffy.ExpectStatus(200), expectString("param-ptr", "bar"))
	tester.AddCall("query-embed", "GET", "/query?param=foo&param-embed=bar", "").Checkers(iffy.ExpectStatus(200), expectString("param-embed", "bar"))

	now, _ := time.Time{}.Add(87 * time.Hour).MarshalText()

	tester.AddCall("query-complex", "GET", fmt.Sprintf("/query?param=foo&param-complex=%s", now), "").Checkers(iffy.ExpectStatus(200), expectString("param-complex", string(now)))

	// Explode.
	tester.AddCall("query-explode", "GET", "/query?param=foo&param-explode=a&param-explode=b&param-explode=c", "").Checkers(iffy.ExpectStatus(200), expectStringArr("param-explode", "a", "b", "c"))
	tester.AddCall("query-explode-disabled-ok", "GET", "/query?param=foo&param-explode-disabled=x,y,z", "").Checkers(iffy.ExpectStatus(200), expectStringArr("param-explode-disabled", "x", "y", "z"))
	tester.AddCall("query-explode-disabled-error", "GET", "/query?param=foo&param-explode-disabled=a&param-explode-disabled=b", "").Checkers(iffy.ExpectStatus(400))
	tester.AddCall("query-explode-string", "GET", "/query?param=foo&param-explode-string=x,y,z", "").Checkers(iffy.ExpectStatus(200), expectString("param-explode-string", "x,y,z"))
	tester.AddCall("query-explode-default", "GET", "/query?param=foo", "").Checkers(iffy.ExpectStatus(200), expectStringArr("param-explode-default", "1", "2", "3"))             // default with explode
	tester.AddCall("query-explode-disabled-default", "GET", "/query?param=foo", "").Checkers(iffy.ExpectStatus(200), expectStringArr("param-explode-disabled-default", "1,2,3")) // default without explode

	tester.Run()
}

func TestPathQueryBackwardsCompatible(t *testing.T) {

	tester := iffy.NewTester(t, r)

	tester.AddCall("query-old-missing-required1", "GET", "/query-old", "").Checkers(iffy.ExpectStatus(400))
	tester.AddCall("query-old-missing-required2", "GET", "/query-old?param=", "").Checkers(iffy.ExpectStatus(400))
	tester.AddCall("query-old-normal", "GET", "/query-old?param=foo", "").Checkers(iffy.ExpectStatus(200), expectString("param", "foo"))
	tester.AddCall("query-old-override-default", "GET", "/query-old?param=foo&param-default=bla", "").Checkers(iffy.ExpectStatus(200), expectString("param-default", "bla"))
	tester.AddCall("query-old-use-default", "GET", "/query-old?param=foo", "").Checkers(iffy.ExpectStatus(200), expectString("param-default", "default"))

	tester.Run()
}

func TestBody(t *testing.T) {

	tester := iffy.NewTester(t, r)

	tester.AddCall("body", "POST", "/body", `{"param": "foo"}`).Checkers(iffy.ExpectStatus(200), expectString("param", "foo"))
	tester.AddCall("body", "POST", "/body", `{}`).Checkers(iffy.ExpectStatus(400))
	tester.AddCall("body", "POST", "/body", `{"param": ""}`).Checkers(iffy.ExpectStatus(400))
	tester.AddCall("body", "POST", "/body", `{"param": "foo", "param-optional": "bar"}`).Checkers(iffy.ExpectStatus(200), expectString("param-optional", "bar"))
	tester.AddCall("body1", "POST", "/body", `{"param": "foo"}`).Checkers(iffy.ExpectStatus(200), expectString("param", "foo"))
	tester.AddCall("body2", "POST", "/body", `{}`).Checkers(iffy.ExpectStatus(400))
	tester.AddCall("body3", "POST", "/body", `{"param": ""}`).Checkers(iffy.ExpectStatus(400))
	tester.AddCall("body4", "POST", "/body", `{"param": "foo", "param-optional": "bar"}`).Checkers(iffy.ExpectStatus(200), expectString("param-optional", "bar"))
	tester.AddCall("body5", "POST", "/body", `{"param": "foo", "param-optional-validated": "ttttt"}`).Checkers(iffy.ExpectStatus(400), expectStringInBody("failed on the 'eq=|eq=foo|gt=10' tag"))
	tester.AddCall("body6", "POST", "/body", `{"param": "foo", "param-optional-validated": "foo"}`).Checkers(iffy.ExpectStatus(200), expectString("param-optional-validated", "foo"))
	tester.AddCall("body7", "POST", "/body", `{"param": "foo", "param-optional-validated": "foobarfoobuz"}`).Checkers(iffy.ExpectStatus(200), expectString("param-optional-validated", "foobarfoobuz"))

	tester.Run()
}

func errorHandler(c *gin.Context) error {
	return errors.New("error")
}

func simpleHandler(c *gin.Context) error {
	return nil
}

func scalarHandler(c *gin.Context) (string, error) {
	return "", nil
}

type pathIn struct {
	Param string `path:"param" json:"param"`
}

func pathHandler(c *gin.Context, in *pathIn) (*pathIn, error) {
	return in, nil
}

type queryIn struct {
	Param                       string    `query:"param" json:"param" validate:"required"`
	ParamOptional               string    `query:"param-optional" json:"param-optional"`
	Params                      []string  `query:"params" json:"params"`
	ParamInt                    int       `query:"param-int" json:"param-int"`
	ParamBool                   bool      `query:"param-bool" json:"param-bool"`
	ParamDefault                string    `query:"param-default" json:"param-default" default:"default" validate:"required"`
	ParamPtr                    *string   `query:"param-ptr" json:"param-ptr"`
	ParamComplex                time.Time `query:"param-complex" json:"param-complex"`
	ParamExplode                []string  `query:"param-explode" json:"param-explode" explode:"true"`
	ParamExplodeDisabled        []string  `query:"param-explode-disabled" json:"param-explode-disabled" explode:"false"`
	ParamExplodeString          string    `query:"param-explode-string" json:"param-explode-string" explode:"true"`
	ParamExplodeDefault         []string  `query:"param-explode-default" json:"param-explode-default" default:"1,2,3" explode:"true"`
	ParamExplodeDefaultDisabled []string  `query:"param-explode-disabled-default" json:"param-explode-disabled-default" default:"1,2,3" explode:"false"`
	*DoubleEmbedded
}

// XXX: deprecated, but ensure backwards compatibility
type queryInOld struct {
	ParamRequired string `query:"param, required" json:"param"`
	ParamDefault  string `query:"param-default,required,default=default" json:"param-default"`
}

type Embedded struct {
	ParamEmbed string `query:"param-embed" json:"param-embed"`
}

type DoubleEmbedded struct {
	Embedded
}

func queryHandler(c *gin.Context, in *queryIn) (*queryIn, error) {
	return in, nil
}

func queryHandlerOld(c *gin.Context, in *queryInOld) (*queryInOld, error) {
	return in, nil
}

type bodyIn struct {
	Param                  string `json:"param" validate:"required"`
	ParamOptional          string `json:"param-optional"`
	ValidatedParamOptional string `json:"param-optional-validated" validate:"eq=|eq=foo|gt=10"`
}

func bodyHandler(c *gin.Context, in *bodyIn) (*bodyIn, error) {
	return in, nil
}

func expectEmptyBody(r *http.Response, body string, obj interface{}) error {
	if len(body) != 0 {
		return fmt.Errorf("Body '%s' should be empty", body)
	}
	return nil
}

func expectString(paramName, value string) func(*http.Response, string, interface{}) error {

	return func(r *http.Response, body string, obj interface{}) error {

		var i map[string]interface{}

		err := json.Unmarshal([]byte(body), &i)
		if err != nil {
			return err
		}
		s, ok := i[paramName]
		if !ok {
			return fmt.Errorf("%s missing", paramName)
		}
		if s != value {
			return fmt.Errorf("%s: expected %s got %s", paramName, value, s)
		}
		return nil
	}
}

func expectBool(paramName string, value bool) func(*http.Response, string, interface{}) error {

	return func(r *http.Response, body string, obj interface{}) error {

		i := map[string]interface{}{paramName: 0}

		err := json.Unmarshal([]byte(body), &i)
		if err != nil {
			return err
		}
		v, ok := i[paramName]
		if !ok {
			return fmt.Errorf("%s missing", paramName)
		}
		vb, ok := v.(bool)
		if !ok {
			return fmt.Errorf("%s not a number", paramName)
		}
		if vb != value {
			return fmt.Errorf("%s: expected %v got %v", paramName, value, vb)
		}
		return nil
	}
}

func expectStringArr(paramName string, value ...string) func(*http.Response, string, interface{}) error {

	return func(r *http.Response, body string, obj interface{}) error {

		var i map[string]interface{}

		err := json.Unmarshal([]byte(body), &i)
		if err != nil {
			return fmt.Errorf("failed to unmarshal json: %s", body)
		}
		s, ok := i[paramName]
		if !ok {
			return fmt.Errorf("%s missing", paramName)
		}
		sArr, ok := s.([]interface{})
		if !ok {
			return fmt.Errorf("%s not a string arr", paramName)
		}
		for n := 0; n < len(value); n++ {
			if n >= len(sArr) {
				return fmt.Errorf("%s too short", paramName)
			}
			if sArr[n] != value[n] {
				return fmt.Errorf("%s: %s does not match", paramName, sArr[n])
			}
		}
		return nil
	}
}

func expectStringInBody(value string) func(*http.Response, string, interface{}) error {

	return func(r *http.Response, body string, obj interface{}) error {
		if !strings.Contains(body, value) {
			return fmt.Errorf("body doesn't contain '%s' (%s)", value, body)
		}
		return nil
	}
}

func expectInt(paramName string, value int) func(*http.Response, string, interface{}) error {

	return func(r *http.Response, body string, obj interface{}) error {

		i := map[string]interface{}{paramName: 0}

		err := json.Unmarshal([]byte(body), &i)
		if err != nil {
			return err
		}
		v, ok := i[paramName]
		if !ok {
			return fmt.Errorf("%s missing", paramName)
		}
		vf, ok := v.(float64)
		if !ok {
			return fmt.Errorf("%s not a number", paramName)
		}
		vInt := int(vf)
		if vInt != value {
			return fmt.Errorf("%s: expected %v got %v", paramName, value, vInt)
		}
		return nil
	}
}
