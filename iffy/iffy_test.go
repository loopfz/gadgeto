package iffy_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/iffy"
	"github.com/loopfz/gadgeto/tonic"
)

func helloHandler(c *gin.Context) (interface{}, error) {
	who, has := c.GetQuery("who")
	if !has {
		return nil, fmt.Errorf("wrong request")
	}
	return &struct {
		Msg string `json:"msg"`
	}{Msg: who}, nil
}
func newFoo(c *gin.Context) error { return nil }
func delFoo(c *gin.Context) error { return nil }

type Foo struct{}

func ExpectValidFoo(r *http.Response, body string, respObject interface{}) error { return nil }

func Test_Tester_Run(t *testing.T) {
	// Instantiate & configure anything that implements http.Handler
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.GET("/hello", tonic.Handler(helloHandler, 200))
	r.POST("/foo", tonic.Handler(newFoo, 201))
	r.DELETE("/foo/:fooid", tonic.Handler(delFoo, 204))

	tester := iffy.NewTester(t, r)

	// Variadic list of checker functions = func(r *http.Response, body string, responseObject interface{}) error
	//
	// Checkers can use closures to trap checker-specific configs -> ExpectStatus(200)
	// Some are provided in the iffy package, but you can use your own Checker functions
	tester.AddCall("helloworld", "GET", "/hello?who=world", "").Checkers(iffy.ExpectStatus(200), iffy.ExpectJSONFields("msg"))
	tester.AddCall("badhello", "GET", "/hello", "").Checkers(iffy.ExpectStatus(400))

	tester.Run()
}
