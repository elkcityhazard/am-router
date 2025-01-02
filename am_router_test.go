package amrouter

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

//go:embed static
var staticDir embed.FS

func Test_AddRoute(t *testing.T) {
	rtr := NewRouter()

	rtr.AddRoute("GET", "/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	}, func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		})
	})

	req := httptest.NewRequest("GET", "/", nil)

	w := httptest.NewRecorder()

	rtr.Routes[0].Handler.ServeHTTP(w, req)

	if w.Result().StatusCode != 200 {
		t.Error("failed")
	}

}

func Test_Use(t *testing.T) {
	rtr := NewRouter()

	rtr.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		})
	})

	if len(rtr.GlobalMiddleware) < 1 {
		t.Error("GlobalMiddleware should equal a length of 1")
	}
}

func Test_AddMiddlewareToHandler(t *testing.T) {
	rtr := NewRouter()

	rtr.AddRoute("GET", "([/]{1})", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello world")
	})

	rtr.Routes[0].Handler = rtr.AddMiddlewareToHandler(rtr.Routes[0].Handler, func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Add-Middleware", "true")
			h.ServeHTTP(w, r)
		})
	})

	req := httptest.NewRequest("GET", "/", nil)

	w := httptest.NewRecorder()

	rtr.Routes[0].Handler.ServeHTTP(w, req)

	if w.Result().Header["X-Add-Middleware"] == nil {
		t.Error("expected custom header but got none")
	}
}

func TestGetField(t *testing.T) {
	rtr := NewRouter()

	tests := []struct {
		name   string
		fields []string
		index  int
		expect string
	}{
		{"Valid field retrieval", []string{"hello-world"}, 0, "hello-world"},
		{"no keys", []string{}, 0, ""},
		{"index greater than len of fields", []string{"hello"}, 1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new HTTP request with a context containing the test fields
			req := &http.Request{}
			ctx := context.WithValue(context.Background(), CtxKey{}, tt.fields)
			req = req.WithContext(ctx)

			// Call the GetField method and compare the result to the expected value
			field := rtr.GetField(req, tt.index)
			if field != tt.expect {
				t.Errorf("TestGetField - %s: expected %v, got %v", tt.name, tt.expect, field)
			}
		})
	}

}

func TestServeHTTP(t *testing.T) {

	// notFoundHandler := func(w http.ResponseWriter, r *http.Request) {
	// 	http.NotFound(w, r)
	// }

	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, World!")
	})

	mockMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Example middleware logic; could manipulate the request or response
			next.ServeHTTP(w, r)
		})
	}

	testRtr := &AMRouter{
		PathToStaticDir:   "static",
		EmbeddedStaticDir: staticDir,
		IsProduction:      false,
		Routes: []AMRoute{
			{
				Path:       regexp.MustCompile("^/$"),
				Method:     "GET",
				Handler:    mockHandler,
				Middleware: []MiddleWareFunc{mockMiddleware},
			},
			{
				Path:       regexp.MustCompile("/hello"),
				Method:     "GET",
				Handler:    mockHandler,
				Middleware: []MiddleWareFunc{mockMiddleware},
			},
			{
				Path:       regexp.MustCompile("/static/test.txt"),
				Method:     "GET",
				Handler:    mockHandler,
				Middleware: []MiddleWareFunc{mockMiddleware},
			},
		},
		GlobalMiddleware: []MiddleWareFunc{func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				h.ServeHTTP(w, r)
			})
		}},
	}

	tests := []struct {
		name           string
		method         string
		path           string
		statusCode     int
		expectResponse string
	}{
		{"Static file", "GET", "/static/test.txt", http.StatusOK, "Static content"},
		{"Valid route", "GET", "/hello", http.StatusOK, "Hello, World!"},
		{"Route not found", "GET", "/not-exists-sorry", http.StatusNotFound, "404 page not found\n"},
		{"Method not allowed", "POST", "/hello", http.StatusMethodNotAllowed, "405 method not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			testRtr.ServeHTTP(w, req)

			res := w.Result()

			t.Log(res.StatusCode)

			if res.StatusCode != tt.statusCode {
				t.Errorf("%s expected status %d, got %d", tt.path, tt.statusCode, res.StatusCode)
			}

		})
	}
}

func Test_ServeStaticDirectory(t *testing.T) {

	rtr := NewRouter()

	rtr.PathToStaticDir = "/static/"
	rtr.EmbeddedStaticDir = staticDir

	rtr.AddRoute("GET", "/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello world")
	})

	req := httptest.NewRequest("GET", "/static/test.txt", nil)

	w := httptest.NewRecorder()

	isStatic := rtr.ServeStaticDirectory(w, req)

	if !isStatic {
		t.Error("expected static file but got none")
	}

	rtr.IsProduction = true

	isStatic = rtr.ServeStaticDirectory(w, req)

	if !isStatic {
		t.Error("expected static file but got none")
	}

	rtr.AddRoute("GET", "/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	})

	req = httptest.NewRequest("GET", "/", nil)

	w = httptest.NewRecorder()

	isStatic = rtr.ServeStaticDirectory(w, req)

	if isStatic {
		t.Error("expected static file but got none")
	}

}

func Test_Custom404Handler(t *testing.T) {

	req := &http.Request{}

	w := httptest.NewRecorder()

	rtr := NewRouter()

	rtr.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		})
	})

	rtr.Custom404Handler(w, req)

	result := w.Result()

	if result.StatusCode != 404 {
		t.Error("expected 404 but got", result.StatusCode)
	}

}
