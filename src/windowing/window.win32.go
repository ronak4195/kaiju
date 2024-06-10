//go:build windows

/******************************************************************************/
/* window.win32.go                                                            */
/******************************************************************************/
/*                           This file is part of:                            */
/*                                KAIJU ENGINE                                */
/*                          https://kaijuengine.org                           */
/******************************************************************************/
/* MIT License                                                                */
/*                                                                            */
/* Copyright (c) 2023-present Kaiju Engine authors (AUTHORS.md).              */
/* Copyright (c) 2015-present Brent Farris.                                   */
/*                                                                            */
/* May all those that this source may reach be blessed by the LORD and find   */
/* peace and joy in life.                                                     */
/* Everyone who drinks of this water will be thirsty again; but whoever       */
/* drinks of the water that I will give him shall never thirst; John 4:13-14  */
/*                                                                            */
/* Permission is hereby granted, free of charge, to any person obtaining a    */
/* copy of this software and associated documentation files (the "Software"), */
/* to deal in the Software without restriction, including without limitation  */
/* the rights to use, copy, modify, merge, publish, distribute, sublicense,   */
/* and/or sell copies of the Software, and to permit persons to whom the      */
/* Software is furnished to do so, subject to the following conditions:       */
/*                                                                            */
/* The above copyright, blessing, biblical verse, notice and                  */
/* this permission notice shall be included in all copies or                  */
/* substantial portions of the Software.                                      */
/*                                                                            */
/* THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS    */
/* OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF                 */
/* MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.     */
/* IN NO EVENT SHALL THE /* AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY    */
/* CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT  */
/* OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE      */
/* OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.                              */
/******************************************************************************/

package windowing

import (
	"syscall"
	"unsafe"

	"golang.design/x/clipboard"
	"golang.org/x/sys/windows"
)

/*
#cgo LDFLAGS: -lgdi32 -lXInput
//#cgo noescape window_main
//#cgo noescape window_show
//#cgo noescape window_destroy
//#cgo noescape window_cursor_standard
//#cgo noescape window_cursor_ibeam
//#cgo noescape window_dpi
//#cgo noescape window_focus
//#cgo noescape window_position
//#cgo noescape window_set_position
//#cgo noescape window_set_size
//#cgo noescape window_remove_border
//#cgo noescape window_add_border
//#cgo noescape window_poll_controller
//#cgo noescape window_poll

#include "windowing.h"
*/
import "C"

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	kernel32               = windows.NewLazySystemDLL("Kernel32.dll")
	procGetModuleHandle    = kernel32.NewProc("GetModuleHandleW")
	procSetClassLongPtr    = user32.NewProc("SetClassLongPtrW")
	procSetCursor          = user32.NewProc("SetCursor")
	procLoadCursor         = user32.NewProc("LoadCursorW")
	procLoadIcon           = user32.NewProc("LoadIconW")
	procRegisterClass      = user32.NewProc("RegisterClassW")
	procAdjustWindowRectEx = user32.NewProc("AdjustWindowRectEx")
	procCreateWindowEx     = user32.NewProc("CreateWindowExW")
	procDestroyWindow      = user32.NewProc("DestroyWindow")
	procPostQuitMessage    = user32.NewProc("PostQuitMessage")
	procGetClientRect      = user32.NewProc("GetClientRect")
	procDefWindowProc      = user32.NewProc("DefWindowProc")
	procShowWindow         = user32.NewProc("ShowWindow")
	procPostMessage        = user32.NewProc("PostMessageW")
	procPostMessageA       = user32.NewProc("PostMessageA")
	procXInputGetState     = user32.NewProc("XInputGetState")
	procPeekMessage        = user32.NewProc("PeekMessage")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procSetCapture         = user32.NewProc("SetCapture")
	procReleaseCapture     = user32.NewProc("ReleaseCapture")
	procMapVirtualKey      = user32.NewProc("MapVirtualKeyW")
	procDispatchMessage    = user32.NewProc("DispatchMessage")
	procGetDpiForWindow    = user32.NewProc("GetDpiForWindow")
	procBringWindowToTop   = user32.NewProc("BringWindowToTop")
	procSetFocus           = user32.NewProc("SetFocus")
	procGetWindowPlacement = user32.NewProc("GetWindowPlacement")
	procSetWindowPos       = user32.NewProc("SetWindowPos")
	procGetWindowLongW     = user32.NewProc("GetWindowLongW")
	procSetWindowLongW     = user32.NewProc("SetWindowLongW")
)

