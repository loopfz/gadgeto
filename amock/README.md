# amock

amock lets you easily mock any HTTP dependency you may have. It respects the http.RoundTripper interface to replace an http client's transport.

Responses are stacked, and indexed by code path: you specify responses for a certain Go function, and when it's invoked the mock object will go up the stack until it reaches max depth, or finds a function for which you specified a response.

The response will be pop'ed, so the next identical call will get the next expected response.

You can specify conditional filters on responses:

- `OnFunc(foo.GetFoo)`: Filter on calls that went through a given go function
- `OnIdentifier("foo")`: Shortcut to filter on requests following a path pattern of /.../foo(/...). It is a reasonable assumption that REST implementations follow that pattern, which makes writing conditions for these simple cases very easy.
- `On(func(c *amock.Context) bool)`: More verbose but possible to express anything. Example that would filter all GET requests:

```go
On(func(c *amock.Context) bool {
    return c.Request.Method == "GET"
})
```

For a working example, see amock_test.go
