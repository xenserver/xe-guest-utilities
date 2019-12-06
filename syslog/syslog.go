// To run 32bit xe-daemon under 64bit system,
// Here we re-implement a syslog writer base on logger CLI.

package syslog

import (
	"io"
	"os"
	"os/exec"
	"time"
)

const (
	waitLoggerQuitSeconds = 5
)

type SysLoggerWriter struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func NewSyslogWriter(topic string, debug bool) (io.Writer, error) {
	// set lower priority by default
	priority := "debug"
	if debug {
		priority = "notice"
	}
	cmd := exec.Command("logger", "-t", topic, "-p", priority)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return &SysLoggerWriter{cmd, stdin}, nil
}

func (s *SysLoggerWriter) Write(data []byte) (int, error) {
	return s.stdin.Write(data)
}

func (s *SysLoggerWriter) Close() error {

	s.stdin.Close()
	s.cmd.Process.Signal(os.Interrupt)

	done := make(chan error, 1)
	go func(c chan<- error) {
		c <- s.cmd.Wait()
	}(done)

	select {
	case <-done:
		return nil
	case <-time.After(waitLoggerQuitSeconds * time.Second):
		return s.cmd.Process.Kill()
	}

	return nil
}
