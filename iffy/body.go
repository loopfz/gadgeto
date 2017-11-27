package iffy

import (
	"io"
)

// Body is an interface implemented by possible body types for an iffy.Call
type Body interface {
	ContentType() string
	GetBody(tmpl TemplaterFunc) (io.Reader, error)
}