const (
	procNULL                = 0
	procFALSE               = 0
	procTRUE                = 1
	procGCLP_HCURSOR        = -12
	procIDC_ARROW           = 32512
	procIDC_IBEAM           = 32513
	procIDC_WAIT            = 32514
	procIDC_CROSS           = 32515
	procIDC_UPARROW         = 32516
	procIDC_SIZENWSE        = 32642
	procIDC_SIZENESW        = 32643
	procIDC_SIZEWE          = 32644
	procIDC_SIZENS          = 32645
	procIDC_SIZEALL         = 32646
	procIDC_NO              = 32648
	procIDC_HAND            = 32649
	procIDC_APPSTARTING     = 32650
	procIDC_HELP            = 32651
	procIDC_PIN             = 32671
	procIDC_PERSON          = 32672
	procIDI_APPLICATION     = 32512
	procWS_OVERLAPPEDWINDOW = 0x00000000 | 0x00C00000 | 0x00080000 | 0x00040000 | 0x00020000 | 0x00010000
	procCW_USEDEFAULT       = 0x80000000
	procSW_SHOW             = 5
	procSWP_NOSIZE          = 0x0001
	procSWP_NOMOVE          = 0x0002
	procSWP_NOZORDER        = 0x0004
	procGWL_STYLE           = -16
	procWS_CAPTION          = 0x00C00000
	procWS_THICKFRAME       = 0x00040000
	procWS_MINIMIZEBOX      = 0x00020000
	procWS_MAXIMIZEBOX      = 0x00010000
	procWS_SYSMENU          = 0x00080000
	procWM_ACTIVATE         = 0x0006
	procWA_ACTIVE           = 1
	procWA_CLICKACTIVE      = 2
	procWA_INACTIVE         = 0
	procWM_MOVE             = 0x0003
	procWM_SIZE             = 0x0005
	procWM_PAINT            = 0x000F
	procPM_REMOVE           = 0x0001
	procWM_QUIT             = 0x0012
	procWM_DESTROY          = 0x0002
	procWM_MOUSEMOVE        = 0x0200
	procWM_LBUTTONDOWN      = 0x0201
	procWM_LBUTTONUP        = 0x0202
	procWM_MBUTTONDOWN      = 0x0207
	procWM_MBUTTONUP        = 0x0208
	procWM_RBUTTONDOWN      = 0x0204
	procWM_RBUTTONUP        = 0x0205
	procWM_XBUTTONDOWN      = 0x020B
	procWM_XBUTTONUP        = 0x020C
	procWM_MOUSEWHEEL       = 0x020A
	procWM_MOUSEHWHEEL      = 0x020E
	procWM_KEYDOWN          = 0x0100
	procWM_SYSKEYDOWN       = 0x0104
	procWM_KEYUP            = 0x0101
	procWM_SYSKEYUP         = 0x0105
	procVK_SHIFT            = 0x10
	procVK_CONTROL          = 0x11
	procVK_MENU             = 0x12
	procVK_RCONTROL         = 0xA3
	procVK_LCONTROL         = 0xA2
	procVK_RMENU            = 0xA5
	procVK_LMENU            = 0xA4
	procMAPVK_VSC_TO_VK_EX  = 3

	procSHARED_MEM_WINDOW_ACTIVITY  = 0xF9
	procSHARED_MEM_WINDOW_MOVE      = 0xFA
	procSHARED_MEM_WINDOW_RESIZE    = 0xFB
	procSHARED_MEM_AWAITING_CONTEXT = 0xFC
	procSHARED_MEM_AWAITING_START   = 0xFD
	procSHARED_MEM_FATAL            = 0xFE
	procSHARED_MEM_QUIT             = 0xFF
	procSHARED_MEM_DATA_START       = 4

	procUWM_SET_CURSOR      = 0x0400 + 0x0001
	procCURSOR_ARROW        = 1
	procCURSOR_IBEAM        = 2
	procCURSOR_WAIT         = 3
	procCURSOR_CROSS        = 4
	procCURSOR_UPARROW      = 5
	procCURSOR_SIZE_NWSE    = 6
	procCURSOR_SIZE_NESW    = 7
	procCURSOR_SIZE_WE      = 8
	procCURSOR_SIZE_NS      = 9
	procCURSOR_SIZE_ALL     = 10
	procCURSOR_NO           = 11
	procCURSOR_HAND         = 12
	procCURSOR_APP_STARTING = 13
	procCURSOR_HELP         = 14
	procCURSOR_PIN          = 15
	procCURSOR_PERSON       = 16

	procMOUSE_WHEEL_HORIZONTAL = -2
	procMOUSE_WHEEL_VERTICAL   = -1
	procMOUSE_BUTTON_LEFT      = 0
	procMOUSE_BUTTON_MIDDLE    = 1
	procMOUSE_BUTTON_RIGHT     = 2
	procMOUSE_BUTTON_X1        = 3
	procMOUSE_BUTTON_X2        = 4

	procMAX_CONTROLLERS = 4
	procERROR_SUCCESS   = 0
)

