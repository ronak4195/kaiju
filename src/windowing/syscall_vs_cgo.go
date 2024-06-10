package windowing

/*
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif

#ifndef UNICODE
#define UNICODE
#endif

#include <windows.h>
*/
import "C"
import (
	"syscall"

	"golang.org/x/sys/windows"
)

var (
	tkernel32               = windows.NewLazySystemDLL("Kernel32.dll")
	procGetCurrentProcessId = tkernel32.NewProc("GetCurrentProcessId")
)

func syscallTest() int {
	r1, _, _ := syscall.SyscallN(procGetCurrentProcessId.Addr())
	return int(r1)
}

func cgoTest() int {
	return int(C.GetCurrentProcessId())
}
