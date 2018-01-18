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
)

// MockRoundTripper implements http.RoundTripper for mocking/testing purposes
type MockRoundTripper struct {
	sync.Mutex
	Responses map[string][]*Response
}

type ResponsePayload interface {
	Payload() ([]byte, error)
}

type Raw []byte

func (r Raw) Payload() ([]byte, error) {
	return []byte(r), nil
}

type JSON struct {
	Obj interface{}
}

func (j JSON) Payload() ([]byte, error) {
	return json.Marshal(j.Obj)
}

type Response struct {
	Status  int
	headers http.Header
	Body    ResponsePayload
	Cond    func(*http.Request) bool
}

func NewMock() *MockRoundTripper {
	return &MockRoundTripper{
		Responses: map[string][]*Response{},
	}
}

func (r *Response) Headers(h http.Header) *Response {
	r.headers = h
	return r
}

func CondAND(fs ...func(*http.Request) bool) func(*http.Request) bool {
	return func(r *http.Request) bool {
		for _, f := range fs {
			if !f(r) {
				return false
			}
		}
		return true
	}
}

func (r *Response) OnIdentifier(ident string) *Response {
	ident = regexp.QuoteMeta(ident)
	matcher := regexp.MustCompile(`/[^/]+/` + ident + `(?:/.*|$)`)
	cond := func(req *http.Request) bool {
		return matcher.MatchString(req.URL.Path)
	}
	if r.Cond != nil {
		r.Cond = CondAND(r.Cond, cond)
	} else {
		r.Cond = cond
	}
	return r
}

func (r *Response) On(f func(*http.Request) bool) *Response {
	if r.Cond != nil {
		r.Cond = CondAND(r.Cond, f)
	} else {
		r.Cond = f
	}
	return r
}

func (mc *MockRoundTripper) Expect(callerFunc interface{}, status int, body interface{}) *Response {
	mc.Lock()
	defer mc.Unlock()

	caller := GetFunctionName(callerFunc)
	bodyPL, ok := body.(ResponsePayload)
	if !ok {
		bodyPL = JSON{body}
	}
	resp := &Response{Status: status, Body: bodyPL}
	mc.Responses[caller] = append(mc.Responses[caller], resp)
	return resp
}

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (mc *MockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {

	mc.Lock()
	defer mc.Unlock()

	caller, err := mc.callerFunc()
	if err != nil {
		return nil, err
	}

	var resp *Response

	for i, rsp := range mc.Responses[caller] {
		if rsp.Cond == nil || rsp.Cond(r) {
			// Delete elem in place
			mc.Responses[caller] = append(mc.Responses[caller][:i], mc.Responses[caller][i+1:]...)
			resp = rsp
			break
		}
	}

	if resp == nil {
		return nil, fmt.Errorf("No response defined for '%s'", caller)
	}

	respBody, err := resp.Body.Payload()
	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: resp.Status,
		Header:     resp.headers,
		Body:       ioutil.NopCloser(bytes.NewReader(respBody)),
	}, nil
}

// TODO: mc.Remaining() to list expected calls that never arrived ?

func (mc *MockRoundTripper) callerFunc() (string, error) {
	callers := make([]uintptr, 10)
	runtime.Callers(3, callers)
	for _, c := range callers {
		name := runtime.FuncForPC(c).Name()
		if len(mc.Responses[name]) != 0 {
			return name, nil
		}
	}
	return "", fmt.Errorf("Unexpected sdk call:\n%s", string(debug.Stack()))
}