type HWND = uintptr
type HINSTANCE = uintptr
type HMODULE = HINSTANCE
type WNDPROC = uintptr
type HICON = uintptr
type HCURSOR = uintptr
type HBRUSH = uintptr
type LPCWSTR = []uint16
type WPARAM = uintptr
type LPARAM = uintptr

var localWindowLookup map[HWND]*Window = make(map[uintptr]*Window)

type WNDCLASS struct {
	Style         uint32
	LpfnWndProc   WNDPROC
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     HINSTANCE
	HIcon         HICON
	HCursor       HCURSOR
	HbrBackground HBRUSH
	LpszMenuName  LPCWSTR
	LpszClassName LPCWSTR
}

type POINT struct {
	X int32
	Y int32
}

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type WINDOWPLACEMENT struct {
	Length           uint32
	Flags            uint32
	ShowCmd          uint32
	PtMinPosition    POINT
	PtMaxPosition    POINT
	RcNormalPosition RECT
}

type MSG struct {
	Hwnd    HWND
	Message uint32
	WParam  WPARAM
	LParam  LPARAM
	Time    uint32
	Pt      POINT
}

type XINPUT_GAMEPAD struct {
	WButtons      uint16
	BLeftTrigger  uint8
	BRightTrigger uint8
	SThumbLX      int16
	SThumbLY      int16
	SThumbRX      int16
	SThumbRY      int16
}

type XINPUT_STATE struct {
	DwPacketNumber int32
	Gamepad        XINPUT_GAMEPAD
}

func procLOWORD(param uintptr) uintptr {
	return param & 0xFFFF
}

func procHIWORD(param uintptr) uintptr {
	return (param >> 16) & 0xffff
}

