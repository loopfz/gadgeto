package iffy

import (
	"net/http"
	"io"
	"bytes"
	"encoding/json"
	"testing"
	"mime/multipart"
	"strings"
	"os"
	"path/filepath"
)

type Call interface {
	BuildHTTPRequest(tmpl TemplaterFunc) (*http.Request, error)
	ResponseObject(respObject interface{}) Call
	Headers(h Headers) Call
	Checkers(ch ...Checker) Call
	UnmarshalResponse(rb []byte) (map[string]interface{}, error)
	GetName() string
	PerformChecks(resp *http.Response, respBody string, t *testing.T)
}

type JSONCall struct {
	Name       string
	Method     string
	QueryStr   string
	Body       string
	headers    Headers
	respObject interface{}
	checkers   []Checker
}

func (c *JSONCall) ResponseObject(respObject interface{}) Call {
	c.respObject = respObject
	return c
}

func (c *JSONCall) Headers(h Headers) Call {
	c.headers = h
	return c
}

func (c *JSONCall) Checkers(ch ...Checker) Call {
	c.checkers = ch
	return c
}

func (c *JSONCall) GetName() string {
	return c.Name
}

func (c *JSONCall) UnmarshalResponse(rb []byte) (map[string]interface{}, error) {
	var err error
	if c.respObject != nil {
		err = json.Unmarshal(rb, c.respObject)
		if err != nil {
			return nil, err
		}
	}
	var retJson map[string]interface{}
	_ = json.Unmarshal(rb, &retJson)
	return retJson, nil
}

func (c JSONCall) BuildHTTPRequest(tmpl TemplaterFunc) (*http.Request, error) {
	var body io.Reader
	if c.Body != "" {
		body = bytes.NewBuffer([]byte(tmpl(c.Body)))
	}
	req, err := http.NewRequest(c.Method, tmpl(c.QueryStr), body)
	if err != nil {
		return nil, err
	}
	if c.Body != "" {
		req.Header.Set("content-type", "application/json")
	}
	if c.headers != nil {
		for k, v := range c.headers {
			req.Header.Set(tmpl(k), tmpl(v))
		}
	}
	return req, err
}

func (c *JSONCall) PerformChecks(resp *http.Response, respBody string, t *testing.T) {
	for _, checker := range c.checkers {
		err := checker(resp, respBody, c.respObject)
		if err != nil {
			t.Errorf("%s: %s", c.Name, err)
		}
	}
}

type MultipartCall struct {
	Name          string
	Method        string
	QueryStr      string
	MultipartForm MultipartForm
	headers       Headers
	respObject    interface{}
	checkers      []Checker
}

func (c *MultipartCall) ResponseObject(respObject interface{}) Call {
	c.respObject = respObject
	return c
}

func (c *MultipartCall) Headers(h Headers) Call {
	c.headers = h
	return c
}

func (c *MultipartCall) Checkers(ch ...Checker) Call {
	c.checkers = ch
	return c
}

func (c *MultipartCall) GetName() string {
	return c.Name
}

func (c *MultipartCall) UnmarshalResponse(rb []byte) (map[string]interface{}, error) {
	var err error
	if c.respObject != nil {
		err = json.Unmarshal(rb, c.respObject)
		if err != nil {
			return nil, err
		}
	}
	var retJson map[string]interface{}
	_ = json.Unmarshal(rb, &retJson)
	return retJson, nil
}

func (c MultipartCall) BuildHTTPRequest(tmpl TemplaterFunc) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	defer func() {
		_ = writer.Close()
	}()
	for key, value := range c.MultipartForm {
		// Considering file will be prefixed by @ (since you could also post regular data in the body)
		if strings.HasPrefix(value, "@") {
			// todo: how can we be sure the @ is not the value we wanted to use ?
			if _, err := os.Stat(value[1:]); !os.IsNotExist(err) {
				part, err := writer.CreateFormFile(key, filepath.Base(value[1:]))
				if err != nil {
					return nil, err
				}
				if err := writeFile(part, value[1:]); err != nil {
					return nil, err
				}
				continue
			}
		}
		if err := writer.WriteField(key, value); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(c.Method, tmpl(c.QueryStr), body)
	if err != nil {
		return nil, err
	}
	if c.headers != nil {
		for k, v := range c.headers {
			req.Header.Set(tmpl(k), tmpl(v))
		}
	}
	req.Header.Set("content-type", writer.FormDataContentType())
	return req, nil
}

func (c *MultipartCall) PerformChecks(resp *http.Response, respBody string, t *testing.T) {
	for _, checker := range c.checkers {
		err := checker(resp, respBody, c.respObject)
		if err != nil {
			t.Errorf("%s: %s", c.Name, err)
		}
	}
}

// writeFile writes the content of the file to an io.Writer
func writeFile(part io.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(part, file)
	return err
}
