package guestmetric

import (
	"testing"
)

func TestCollector(t *testing.T) {
	c := Collector{
		Client: nil,
	}

	funcs := []CollectFunc{
		c.CollectDisk,
		c.CollectMemory,
		c.CollectMisc,
		c.CollectNetworkAddr,
	}

	for _, f := range funcs {
		metric, err := f()
		if err != nil {
			t.Errorf("%#v error: %#v\n", f, err)
		}
		t.Logf("%#v return %#v", f, metric)
	}
}

func doBenchmark(b *testing.B, f CollectFunc) {
	b.Logf("doBenchmark 1000 for %#v", f)
	for i := 0; i < 1000; i++ {
		if _, err := f(); err != nil {
			b.Errorf("%#v error: %#v\n", f, err)
		}
	}
}

func BenchmarkCollectorDisk(b *testing.B) {
	c := Collector{
		Client: nil,
	}
	doBenchmark(b, c.CollectDisk)
}

func BenchmarkCollectMemory(b *testing.B) {
	c := Collector{
		Client: nil,
	}
	doBenchmark(b, c.CollectMemory)
}

func BenchmarkCollectMisc(b *testing.B) {
	c := Collector{
		Client: nil,
	}
	doBenchmark(b, c.CollectMisc)
}

func BenchmarkCollectNetwork(b *testing.B) {
	c := Collector{
		Client: nil,
	}
	doBenchmark(b, c.CollectNetworkAddr)
}