func asEventType(msg uint32) eventType {
	switch msg {
	case 0x0002:
		fallthrough
	case 0x0012:
		return evtQuit
	case 0x0003:
		return evtMove
	case 0x0005:
		return evtResize
	case 0x0006:
		return evtActivity
	case 0x0104:
		fallthrough
	case 0x0100:
		return evtKeyDown
	case 0x0105:
		fallthrough
	case 0x0101:
		return evtKeyUp
	case 512:
		return evtMouseMove
	case 513:
		return evtLeftMouseDown
	case 514:
		return evtLeftMouseUp
	case 516:
		return evtRightMouseDown
	case 517:
		return evtRightMouseUp
	case 519:
		return evtMiddleMouseDown
	case 520:
		return evtMiddleMouseUp
	case 523:
		return evtX1MouseDown
	case 524:
		return evtX1MouseUp
	case 0x020A:
		return evtMouseWheelVertical
	case 0x020E:
		return evtMouseWheelHorizontal
	case 0xFFFFFFFF - 1:
		return evtControllerStates
	default:
		return evtUnknown
	}
}

func scaleScrollDelta(delta float32) float32 {
	return delta / 120.0
}

func createWindow(w *Window, windowName string, width, height, x, y int) {
	//C.window_main(title, C.int(width), C.int(height),
	//	C.int(x), C.int(y), w.evtSharedMem.AsPointer(), evtSharedMemSize)

	title, _ := syscall.UTF16FromString(windowName)
	className, _ := syscall.UTF16FromString("Kaiju Window Class")
	r1, _, _ := syscall.SyscallN(procGetModuleHandle.Addr(), procNULL)
	hInstance := HMODULE(r1)
	wc := WNDCLASS{}
	wc.LpfnWndProc = syscall.NewCallback(windowWndProc)
	wc.HInstance = hInstance
	wc.LpszClassName = className
	r1, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_ARROW)
	wc.HCursor = HCURSOR(r1)
	r1, _, _ = syscall.SyscallN(procLoadIcon.Addr(), procNULL, procIDI_APPLICATION)
	wc.HIcon = HICON(r1)
	syscall.SyscallN(procRegisterClass.Addr(), uintptr(unsafe.Pointer(&wc)))
	clientArea := RECT{0, 0, int32(width), int32(height)}
	syscall.SyscallN(procAdjustWindowRectEx.Addr(),
		uintptr(unsafe.Pointer(&clientArea)), procWS_OVERLAPPEDWINDOW, procFALSE, 0)
	width = int(clientArea.Right - clientArea.Left)
	height = int(clientArea.Bottom - clientArea.Top)
	if x < 0 {
		x = procCW_USEDEFAULT
	}
	if y < 0 {
		x = procCW_USEDEFAULT
	}
	r1, _, _ = syscall.SyscallN(procCreateWindowEx.Addr(),
		0,                                      // Optional window styles.
		uintptr(unsafe.Pointer(&className[0])), // Window class
		uintptr(unsafe.Pointer(&title[0])),     // Window text
		procWS_OVERLAPPEDWINDOW,                // Window style
		uintptr(x),                             // Position Y
		uintptr(y),                             // Position X
		uintptr(width),                         // Size X
		uintptr(height),                        // Size Y
		procNULL,                               // Parent window
		procNULL,                               // Menu
		hInstance,                              // Instance handle
		procNULL,                               // Additional application data
	)
	hwnd := HWND(r1)
	if hwnd == procNULL {
		panic("Failed to create window.")
	}
	w.handle = unsafe.Pointer(hwnd)
	w.instance = unsafe.Pointer(hInstance)
	w.CursorStandard()
	w.clientWidth = int32(width)
	w.clientHeight = int32(height)
	localWindowLookup[hwnd] = w
}

