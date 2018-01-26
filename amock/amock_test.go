package amock

import (
	"fmt"
	"testing"

	"github.com/loopfz/gadgeto/amock/foo"
)

func TestMock(t *testing.T) {

	mock := NewMock()
	mock.Expect(200, foo.Foo{Identifier: "f1234", BarCount: 42}).OnIdentifier("f1234").OnFunc(foo.GetFoo)
	foo.Client.Transport = mock

	fmt.Println("get a foo with an identifier not matching the expected one")

	f, err := foo.GetFoo("f1")
	if err == nil {
		t.Error("Should not have returned foo object with non-matching ident")
	}

	fmt.Println("returned:", err)

	fmt.Println("----------------------------------------------------------------------")

	fmt.Println("get a foo with the correct identifier but going through an unexpected code path")

	f, err = foo.GetFoo2("f1234")
	if err == nil {
		t.Error("Should not have returned foo object with non-matching func")
	}

	fmt.Println("returned:", err)

	fmt.Println("----------------------------------------------------------------------")

	fmt.Println("get a foo with the correct identifier")

	f, err = foo.GetFoo("f1234")
	if err != nil {
		t.Error(err)
	}

	fmt.Println("returned:", f)

	fmt.Println("----------------------------------------------------------------------")

	fmt.Println("update foo object and call an update _method_")

	f.BarCount = 43

	mock.Expect(200, f).OnIdentifier(f.Identifier).OnFunc(f.UpdateFoo)

	f, err = f.UpdateFoo()
	if err != nil {
		t.Error(err)
	}

	fmt.Println("returned:", f)

	fmt.Println("----------------------------------------------------------------------")

	fmt.Println("make the mock simulate a 503, get a foo expecting an error")

	mock.Expect(503, Raw([]byte(`<html><body><h1>503 Service Unavailable</h1>
No server is available to handle this request.
</body></html>`))).Sticky()

	f, err = foo.GetFoo("f2")
	if err == nil {
		t.Error("expected a 503 error")
	}

	fmt.Println("returned:", err)

	fmt.Println("----------------------------------------------------------------------")

	fmt.Println("previous response is sticky, retry and expect same error")

	f, err2 := foo.GetFoo("f2")
	if err2 == nil {
		t.Error("expected a 503 error")
	}
	if err.Error() != err2.Error() {
		t.Errorf("Errors mismatched: '%s' // '%s'", err.Error(), err2.Error())
	}

	fmt.Println("returned:", err2)

	mock.AssertEmpty(t)
}
