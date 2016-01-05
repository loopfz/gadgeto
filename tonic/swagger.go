package tonic

import (
	"fmt"
	"os"
	"strings"

	"github.com/loopfz/gadgeto/tonic/swagger"

	"reflect"
)

func (s *SchemaGenerator) setOperationResponse(op *swagger.Operation, t reflect.Type) error {

	//Just give every method a 200 response for now
	//This could be improved given for example 201 for post
	//methods etc
	schema := swagger.Schema{}
	schemaType := s.generateSwagModel(t, nil)
	if strings.Contains(schemaType, "#/") {
		schema.Ref = schemaType
	} else {
		schema.Type = schemaType
	}

	response := swagger.Response{}
	if schema.Type != "void" {
		//For void params swagger def can't be "void"
		//No schema at all is fine
		response.Schema = &schema
	}

	op.Responses = map[string]swagger.Response{
		"200": response,
	}

	return nil
}

func (s *SchemaGenerator) setOperationParams(op *swagger.Operation, in reflect.Type) error {

	if in == nil || in.Kind() != reflect.Struct {
		return nil
	}

	var body *swagger.Model

	for i := 0; i < in.NumField(); i++ {
		p := s.newParamFromStructField(in.Field(i), &body)
		if p != nil {
			if doc := s.docInfos.StructFieldsDoc[in.Name()]; doc != nil {
				if fieldDoc := doc[in.Field(i).Name]; fieldDoc != "" {
					p.Description = strings.TrimSuffix(fieldDoc, "\n")
				}
			}
			op.AddParameter(*p)
		}
	}
	if body != nil {
		body.Id = "Input" + op.Nickname + "In"
		s.apiDeclaration.AddModel(*body)
		bodyParam := swagger.NewParameter("body", "body", "body request", true, false, "", "", "")
		bodyParam.Schema.Ref = "#/definitions/" + body.Id
		op.AddParameter(bodyParam)
	}

	return nil

}

func (s *SchemaGenerator) newParamFromStructField(f reflect.StructField, bodyModel **swagger.Model) *swagger.Parameter {

	s.generateSwagModel(f.Type, nil)

	name := paramName(f)
	paramType := paramType(f)

	if paramType == "" {
		if *bodyModel == nil {
			m := swagger.NewModel("Input")
			*bodyModel = &m
		}
		(*bodyModel).Properties[name] = s.fieldToModelProperty(f)
		return nil
	}

	_, allowMultiple := paramTargetTypeAllowMultiple(f)
	format, dataType, refId := paramFormatDataTypeRefId(f)

	p := swagger.NewParameter(
		paramType,
		name,
		paramDescription(f),
		paramRequired(f),
		allowMultiple,
		dataType,
		format,
		refId,
	)

	return &p
}

func (s *SchemaGenerator) generateSwagModel(t reflect.Type, ns nameSetter) string {
	// nothing to generate
	if t == nil {
		s.generatedTypes[t] = "void"
		return "void"
	}

	//Check if we alredy seen this type
	if finalName, ok := s.generatedTypes[t]; ok {
		return finalName
	}

	if s.genesis[t] {
		if s.genesisDefer[t] == nil {
			s.genesisDefer[t] = []nameSetter{ns}
		} else {
			s.genesisDefer[t] = append(s.genesisDefer[t], ns)
		}
		return "defered"
	}

	s.genesis[t] = true
	defer func() { s.genesis[t] = false }()

	if "Time" == t.Name() && t.PkgPath() == "time" {
		s.generatedTypes[t] = "dateTime (sdk borken)"
		return "dateTime (sdk borken)" //TODO: this is wrong, if a function has to return time we would need type + format
	}

	// let's treat pointed type
	if t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return s.generateSwagModel(t.Elem(), ns)
	}

	// we have either a simple type or a constant/aliased one
	if t.Kind() != reflect.Struct {
		if t.Kind().String() != t.Name() {
			return t.Kind().String()
		}
		typeName, _, _ := swagger.GoTypeToSwagger(t)
		s.generatedTypes[t] = typeName
		return typeName
	}

	modelName := swagger.ModelName(t)
	if modelName == "" { // Bozo : I can't find the guilty type coming in, probably wosk.Null but ...
		s.generatedTypes[t] = "void"
		return "void"
	}

	if _, ok := s.swaggedTypesMap[t]; ok {
		s.generatedTypes[t] = modelName
		return modelName
	}

	m := swagger.NewModel(modelName)
	if t.Kind() == reflect.Struct {
		structFields := s.getStructFields(t, ns)
		if len(structFields) > 0 {
			for name, property := range structFields {
				m.Properties[name] = property
			}
			s.apiDeclaration.AddModel(m)
		}
	}

	s.swaggedTypesMap[t] = &m
	s.generatedTypes[t] = "#/definitions/" + m.Id

	return "#/definitions/" + m.Id
}

