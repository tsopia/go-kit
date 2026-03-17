package prometheus

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Config 描述 metrics 路由注册配置。
type Config struct {
	Path      string
	Collector *Collector
}

// Register 显式注册 metrics 路由。
func Register(r gin.IRoutes, config Config) {
	path := config.Path
	if path == "" {
		path = "/metrics"
	}

	collector := config.Collector
	if collector == nil {
		collector = defaultCollector
	}

	r.GET(path, gin.WrapH(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(collector.render()))
	})))
}

func statusCodeString(status int) string {
	if status == 0 {
		status = http.StatusOK
	}

	return strconv.Itoa(status)
}

func uint64String(value uint64) string {
	return strconv.FormatUint(value, 10)
}

func durationSeconds(duration time.Duration) string {
	return strconv.FormatFloat(duration.Seconds(), 'f', 6, 64)
}
