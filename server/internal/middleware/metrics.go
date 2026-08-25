package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	reqTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP 请求总数（按 method/status/path）",
		},
		[]string{"method", "status", "path"},
	)
	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP 请求耗时（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	paySuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_success_total",
			Help: "支付成功回调总数（按渠道）",
		},
		[]string{"method"},
	)
	cronJobRuns = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cron_job_runs_total",
			Help: "cron 任务执行总数（按 job/status: success|error|skipped）",
		},
		[]string{"job", "status"},
	)
	cronJobDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cron_job_duration_seconds",
			Help:    "cron 任务执行耗时（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"job"},
	)
)

func init() {
	prometheus.MustRegister(reqTotal, reqDuration, paySuccess, cronJobRuns, cronJobDuration)
}

// Metrics 请求指标采集（QPS/延迟直方图）。
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		reqTotal.WithLabelValues(c.Request.Method, strconv.Itoa(c.Writer.Status()), path).Inc()
		reqDuration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}

// PaySuccessInc 记录支付成功回调（供支付成功率监控）。
func PaySuccessInc(method string) {
	paySuccess.WithLabelValues(method).Inc()
}

// CronJobInc 记录 cron 任务执行结果（status: success/error/skipped，worker 进程打点）。
func CronJobInc(job, status string) {
	cronJobRuns.WithLabelValues(job, status).Inc()
}

// CronJobDuration 记录 cron 任务执行耗时（worker 进程打点）。
func CronJobDuration(job string, d time.Duration) {
	cronJobDuration.WithLabelValues(job).Observe(d.Seconds())
}
