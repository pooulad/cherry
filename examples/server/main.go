package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/pooulad/cherry"
	"golang.org/x/net/context"
)

// Simpel example how to use cherry with a "datastore" by making use
// of cherry.Context to pass information between middleware and handlers

func main() {
	listen := flag.Int("listen", 3000, "listen address of the application")
	flag.Parse()

	app := cherry.New()

	// centralizing our errors returned from middleware and request handlers
	app.SetErrorHandler(errorHandler)

	app.Get("/hello/:name", greetingHandler)
	app.Use(dbContextHandler)

	// make a subrouter and register some middleware for it
	admin := app.Group("/admin")
	admin.Get("/:name", adminGreetingHandler)
	admin.Use(authenticate)

	log.Fatal(app.Serve(*listen))
}

type datastore struct {
	name string
}

type dbContext struct {
	context.Context
	ds *datastore
}

func (c *dbContext) Value(key interface{}) interface{} {
	if key == "datastore" {
		return c.ds
	}
	return c.Context.Value(key)
}

func newDatastoreContext(parent context.Context, ds *datastore) context.Context {
	return &dbContext{parent, ds}
}

func dbContextHandler(ctx *cherry.Context) error {
	db := datastore{"mydatabase"}
	ctx.Context = newDatastoreContext(ctx.Context, &db)
	return nil
}

// Only the powerfull have access to the admin routes
func authenticate(ctx *cherry.Context) error {
	admins := []string{"toby", "master iy", "c.froome"}
	name := ctx.Param("name")

	for _, admin := range admins {
		if admin == name {
			return nil
		}
	}
	return errors.New("access forbidden")
}

// context helper function to stay lean and mean in your handlers
func datastoreFromContext(ctx context.Context) *datastore {
	return ctx.Value("datastore").(*datastore)
}

func greetingHandler(ctx *cherry.Context) error {
	name := ctx.Param("name")
	db := datastoreFromContext(ctx.Context)
	greeting := fmt.Sprintf("Greetings, %s\nYour database %s is ready", name, db.name)
	return ctx.Text(http.StatusOK, greeting)
}

func adminGreetingHandler(ctx *cherry.Context) error {
	name := ctx.Param("name")
	db := datastoreFromContext(ctx.Context)
	greeting := fmt.Sprintf("Greetings powerfull admin, %s\nYour database %s is ready", name, db.name)
	return ctx.Text(http.StatusOK, greeting)
}

// custom centralized error handling
func errorHandler(ctx *cherry.Context, err error) {
	http.Error(ctx.Response(), "Hey some error occured: "+err.Error(), http.StatusInternalServerError)
}
