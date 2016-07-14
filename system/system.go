package sys

import (
	"math"
	"syscall"
	"time"
	"unsafe"
)

const (
	CLOCK_REALTIME          = 0
	TFD_CLOEXEC             = 02000000
	TFD_TIMER_ABSTIME       = 1
	TFD_TIMER_CANCEL_ON_SET = 2
)

type ITimerSpec struct {
	Interval syscall.Timespec
	Value    syscall.Timespec
}

// System call wrapper for timerfd_create, generated with mksyscall.pl
func timerfdCreate(clockid int, flags int) (fd int, err error) {
	r0, _, e1 := syscall.Syscall(syscall.SYS_TIMERFD_CREATE,
		uintptr(clockid), uintptr(flags), 0)
	fd = int(r0)
	if e1 != 0 {
		err = e1
	}
	return
}

// System call wrapper for timerfd_settime, generated with mksyscall.pl
func timerfdSettime(fd int, flags int, new_value *ITimerSpec,
	old_value *ITimerSpec) (err error) {
	_, _, e1 := syscall.Syscall6(syscall.SYS_TIMERFD_SETTIME, uintptr(fd),
		uintptr(flags),
		uintptr(unsafe.Pointer(new_value)),
		uintptr(unsafe.Pointer(old_value)), 0, 0)
	if e1 != 0 {
		err = e1
	}
	return
}

/*
 * Send a notification on @c when the system has just been resumed after
 * sleep. This is implemented by watching for a change in real time compared
 * with monotonic time. This may cause a spurious notification if the time
 * is changed by a user or NTP jump.
 */
func NotifyResumed(c chan int) {
	ts := ITimerSpec{Interval: syscall.Timespec{math.MaxInt32, 0},
		Value: syscall.Timespec{0, 0}}
	buf := make([]byte, 8)
	for {
		fd, err := timerfdCreate(CLOCK_REALTIME, TFD_CLOEXEC)
		if err != nil {
			return
		}

		err = timerfdSettime(fd, TFD_TIMER_ABSTIME|TFD_TIMER_CANCEL_ON_SET, &ts, nil)
		if err != nil {
			return
		}

		_, err = syscall.Read(fd, buf)
		if err == syscall.ECANCELED {
			// Wait a bit for the system to settle down after resuming
			time.Sleep(time.Duration(1) * time.Second)
			c <- 1
		}

		syscall.Close(fd)
	}
}
