package everquest

import (
	"errors"
	"fmt"
	"github.com/lxn/win"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

var (
	libUser32, _               = syscall.LoadLibrary("user32.dll")
	funcGetDesktopWindow, _    = syscall.GetProcAddress(syscall.Handle(libUser32), "GetDesktopWindow")
	funcEnumDisplayMonitors, _ = syscall.GetProcAddress(syscall.Handle(libUser32), "EnumDisplayMonitors")
)

var ewMutex sync.Mutex
var ewMapping = make(map[win.HWND]string)
var ewProc = syscall.NewCallback(func(handle win.HWND, _ uintptr) uintptr {
	iwv, _, _ := isWindowVisible.Call(uintptr(handle))
	if iwv != 0 {
		length, _, _ := getWindowTextLength.Call(uintptr(handle))
		buffer := make([]uint16, length+1)
		rl, _, _ := getWindowText.Call(uintptr(handle), uintptr(unsafe.Pointer(&buffer[0])), uintptr(length+1))
		if rl != 0 {
			buffer = buffer[:rl]
			ewMapping[handle] = syscall.UTF16ToString(buffer)
		}
	}
	return 1
})

func enumerateWindows() (mapping map[win.HWND]string, err error) {
	ewMutex.Lock()
	defer ewMutex.Unlock()
	ewMapping = make(map[win.HWND]string)
	rv, _, errTmp := enumWindows.Call(ewProc, 0)
	if rv == 0 {
		err = errTmp
	}
	mapping = ewMapping
	return
}

// Get the window handle for EverQuest
func findEverQuest() (win.HWND, error) {
	mapping, err := enumerateWindows()
	if err != nil {
		return 0, fmt.Errorf("Failed to enumerate windows: %v", err)
	}
	for handle, name := range mapping {
		if strings.Compare("EverQuest", name) == 0 {
			return handle, nil
		}
	}
	return 0, errors.New("No EverQuest window found")
}

// Bring the EverQuest window to the foreground
func raiseEverquest() error {
	handle, err := findEverQuest()
	if err != nil {
		return err
	}
	rv, _, tmpErr := setForegroundWindow.Call(uintptr(handle))
	if rv == 0 {
		return fmt.Errorf("Failed to set foreground window: %v", tmpErr)
	} else {
		return nil
	}
}

func getEqClientArea() (x int, y int, width int, height int, err error) {
	handle, err := findEverQuest()
	if err != nil {
		return
	}
	pt := point{0, 0}
	rv, _, tmpErr := screenToClient.Call(uintptr(handle), uintptr(unsafe.Pointer(&pt)))
	if rv == 0 {
		err = tmpErr
		return
	}
	x = -int(x)
	y = -int(y)
	wrect := rect{}
	rv, _, tmpErr = getClientRect.Call(uintptr(handle), uintptr(unsafe.Pointer(&wrect)))
	if rv == 0 {
		err = tmpErr
		return
	}
	width = int(wrect.right - wrect.left)
	height = int(wrect.bottom - wrect.top)
	return
}

func getDesktopWindow() win.HWND {
	ret, _, _ := syscall.Syscall(funcGetDesktopWindow, 0, 0, 0, 0)
	return win.HWND(ret)
}
