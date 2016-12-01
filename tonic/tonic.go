// tonic lets you write simpler gin handlers.
// The way it works is that it generates wrapping gin-compatible handlers,
// that do all the repetitive work and wrap the call to your simple tonic handler.
//
// tonic handles path/query/body parameter binding in a single consolidated input object
// which allows you to remove all the boilerplate code that retrieves and tests the presence
// of various parameters.
//
// Example input object:
// type MyInput struct {
//     Foo int `path:"foo, required"`
//     Bar float `query:"bar, default=foobar"`
//     Baz string `json:"baz" binding:"required"`
// }
//
// Output objects can be of any type, and will be marshaled to JSON.
//
// The handler can return an error, which will be returned to the caller.
package tonic

import (
	"encoding"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	query_tag = "query"
	path_tag  = "path"
)

// An ErrorHook lets you interpret errors returned by your handlers.
// After analysis, the hook should return a suitable http status code and
// error payload.
// This lets you deeply inspect custom error types.
// See sub-package 'jujuerrhook' for a ready-to-use implementation
// that relies on juju/errors (https://github.com/juju/errors).
type ErrorHook func(error) (int, interface{})

type ExecHook func(*gin.Context, gin.HandlerFunc, string)

type BindHook func(*gin.Context, interface{}) error

var (
	errorHook ErrorHook = DefaultErrorHook
	execHook  ExecHook  = DefaultExecHook
	bindHook  BindHook  = DefaultBindingHook
	routes              = make(map[string]*Route)
)

func GetRoutes() map[string]*Route {
	return routes
}

// SetErrorHook lets you set your own error hook.
func SetErrorHook(eh ErrorHook) {
	if eh != nil {
		errorHook = eh
	}
}

func GetErrorHook() ErrorHook {
	return errorHook
}

func SetExecHook(eh ExecHook) {
	if eh != nil {
		execHook = eh
	}
}

func GetExecHook() ExecHook {
	return execHook
}

func SetBindHook(bh BindHook) {
	if bh != nil {
		bindHook = bh
	}
}

func GetBindHook() BindHook {
	return bindHook
}

func DefaultExecHook(c *gin.Context, h gin.HandlerFunc, fname string) { h(c) }
func DefaultErrorHook(e error) (int, interface{})                     { return 400, e.Error() }
func DefaultBindingHook(c *gin.Context, i interface{}) error {
	if c.Request.ContentLength == 0 {
		return nil
	}
	err := c.Bind(i)
	if err != nil {
		return fmt.Errorf("Error parsing request body: %s", err.Error())
	}
	return nil
}

// Handler returns a wrapping gin-compatible handler that calls the tonic handler
// passed in parameter.
// The tonic handler may use the following signature:
// func(*gin.Context, [input object ptr]) ([output object], error)
// Input and output objects are both optional (tonic analyzes the handler signature reflexively).
// As such, the minimal accepted signature is:
// func(*gin.Context) error
// The wrapping gin-handler will handle the binding code (JSON + path/query)
// and the error handling.
// This will PANIC if the tonic-handler is of incompatible type.
func Handler(f interface{}, retcode int) gin.HandlerFunc {

	fval := reflect.ValueOf(f)
	ftype := fval.Type()
	fname := runtime.FuncForPC(fval.Pointer()).Name()

	if fval.Kind() != reflect.Func {
		panic(fmt.Sprintf("Handler parameter not a function: %T", f))
	}

	var typeIn reflect.Type
	var typeOut reflect.Type

	// Check tonic-handler inputs
	numin := ftype.NumIn()
	if numin < 1 || numin > 2 {
		panic(fmt.Sprintf("Incorrect number of handler input params: %d", numin))
	}
	hasIn := (numin == 2)
	if !ftype.In(0).ConvertibleTo(reflect.TypeOf(&gin.Context{})) {
		panic(fmt.Sprintf("Unsupported type for handler input parameter: %v. Should be gin.Context.", ftype.In(0)))
	}
	if hasIn {
		if ftype.In(1).Kind() != reflect.Ptr && ftype.In(1).Elem().Kind() != reflect.Struct {
			panic(fmt.Sprintf("Unsupported type for handler input parameter: %v. Should be struct ptr.", ftype.In(1)))
		} else {
			typeIn = ftype.In(1)
		}
	}

	// Check tonic handler outputs
	numout := ftype.NumOut()
	if numout < 1 || numout > 2 {
		panic(fmt.Sprintf("Incorrect number of handler output params: %d", numout))
	}
	hasOut := (numout == 2)
	errIdx := 0
	if hasOut {
		errIdx += 1
		typeOut = ftype.Out(0)
	}
	typeOfError := reflect.TypeOf((*error)(nil)).Elem()
	if !ftype.Out(errIdx).Implements(typeOfError) {
		panic(fmt.Sprintf("Unsupported type for handler output parameter: %v. Should be error.", ftype.Out(errIdx)))
	}

	// Wrapping gin-handler
	retfunc := func(c *gin.Context) {

		funcIn := []reflect.Value{reflect.ValueOf(c)}

		if hasIn {
			// tonic-handler has custom input object, handle binding
			input := reflect.New(typeIn.Elem())
			err := bindHook(c, input.Interface())
			if err != nil {
				c.JSON(400, gin.H{`error`: err.Error()})
				return
			}
			err = bindQueryPath(c, input, query_tag, extractQuery)
			if err != nil {
				c.JSON(400, gin.H{`error`: err.Error()})
				return
			}
			err = bindQueryPath(c, input, path_tag, extractPath)
			if err != nil {
				c.JSON(400, gin.H{`error`: err.Error()})
				return
			}
			funcIn = append(funcIn, input)
		}

		// Call tonic-handler
		ret := fval.Call(funcIn)
		var errOut interface{}
		var outval interface{}
		if hasOut {
			outval = ret[0].Interface()
			errOut = ret[1].Interface()
		} else {
			errOut = ret[0].Interface()
		}
		// Raised error, handle it
		if errOut != nil {
			reterr := errOut.(error)
			c.Error(reterr)
			errcode, errpl := errorHook(reterr)
			if errpl != nil {
				c.JSON(errcode, gin.H{`error`: errpl})
			} else {
				c.String(errcode, "")
			}
			return
		}
		// Normal output, either serialize custom output object or send empty body
		if hasOut {
			c.JSON(retcode, outval)
		} else {
			c.String(retcode, "")
		}
	}

	routes[fname] = &Route{
		defaultStatusCode: retcode,
		handler:           fval,
		handlerType:       ftype,
		inputType:         typeIn,
		outputType:        typeOut,
	}

	return func(c *gin.Context) { execHook(c, retfunc, fname) }
}

