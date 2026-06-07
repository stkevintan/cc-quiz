package routes

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"classical-chinese-quiz/internal/config"
	"github.com/gin-gonic/gin"
)

type stubFrontendAssets struct {
	dist     map[string][]byte
	fallback []byte
}

func (assets stubFrontendAssets) HasDistIndex() bool {
	_, ok := assets.dist["index.html"]
	return ok
}

func (assets stubFrontendAssets) HasDistAsset(name string) bool {
	_, ok := assets.dist[name]
	return ok
}

func (assets stubFrontendAssets) ReadDistAsset(name string) ([]byte, error) {
	body, ok := assets.dist[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return body, nil
}

func (assets stubFrontendAssets) FallbackIndexHTML() ([]byte, error) {
	return assets.fallback, nil
}

func TestNewRouterServesEmbeddedFrontendWithoutRedirects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newRouter(config.Config{AppEnv: "test"}, nil, stubFrontendAssets{
		dist: map[string][]byte{
			"index.html":     []byte(`<!doctype html><html><head><title>Quiz</title><script type="module" src="/assets/app.js"></script></head><body><div id="root"></div></body></html>`),
			"assets/app.js":  []byte(`console.log("quiz-app")`),
			"assets/app.css": []byte(`body{font-family:sans-serif;}`),
		},
		fallback: []byte("<html>fallback</html>"),
	})

	cases := []struct {
		name                string
		method              string
		target              string
		wantStatus          int
		wantContentTypePart string
		wantBodyContains    string
		wantLocation        string
	}{
		{
			name:                "root serves index",
			method:              http.MethodGet,
			target:              "/",
			wantStatus:          http.StatusOK,
			wantContentTypePart: "text/html",
			wantBodyContains:    `/assets/app.js`,
		},
		{
			name:                "index html serves directly",
			method:              http.MethodGet,
			target:              "/index.html",
			wantStatus:          http.StatusOK,
			wantContentTypePart: "text/html",
			wantBodyContains:    `<div id="root"></div>`,
		},
		{
			name:                "spa route falls back to index",
			method:              http.MethodGet,
			target:              "/student/dashboard",
			wantStatus:          http.StatusOK,
			wantContentTypePart: "text/html",
			wantBodyContains:    `<title>Quiz</title>`,
		},
		{
			name:                "asset serves bytes",
			method:              http.MethodGet,
			target:              "/assets/app.js",
			wantStatus:          http.StatusOK,
			wantContentTypePart: "javascript",
			wantBodyContains:    `console.log("quiz-app")`,
		},
		{
			name:                "head request returns headers only",
			method:              http.MethodHead,
			target:              "/",
			wantStatus:          http.StatusOK,
			wantContentTypePart: "text/html",
		},
		{
			name:       "missing file asset stays not found",
			method:     http.MethodGet,
			target:     "/assets/missing.js",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("%s %s status = %d, want %d, body=%q", tc.method, tc.target, rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantContentTypePart != "" && !strings.Contains(rec.Header().Get("Content-Type"), tc.wantContentTypePart) {
				t.Fatalf("%s %s content-type = %q, want substring %q", tc.method, tc.target, rec.Header().Get("Content-Type"), tc.wantContentTypePart)
			}
			if tc.wantBodyContains != "" && !strings.Contains(rec.Body.String(), tc.wantBodyContains) {
				t.Fatalf("%s %s body = %q, want substring %q", tc.method, tc.target, rec.Body.String(), tc.wantBodyContains)
			}
			if tc.wantLocation != "" && rec.Header().Get("Location") != tc.wantLocation {
				t.Fatalf("%s %s location = %q, want %q", tc.method, tc.target, rec.Header().Get("Location"), tc.wantLocation)
			}
			if tc.wantLocation == "" && rec.Header().Get("Location") != "" {
				t.Fatalf("%s %s location = %q, want empty", tc.method, tc.target, rec.Header().Get("Location"))
			}
			if tc.method == http.MethodHead && rec.Body.Len() != 0 {
				t.Fatalf("HEAD %s wrote body %q, want empty", tc.target, rec.Body.String())
			}
		})
	}
}

func TestNewRouterUsesFallbackWhenEmbeddedDistMissingAndPreservesAPI404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newRouter(config.Config{AppEnv: "test"}, nil, stubFrontendAssets{
		dist:     map[string][]byte{},
		fallback: []byte("<html><body>fallback page</body></html>"),
	})

	pageReq := httptest.NewRequest(http.MethodGet, "/teacher/reports", nil)
	pageRec := httptest.NewRecorder()
	router.ServeHTTP(pageRec, pageReq)
	if pageRec.Code != http.StatusOK {
		t.Fatalf("GET /teacher/reports status = %d, want %d", pageRec.Code, http.StatusOK)
	}
	if !strings.Contains(pageRec.Body.String(), "fallback page") {
		t.Fatalf("GET /teacher/reports body = %q, want fallback page", pageRec.Body.String())
	}

	apiReq := httptest.NewRequest(http.MethodGet, "/api/missing", nil)
	apiRec := httptest.NewRecorder()
	router.ServeHTTP(apiRec, apiReq)
	if apiRec.Code != http.StatusNotFound {
		t.Fatalf("GET /api/missing status = %d, want %d", apiRec.Code, http.StatusNotFound)
	}
	if !strings.Contains(apiRec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("GET /api/missing content-type = %q, want application/json", apiRec.Header().Get("Content-Type"))
	}
	if strings.TrimSpace(apiRec.Body.String()) != `{"error":"not found"}` {
		t.Fatalf("GET /api/missing body = %q, want JSON not found response", apiRec.Body.String())
	}
}
