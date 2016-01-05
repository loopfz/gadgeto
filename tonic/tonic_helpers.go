package tonic

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/loopfz/gadgeto/tonic/swagger"
)

// why two different funcs for the same ???
func getFieldName(field reflect.StructField) *string {
	// name is taken from json tag if present
	name := field.Tag.Get("json")
	name = strings.Replace(name, ",omitempty", "", -1)
	if name == "-" {
		return nil
	}
	if name == "" {
		name = field.Name
	}

	return &name
}

func paramName(f reflect.StructField) string {
	// FIXME no more wosk
	woskDirectives := f.Tag.Get("wosk")
	name := f.Tag.Get("json")
	name = strings.Replace(name, ",omitempty", "", -1)
	if name == "" {
		name = f.Name
	}
	re := regexp.MustCompile(`name=([^"',]+)`)
	matches := re.FindStringSubmatch(woskDirectives)
	if len(matches) > 0 {
		name = matches[1]
	}
	return name
}

func paramDescription(f reflect.StructField) string {
	return f.Tag.Get("description")
}

func paramRequired(f reflect.StructField) bool {
	// FIXME no more wosk
	woskDirectives := f.Tag.Get("wosk")
	required := strings.Index(woskDirectives, "required=false") == -1
	return required
}

func paramType(f reflect.StructField) string {
	// FIXME no more wosk
	woskDirectives := f.Tag.Get("wosk")
	var paramType string
	// dataType value MUST be one of these values: "path", "query", "body", "header", "form"
	if strings.Index(woskDirectives, "location=path") != -1 {
		paramType = "path"
	} else if strings.Index(woskDirectives, "location=query") != -1 {
		paramType = "query"
	} else if strings.Index(woskDirectives, "location=header") != -1 {
		paramType = "header"
	}
	return paramType
}

func paramTargetTypeAllowMultiple(f reflect.StructField) (reflect.Type, bool) {
	targetType := f.Type
	allowMultiple := false
	if f.Type.Kind() == reflect.Slice || f.Type.Kind() == reflect.Array {
		targetType = f.Type.Elem()
		allowMultiple = true
	}
	return targetType, allowMultiple
}

func paramFormatDataTypeRefId(f reflect.StructField) (string, string, string) {
	var format, dataType, refId string
	if f.Tag.Get("swagger-type") != "" {
		//Swagger type defined on the original struct, no need to infer it
		//format is: swagger-type:type[,format]
		tagValue := f.Tag.Get("swagger-type")
		tagTypes := strings.Split(tagValue, ",")
		switch len(tagTypes) {
		case 1:
			dataType = tagTypes[0]
		case 2:
			dataType = tagTypes[0]
			format = tagTypes[1]
		default:
			panic(fmt.Sprintf("Error: bad swagger-type definition on %s (%s)", f.Name, tagValue))
		}
	} else {
		targetType, _ := paramTargetTypeAllowMultiple(f)
		dataType, format, refId = swagger.GoTypeToSwagger(targetType)
	}
	return format, dataType, refId
}
