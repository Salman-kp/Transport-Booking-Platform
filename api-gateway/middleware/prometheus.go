package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsRegisterOnce sync.Once
	httpRequestTotal    = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_total",
			Help: "Total Request Count",
		},
		[]string{"method", "path", "status"},
	)
)

var httpRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "HTTP Request latency",
	},
	[]string{"method", "path", "status"},
)

func PrometheusHTTP() gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		metricsRegisterOnce.Do(func() {
			prometheus.MustRegister(httpRequestTotal, httpRequestDuration)
		})

		start := time.Now()

		c.Next()

		duration := time.Since(start)

		method := c.Request.Method

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		status := strconv.Itoa(c.Writer.Status())

		httpRequestTotal.WithLabelValues(method, route, status).Inc()

		httpRequestDuration.WithLabelValues(method, route, status).Observe(duration.Seconds())
	}
}
