package main

import (
	guestmetric "../guestmetric"
	xenstoreclient "../xenstoreclient"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func write_pid_file(pid_file string) error {
	f, err := os.Create(pid_file)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "%d\n", os.Getpid())
	return nil
}

func main() {
	var err error

	sleepInterval := flag.Int("i", 60, "Interval between updates (in seconds)")
	debugFlag := flag.Bool("d", false, "Update to stdout rather than xenstore")
	balloonFlag := flag.Bool("B", true, "Do not report that ballooning is supported")
	pid := flag.String("p", "", "Write the PID to FILE")

	flag.Parse()

	if *pid != "" {
		write_pid_file(*pid)
	}

	logger := log.New(os.Stderr, "xe-daemon", 0)

	exitChannel := make(chan os.Signal, 1)
	signal.Notify(exitChannel, syscall.SIGTERM, syscall.SIGINT)

	xs, err := xenstoreclient.NewCachedXenstore(0)
	if err != nil {
		logger.Printf("NewCachedXenstore error: %v", err)
		return
	}

	collector := &guestmetric.Collector{
		Client: xs,
		Ballon: *balloonFlag,
		Debug:  *debugFlag,
	}

	collectors := []struct {
		divisor int
		name    string
		Collect func() (guestmetric.GuestMetric, error)
	}{
		{1, "CollectOS", collector.CollectOS},
		{1, "CollectMisc", collector.CollectMisc},
		{1, "CollectNetworkAddr", collector.CollectNetworkAddr},
		{1, "CollectDisk", collector.CollectDisk},
		{2, "CollectMemory", collector.CollectMemory},
	}

	lastUniqueID, err := xs.Read("unique-domain-id")
	if err != nil {
		logger.Printf("xenstore.Read unique-domain-id error: %v", err)
	}

	for count := 0; ; count += 1 {
		uniqueID, err := xs.Read("unique-domain-id")
		if err != nil {
			logger.Printf("xenstore.Read unique-domain-id error: %v", err)
			return
		}
		if uniqueID != lastUniqueID {
			// VM has just resume, cache state now invalid
			lastUniqueID = uniqueID
			if cx, ok := xs.(*xenstoreclient.CachedXenStore); ok {
				cx.Clear()
			}
		}

		// invoke collectors
		for _, collector := range collectors {
			if count%collector.divisor == 0 {
				logger.Printf("Running %s ...", collector.name)
				result, err := collector.Collect()
				if err != nil {
					logger.Printf("%s error: %#v", collector.name, err)
				} else {
					for name, value := range result {
						err := xs.Write(name, value)
						if err != nil {
							logger.Printf("xenstore.Write error: %v", err)
						} else {
							logger.Printf("xenstore.Write OK: %#v: %#v", name, value)
						}
					}
				}
			}
		}

		xs.Write("data/updated", time.Now().Format("Mon Jan _2 15:04:05 2006"))

		select {
		case <-exitChannel:
			logger.Printf("Received an interrupt, stopping services...")
			return

		case <-time.After(time.Duration(*sleepInterval) * time.Second):
			continue
		}
	}
}
