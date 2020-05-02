package runtime_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/internal/utilities"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMuxServeHTTP(t *testing.T) {
	type stubPattern struct {
		method string
		ops    []int
		pool   []string
		verb   string
	}
	for _, spec := range []struct {
		name string

		patterns    []stubPattern
		patternOpts []runtime.PatternOpt

		reqMethod string
		reqPath   string
		headers   map[string]string

		respStatus  int
		respContent string

		disablePathLengthFallback bool
		muxOpts                   []runtime.ServeMuxOption
	}{
		{
			name:       "GET to unregistered path with no registered paths should return 501 Not Implemented",
			patterns:   nil,
			reqMethod:  "GET",
			reqPath:    "/",
			respStatus: http.StatusNotImplemented,
		},
		{
			name: "GET to registered path should return GET 200 OK",
			patterns: []stubPattern{
				{
					method: "GET",
					ops:    []int{int(utilities.OpLitPush), 0},
					pool:   []string{"foo"},
				},
			},
			reqMethod:   "GET",
			reqPath:     "/foo",
			respStatus:  http.StatusOK,
			respContent: "GET /foo",
		},
		{
			name: "GET to unregistered path with should return 501 Not Implemented",
			patterns: []stubPattern{
				{
					method: "GET",
					ops:    []int{int(utilities.OpLitPush), 0},
					pool:   []string{"foo"},
				},
			},
			reqMethod:  "GET",
			reqPath:    "/bar",
			respStatus: http.StatusNotImplemented,
		},
		{
			name: "GET to registered path with two registered paths should return GET 200 OK",
			patterns: []stubPattern{
				{
					method: "GET",
					ops:    []int{int(utilities.OpLitPush), 0},
					pool:   []string{"foo"},
				},
				{
					method: "GET",
					ops:    []int{int(utilities.OpPush), 0},
				},
			},
			reqMethod:   "GET",
			reqPath:     "/foo",
			respStatus:  http.StatusOK,
			respContent: "GET /foo",
		},
		{
			name: "POST to registered path with GET also registered should return POST 200 OK",
			patterns: []stubPattern{
				{
					method: "GET",
					ops:    []int{int(utilities.OpLitPush), 0},
					pool:   []string{"foo"},
				},
				{
					method: "POST",
					ops:    []int{int(utilities.OpLitPush), 0},
					pool:   []string{"foo"},
				},
			},
			reqMethod:   "POST",
			reqPath:     "/foo",
			respStatus:  http.StatusOK,
			respContent: "POST /foo",
		},
		{
			name: "DELETE to path with GET registered should return 501 NotImplemented",
			patterns: []stubPattern{
				{
					method: "GET",
					ops:    []int{int(utilities.OpLitPush), 0},
					pool:   []string{"foo"},
				},
			},
			reqMethod:  "DELETE",
			reqPath:    "/foo",
			respStatus: http.StatusNotImplemented,
		},
		{
			name: "POST with path length fallback to registered path with GET should return GET 200 OK",
			patterns: []stubPattern{
				{
					method: "GET",
					ops:    []int{int(utilities.OpLitPush), 0},
					pool:   []string{"foo"},
				},
			},
			reqMethod: "POST",
			reqPath:   "/foo",
			headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			respStatus:  http.StatusOK,
			respContent: "GET /foo",
		},
		{
			name: "POST with path length fallback to registered path with GET with path length fallback disabled should return 501 Not Implemented",
			patterns: []stubPattern{
				{
					method: "GET",
					ops:    []int{int(utilities.OpLitPush), 0},
					pool:   []string{"foo"},
				},
			},
			reqMethod: "POST",
			reqPath:   "/foo",
			headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			respStatus:                http.StatusNotImplemented,
			disablePathLengthFallback: true,
		},
		{
			name: "POST with path length fallback to registered path with POST with path length fallback disabled should return POST 200 OK",
			patterns: []stubPattern{
				{
					method: "GET",
					ops:    []int{int(utilities.OpLitPush), 0},
					pool:   []string{"foo"},
				},
				{
					method: "POST",
					ops:    []int{int(utilities.OpLitPush), 0},
					pool:   []string{"foo"},
				},
			},
			reqMethod: "POST",
			reqPath:   "/foo",
			headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			respStatus:                http.StatusOK,
			respContent:               "POST /foo",
			disablePathLengthFallback: true,
		},
		{
			name: "POST with path length fallback and method override to registered path with GET should return GET 200 OK",
			patterns: []stubPattern{
				{
					method: "GET",
					ops:    []int{int(utilities.OpLitPush), 0},
					pool:   []string{"foo"},
				},
				{
					method: "POST",
					ops:    []int{int(utilities.OpLitPush), 0},
					pool:   []string{"foo"},
				},
			},
			reqMethod: "POST",
			reqPath:   "/foo",
			headers: map[string]string{
				"Content-Type":           "application/x-www-form-urlencoded",
				"X-HTTP-Method-Override": "GET",
			},
			respStatus:  http.StatusOK,
			respContent: "GET /foo",
		},
		{
			name: "POST to registered path with GET should return 501 NotImplemented",
			patterns: []stubPattern{
				{
					method: "GET",
					ops:    []int{int(utilities.OpLitPush), 0},
					pool:   []string{"foo"},
				},
			},
			reqMethod: "POST",
			reqPath:   "/foo",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			respStatus: http.StatusNotImplemented,
		},
		{
			name: "POST to registered path with verb should return POST 200 OK",
			patterns: []stubPattern{
				{
					method: "POST",
					ops:    []int{int(utilities.OpLitPush), 0},
					pool:   []string{"foo"},
					verb:   "bar",
				},
			},
			reqMethod: "POST",
			reqPath:   "/foo:bar",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			respStatus:  http.StatusOK,
			respContent: "POST /foo:bar",
		},
		{
			name: "POST to registered path with verb and non-verb should return POST 200 OK",
			patterns: []stubPattern{
				{
					method: "GET",
					ops:    []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1},
					pool:   []string{"foo", "id"},
				},
				{
					method: "GET",
					ops:    []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1},
					pool:   []string{"foo", "id"},
					verb:   "verb",
				},
			},
			reqMethod: "GET",
			reqPath:   "/foo/bar:verb",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			respStatus:  http.StatusOK,
			respContent: "GET /foo/{id=*}:verb",
		},
		{
			name: "GET to registered path without AssumeColonVerb should return GET 200 OK",
			patterns: []stubPattern{
				{
					method: "GET",
					ops:    []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1},
					pool:   []string{"foo", "id"},
				},
			},
			patternOpts: []runtime.PatternOpt{runtime.AssumeColonVerbOpt(false)},
			reqMethod:   "GET",
			reqPath:     "/foo/bar",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			respStatus:  http.StatusOK,
			respContent: "GET /foo/{id=*}",
		},
		{
			name: "GET to registered path without AssumeColonVerb with colon verb should return GET 200 OK",
			patterns: []stubPattern{
				{
					method: "GET",
					ops:    []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1},
					pool:   []string{"foo", "id"},
				},
			},
			patternOpts: []runtime.PatternOpt{runtime.AssumeColonVerbOpt(false)},
			reqMethod:   "GET",
			reqPath:     "/foo/bar:123",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			respStatus:  http.StatusOK,
			respContent: "GET /foo/{id=*}",
		},
		{
			name: "POST to registered path without AssumeColonVerb with LastMatchWins should match the correctly",
			patterns: []stubPattern{
				{
					method: "POST",
					ops:    []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1},
					pool:   []string{"foo", "id"},
				},
				{
					method: "POST",
					ops:    []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1},
					pool:   []string{"foo", "id"},
					verb:   "verb",
				},
			},
			patternOpts: []runtime.PatternOpt{runtime.AssumeColonVerbOpt(false)},
			reqMethod:   "POST",
			reqPath:     "/foo/bar:verb",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			respStatus:  http.StatusOK,
			respContent: "POST /foo/{id=*}:verb",
			muxOpts:     []runtime.ServeMuxOption{runtime.WithLastMatchWins()},
		},
	} {
		t.Run(spec.name, func(t *testing.T) {
			opts := spec.muxOpts
			if spec.disablePathLengthFallback {
				opts = append(opts, runtime.WithDisablePathLengthFallback())
			}
			mux := runtime.NewServeMux(opts...)
			for _, p := range spec.patterns {
				func(p stubPattern) {
					pat, err := runtime.NewPattern(1, p.ops, p.pool, p.verb, spec.patternOpts...)
					if err != nil {
						t.Fatalf("runtime.NewPattern(1, %#v, %#v, %q) failed with %v; want success", p.ops, p.pool, p.verb, err)
					}
					mux.Handle(p.method, pat, func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
						if r.URL.Path == "/unimplemented" {
							// simulate method returning "unimplemented" error
							_, m := runtime.MarshalerForRequest(mux, r)
							st := status.New(codes.Unimplemented, http.StatusText(http.StatusNotImplemented))
							m.NewEncoder(w).Encode(st.Proto())
							w.WriteHeader(http.StatusNotImplemented)
							return
						}
						fmt.Fprintf(w, "%s %s", p.method, pat.String())
					})
				}(p)
			}

			url := fmt.Sprintf("http://host.example%s", spec.reqPath)
			r, err := http.NewRequest(spec.reqMethod, url, bytes.NewReader(nil))
			if err != nil {
				t.Fatalf("http.NewRequest(%q, %q, nil) failed with %v; want success", spec.reqMethod, url, err)
			}
			for name, value := range spec.headers {
				r.Header.Set(name, value)
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)

			if got, want := w.Code, spec.respStatus; got != want {
				t.Errorf("w.Code = %d; want %d; patterns=%v; req=%v", got, want, spec.patterns, r)
			}
			if spec.respContent != "" {
				if got, want := w.Body.String(), spec.respContent; got != want {
					t.Errorf("w.Body = %q; want %q; patterns=%v; req=%v", got, want, spec.patterns, r)
				}
			}
		})
	}
}

var defaultHeaderMatcherTests = []struct {
	name     string
	in       string
	outValue string
	outValid bool
}{
	{
		"permanent HTTP header should return prefixed",
		"Accept",
		"grpcgateway-Accept",
		true,
	},
	{
		"key prefixed with MetadataHeaderPrefix should return without the prefix",
		"Grpc-Metadata-Custom-Header",
		"Custom-Header",
		true,
	},
	{
		"non-permanent HTTP header key without prefix should not return",
		"Custom-Header",
		"",
		false,
	},
}

func TestDefaultHeaderMatcher(t *testing.T) {
	for _, tt := range defaultHeaderMatcherTests {
		t.Run(tt.name, func(t *testing.T) {
			out, valid := runtime.DefaultHeaderMatcher(tt.in)
			if out != tt.outValue {
				t.Errorf("got %v, want %v", out, tt.outValue)
			}
			if valid != tt.outValid {
				t.Errorf("got %v, want %v", valid, tt.outValid)
			}
		})
	}
}
