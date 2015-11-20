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
	"fmt"
	"reflect"
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
type ErrorHook func(error) (int, interface{})

var (
	errorHook ErrorHook = func(e error) (int, interface{}) { return 400, e.Error() }
)

// SetErrorHook lets you set your own error hook.
func SetErrorHook(eh ErrorHook) {
	errorHook = eh
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
func Handler(f interface{}, retcode int) func(*gin.Context) {

	fval := reflect.ValueOf(f)
	ftype := fval.Type()

	if fval.Kind() != reflect.Func {
		panic(fmt.Sprintf("Handler parameter not a function: %T", f))
	}

	// Check tonic-handler inputs
	numin := ftype.NumIn()
	if numin < 1 || numin > 2 {
		panic(fmt.Sprintf("Incorrect number of handler input params: %d", numin))
	}
	hasIn := (numin == 2)
	if !ftype.In(0).ConvertibleTo(reflect.TypeOf(&gin.Context{})) {
		panic(fmt.Sprintf("Unsupported type for handler input parameter: %v. Should be gin.Context.", ftype.In(0)))
	}
	if hasIn && ftype.In(1).Kind() != reflect.Ptr && ftype.In(1).Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("Unsupported type for handler input parameter: %v. Should be struct ptr.", ftype.In(1)))
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
	}
	typeOfError := reflect.TypeOf((*error)(nil)).Elem()
	if !ftype.Out(errIdx).Implements(typeOfError) {
		panic(fmt.Sprintf("Unsupported type for handler output parameter: %v. Should be error.", ftype.Out(errIdx)))
	}

	// Store custom input type to New it later
	var typeIn reflect.Type
	if hasIn {
		typeIn = ftype.In(1).Elem()
	}

	// Wrapping gin-handler
	return func(c *gin.Context) {

		funcIn := []reflect.Value{reflect.ValueOf(c)}

		if hasIn {
			// tonic-handler has custom input object, handle binding
			input := reflect.New(typeIn)
			err := c.Bind(input.Interface())
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
			errcode, errpl := errorHook(errOut.(error))
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
}

func bindQueryPath(c *gin.Context, in reflect.Value, targetTag string, extractor func(*gin.Context, string) (string, error)) error {

	inType := in.Type().Elem()

	for i := 0; i < in.Elem().NumField(); i++ {
		fieldType := inType.Field(i)
		tag := fieldType.Tag.Get(targetTag)
		if tag == "" {
			continue
		}
		str, err := extractor(c, tag)
		if err != nil {
			return err
		}
		if str == "" {
			continue
		}
		field := in.Elem().Field(i)
		if field.Kind() == reflect.Ptr {
			f := reflect.New(field.Type().Elem())
			field.Set(f)
			field = field.Elem()
		}
		err = bindValue(str, field)
		if err != nil {
			return err
		}
	}

	return nil
}

func extractQuery(c *gin.Context, tag string) (string, error) {

	name, required, defval, err := extractTag(tag, true)
	if err != nil {
		return "", err
	}

	var out string
	if defval != "" {
		out = c.DefaultQuery(name, defval)
	} else {
		out = c.Query(name)
	}

	if required && out == "" {
		return "", fmt.Errorf("Field %s is missing: required.", name)
	}

	return out, nil
}

func extractPath(c *gin.Context, tag string) (string, error) {

	name, required, _, err := extractTag(tag, false)
	if err != nil {
		return "", err
	}

	out := c.Param(name)
	if required && out == "" {
		return "", fmt.Errorf("Field %s is missing: required.", name)
	}
	return out, nil
}

func extractTag(tag string, defaultValue bool) (string, bool, string, error) {

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
