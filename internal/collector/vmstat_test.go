package collector

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewVmstatCollector(t *testing.T) {
	c, err := NewVmstatCollector()
	if err != nil {
		t.Fatalf("NewVmstatCollector() error = %v", err)
	}
	if c == nil {
		t.Fatal("NewVmstatCollector() returned nil")
	}
}

func TestVmstatCollectorUpdate(t *testing.T) {
	// This is a basic test to ensure the collector runs without panicking.
	c, err := NewVmstatCollector()
	if err != nil {
		t.Fatalf("NewVmstatCollector() error = %v", err)
	}

	ch := make(chan prometheus.Metric)
	go func() {
		for range ch {
			// consume metrics
		}
	}()

	err = c.Update(ch, 10*time.Second)
	if err != nil {
		t.Errorf("VmstatCollector.Update() error = %v", err)
	}
	close(ch)
}
