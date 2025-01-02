
# Introduction
AMROuter uses no dependencies except Go's Standard Library.  

It has support for Global Middleware and Per Route MIddleware.

I would not use it in production :-)


## How To Use

```
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	amrouter "github.com/elkcityhazard/am-router"
)

func main() {

	rtr := amrouter.NewRouter()
	rtr.PathToStaticDir = "/static"
	rtr.Use(addTrailingSlash)
	rtr.AddRoute("GET", "/", http.HandlerFunc(homeHandler), homeMiddleWare)
	rtr.AddRoute("GET", "(^/[\\w-]+)/?$", func(w http.ResponseWriter, r *http.Request) {
		key := rtr.GetField(r, 0)
		fmt.Fprint(w, key)
	})
	srv := &http.Server{
		Addr:    ":8080",
		Handler: rtr,
	}
	fmt.Println("running")
	if err := srv.ListenAndServe(); err != nil {
		log.Panic(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "home handler")
}

func homeMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("I am the home middleware")
		next.ServeHTTP(w, r)
	})
}

func addTrailingSlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		fmt.Println("add trailing slash")

		if !strings.HasSuffix(r.URL.Path, "/") {
			http.Redirect(w, r.WithContext(r.Context()), r.URL.Path+"/", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)

	})
}

```
