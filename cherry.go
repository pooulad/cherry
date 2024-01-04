package cherry

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/pooulad/cherry/utils"
)

//go:embed assets/banner.txt
var banner []byte

// errorHandler is the default error handler for cherry.
var errorHandler = func(ctx *Context, err error) {
	http.Error(ctx.Response(), err.Error(), http.StatusInternalServerError)
}

// ErrorHandlerFunc used for centralize error handling when an error happens in Handler.
type ErrorHandlerFunc func(ctx *Context, err error)

// Handler is a cherry idiom for handling http.Requests.
type Handler func(ctx *Context) error

// Cherry is a web framework for making fast and simple
// web applications in the Go programming language.
// Cherry supports by one of the fastest request router in Golang.
// It is also provides a graceful web server that can serve TLS encrypted requests as well.
type Cherry struct {
	ErrorHandler ErrorHandlerFunc

	// Output writes the access-log and debug parameters for web-server
	Output io.Writer

	// HasAccessLog enables access-log for cherry. The default is false
	HasAccessLog bool

	// HTTP2 enables the HTTP2 protocol on the server(TLS)
	HTTP2 bool

	router     *httprouter.Router
	middleware []Handler
	prefix     string
	context    context.Context
}

// New returns a new Cherry object.
func New() *Cherry {
	return &Cherry{
		router:       httprouter.New(),
		Output:       os.Stderr,
		ErrorHandler: errorHandler,
		HasAccessLog: false,
	}
}

// Serve method serves the cherry web server on the given port.
func (c *Cherry) Serve(port int) error {
	srv := newServer(fmt.Sprintf(":%d", port), c, c.HTTP2)
	return c.serve(srv)
}

// ServeTLS method serves the application one the given port with TLS encryption.
func (c *Cherry) ServeTLS(port int, certFile, keyFile string) error {
	srv := newServer(fmt.Sprintf(":%d", port), c, c.HTTP2)
	return c.serve(srv, certFile, keyFile)
}

// ServeCustom method serves the application with custom server configuration.
func (c *Cherry) ServeCustom(s *http.Server) error {
	return c.serve(s)
}

// ServeCustomTLS method serves the application with TLS encryption and custom server configuration.
func (c *Cherry) ServeCustomTLS(s *http.Server, certFile, keyFile string) error {
	return c.serve(s, certFile, keyFile)
}

func (c *Cherry) serve(s *http.Server, files ...string) error {
	srv := &server{
		Server: s,
		quit:   make(chan struct{}, 1),
		fquit:  make(chan struct{}, 1),
	}

	packagePath, _ := os.Executable()
	packageDir := filepath.Dir(packagePath)
	err := os.Chdir(packageDir)
	if err != nil {
		return err
	}

	fmt.Fprint(c.Output, utils.Colorize(utils.ColorRed, string(banner))+"\n")

	if len(files) == 0 {
		fmt.Fprintf(c.Output, "Cherry🍒 listening on 0.0.0.0:%s\n", s.Addr)
		return srv.ListenAndServe()
	}
	if len(files) == 2 {
		fmt.Fprintf(c.Output, "Cherry🍒 listening TLS on 0.0.0.0:%s\n", s.Addr)
		return srv.ListenAndServeTLS(files[0], files[1])
	}
	return errors.New("invalid server configuration detected")
}

// Handle adapts the usage of an http.Handler and will be invoked when
// the router matches the prefix and request method.
func (c *Cherry) Handle(method, path string, h http.Handler) {
	c.router.Handler(method, path, h)
}

// Get invokes when request method in handler is set to GET.
func (c *Cherry) Get(route string, h Handler) {
	c.add("GET", route, h)
}

// Post invokes when request method in handler is set to POST.
func (c *Cherry) Post(route string, h Handler) {
	c.add("POST", route, h)
}

// Put invokes when request method in handler is set to PUT.
func (c *Cherry) Put(route string, h Handler) {
	c.add("PUT", route, h)
}

// Delete invokes when request method in handler is set to DELETE.
func (c *Cherry) Delete(route string, h Handler) {
	c.add("DELETE", route, h)
}

// Head invokes when request method in handler is set to HEAD.
func (c *Cherry) Head(route string, h Handler) {
	c.add("HEAD", route, h)
}

// Options invokes when request method in handler is set to OPTIONS.
func (c *Cherry) Options(route string, h Handler) {
	c.add("OPTIONS", route, h)
}

// Static registers the prefix to the router and start to act as a fileserver
//
// app.Static("/public", "./assets").
func (c *Cherry) Static(prefix, dir string) {
	c.router.ServeFiles(path.Join(prefix, "*filepath"), http.Dir(dir))
}

// BindContext lets you provide a context that will live a full http roundtrip
// BindContext is mostly used in a func main() to provide init variables that
// may be created only once, like a database connection. If BindContext is not
// called, Cherry will use a context.Background().
func (c *Cherry) BindContext(ctx context.Context) {
	c.context = ctx
}

// Use appends a Handler to the middleware. Different middleware can be set
// for each sub-router.
func (c *Cherry) Use(handlers ...Handler) {
	c.middleware = append(c.middleware, handlers...)
}

// Group returns a new Group that will inherit all of its parents middleware.
// you can reset the middleware registered to the group by calling Reset().
func (c *Cherry) Group(prefix string) *Group {
	g := &Group{*c}
	g.Cherry.prefix += prefix
	return g
}

// Group act as a sub-router and wil inherit all of its parents middleware.
type Group struct {
	Cherry
}

