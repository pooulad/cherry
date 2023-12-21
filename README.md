# Cherry🍒 [![Go](https://img.shields.io/badge/--00ADD8?logo=go&logoColor=ffffff)](https://golang.org/)

🚨Cherry🍒 is a web framework for Go language.

## Installation
go get github.com/pooulad/cherry

## Features

- fast route dispatching backed by httprouter
- easy to add middleware handlers
- subrouting with seperated middleware handlers
- central based error handling
- build in template engine
- fast, lightweight and extendable

## Basic usage

```go
package main
import "github.com/pooulad/cherry"

func main() {
    app := cherry.New()

    app.Get("/foo", fooHandler)
    app.Post("/bar", barHandler)
    app.Use(middleware1, middleware2)

    friends := app.Group("/friends")
    friends.Get("/profile", profileHandler)
    friends.Use(middleware3, middleware4)
    
    app.Serve(8080)
}
```
More complete examples can be found in the examples folder

## Routes

```go
app := cherry.New()

app.Get("/", func(ctx *cherry.Context) error {
   .. do something .. 
})
app.Post("/", func(ctx *cherry.Context) error {
   .. do something .. 
})
app.Put("/", func(ctx *cherry.Context) error {
   .. do something .. 
})
app.Delete("/", func(ctx *cherry.Context) error {
   .. do something .. 
})
```
get named url parameters

```go
app.Get("/hello/:name", func(ctx *cherry.Context) error {
    name := ctx.Param("name")
})
```

## Group (subrouting)

Group lets you manage routes, contexts and middleware separate from each other.

Create a new cherry object and attach some middleware and context to it.

```go
app := cherry.New()
app.BindContext(context.WithValue(context.Background(), "foo", "bar")
app.Get("/", somHandler)
app.Use(middleware1, middleware2)
```

Create a group and attach its own middleware and context to it

```go
friends := app.Group("/friends")
app.BindContext(context.WithValue(context.Background(), "friend1", "john")
friends.Post("/create", someHandler)
friends.Use(middleware3, middleware4)
```
In this case group friends will inherit middleware1 and middleware2 from its parent app. We can reset the middleware from its parent by calling ```Reset()```

```go
friends := app.Group("/friends").Reset()
friends.Use(middleware3, middleware4)
```
Now group friends will have only middleware3 and middleware4 attached.

## Static files
Make our assets are accessable trough /assets/styles.css

```go
app := cherry.New()
app.Static("/assets", "public/assets")
```

## Handlers
### A definition of a cherry.Handler

```go
func(ctx *cherry.Context) error
```

Cherry only accepts handlers of type ```cherry.Handler``` to be passed as functions in routes. You can convert any type of handler to a ```cherry.Handler```.

```go
func myHandler(name string) cherry.Handler{
    .. do something ..
   return func(ctx *cherry.Context) error {
        return ctx.Text(w, http.StatusOK, name)
   }
}
```

### Returning errors
Each handler requires an error to be returned. This is personal idiom but it brings some benifits for handling your errors inside request handlers.

```go
func someHandler(ctx *cherry.Context) error {
    // simple error handling by returning all errors 
    err := someFunc(); err != nil {
        return err
    }
    ...
    req, err := http.NewRequest(...)
    if err != nil {
        return err
    }
}
```

A cherry ErrorHandlerFunc

```go
func(ctx *cherry.Context, err error)
```

Handle all errors returned by adding a custom errorHandler for our application.

```go
app := cherry.New()
errHandler := func(ctx *cherry.Context, err error) {
    .. handle the error ..
}
app.SetErrorHandler(errHandler)
```

## Context
Context is a request based object helping you with a series of functions performed against the current request scope.

### Passing values arround middleware functions
Context provides a context.Context for passing request scoped values arround middleware functions.

Create a new context and pass some values

```go
func someMiddleware(ctx *cherry.Context) error {
    ctx.Context = context.WithValue(ctx.Context, "foo", "bar")
    return someMiddleware2(ctx)
}
```

Get the value back from the context in another middleware function

```go
func someMiddleware2(ctx *cherry.Context) error {
    value := ctx.Context.Value("foo").(string)
    ..
}
```

### Binding a context
In some cases you want to intitialize a context from the the main function, like a datastore for example. You can set a context out of a request scope by calling ```BindContext()```.

```go
app.BindContext(context.WithValue(context.Background(), "foo", "bar"))
```

As mentioned in the Group section, you can add different contexts to different groups.

```go
myGroup := app.Group("/foo", ..)
myGroup.BindContext(..)
```

### Helper functions
Context also provides a series of helper functions like responding JSON en text, JSON decoding etc..

```go
func createUser(ctx *cherry.Context) error {
    user := model.User{}
    if err := ctx.DecodeJSON(&user); err != nil {
        return errors.New("failed to decode the response body")
    }
    ..
    return ctx.JSON(http.StatusCreated, user)
}

func login(ctx *cherry.Context) error {
    token := ctx.Header("x-hmac-token")
    if token == "" {
        ctx.Redirect("/login", http.StatusMovedPermanently)
        return nil
    }
    ..
}
```

## Logging

### Access Log

Cherry provides an access-log in an Apache log format for each incomming request. The access-log is disabled by default, to enable the access-log set ```app.HasAccessLog = true```.

```
127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326
```

## Server
Cherry HTTP server is a wrapper arround the default std HTTP server, the only difference is that it provides a gracefull shutdown. Cherry provides both HTTP and HTTPS (TLS).

```go
app := cherry.New()
app.ServeTLS(8080, cert, key)
// or 
app.Serve(8080)
```
### Gracefull stopping a cherry app

Gracefull stopping a cherry app is done by sending one of these signals to the process.

- SIGINT
- SIGQUIT
- SIGTERM

You can also force-quit your app by sending it SIGKILL signal

SIGUSR2 signal is not yet implemented. Reloading a new binary by forking the main process is something that wil be implemented when the need for it is there. Feel free to give some feedback on this feature if you think it can provide a bonus to the package.

## Screenshots

![App Screenshot](https://github.com/pooulad/cherry/blob/main/images/test_app.png)

![App Screenshot](https://github.com/pooulad/cherry/blob/main/images/start_app.png)


## Support

If you like Cherry🍒 buy me a coffee☕ or star project🌟

<a href="https://www.coffeebede.com/poulad"><img size="small" class="img-fluid" src="https://coffeebede.ir/DashboardTemplateV2/app-assets/images/banner/default-yellow.svg" /></a>
