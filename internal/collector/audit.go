package collector

import (
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/hpcloud/tail"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	k8sAuditRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "k8s_audit_requests_total",
			Help: "Total number of audit events.",
		},
		[]string{"verb", "user", "resource", "namespace", "code"},
	)
	k8sAuditRequestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "k8s_audit_request_latency_seconds",
			Help: "API request latency.",
		},
		[]string{"verb", "resource", "namespace"},
	)
)

type AuditCollector struct {
	LogPath string
	once    sync.Once
}

func NewAuditCollector(logPath string) (Collector, error) {
	return &AuditCollector{LogPath: logPath}, nil
}

func (c *AuditCollector) Update(ch chan<- prometheus.Metric) error {
	c.once.Do(func() {
		go c.tailLog()
	})

	k8sAuditRequestsTotal.Collect(ch)
	k8sAuditRequestLatency.Collect(ch)

	return nil
}

func (c *AuditCollector) tailLog() {
	t, err := tail.TailFile(c.LogPath, tail.Config{Follow: true})
	if err != nil {
		log.Printf("Error tailing audit log: %v", err)
		return
	}
	for line := range t.Lines {
		var event struct {
			Verb                     string `json:"verb"`
			User                     struct {
				Username string `json:"username"`
			} `json:"user"`
			ObjectRef                struct {
				Resource  string `json:"resource"`
				Namespace string `json:"namespace"`
			} `json:"objectRef"`
			ResponseStatus           struct {
				Code int `json:"code"`
			} `json:"responseStatus"`
			StageTimestamp           string `json:"stageTimestamp"`
			RequestReceivedTimestamp string `json:"requestReceivedTimestamp"`
		}
		if err := json.Unmarshal([]byte(line.Text), &event); err != nil {
			log.Printf("Error parsing audit log event: %v", err)
			continue
		}

		k8sAuditRequestsTotal.WithLabelValues(
			event.Verb,
			event.User.Username,
			event.ObjectRef.Resource,
			event.ObjectRef.Namespace,
			strconv.Itoa(event.ResponseStatus.Code),
		).Inc()

		if event.StageTimestamp != "" && event.RequestReceivedTimestamp != "" {
			stageTime, err := time.Parse(time.RFC3339Nano, event.StageTimestamp)
			if err != nil {
				log.Printf("Error parsing stage timestamp: %v", err)
				continue
			}
			receivedTime, err := time.Parse(time.RFC3339Nano, event.RequestReceivedTimestamp)
			if err != nil {
				log.Printf("Error parsing received timestamp: %v", err)
				continue
			}
			latency := stageTime.Sub(receivedTime).Seconds()
			k8sAuditRequestLatency.WithLabelValues(
				event.Verb,
				event.ObjectRef.Resource,
				event.ObjectRef.Namespace,
			).Observe(latency)
		}
	}
}
