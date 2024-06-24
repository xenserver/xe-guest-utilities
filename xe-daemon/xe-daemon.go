package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	guestmetric "github.com/xenserver/xe-guest-utilities/guestmetric"
	syslog "github.com/xenserver/xe-guest-utilities/syslog"
	system "github.com/xenserver/xe-guest-utilities/system"
	xenstoreclient "github.com/xenserver/xe-guest-utilities/xenstoreclient"
)

const (
	LoggerName           string = "xe-daemon"
	DivisorOne           int    = 1
	DivisorTwo           int    = 2
	DivisorLeastMultiple int    = 2 // The least common multiple, ensure every collector done before executing InvalidCacheFlush.
	SuspendStatusPath    string = "control/process_suspend_status"
	SysFreezeTimeoutPath string = "/sys/power/pm_freeze_timeout"
	DefaultTimeout       string = "20000"
	ExtendedTimeout      string = "300000"
)

func main() {
	var err error

	sleepInterval := flag.Int("i", 60, "Interval between updates (in seconds)")
	debugFlag := flag.Bool("d", false, "Update to log in addition to xenstore")
	balloonFlag := flag.Bool("B", true, "Do not report that ballooning is supported")
	pid := flag.String("p", "", "Write the PID to FILE")

	flag.Parse()

	if *pid != "" {
		if err = ioutil.WriteFile(*pid, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Write pid to %s error: %s\n", *pid, err)
			return
		}
	}

	var loggerWriter io.Writer = os.Stderr
	var topic string = LoggerName
	if w, err := syslog.NewSyslogWriter(topic, *debugFlag); err == nil {
		loggerWriter = w
		topic = ""
	} else {
		fmt.Fprintf(os.Stderr, "NewSyslogWriter(%s) error: %s, use stderr logging\n", topic, err)
		topic = LoggerName + ": "
	}

	logger := log.New(loggerWriter, topic, 0)

	exitChannel := make(chan os.Signal, 1)
	signal.Notify(exitChannel, syscall.SIGTERM, syscall.SIGINT)

	resumedChannel := make(chan int)
	go system.NotifyResumed(resumedChannel)

	xs, err := xenstoreclient.NewCachedXenstore(0)
	if err != nil {
		message := fmt.Sprintf("NewCachedXenstore error: %v\n", err)
		logger.Print(message)
		fmt.Fprint(os.Stderr, message)
		return
	}

	// Reset pm_freeze_timeout to default value every time the daemon starts
	if err := ioutil.WriteFile(SysFreezeTimeoutPath, []byte(DefaultTimeout), 0644); err != nil {
		logger.Printf("Reset pm freeze timeout failed: %v", err)
	}

	watchChannel, watch_err := xs.Watch([]string{SuspendStatusPath})
	if watch_err != nil {
		message := fmt.Sprintf("watch xenstore error: %v\n", err)
		logger.Print(message)
		fmt.Fprint(os.Stderr, message)
	}
	// Ignore the first event trigger by xenstore-watch
	<-watchChannel

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
		{DivisorOne, "CollectOS", collector.CollectOS},
		{DivisorOne, "CollectMisc", collector.CollectMisc},
		{DivisorOne, "CollectNetworkAddr", collector.CollectNetworkAddr},
		{DivisorOne, "CollectDisk", collector.CollectDisk},
		{DivisorTwo, "CollectMemory", collector.CollectMemory},
	}

	lastUniqueID, err := xs.Read("unique-domain-id")
	if err != nil {
		logger.Printf("xenstore.Read unique-domain-id error: %v\n", err)
	}

	for count := 0; ; count += 1 {
		uniqueID, err := xs.Read("unique-domain-id")
		if err != nil {
			logger.Printf("xenstore.Read unique-domain-id error: %v\n", err)
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
		updated := false
		for _, collector := range collectors {
			if count%collector.divisor == 0 {
				if *debugFlag {
					logger.Printf("Running %s ...\n", collector.name)
				}
				result, err := collector.Collect()
				if err != nil {
					logger.Printf("%s error: %#v\n", collector.name, err)
				} else {
					for name, value := range result {
						err := xs.Write(name, value)
						if err != nil {
							logger.Printf("xenstore.Write error: %v\n", err)
						} else {
							if *debugFlag {
								logger.Printf("xenstore.Write OK: %#v: %#v\n", name, value)
							}
							updated = true
						}
					}
				}
			}
		}
		if count%DivisorLeastMultiple == 0 {
			if cx, ok := xs.(*xenstoreclient.CachedXenStore); ok {
				err := cx.InvalidCacheFlush()
				if err != nil {
					logger.Printf("InvalidCacheFlush error: %#v\n", err)
				}
			}
		}

		if updated {
			xs.Write("data/updated", time.Now().Format("Mon Jan _2 15:04:05 2006"))
		}

		select {
		case event := <-watchChannel:
				if event.Path == SuspendStatusPath{
					if err := ioutil.WriteFile(SysFreezeTimeoutPath, []byte(ExtendedTimeout), 0644); err != nil {
						logger.Printf("Dump pm freeze timeout failed: %v", err)
					}
				}
			continue
		case <-exitChannel:
			xs.StopWatch()
			logger.Printf("Received an interrupt, stopping services...\n")
			if c, ok := loggerWriter.(io.Closer); ok {
				if err := c.Close(); err != nil {
					fmt.Fprintf(os.Stderr, "logger close error: %s\n", err)
				}
			}
			return

		case <-resumedChannel:
			if err := ioutil.WriteFile(SysFreezeTimeoutPath, []byte(DefaultTimeout), 0644); err != nil {
				logger.Printf("Dump pm freeze timeout failed: %v", err)
			}
			logger.Printf("Trigger refresh after system resume\n")
			continue

		case <-time.After(time.Duration(*sleepInterval) * time.Second):
			continue
		}
	}
}
