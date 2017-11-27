package iffy

type Call struct {
	Name       string
	Method     string
	QueryStr   string
	Body       Body
	headers    Headers
	respObject interface{}
	checkers   []Checker
}

func (c *Call) ResponseObject(respObject interface{}) *Call {
	c.respObject = respObject
	return c
}

func (c *Call) Headers(h Headers) *Call {
	c.headers = h
	return c
}

func (c *Call) Checkers(ch ...Checker) *Call {
	c.checkers = ch
	return c
}