func windowWndProc(hwnd HWND, uMsg uint32, wParam uintptr, lParam uintptr) uintptr {
	w := localWindowLookup[hwnd]
	switch uMsg {
	case procWM_DESTROY:
		w.evtSharedMem.SetWriteState(procSHARED_MEM_QUIT)
		syscall.SyscallN(procPostQuitMessage.Addr(), 0)
		return 0
	case procWM_ACTIVATE:
		switch procLOWORD(wParam) {
		case procWA_ACTIVE:
			fallthrough
		case procWA_CLICKACTIVE:
			w.evtSharedMem.toEnumEvent().value = 1
		case procWA_INACTIVE:
			w.evtSharedMem.toEnumEvent().value = 0
		}
		w.evtSharedMem.SetWriteState(procSHARED_MEM_WINDOW_ACTIVITY)
	case procWM_MOVE:
		evt := w.evtSharedMem.toWindowMoveEvent()
		evt.x = int32(procLOWORD(lParam))
		evt.y = int32(procHIWORD(lParam))
		w.evtSharedMem.SetWriteState(procSHARED_MEM_WINDOW_MOVE)
	case procWM_SIZE:
		clientArea := RECT{}
		syscall.SyscallN(procGetClientRect.Addr(),
			uintptr(w.handle), uintptr(unsafe.Pointer(&clientArea)))
		width := clientArea.Right - clientArea.Left
		height := clientArea.Bottom - clientArea.Top
		if w.clientWidth != width || w.clientHeight != height {
			w.clientWidth = width
			w.clientHeight = height
			evt := w.evtSharedMem.toWindowResizeEvent()
			evt.width = width
			evt.height = height
			w.evtSharedMem.SetWriteState(procSHARED_MEM_WINDOW_RESIZE)
		}
		syscall.SyscallN(procPostMessage.Addr(), uintptr(w.handle), procWM_PAINT, 0, 0)
	}
	r1, _, _ := syscall.SyscallN(procDefWindowProc.Addr(),
		uintptr(w.handle), uintptr(uMsg), wParam, lParam)
	return r1
}

func (w *Window) showWindow(evtSharedMem *evtMem) {
	//C.window_show(w.handle)
	syscall.SyscallN(procShowWindow.Addr(), uintptr(w.handle), procSW_SHOW)
}

func (w *Window) destroy() {
	//C.window_destroy(w.handle)
	syscall.SyscallN(procDestroyWindow.Addr(), uintptr(w.handle))
}

func (w *Window) pollController() bool {
	readControllerStates := false
	var dwResult int32
	evt := w.evtSharedMem.toControllerEvent()
	evt.clear()
	for i := 0; i < procMAX_CONTROLLERS; i++ {
		state := XINPUT_STATE{}
		// Simply get the state of the controller from XInput.
		r1, _, _ := syscall.SyscallN(procXInputGetState.Addr(),
			uintptr(i), uintptr(unsafe.Pointer(&state)))
		dwResult = int32(r1)
		if dwResult == procERROR_SUCCESS {
			evt.controllerStates[i].buttons = state.Gamepad.WButtons
			evt.controllerStates[i].leftTrigger = state.Gamepad.BLeftTrigger
			evt.controllerStates[i].rightTrigger = state.Gamepad.BRightTrigger
			evt.controllerStates[i].thumbLX = state.Gamepad.SThumbLX
			evt.controllerStates[i].thumbLY = state.Gamepad.SThumbLY
			evt.controllerStates[i].thumbRX = state.Gamepad.SThumbRX
			evt.controllerStates[i].thumbRY = state.Gamepad.SThumbRY
			evt.controllerStates[i].isConnected = 1
			readControllerStates = true
		} else {
			// TODO:  readControllerStates would be true here too, but
			// no need to spam the event if no controllers are available?
			// Probably means the state of the controllers need tracking in C...
			evt.controllerStates[i].isConnected = 0
		}
	}
	return readControllerStates
}

func (w *Window) pollInternal() uint32 {
	// Run the message loop.
	msg := MSG{}
	r1, _, _ := syscall.SyscallN(procPeekMessage.Addr(), uintptr(w.handle), 0, 0, procPM_REMOVE)
	if r1 > 0 {
		syscall.SyscallN(procTranslateMessage.Addr(), uintptr(unsafe.Pointer(&msg)))
		// TODO:  Window resize happens in here, but would be clobbered by &msg which is different
		syscall.SyscallN(procDispatchMessage.Addr(), uintptr(unsafe.Pointer(&msg)))
		w.processMessage(&msg)
		return msg.Message
	} else {
		return 0
	}
}

