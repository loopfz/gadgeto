package iffy

import (
	"bytes"
	"io"
)

type NoopBody struct {
}

func (nb NoopBody) ContentType() string {
	return ""
}

func (nb NoopBody) GetBody(tmpl TemplaterFunc) (io.Reader, error) {
	return &bytes.Buffer{}, nil
}
