package status

import (
	"syscall"
	"time"
	"unsafe"
)

// Monotonic returns CLOCK_MONOTONIC, the clock systemd uses for
// *TimestampMonotonic properties (DATA-08).
func Monotonic() time.Duration {
	var ts syscall.Timespec
	const clockMonotonic = 1
	if _, _, e := syscall.Syscall(syscall.SYS_CLOCK_GETTIME, clockMonotonic, uintptr(unsafe.Pointer(&ts)), 0); e != 0 {
		return 0
	}
	return time.Duration(ts.Sec)*time.Second + time.Duration(ts.Nsec)
}
