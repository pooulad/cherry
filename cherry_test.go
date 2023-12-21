package cherry

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/context"
)

var noopHandler = func(ctx *Context) error { return nil }

func TestHandle(t *testing.T) {
	c := New()
	handler := http.HandlerFunc(func(c http.ResponseWriter, r *http.Request) {})
	for _, method := range []string{"GET", "PUT", "POST", "DELETE"} {
		c.Handle(method, "/", handler)
		code, _ := doRequest(t, method, "/", nil, c)
		isHTTPStatusOK(t, code)
	}
}

func TestMethodGet(t *testing.T) {
	c := New()
	c.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/", nil, c)
	isHTTPStatusOK(t, code)
}

func TestMethodPost(t *testing.T) {
	c := New()
	c.Post("/", noopHandler)
	code, _ := doRequest(t, "POST", "/", nil, c)
	isHTTPStatusOK(t, code)
}

func TestMethodPut(t *testing.T) {
	c := New()
	c.Put("/", noopHandler)
	code, _ := doRequest(t, "PUT", "/", nil, c)
	isHTTPStatusOK(t, code)
}

func TestMethodDelete(t *testing.T) {
	c := New()
	c.Delete("/", noopHandler)
	code, _ := doRequest(t, "DELETE", "/", nil, c)
	isHTTPStatusOK(t, code)
}

func TestMethodHead(t *testing.T) {
	c := New()
	c.Head("/", noopHandler)
	code, _ := doRequest(t, "HEAD", "/", nil, c)
	isHTTPStatusOK(t, code)
}

func TestMethodOptions(t *testing.T) {
	c := New()
	c.Options("/", noopHandler)
	code, _ := doRequest(t, "OPTIONS", "/", nil, c)
	isHTTPStatusOK(t, code)
}

func TestBox(t *testing.T) {
	c := New()
	sr := c.Group("/foo")
	sr.Get("/bar", noopHandler)
	code, _ := doRequest(t, "GET", "/foo/bar", nil, c)
	isHTTPStatusOK(t, code)
}

func TestStatic(t *testing.T) {
	c := New()
	c.Static("/public", "./")
	code, body := doRequest(t, "GET", "/public/README.md", nil, c)
	isHTTPStatusOK(t, code)
	if len(body) == 0 {
		t.Error("body cannot be empty")
	}
	if !strings.Contains(body, "cherry") {
		t.Error("expecting body containing string (cherry)")
	}

	code, _ = doRequest(t, "GET", "/public/nofile", nil, c)
	if code != http.StatusNotFound {
		t.Errorf("expecting status 404 got %d", code)
	}
}

func TestContext(t *testing.T) {
	c := New()
	c.Get("/", checkContext(t, "m1", "m1"))
	c.Use(func(ctx *Context) error {
		ctx.Context = context.WithValue(ctx.Context, "m1", "m1")
		return nil
	})
	code, _ := doRequest(t, "GET", "/", nil, c)
	isHTTPStatusOK(t, code)

	c.Get("/some", checkContext(t, "m1", "m2"))
	c.Use(func(ctx *Context) error {
		ctx.Context = context.WithValue(ctx.Context, "m1", "m2")
		ctx.Response().WriteHeader(http.StatusBadRequest)
		return nil
	})
	code, _ = doRequest(t, "GET", "/some", nil, c)
	if code != http.StatusBadRequest {
		t.Errorf("expecting %d, got %d", http.StatusBadRequest, code)
	}
}

func TestContextWithSubrouter(t *testing.T) {
	c := New()
	sub := c.Group("/test")
	sub.Get("/", checkContext(t, "a", "b"))
	sub.Use(func(ctx *Context) error {
		ctx.Context = context.WithValue(ctx.Context, "a", "b")
		return nil
	})
	code, _ := doRequest(t, "GET", "/test", nil, c)
	if code != http.StatusOK {
		t.Errorf("expected status code 200 got %d", code)
	}
}

func TestBindContext(t *testing.T) {
	c := New()
	c.BindContext(context.WithValue(context.Background(), "a", "b"))

	c.Get("/", checkContext(t, "a", "b"))

	sub := c.Group("/foo")
	sub.Get("/", checkContext(t, "a", "b"))

	code, _ := doRequest(t, "GET", "/", nil, c)
	isHTTPStatusOK(t, code)
	code, _ = doRequest(t, "GET", "/foo", nil, c)
	isHTTPStatusOK(t, code)
}

func TestBindContextSubrouter(t *testing.T) {
	c := New()
	sub := c.Group("/foo")
	sub.Get("/", checkContext(t, "foo", "bar"))
	sub.BindContext(context.WithValue(context.Background(), "foo", "bar"))

	code, _ := doRequest(t, "GET", "/foo", nil, c)
	isHTTPStatusOK(t, code)
}

func checkContext(t *testing.T, key, expect string) Handler {
	return func(ctx *Context) error {
		value := ctx.Context.Value(key).(string)
		if value != expect {
			t.Errorf("expected %s got %s", expect, value)
		}
		return nil
	}
}

func TestMiddleware(t *testing.T) {
	buf := &bytes.Buffer{}
	c := New()
	c.Use(func(ctx *Context) error {
		buf.WriteString("a")
		return nil
	})
	c.Use(func(ctx *Context) error {
		buf.WriteString("b")
		return nil
	})
	c.Use(func(ctx *Context) error {
		buf.WriteString("c")
		return nil
	})
	c.Use(func(ctx *Context) error {
		buf.WriteString("d")
		return nil
	})
	c.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/", nil, c)
	isHTTPStatusOK(t, code)
	if buf.String() != "abcd" {
		t.Errorf("expecting abcd got %s", buf.String())
	}
}

