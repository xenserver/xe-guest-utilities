package guestmetric

import (
	"bytes"
	"os/exec"
)

type GuestMetric map[string]string

type CollectFunc func() (GuestMetric, error)

type GuestMetricsCollector interface {
	CollectOS() (GuestMetric, error)
	CollectMisc() (GuestMetric, error)
	CollectNetworkAddr() (GuestMetric, error)
	CollectDisk() (GuestMetric, error)
	CollectMemory() (GuestMetric, error)
}

func runCmd(name string, args ...string) (output string, err error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	output = out.String()
	return output, nil
}

func prefixKeys(prefix string, m GuestMetric) GuestMetric {
	m1 := make(GuestMetric, 0)
	for k, v := range m {
		m1[prefix+k] = v
	}
	return m1
}
