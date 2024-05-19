package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"
)

type (
	middleware func(handler http.Handler) http.Handler
	router     struct {
		*http.ServeMux
		chain []middleware
	}
)

func NewRouter(mx ...middleware) *router {
	return &router{ServeMux: &http.ServeMux{}, chain: mx}
}

func (r *router) Use(mx ...middleware) {
	r.chain = append(r.chain, mx...)
}

func (r *router) Group(fn func(r *router)) {
	fn(&router{ServeMux: r.ServeMux, chain: slices.Clone(r.chain)})
}

func (r *router) Get(path string, fn http.HandlerFunc, mx ...middleware) {
	r.handle(http.MethodGet, path, fn, mx)
}

func (r *router) Post(path string, fn http.HandlerFunc, mx ...middleware) {
	r.handle(http.MethodPost, path, fn, mx)
}

func (r *router) Put(path string, fn http.HandlerFunc, mx ...middleware) {
	r.handle(http.MethodPut, path, fn, mx)
}

func (r *router) Delete(path string, fn http.HandlerFunc, mx ...middleware) {
	r.handle(http.MethodDelete, path, fn, mx)
}

func (r *router) Head(path string, fn http.HandlerFunc, mx ...middleware) {
	r.handle(http.MethodHead, path, fn, mx)
}

func (r *router) Options(path string, fn http.HandlerFunc, mx ...middleware) {
	r.handle(http.MethodOptions, path, fn, mx)
}

func (r *router) handle(method, path string, fn http.HandlerFunc, mx []middleware) {
	r.Handle(method+" "+path, r.wrap(fn, mx))
}

func (r *router) wrap(fn http.HandlerFunc, mx []middleware) (out http.Handler) {
	out, mx = http.Handler(fn), append(slices.Clone(r.chain), mx...)

	slices.Reverse(mx)

	for _, m := range mx {
		out = m(out)
	}

	return
}

func NumberShow(i int) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("middleware nº ", i, " começou")
			next.ServeHTTP(w, r)
			fmt.Println("middleware nº ", i, " acabou")
		})
	}
}

func formHandler(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "Hello, %s!\n", r.FormValue("name"))
	fmt.Fprintf(w, "Your address is %s!", r.FormValue("address"))
}

func Auth() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "secret" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func Logger() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()
			next.ServeHTTP(w, r)
			elapsedTime := time.Since(startTime)

			slog.Info("http request", slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.String("elapsed_time", elapsedTime.String()))
		})
	}
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/hello" {
		http.NotFound(w, r)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprintf(w, "Hello, World!")
}

func main() {
	r := NewRouter(Logger())

	r.Group(func(r *router) {
		r.Use(NumberShow(1), NumberShow(2), Auth())
		r.Get("/foo/", helloHandler)
	})

	r.Group(func(r *router) {
		r.Use(NumberShow(3))
		r.Get("/bar/", helloHandler, NumberShow(4))
		r.Get("/baz/", helloHandler, NumberShow(5))
	})

	fileServer := http.FileServer(http.Dir("./static"))

	r.Get("/", fileServer.ServeHTTP)
	r.Post("/form", formHandler)
	r.Get("/hello", helloHandler)

	slog.Info("Server is running on localhost:7100")
	http.ListenAndServe("localhost:7100", r)
}