func bindQueryPath(c *gin.Context, in reflect.Value, targetTag string, extractor func(*gin.Context, string) (string, []string, error)) error {

	inType := in.Type().Elem()

	for i := 0; i < in.Elem().NumField(); i++ {
		fieldType := inType.Field(i)
		tag := fieldType.Tag.Get(targetTag)
		if tag == "" {
			continue
		}
		name, values, err := extractor(c, tag)
		if err != nil {
			return err
		}
		if len(values) == 0 {
			continue
		}
		field := in.Elem().Field(i)
		if field.Kind() == reflect.Ptr {
			f := reflect.New(field.Type().Elem())
			field.Set(f)
			field = field.Elem()
		}
		if field.Kind() == reflect.Slice {
			for _, v := range values {
				newV := reflect.New(field.Type().Elem()).Elem()
				err := bindValue(v, newV)
				if err != nil {
					return err
				}
				field.Set(reflect.Append(field, newV))
			}
			return nil
		} else if len(values) > 1 {
			return fmt.Errorf("Query parameter '%s' does not support multiple values", name)
		} else {
			err = bindValue(values[0], field)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func extractQuery(c *gin.Context, tag string) (string, []string, error) {

	name, required, defval, err := ExtractTag(tag, true)
	if err != nil {
		return "", nil, err
	}

	q := c.Request.URL.Query()[name]

	if defval != "" && len(q) == 0 {
		q = []string{defval}
	}

	if required && len(q) == 0 {
		return "", nil, fmt.Errorf("Field %s is missing: required.", name)
	}

	return name, q, nil
}

func extractPath(c *gin.Context, tag string) (string, []string, error) {

	name, required, _, err := ExtractTag(tag, false)
	if err != nil {
		return "", nil, err
	}

	out := c.Param(name)
	if required && out == "" {
		return "", nil, fmt.Errorf("Field %s is missing: required.", name)
	}
	return name, []string{out}, nil
}

func ExtractTag(tag string, defaultValue bool) (string, bool, string, error) {

	parts := strings.SplitN(tag, ",", -1)
	name := parts[0]
	options := parts[1:]

	var defval string
	var required bool
	for _, o := range options {
		o = strings.TrimSpace(o)
		if o == "required" {
			required = true
		} else if defaultValue && strings.HasPrefix(o, "default=") {
			o = strings.TrimPrefix(o, "default=")
			defval = o
		} else {
			return "", false, "", fmt.Errorf("Malformed tag for param '%s': unknown opt '%s'", name, o)
		}
	}
	return name, required, defval, nil
}

func bindValue(s string, v reflect.Value) error {

	vIntf := reflect.New(v.Type()).Interface()
	unmarshaler, ok := vIntf.(encoding.TextUnmarshaler)
	if ok {
		err := unmarshaler.UnmarshalText([]byte(s))
		if err != nil {
			return err
		}
		v.Set(reflect.Indirect(reflect.ValueOf(unmarshaler)))
		return nil
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(i)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.SetBool(b)
	default:
		return fmt.Errorf("Unsupported type for query param bind: %v", v.Kind())
	}
	return nil
}