func TestBoxMiddlewareReset(t *testing.T) {
	buf := &bytes.Buffer{}
	c := New()
	c.Use(func(ctx *Context) error {
		buf.WriteString("a")
		return nil
	})
	c.Use(func(ctx *Context) error {
		buf.WriteString("b")
		return nil
	})
	sub := c.Group("/sub").Reset()
	sub.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/sub", nil, c)
	isHTTPStatusOK(t, code)
	if buf.String() != "" {
		t.Errorf("expecting empty buffer got %s", buf.String())
	}
}

func TestBoxMiddlewareInheritsParent(t *testing.T) {
	buf := &bytes.Buffer{}
	c := New()
	c.Use(func(ctx *Context) error {
		buf.WriteString("a")
		return nil
	})
	c.Use(func(ctx *Context) error {
		buf.WriteString("b")
		return nil
	})
	sub := c.Group("/sub")
	sub.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/sub", nil, c)
	isHTTPStatusOK(t, code)
	if buf.String() != "ab" {
		t.Errorf("expecting ab got %s", buf.String())
	}
}

func TestErrorHandler(t *testing.T) {
	c := New()
	errorMsg := "oops! something went wrong"
	c.SetErrorHandler(func(ctx *Context, err error) {
		ctx.Response().WriteHeader(http.StatusInternalServerError)
		if err.Error() != errorMsg {
			t.Errorf("expecting %s, got %s", errorMsg, err.Error())
		}
	})
	c.Use(func(ctx *Context) error {
		return errors.New(errorMsg)
	})
	c.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/", nil, c)
	if code != http.StatusInternalServerError {
		t.Errorf("expecting code 500 got %d", code)
	}
}

func TestWeaveboxHandler(t *testing.T) {
	c := New()
	handle := func(respStr string) Handler {
		return func(ctx *Context) error {
			return ctx.Text(http.StatusOK, respStr)
		}
	}
	c.Get("/a", handle("a"))
	c.Get("/b", handle("b"))
	c.Get("/c", handle("c"))

	for _, r := range []string{"a", "b", "c"} {
		code, body := doRequest(t, "GET", "/"+r, nil, c)
		isHTTPStatusOK(t, code)
		if body != r {
			t.Errorf("expecting %s got %s", r, body)
		}
	}
}

func TestNotFoundHandler(t *testing.T) {
	c := New()
	code, body := doRequest(t, "GET", "/", nil, c)
	if code != http.StatusNotFound {
		t.Errorf("expecting code 404 got %d", code)
	}
	if !strings.Contains(body, "404 page not found") {
		t.Errorf("expecting body: 404 page not found got %s", body)
	}
}

func TestSetNotFound(t *testing.T) {
	c := New()
	notFoundMsg := "hey! not found"
	h := http.HandlerFunc(func(c http.ResponseWriter, r *http.Request) {
		c.WriteHeader(http.StatusNotFound)
		c.Write([]byte(notFoundMsg))
	})
	c.SetNotFound(h)

	code, body := doRequest(t, "GET", "/", nil, c)
	if code != http.StatusNotFound {
		t.Errorf("expecting code 404 got %d", code)
	}
	if !strings.Contains(body, notFoundMsg) {
		t.Errorf("expecting body: %s got %s", notFoundMsg, body)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	c := New()
	c.Get("/", noopHandler)
	code, body := doRequest(t, "POST", "/", nil, c)
	if code != http.StatusMethodNotAllowed {
		t.Errorf("expecting code 405 got %d", code)
	}
	if !strings.Contains(body, "Method Not Allowed") {
		t.Errorf("expecting body: Method Not Allowed got %s", body)
	}
}

func TestSetMethodNotAllowed(t *testing.T) {
	c := New()
	handler := http.HandlerFunc(func(c http.ResponseWriter, r *http.Request) {
		c.WriteHeader(http.StatusMethodNotAllowed)
		c.Write([]byte("foo"))
	})
	c.SetMethodNotAllowed(handler)
	c.Get("/", noopHandler)

	code, body := doRequest(t, "POST", "/", nil, c)
	if code != http.StatusMethodNotAllowed {
		t.Errorf("expecting code 405 got %d", code)
	}
	if !strings.Contains(body, "foo") {
		t.Errorf("expecting body: foo got %s", body)
	}
}

func TestContextURLQuery(t *testing.T) {
	req, _ := http.NewRequest("GET", "/?name=anthony", nil)
	ctx := &Context{request: req}
	if ctx.Query("name") != "anthony" {
		t.Errorf("expected anthony got %s", ctx.Query("name"))
	}
}

func TestContextForm(t *testing.T) {
	values := url.Values{}
	values.Set("email", "john@gmail.com")
	req, _ := http.NewRequest("POST", "/", strings.NewReader(values.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	ctx := &Context{request: req}
	if ctx.Form("email") != "john@gmail.com" {
		t.Errorf("expected john@gmail.com got %s", ctx.Form("email"))
	}
}

func TestContextHeader(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("x-test", "test")
	ctx := &Context{request: req}
	if ctx.Header("x-test") != "test" {
		t.Errorf("expected header to be (test) got %s", ctx.Header("x-test"))
	}
}

func isHTTPStatusOK(t *testing.T, code int) {
	if code != http.StatusOK {
		t.Errorf("Expecting status 200 got %d", code)
	}
}

func doRequest(t *testing.T, method, route string, body io.Reader, c *Cherry) (int, string) {
	r, err := http.NewRequest(method, route, body)
	if err != nil {
		t.Fatal(err)
	}
	rw := httptest.NewRecorder()
	c.ServeHTTP(rw, r)
	return rw.Code, rw.Body.String()
}