func (s *SchemaGenerator) getStructFields(t reflect.Type, ns nameSetter) map[string]*swagger.ModelProperty {
	structFields := make(map[string]*swagger.ModelProperty)

	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Type.Kind() == reflect.Func {
			//Ignore functions
			continue
		}

		name := getFieldName(t.Field(i))
		if name == nil {
			continue
		}

		if *name == "DBModel" {
			//For properties with name DBModel we flatten the structure, ie, we add its fields
			//to the parent model.
			dbModelFields := s.getStructFields(t.Field(i).Type, ns)
			for fieldName, property := range dbModelFields {
				structFields[fieldName] = property
			}

		} else {
			// for fields that are not of simple types, we "program" generation
			s.generateSwagModel(t.Field(i).Type, ns)
			property := s.fieldToModelProperty(t.Field(i))
			structFields[*name] = property
		}

	}

	return structFields
}

func (s *SchemaGenerator) getNestedItemType(t reflect.Type, p *swagger.ModelProperty) *swagger.NestedItems {
	arrayItems := &swagger.NestedItems{}

	if t.Kind() == reflect.Struct {
		arrayItems.RefId = s.generateSwagModel(t, func(a string) { arrayItems.RefId = a })
	} else if t.Kind() == reflect.Map {
		arrayItems.AdditionalProperties = s.getNestedItemType(t.Elem(), p)
		arrayItems.Type = "object"
	} else if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		arrayItems.Items = s.getNestedItemType(t.Elem(), p)
		arrayItems.Type = "array"
	} else {
		arrayItems.Type = s.generateSwagModel(t, nil)
	}
	return arrayItems
}

// Turns a field of a struct to a model property
func (s *SchemaGenerator) fieldToModelProperty(f reflect.StructField) *swagger.ModelProperty {
	// TODO we should know whether struct is inbound or outbound
	p := &swagger.ModelProperty{Required: true}
	if f.Tag.Get("wosk") != "" {
		if strings.Index(f.Tag.Get("wosk"), "required=false") != -1 {
			p.Required = false
		}
	}

	if f.Tag.Get("description") != "" {
		p.Description = f.Tag.Get("description")
	}

	if f.Tag.Get("swagger-type") != "" {
		//Swagger type defined on the original struct, no need to infer it
		//format is: swagger-type:type[,format]
		tagValue := f.Tag.Get("swagger-type")
		tagTypes := strings.Split(tagValue, ",")
		switch len(tagTypes) {
		case 1:
			p.Type = tagTypes[0]
		case 2:
			p.Type = tagTypes[0]
			p.Format = tagTypes[1]
		default:
			panic(fmt.Sprintf("Error: bad swagger-type definition on %s (%s)", f.Name, tagValue))
		}
	} else {

		if f.Type.Kind() == reflect.Slice || f.Type.Kind() == reflect.Array {
			p.Type = "array"
			targetType := f.Type.Elem()
			if targetType.Kind() == reflect.Ptr {
				targetType = targetType.Elem()
			}
			if targetType.Kind() == reflect.Map {
				p.Items = &swagger.NestedItems{}
				nestedItem := s.getNestedItemType(targetType.Elem(), p)
				p.Items.AdditionalProperties = nestedItem
				p.Items.Type = "object"

			} else {
				p.Items = s.getNestedItemType(targetType, p)
			}
		} else if f.Type.Kind() == reflect.Map {
			if f.Type.Key().Kind() != reflect.String {
				fmt.Fprintln(os.Stderr, "Type not supported, only map with string keys, got %: ", f.Type.Key())
			}
			p.Type = "object"
			p.AdditionalProperties = &swagger.NestedItems{}
			targetType := f.Type.Elem()
			if targetType.Kind() == reflect.Ptr {
				targetType = targetType.Elem()
			}
			typ := s.generateSwagModel(targetType, nil)
			if targetType.Kind() == reflect.Struct {
				p.AdditionalProperties.RefId = typ
			} else if targetType.Kind() == reflect.Slice || targetType.Kind() == reflect.Array {
				nestedItem := s.getNestedItemType(targetType.Elem(), p)
				p.AdditionalProperties.Items = nestedItem
				p.AdditionalProperties.Type = "array"
			} else {
				p.AdditionalProperties.Type = typ
			}

		} else {
			t := f.Type
			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			if "Time" == t.Name() && t.PkgPath() == "time" {
				p.Type = "string"
				p.Format = "dateTime"
			} else if t.Kind() == reflect.Struct {
				p.RefId = s.generateSwagModel(t, func(a string) { p.RefId = a })
			} else { // if it's a constant, maybe it's an enum
				s.generateSwagModel(t, nil)
				if list, ok := s.docInfos.Constants[t.String()]; ok {
					values := []string{}
					for _, co := range list.ListC {
						// TODO this is WRONG !
						// TODO I only copy names of constants, we'd need to actually evaluate value
						// I can only think of generating a script, go run it to get values :(
						values = append(values, co)
					}
					p.Enum = values
					p.Description = "WARNING: constants are constants names, they should be values (swagger generator incomplete)"
				}
				p.Type = s.generateSwagModel(t, func(n string) { p.Type = n })
			}
		}
	}
	return p
}
