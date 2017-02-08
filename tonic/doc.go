/*
Package tonic lets you write simpler gin handlers.
The way it works is that it generates wrapping gin-compatible handlers,
that do all the repetitive work and wrap the call to your simple tonic handler.

Package tonic handles path/query/body parameter binding in a single consolidated input object
which allows you to remove all the boilerplate code that retrieves and tests the presence
of various parameters.

Here is an example input object.

	type MyInput struct {
		Foo int    `path:"foo,required"`
		Bar float  `query:"bar,default=foobar"`
		Baz string `json:"baz" binding:"required"`
	}

Output objects can be of any type, and will be marshaled to JSON.

The handler can return an error, which will be returned to the caller.

Here is a basic application that greets a user on http://localhost:8080/hello/me

	type GreetUserInput struct {
		Name string `path:"name,required" description:"User name"`
	}

	func GreetUser(c *gin.Context, in *GreetUserInput) (*gin.H, error) {
		if in.Name == "satan" {
			return nil, fmt.Errorf("go to hell")
		}
		return &gin.H{"message": fmt.Sprintf("Hello %s!", in.Name)}, nil
	}

	func main() {
		r := gin.Default()
		r.GET("/hello/:name", tonic.Handler(GreetUser, 200))
		r.Run(":8080")
	}
*/
package tonic
