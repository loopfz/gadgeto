package iffy

import (
	"bytes"
	"io"
)

type StringBody struct {
	Data string
}

func (sb StringBody) ContentType() string {
	return "application/json"
}

func (sb StringBody) GetBody(tmpl TemplaterFunc) (io.Reader, error) {
	return bytes.NewBuffer([]byte(tmpl(sb.Data))), nil
}
