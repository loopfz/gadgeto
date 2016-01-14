package tonic

import (
	"fmt"
	"reflect"
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
	var tag string
	qTag := f.Tag.Get(query_tag)
	if qTag != "" {
		tag = qTag
	}
	pTag := f.Tag.Get(path_tag)
	if pTag != "" {
		tag = pTag
	}
	jTag := f.Tag.Get("json")
	if jTag != "" {
		tag = jTag
	}
	tag = strings.Replace(tag, " ", "", -1)
	tag = strings.Replace(tag, ",required", "", -1)
	tag = strings.Replace(tag, ",omitempty", "", -1)
	if tag == "" {
		tag = f.Name
	}
	return tag
}

func paramDescription(f reflect.StructField) string {
	return f.Tag.Get("description")
}

func paramRequired(f reflect.StructField) bool {
	var tag string
	qTag := f.Tag.Get(query_tag)
	if qTag != "" {
		tag = qTag
	}
	pTag := f.Tag.Get(path_tag)
	if pTag != "" {
		tag = pTag
	}
	bTag := f.Tag.Get("binding")
	if bTag != "" {
		tag = bTag
	}
	return strings.Index(tag, "required") != -1
}

func paramType(f reflect.StructField) string {
	qTag := f.Tag.Get(query_tag)
	if qTag != "" {
		return query_tag
	}
	pTag := f.Tag.Get(path_tag)
	if pTag != "" {
		return path_tag
	}
	return "body"
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
