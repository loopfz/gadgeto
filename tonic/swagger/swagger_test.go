package swagger_test

import (
	"fmt"
	"testing"

	"stash.ovh.net/xander/gogadget/sdkgen/swag/swagger"
)

func TestXX(t *testing.T) {

	apiDecl := swagger.NewApiDeclaration("test1.2", "/", "/resourcePath")

	api := apiDecl.GetApi("/")
	if api != nil {
		t.Fatal("getApi return phantom apis")
	}

	a := swagger.NewApi("/foo", "a fake api")
	apiDecl.AddApi(a)

	api = apiDecl.GetApi("/foo")
	if api == nil {
		t.Fatal("getApi not able to get back added api")
	}
	api.AddOperation(getOp())
	if len(api.Operations) != 1 {
		t.Fatal("AddOperation not adding ?")
	}

	if len(apiDecl.GetApi("/foo").Operations) != 1 {
		t.Logf("%+v\n", apiDecl.GetApi("/foo"))
		t.Fatal("api not inside decl ?")
	}

}

var staticNum = 0

func getOp() swagger.Operation {
	op := swagger.NewOperation("POST",
		fmt.Sprintf("functionName%d", staticNum),
		"OutputType",
		"test ope ",
		"does nix")

	staticNum++
	return op

}
