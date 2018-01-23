package amock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
)

// MockRoundTripper implements http.RoundTripper for mocking/testing purposes
type MockRoundTripper struct {
	sync.Mutex
	Responses        []*Response
	potentialCallers map[string]struct{}
}

// ResponsePayload is an interface that the Body object you pass in your expected responses can respect.
// It lets you customize the way your body is handled. If you pass an object that does NOT respect ResponsePayload,
// JSON is the default.
type ResponsePayload interface {
	Payload() ([]byte, error)
}

// Raw respects the ResponsePayload interface. It lets you define a Body object with raw bytes.
type Raw []byte

func (r Raw) Payload() ([]byte, error) {
	return []byte(r), nil
}

// JSON respects the ResponsePayload interface. It encloses another object and marshals it into json.
// This is used if your body object does not respect ResponsePayload.
type JSON struct {
	Obj interface{}
}

func (j JSON) Payload() ([]byte, error) {
	return json.Marshal(j.Obj)
}

// An expected mocked response. Defining traits are status and body.
// Optionally includes conditional filter function defined by one or several On(...) or OnIdentifier(...) calls.
type Response struct {
	Status  int
	headers http.Header
	Body    ResponsePayload
	Cond    func(*Context) bool
	sticky  bool
	Mock    *MockRoundTripper
}

// Context describes the context of the current call to conditional filter functions
type Context struct {
	Request *http.Request
	callers map[string]struct{}
	mock    *MockRoundTripper
}

// Callers returns the functions in the current stack that may be of interest to the conditional filter funcs
func (c *Context) Callers() map[string]struct{} {
	if c.callers == nil {
		c.callers = c.mock.callers()
	}
	return c.callers
}

// NewMock creates a MockRoundTripper object
func NewMock() *MockRoundTripper {
	return &MockRoundTripper{
		potentialCallers: map[string]struct{}{},
	}
}

// Sticky marks the response as reusable. It will not get consumed whenever it is returned.
func (r *Response) Sticky() *Response {
	r.Mock.Lock()
	defer r.Mock.Unlock()
	r.sticky = true
	return r
}

// Headers adds http headers to the response
func (r *Response) Headers(h http.Header) *Response {
	r.Mock.Lock()
	defer r.Mock.Unlock()
	r.headers = h
	return r
}

// merges two conditional filter functions into a composite one (logical AND)
func condAND(fs ...func(*Context) bool) func(*Context) bool {
	return func(c *Context) bool {
		for _, f := range fs {
			if !f(c) {
				return false
			}
		}
		return true
	}
}

// addCond merges a conditional filter with the existing ones on a Response.
func (r *Response) addCond(cond func(*Context) bool) {
	if r.Cond != nil {
		r.Cond = condAND(r.Cond, cond)
	} else {
		r.Cond = cond
	}
}

// OnFunc matches calls that went throug a given go function.
// It accepts a reference to a function as input, and panics otherwise.
func (r *Response) OnFunc(callerFunc interface{}) *Response {
	caller := getFunctionName(callerFunc)
	r.Mock.potentialCaller(caller)
	cond := func(c *Context) bool {
		callers := c.Callers()
		_, ok := callers[caller]
		return ok
	}
	r.addCond(cond)
	return r
}

// OnIdentifier adds a conditional filter to the response.
// The response will be selected only if the HTTP path of the request contains
// "/.../IDENT(/...)": the identifier enclosed in a distinct path segment
func (r *Response) OnIdentifier(ident string) *Response {
	r.Mock.Lock()
	defer r.Mock.Unlock()
	ident = regexp.QuoteMeta(ident)
	matcher := regexp.MustCompile(`/[^/]+/` + ident + `(?:/.*|$)`)
	cond := func(c *Context) bool {
		return matcher.MatchString(c.Request.URL.Path)
	}
	r.addCond(cond)
	return r
}

