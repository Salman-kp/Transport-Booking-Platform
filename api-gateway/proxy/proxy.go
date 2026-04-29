package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

func To(baseURL string) gin.HandlerFunc {
	targetURL, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalf("Invalid base URL for proxy: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		if xUserID := req.Header.Get("X-User-ID"); xUserID != "" {
			req.Header.Set("X-User-ID", xUserID)
		}
	}

	return func(c *gin.Context) {
		log.Printf("Proxying %s to %s%s", c.Request.Method, baseURL, c.Request.URL.Path)
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
