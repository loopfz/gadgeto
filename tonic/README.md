# tonic

tonic lets you write simpler gin handlers. The way it works is that it generates wrapping gin-compatible handlers, that do all the repetitive work and wrap the call to your simple tonic handler.

Package tonic handles path/query/body parameter binding in a single consolidated input object which allows you to remove all the boilerplate code that retrieves and tests the presence of various parameters.

## Table of contents

1. [Usage](#usage)
   - [Inputs](#inputs)
   - [Outputs](#outputs)
   - [Validation](#validation)
     - [Enum Validation](#enum-validation)
2. [Basic Example](#basic-example)
3. [Customization](#customization)
   - [Custom Errors Example](#custom-errors-example)
4. [OpenAPI / Swagger Documentation](#openapi--swagger-documentation)

## Usage

### Inputs

Here is an example input object.

```go
type MyInput struct {
    Foo int    `path:"foo"`
    Bar string `query:"bar" default:"foobar"`
    Baz string `json:"baz"`
}
```

### Outputs

Output objects can be of any type, and will be marshaled to JSON.

### Validation

Input validation is performed after binding into the object using the validator library
(<https://github.com/go-playground/validator>). You can use the tag 'validate' on your object
definition to perform validation inside the tonic handler phase.

```go
type MyInput struct {
    Foo int    `path:"foo" validate:"required,gt=10"`
    Bar string `query:"bar" default:"foobar" validate:"nefield=Baz"`
    Baz string `json:"baz" validate:"required,email"`
}
```

#### Enum validation

In addition to the validation provided by the validator library, enum input validation is also implemented natively by tonic, and can check that the provided input value corresponds to one of the expected enum values.

```go
type MyInput struct {
    Bar string `query:"bar" enum:"foo,buz,biz"`
}
```

The handler can return an error, which will be returned to the caller.

## Basic Example

Here is a basic application that greets a user on <http://localhost:8080/hello/me>

```go
import (
    "errors"
    "fmt"

    "github.com/gin-gonic/gin"
    "github.com/loopfz/gadgeto/tonic"
)

type GreetUserInput struct {
    Name string `path:"name" validate:"required,gt=3" description:"User name"`
}

type GreetUserOutput struct {
    Message string `json:"message"`
}

func GreetUser(c *gin.Context, in *GreetUserInput) (*GreetUserOutput, error) {
    if in.Name == "satan" {
        return nil, errors.New("go to hell")
    }
    return &GreetUserOutput{Message: fmt.Sprintf("Hello %s!", in.Name)}, nil
}

func main() {
    r := gin.Default()
    r.GET("/hello/:name", tonic.Handler(GreetUser, 200))
    r.Run(":8080")
}
```

## Customization

If needed, you can also override different parts of the logic via certain available hooks in tonic:

- binding
  - default: bind from JSON
- error handling
  - default: http status 400
- render
  - default: render into JSON

You will probably want to customize the error hook to produce finer grained error status codes.

The role of this error hook is to inspect the returned error object and deduce the http specifics from it. We provide a ready-to-use error hook that depends on the juju/errors package (richer errors): <https://github.com/loopfz/gadgeto/tree/master/tonic/utils/jujerr>

## Custom Errors Example

Example of the same application as before, using juju errors:

```go
import (
    "fmt"

    "github.com/gin-gonic/gin"
    "github.com/juju/errors"
    "github.com/loopfz/gadgeto/tonic"
    "github.com/loopfz/gadgeto/tonic/utils/jujerr"
)

type GreetUserInput struct {
    Name string `path:"name" description:"User name" validate:"required"`
}

type GreetUserOutput struct {
    Message string `json:"message"`
}

func GreetUser(c *gin.Context, in *GreetUserInput) (*GreetUserOutput, error) {
    if in.Name == "satan" {
        return nil, errors.NewForbidden(nil, "go to hell")
    }
    return &GreetUserOutput{Message: fmt.Sprintf("Hello %s!", in.Name)}, nil
}

func main() {
    tonic.SetErrorHook(jujerr.ErrHook)
    r := gin.Default()
    r.GET("/hello/:name", tonic.Handler(GreetUser, 200))
    r.Run(":8080")
}
```

## OpenAPI / Swagger Documentation

You can also easily serve auto-generated swagger documentation (using tonic data) with <https://github.com/wi2l/fizz>