// Reset clears all middleware.
func (g *Group) Reset() *Group {
	g.Cherry.middleware = nil
	return g
}

// SetNotFound sets a custom handler that is invoked whenever the
// router could not match a route against the request url.
func (c *Cherry) SetNotFound(h http.Handler) {
	c.router.NotFound = h
}

// SetMethodNotAllowed sets a custom handler that is invoked whenever the router
// could not match the method against the predefined routes.
func (c *Cherry) SetMethodNotAllowed(h http.Handler) {
	c.router.MethodNotAllowed = h
}

// SetErrorHandler sets a centralized errorHandler that is invoked whenever
// a Handler returns an error.
func (c *Cherry) SetErrorHandler(h ErrorHandlerFunc) {
	c.ErrorHandler = h
}

// ServeHTTP satisfies the http.Handler interface.
func (c *Cherry) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if rw != nil {
		rw.Header().Set("Server", "Cherry🍒/1.0")
	}
	if c.HasAccessLog {
		start := time.Now()
		logger := &responseLogger{c: rw}
		c.router.ServeHTTP(logger, r)
		c.writeLog(r, start, logger.Status(), logger.Size())
		// saves an allocation by separating the whole logger if log is disabled
	} else {
		c.router.ServeHTTP(rw, r)
	}
}

func (c *Cherry) add(method, route string, h Handler) {
	path := path.Join(c.prefix, route)
	c.router.Handle(method, path, c.makeHttpRouterHandle(h))
}

func (c *Cherry) makeHttpRouterHandle(h Handler) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
		if c.context == nil {
			c.context = context.Background()
		}
		ctx := &Context{
			Context:  c.context,
			vars:     params,
			response: rw,
			request:  r,
			cherry:   c,
		}
		for _, handler := range c.middleware {
			if err := handler(ctx); err != nil {
				c.ErrorHandler(ctx, err)
				return
			}
		}
		if err := h(ctx); err != nil {
			c.ErrorHandler(ctx, err)
			return
		}
	}
}

func (c *Cherry) writeLog(r *http.Request, start time.Time, status, size int) {
	host, _, _ := net.SplitHostPort(r.Host)
	username := "-"
	if r.URL.User != nil {
		if name := r.URL.User.Username(); name != "" {
			username = name
		}
	}
	fmt.Fprintf(c.Output, "%s - %s [%s] \"%s %s %s\" %d %d\n",
		host,
		username,
		start.Format("02/Jan/2006:15:04:05 -0700"),
		r.Method,
		r.RequestURI,
		r.Proto,
		status,
		size,
	)
}

// Context is required in each cherry Handler and can be used to pass information
// between requests.
type Context struct {
	// Context is a idiomatic way to pass information between requests.
	// More information about context.Context can be found here:
	// https://godoc.org/golang.org/x/net/context
	Context  context.Context
	response http.ResponseWriter
	request  *http.Request
	vars     httprouter.Params
	cherry   *Cherry
}

// Response returns a default http.ResponseWriter.
func (c *Context) Response() http.ResponseWriter {
	return c.response
}

// Request returns a default http.Request ptr.
func (c *Context) Request() *http.Request {
	return c.request
}

// JSON is a helper function for writing a JSON encoded representation of v to
// the ResponseWriter.
func (c *Context) JSON(code int, v interface{}) error {
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().WriteHeader(code)
	return json.NewEncoder(c.Response()).Encode(v)
}

// Text is a helper function for writing a text/plain string to the ResponseWriter.
func (c *Context) Text(code int, text string) error {
	c.Response().Header().Set("Content-Type", "text/plain")
	c.Response().WriteHeader(code)
	c.Response().Write([]byte(text))
	return nil
}

// DecodeJSON is a helper that decodes the request Body to v.
// For a more in depth use of decoding and encoding JSON, use the std JSON package.
func (c *Context) DecodeJSON(v interface{}) error {
	return json.NewDecoder(c.Request().Body).Decode(v)
}

// Param returns the url named parameter given in the route prefix by its name
// app.Get("/:name", ..) => ctx.Param("name").
func (c *Context) Param(name string) string {
	return c.vars.ByName(name)
}

// Query returns the url query parameter by its name.
// app.Get("/api?limit=25", ..) => ctx.Query("limit").
func (c *Context) Query(name string) string {
	return c.request.URL.Query().Get(name)
}

// Form returns the form parameter by its name.
func (c *Context) Form(name string) string {
	return c.request.FormValue(name)
}

// Header returns the request header by name.
func (c *Context) Header(name string) string {
	return c.request.Header.Get(name)
}

// Redirect redirects the request to the provided URL with the given status code.
func (c *Context) Redirect(url string, code int) error {
	if code < http.StatusMultipleChoices || code > http.StatusTemporaryRedirect {
		return errors.New("invalid redirect code")
	}
	http.Redirect(c.response, c.request, url, code)
	return nil
}

type responseLogger struct {
	c      http.ResponseWriter
	status int
	size   int
}

func (l *responseLogger) Write(p []byte) (int, error) {
	if l.status == 0 {
		l.status = http.StatusOK
	}
	size, err := l.c.Write(p)
	l.size += size
	return size, err
}

func (l *responseLogger) Header() http.Header {
	return l.c.Header()
}

func (l *responseLogger) WriteHeader(code int) {
	l.c.WriteHeader(code)
	l.status = code
}

func (l *responseLogger) Status() int {
	return l.status
}

func (l *responseLogger) Size() int {
	return l.size
}