// On adds a conditional filter to the response.
func (r *Response) On(f func(*Context) bool) *Response {
	r.Mock.Lock()
	defer r.Mock.Unlock()
	r.addCond(f)
	return r
}

// potentialCaller marks a function as worthy of consideration when going through the stack.
// It is called by the OnFunc() filter.
func (mc *MockRoundTripper) potentialCaller(caller string) {
	mc.potentialCallers[caller] = struct{}{}
}

// callers scans the stack for functions defined in "potentialCallers".
// potentialCallers are the aggregated values passed to OnFunc() filters of the responses
// attached to this mock.
func (mc *MockRoundTripper) callers() map[string]struct{} {
	ret := map[string]struct{}{}
	callers := make([]uintptr, 50)
	runtime.Callers(3, callers)
	frames := runtime.CallersFrames(callers)
	for {
		frame, more := frames.Next()
		_, ok := mc.potentialCallers[frame.Function]
		if ok {
			ret[frame.Function] = struct{}{}
		}
		if !more {
			break
		}
	}
	return ret
}

// Expect adds a new expected response, specifying status and body. The other components (headers, conditional filters)
// can be further specified by chaining setter calls on the response object.
func (mc *MockRoundTripper) Expect(status int, body interface{}) *Response {
	mc.Lock()
	defer mc.Unlock()

	bodyPL, ok := body.(ResponsePayload)
	if !ok {
		bodyPL = JSON{body}
	}
	resp := &Response{Status: status, Body: bodyPL, Mock: mc}
	mc.Responses = append(mc.Responses, resp)
	return resp
}

// Hack to fix method vs function references
//
// var f foo.Foo
// f.UpdateFoo and (*foo.Foo).UpdateFoo have a different uintptr
// Their stringified name is suffixed with -... (illegal in identifier names)
var regexTrimMethodSuffix = regexp.MustCompile(`-[^/\.]+$`)

func getFunctionName(i interface{}) string {
	name := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	name = regexTrimMethodSuffix.ReplaceAllString(name, "")
	return name
}

// RoundTrip respects http.RoundTripper. It finds the code path taken to get to here, and returns the first matching expected response.
func (mc *MockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {

	mc.Lock()
	defer mc.Unlock()

	if len(mc.Responses) == 0 {
		return nil, ErrUnexpectedCall("no more expected responses")
	}

	ctx := &Context{Request: r, mock: mc}

	var resp *Response

	for i, rsp := range mc.Responses {
		if rsp.Cond == nil || rsp.Cond(ctx) {
			// Delete elem in place
			if !rsp.sticky {
				mc.Responses = append(mc.Responses[:i], mc.Responses[i+1:]...)
			}
			resp = rsp
			break
		}
	}

	if resp == nil {
		return nil, ErrUnexpectedCall("remaining responses have unmet conditions")
	}

	var respBody []byte
	var err error

	if resp.Body != nil {
		respBody, err = resp.Body.Payload()
		if err != nil {
			return nil, err
		}
	}

	return &http.Response{
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Status:        http.StatusText(resp.Status),
		StatusCode:    resp.Status,
		Header:        resp.headers,
		Body:          ioutil.NopCloser(bytes.NewReader(respBody)),
		Request:       r,
		ContentLength: int64(len(respBody)),
	}, nil
}

// AssertEmpty ensures all expected responses have been consumed.
// It will call t.Error() detailing the remaining unconsumed responses.
func (mc *MockRoundTripper) AssertEmpty(t *testing.T) {
	mc.Lock()
	defer mc.Unlock()

	i := 0
	for _, r := range mc.Responses {
		// ignore sticky responses
		if !r.sticky {
			i++
		}
	}

	if i > 0 {
		t.Error(fmt.Sprintf("%d expected responses remaining", i))
	}
}

// ErrUnexpectedCall crafts an error including a stack trace, to pinpoint a call that did not match
// any of the configured responses
func ErrUnexpectedCall(reason string) error {
	return fmt.Errorf("unexpected call: %s\n%s", reason, string(debug.Stack()))
}
