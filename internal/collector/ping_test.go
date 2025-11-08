package collector

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewPingCollector(t *testing.T) {
	targets := []string{"localhost"}
	c, err := NewPingCollector(targets)
	if err != nil {
		t.Fatalf("NewPingCollector() error = %v", err)
	}
	if c == nil {
		t.Fatal("NewPingCollector() returned nil")
	}
}

func TestPingCollectorUpdate(t *testing.T) {
	// This is a basic test to ensure the collector runs without panicking.
	// It does not validate the metrics as that would require a live network.
	targets := []string{"127.0.0.1"}
	c, err := NewPingCollector(targets)
	if err != nil {
		t.Fatalf("NewPingCollector() error = %v", err)
	}

	ch := make(chan prometheus.Metric)
	go func() {
		for range ch {
			// consume metrics
		}
	}()

	err = c.Update(ch, 10*time.Second)
	if err != nil {
		// We expect this to fail in some environments where privileged ping is not allowed.
		// So we don't fail the test, but we can log it.
		t.Logf("Ping collector update returned an error (this may be expected): %v", err)
	}
	close(ch)
}