func procGET_X_LPARAM(param LPARAM) int32 {
	return int32(procLOWORD(param))
}

func procGET_Y_LPARAM(param LPARAM) int32 {
	return int32(procHIWORD(param))
}

func procGET_WHEEL_DELTA_WPARAM(param WPARAM) int32 {
	return int32(procHIWORD(param))
}

func setMouseEvent(evt *mouseEvent, lParam LPARAM, buttonId int32) {
	evt.buttonId = buttonId
	evt.x = procGET_X_LPARAM(lParam)
	evt.y = procGET_Y_LPARAM(lParam)
}

func (w *Window) processMessage(msg *MSG) {
	w.evtSharedMem.SetEventType(msg.Message)
	switch msg.Message {
	case procWM_QUIT:
		fallthrough
	case procWM_DESTROY:
		w.evtSharedMem.SetWriteState(procSHARED_MEM_QUIT)
	case procWM_MOUSEMOVE:
		setMouseEvent(w.evtSharedMem.toMouseEvent(), msg.LParam, -1)
	case procWM_LBUTTONDOWN:
		syscall.SyscallN(procSetCapture.Addr(), uintptr(w.handle))
		setMouseEvent(w.evtSharedMem.toMouseEvent(), msg.LParam, procMOUSE_BUTTON_LEFT)
	case procWM_LBUTTONUP:
		syscall.SyscallN(procReleaseCapture.Addr())
		setMouseEvent(w.evtSharedMem.toMouseEvent(), msg.LParam, procMOUSE_BUTTON_LEFT)
	case procWM_MBUTTONDOWN:
		syscall.SyscallN(procSetCapture.Addr(), uintptr(w.handle))
		setMouseEvent(w.evtSharedMem.toMouseEvent(), msg.LParam, procMOUSE_BUTTON_MIDDLE)
	case procWM_MBUTTONUP:
		syscall.SyscallN(procReleaseCapture.Addr())
		setMouseEvent(w.evtSharedMem.toMouseEvent(), msg.LParam, procMOUSE_BUTTON_MIDDLE)
	case procWM_RBUTTONDOWN:
		syscall.SyscallN(procSetCapture.Addr(), uintptr(w.handle))
		setMouseEvent(w.evtSharedMem.toMouseEvent(), msg.LParam, procMOUSE_BUTTON_RIGHT)
	case procWM_RBUTTONUP:
		syscall.SyscallN(procReleaseCapture.Addr())
		setMouseEvent(w.evtSharedMem.toMouseEvent(), msg.LParam, procMOUSE_BUTTON_RIGHT)
	case procWM_XBUTTONDOWN:
		fallthrough
	case procWM_XBUTTONUP:
		if (msg.WParam & 0x0010000) != 0 {
			setMouseEvent(w.evtSharedMem.toMouseEvent(), msg.LParam, procMOUSE_BUTTON_X1)
		} else if (msg.WParam & 0x0020000) != 0 {
			setMouseEvent(w.evtSharedMem.toMouseEvent(), msg.LParam, procMOUSE_BUTTON_X2)
		}
	case procWM_MOUSEWHEEL:
		evt := w.evtSharedMem.toMouseEvent()
		setMouseEvent(evt, msg.LParam, procMOUSE_WHEEL_VERTICAL)
		evt.delta = procGET_WHEEL_DELTA_WPARAM(msg.WParam)
	case procWM_MOUSEHWHEEL:
		evt := w.evtSharedMem.toMouseEvent()
		setMouseEvent(evt, msg.LParam, procMOUSE_WHEEL_HORIZONTAL)
		evt.delta = procGET_WHEEL_DELTA_WPARAM(msg.WParam)
	case procWM_KEYDOWN:
		fallthrough
	case procWM_SYSKEYDOWN:
		fallthrough
	case procWM_KEYUP:
		fallthrough
	case procWM_SYSKEYUP:
		evt := w.evtSharedMem.toKeyboardEvent()
		switch msg.WParam {
		case procVK_SHIFT:
			scancode := (msg.LParam & 0x00FF0000) >> 16
			r1, _, _ := syscall.SyscallN(procMapVirtualKey.Addr(),
				uintptr(scancode), procMAPVK_VSC_TO_VK_EX)
			evt.key = int32(r1)
		case procVK_CONTROL:
			if (msg.LParam & 0x01000000) != 0 {
				evt.key = procVK_RCONTROL
			} else {
				evt.key = procVK_LCONTROL
			}
		case procVK_MENU:
			if (msg.LParam & 0x01000000) != 0 {
				evt.key = procVK_RMENU
			} else {
				evt.key = procVK_LMENU
			}
		default:
			evt.key = int32(msg.WParam)
		}
	case procUWM_SET_CURSOR:
		{
			c := HCURSOR(procNULL)
			switch msg.WParam {
			case procCURSOR_ARROW:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_ARROW)
			case procCURSOR_IBEAM:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_IBEAM)
			case procCURSOR_WAIT:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_WAIT)
			case procCURSOR_CROSS:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_CROSS)
			case procCURSOR_UPARROW:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_UPARROW)
			case procCURSOR_SIZE_NWSE:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_SIZENWSE)
			case procCURSOR_SIZE_NESW:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_SIZENESW)
			case procCURSOR_SIZE_WE:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_SIZEWE)
			case procCURSOR_SIZE_NS:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_SIZENS)
			case procCURSOR_SIZE_ALL:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_SIZEALL)
			case procCURSOR_NO:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_NO)
			case procCURSOR_HAND:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_HAND)
			case procCURSOR_APP_STARTING:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_APPSTARTING)
			case procCURSOR_HELP:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_HELP)
			case procCURSOR_PIN:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_PIN)
			case procCURSOR_PERSON:
				c, _, _ = syscall.SyscallN(procLoadCursor.Addr(), procNULL, procIDC_PERSON)
			}
			if c != procNULL {
				hCursor := procGCLP_HCURSOR
				syscall.SyscallN(procSetCursor.Addr(), c)
				syscall.SyscallN(procSetClassLongPtr.Addr(), uintptr(w.handle), uintptr(hCursor), c)
			}
		}
	}
}

