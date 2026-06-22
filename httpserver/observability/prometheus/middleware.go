package prometheus

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	httpmiddleware "github.com/tsopia/go-kit/httpserver/middleware"
	"github.com/tsopia/go-kit/utils"
)

type metricKey struct {
	method string
	route  string
	status string
}

type requestMetric struct {
	count       uint64
	durationSum time.Duration
}

// Collector 保存 HTTP 指标。
type Collector struct {
	mu          sync.RWMutex
	metrics     map[metricKey]requestMetric
	streamGauge map[string]int64
}

var defaultCollector = NewCollector()

// NewCollector 创建新的指标收集器。
func NewCollector() *Collector {
	return &Collector{
		metrics:     make(map[metricKey]requestMetric),
		streamGauge: make(map[string]int64),
	}
}

// Middleware 返回默认 collector 的指标中间件。
func Middleware() gin.HandlerFunc {
	return defaultCollector.Middleware()
}

// Middleware 返回 collector 自身的指标中间件。
func (c *Collector) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startedAt := time.Now()

		ctx.Request = ctx.Request.WithContext(
			httpmiddleware.WithStreamObserver(ctx.Request.Context(), collectorStreamObserver{collector: c}),
		)

		ctx.Next()

		if ctx.GetString(utils.StreamingKey) != "" {
			return
		}

		route := ctx.FullPath()
		if route == "" {
			route = ctx.Request.URL.Path
		}
		if route == "" {
			route = "/"
		}

		key := metricKey{
			method: ctx.Request.Method,
			route:  route,
			status: http.StatusText(ctx.Writer.Status()),
		}
		if key.status == "" {
			key.status = http.StatusText(http.StatusOK)
		}
		key.status = strings.TrimSpace(key.status)
		key.status = strings.ReplaceAll(key.status, " ", "_")

		c.observe(metricKey{
			method: key.method,
			route:  key.route,
			status: statusCodeString(ctx.Writer.Status()),
		}, time.Since(startedAt))
	}
}

func (c *Collector) observe(key metricKey, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	metric := c.metrics[key]
	metric.count++
	metric.durationSum += duration
	c.metrics[key] = metric
}

// IncStream 活跃流式连接 +1。
func (c *Collector) IncStream(transport string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.streamGauge[transport]++
}

// DecStream 活跃流式连接 -1，归零时移除键以保持 render 干净。
func (c *Collector) DecStream(transport string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.streamGauge[transport]--
	if c.streamGauge[transport] <= 0 {
		delete(c.streamGauge, transport)
	}
}

type collectorStreamObserver struct {
	collector *Collector
}

func (o collectorStreamObserver) OnConnect(transport string)    { o.collector.IncStream(transport) }
func (o collectorStreamObserver) OnDisconnect(transport string) { o.collector.DecStream(transport) }

func (c *Collector) snapshot() map[metricKey]requestMetric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := make(map[metricKey]requestMetric, len(c.metrics))
	for key, metric := range c.metrics {
		snapshot[key] = metric
	}

	return snapshot
}

func (c *Collector) render() string {
	snapshot := c.snapshot()
	keys := make([]metricKey, 0, len(snapshot))
	for key := range snapshot {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].route != keys[j].route {
			return keys[i].route < keys[j].route
		}
		if keys[i].method != keys[j].method {
			return keys[i].method < keys[j].method
		}
		return keys[i].status < keys[j].status
	})

	var builder strings.Builder
	builder.WriteString("# HELP http_requests_total Total number of HTTP requests.\n")
	builder.WriteString("# TYPE http_requests_total counter\n")
	for _, key := range keys {
		metric := snapshot[key]
		builder.WriteString("http_requests_total{")
		builder.WriteString(metricLabels(key))
		builder.WriteString("} ")
		builder.WriteString(uint64String(metric.count))
		builder.WriteString("\n")
	}

	builder.WriteString("# HELP http_request_duration_seconds_sum Cumulative request duration in seconds.\n")
	builder.WriteString("# TYPE http_request_duration_seconds_sum counter\n")
	for _, key := range keys {
		metric := snapshot[key]
		builder.WriteString("http_request_duration_seconds_sum{")
		builder.WriteString(metricLabels(key))
		builder.WriteString("} ")
		builder.WriteString(durationSeconds(metric.durationSum))
		builder.WriteString("\n")
	}

	c.mu.RLock()
	gauge := make(map[string]int64, len(c.streamGauge))
	for transport, value := range c.streamGauge {
		gauge[transport] = value
	}
	c.mu.RUnlock()

	transports := make([]string, 0, len(gauge))
	for transport := range gauge {
		transports = append(transports, transport)
	}
	sort.Strings(transports)

	builder.WriteString("# HELP streaming_active_connections Currently active streaming connections.\n")
	builder.WriteString("# TYPE streaming_active_connections gauge\n")
	for _, transport := range transports {
		builder.WriteString(`streaming_active_connections{transport="`)
		builder.WriteString(escapeLabelValue(transport))
		builder.WriteString(`"} `)
		builder.WriteString(strconv.FormatInt(gauge[transport], 10))
		builder.WriteString("\n")
	}

	return builder.String()
}

func metricLabels(key metricKey) string {
	return `method="` + escapeLabelValue(key.method) +
		`",route="` + escapeLabelValue(key.route) +
		`",status="` + escapeLabelValue(key.status) + `"`
}

func escapeLabelValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)

	return value
}
