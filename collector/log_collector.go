package metric

import (
	"fmt"
	"log"
	"strings"

	"github.com/hpcloud/tail"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/satyrius/gonx"
	"github.com/songjiayang/nginx-log-exporter/config"
)

// Collector is a struct containing pointers to all metrics that should be
// exposed to Prometheus
type Collector struct {
	sessionsSeconds       *prometheus.HistogramVec
	sessionsBytesReceived *prometheus.HistogramVec
	sessionsBytesSent     *prometheus.HistogramVec

	staticValues    []string
	dynamicLabels   []string
	dynamicValueLen int

	cfg    *config.AppConfig
	parser *gonx.Parser
}

func NewCollector(cfg *config.AppConfig) *Collector {
	staticLabels, staticValues := cfg.StaticLabelValues()
	dynamicLabels := cfg.DynamicLabels()

	labels := append(staticLabels, dynamicLabels...)

	return &Collector{
		sessionsSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: cfg.Name,
			Name:      "stream_sessions_seconds",
			Help:      "Duration of reverse proxy stream session",
		}, labels),

		sessionsBytesReceived: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: cfg.Name,
			Name:      "stream_sessions_bytes_received",
			Help:      "Bytes received during reverse proxy stream session",
		}, labels),

		sessionsBytesSent: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: cfg.Name,
			Name:      "stream_sessions_bytes_sent",
			Help:      "Bytes sent upstream during reverse proxy stream session",
		}, labels),

		staticValues:    staticValues,
		dynamicLabels:   dynamicLabels,
		dynamicValueLen: len(dynamicLabels),

		cfg:    cfg,
		parser: gonx.NewParser(cfg.Format),
	}
}

func (c *Collector) Run() {
	c.cfg.Prepare()

	// register to prometheus
	prometheus.MustRegister(c.sessionsSeconds)
	prometheus.MustRegister(c.sessionsBytesReceived)
	prometheus.MustRegister(c.sessionsBytesSent)

	for _, f := range c.cfg.SourceFiles {
		t, err := tail.TailFile(f, tail.Config{
			Follow: true,
			ReOpen: true,
			Poll:   true,
		})

		if err != nil {
			log.Panic(err)
		}

		go func() {
			for line := range t.Lines {
				entry, err := c.parser.ParseString(line.Text)
				if err != nil {
					fmt.Printf("error while parsing line '%s': %s", line.Text, err)
					continue
				}

				dynamicValues := make([]string, c.dynamicValueLen)

				for i, label := range c.dynamicLabels {
					if s, err := entry.Field(label); err == nil {
						dynamicValues[i] = c.formatValue(label, s)
					}
				}

				labelValues := append(c.staticValues, dynamicValues...)

				if sessionTime, err := entry.FloatField("session_time"); err == nil {
					c.sessionsSeconds.WithLabelValues(labelValues...).Observe(sessionTime)
				}

				if bytes, err := entry.FloatField("bytes_sent"); err == nil {
					c.sessionsBytesSent.WithLabelValues(labelValues...).Observe(bytes)
				}

				if bytes, err := entry.FloatField("bytes_received"); err == nil {
					c.sessionsBytesReceived.WithLabelValues(labelValues...).Observe(bytes)
				}
			}
		}()
	}
}

func (c *Collector) formatValue(label, value string) string {
	replacement, ok := c.cfg.RelabelConfig.Replacement[label]
	if !ok {
		return value
	}

	if replacement.Trim != "" {
		value = strings.Split(value, replacement.Trim)[0]
	}

	for _, target := range replacement.Repace {
		if target.Regexp().MatchString(value) {
			return target.Value
		}
	}

	return value
}
