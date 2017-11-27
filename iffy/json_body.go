package iffy

import (
	"bytes"
	"encoding/json"
	"io"
)

type JSONBody struct {
	Data interface{}
}

func NewJSONBody(v interface{}) Body {
	return &JSONBody{Data: v}
}

func (jb JSONBody) ContentType() string {
	return "application/json"
}

func (jb JSONBody) GetBody(tmpl TemplaterFunc) (io.Reader, error) {
	body, err := json.Marshal(jb.Data)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer([]byte(tmpl(string(body)))), nil
}