func (w *Window) poll() {
	if w.pollController() {
		w.processControllerEvent(evtControllerStates)
	}
	evtType := uint32(1)
	for evtType != 0 && !w.evtSharedMem.IsQuit() {
		evtType = w.pollInternal()
		t := asEventType(evtType)
		if w.evtSharedMem.IsResize() {
			t = evtResize
			w.evtSharedMem.ResetHeader()
		} else if w.evtSharedMem.IsMove() {
			t = evtMove
			w.evtSharedMem.ResetHeader()
		} else if w.evtSharedMem.IsActivity() {
			t = evtActivity
			w.evtSharedMem.ResetHeader()
		}
		if t != evtUnknown {
			w.processEvent(t)
		}
	}
}

func (w *Window) setCursorTo(id uintptr) {
	syscall.SyscallN(procPostMessageA.Addr(), uintptr(w.handle),
		procUWM_SET_CURSOR, id, 0)
}

func (w *Window) cursorStandard() {
	//C.window_cursor_standard(w.handle)
	w.setCursorTo(procCURSOR_ARROW)
}

func (w *Window) cursorIbeam() {
	//C.window_cursor_ibeam(w.handle)
	w.setCursorTo(procCURSOR_IBEAM)
}

func (w *Window) cursorSizeAll() {
	//C.window_cursor_size_all(w.handle)
	w.setCursorTo(procCURSOR_SIZE_ALL)
}

func (w *Window) cursorSizeNS() {
	//C.window_cursor_size_ns(w.handle)
	w.setCursorTo(procCURSOR_SIZE_NS)
}

