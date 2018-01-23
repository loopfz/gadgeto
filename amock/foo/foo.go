package foo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

var Client = &http.Client{}

type Foo struct {
	Identifier string `json:"identifier"`
	BarCount   int    `json:"bar_count"`
}

func GetFoo(ident string) (*Foo, error) {
	return doGetFoo(ident)
}

func GetFoo2(ident string) (*Foo, error) {
	return doGetFoo(ident)
}

func doGetFoo(ident string) (*Foo, error) {
	resp, err := Client.Get("http://www.foo.com/foo/" + ident)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("got http error %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ret := &Foo{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (f *Foo) UpdateFoo() (*Foo, error) {

	body, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", "http://www.foo.com/foo/"+f.Identifier, bytes.NewReader(body))
	resp, err := Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("got http error %d", resp.StatusCode)
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ret := &Foo{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