func (w *Window) cursorSizeWE() {
	//C.window_cursor_size_we(w.handle)
	w.setCursorTo(procCURSOR_SIZE_WE)
}

func (w *Window) copyToClipboard(text string) {
	clipboard.Write(clipboard.FmtText, []byte(text))
}

func (w *Window) clipboardContents() string {
	return string(clipboard.Read(clipboard.FmtText))
}

func (w *Window) sizeMM() (int, int, error) {
	r1, _, _ := syscall.SyscallN(procGetDpiForWindow.Addr(), uintptr(w.handle))
	dpi := float64(r1)
	//dpi := float64(C.window_dpi(w.handle))
	mm := dpi / 25.4
	return int(float64(w.width) * mm), int(float64(w.height) * mm), nil
}

func (w *Window) cHandle() unsafe.Pointer   { return w.handle }
func (w *Window) cInstance() unsafe.Pointer { return w.instance }

func (w *Window) focus() {
	//C.window_focus(w.handle)
	syscall.SyscallN(procBringWindowToTop.Addr(), uintptr(w.handle))
	syscall.SyscallN(procSetFocus.Addr(), uintptr(w.handle))
}

func (w *Window) position() (x, y int) {
	//C.window_position(w.handle, (*C.int)(unsafe.Pointer(&x)), (*C.int)(unsafe.Pointer(&y)))
	//return x, y
	wp := WINDOWPLACEMENT{}
	wp.Length = uint32(unsafe.Sizeof(wp))
	r1, _, _ := syscall.SyscallN(procGetWindowPlacement.Addr(),
		uintptr(w.handle), uintptr(unsafe.Pointer(&wp)))
	if r1 == 0 {
		return 0, 0
	}
	return int(wp.RcNormalPosition.Left), int(wp.RcNormalPosition.Top)
}

func (w *Window) setPosition(x, y int) {
	//C.window_set_position(w.handle, C.int(x), C.int(y))
	syscall.SyscallN(procSetWindowPos.Addr(), uintptr(w.handle),
		0, uintptr(unsafe.Pointer(&x)), uintptr(unsafe.Pointer(&y)),
		0, 0, procSWP_NOSIZE|procSWP_NOZORDER)
}

func (w *Window) setSize(width, height int) {
	//C.window_set_size(w.handle, C.int(width), C.int(height))
	syscall.SyscallN(procSetWindowPos.Addr(), uintptr(w.handle),
		0, 0, 0, uintptr(unsafe.Pointer(&width)),
		uintptr(unsafe.Pointer(&height)), procSWP_NOMOVE|procSWP_NOZORDER)
}

func (w *Window) removeBorder() {
	//C.window_remove_border(w.handle)
	gwlStyle := procGWL_STYLE
	r1, _, _ := syscall.SyscallN(procGetWindowLongW.Addr(),
		uintptr(w.handle), uintptr(gwlStyle))
	style := int32(r1)
	style &= ^procWS_CAPTION
	style &= ^procWS_THICKFRAME
	style &= ^procWS_MINIMIZEBOX
	style &= ^procWS_MAXIMIZEBOX
	style &= ^procWS_SYSMENU
	syscall.SyscallN(procSetWindowLongW.Addr(),
		uintptr(w.handle), uintptr(gwlStyle), uintptr(style))
}

func (w *Window) addBorder() {
	//C.window_add_border(w.handle)
	gwlStyle := procGWL_STYLE
	r1, _, _ := syscall.SyscallN(procGetWindowLongW.Addr(),
		uintptr(w.handle), uintptr(gwlStyle))
	style := int32(r1)
	style |= procWS_CAPTION
	style |= procWS_THICKFRAME
	style |= procWS_MINIMIZEBOX
	style |= procWS_MAXIMIZEBOX
	style |= procWS_SYSMENU
	syscall.SyscallN(procSetWindowLongW.Addr(),
		uintptr(w.handle), uintptr(gwlStyle), uintptr(style))
}
