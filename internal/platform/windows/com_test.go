//go:build windows

package windows

import (
	"bytes"
	"log"
	"runtime"
	"strings"
	"testing"
	"unsafe"
)

var clsidFromString = ole32.NewProc("CLSIDFromString")

// apartmentThread keeps the test on one thread, a single-threaded COM apartment as the main thread is, until the function it returns is called.
func apartmentThread() func() {
	runtime.LockOSThread()
	const coinitApartmentThreaded = 2
	coInitializeEx.Call(0, coinitApartmentThreaded)
	return runtime.UnlockOSThread
}

// TestTaskbarListIDs holds the taskbar list's class and interface IDs to the ones shobjidl_core.h documents, as Windows parses them: with a wrong IID, CoCreateInstance answers E_NOINTERFACE and taskbar buttons show no badge or progress.
func TestTaskbarListIDs(t *testing.T) {
	for _, test := range []struct {
		name       string
		id         *guid
		documented string
	}{
		{"CLSID_TaskbarList", &clsidTaskbarList, "{56FDF344-FD6D-11D0-958A-006097C9A090}"},
		{"IID_ITaskbarList3", &iidTaskbarList3, "{EA1AFB91-9E28-4B86-90E9-9E9F8A5EEFAF}"},
	} {
		var want guid
		if hr, _, _ := clsidFromString.Call(uintptr(unsafe.Pointer(wide(test.documented))), uintptr(unsafe.Pointer(&want))); failed(hr) {
			t.Fatalf("CLSIDFromString(%s): HRESULT %#x", test.documented, uint32(hr))
		}
		if *test.id != want {
			t.Errorf("%s is %s, want %s", test.name, formatGUID(*test.id), test.documented)
		}
	}
}

// TestTaskbarListIsMade checks that the shell makes its taskbar list as the taskbar asks for it, the class with that interface.
func TestTaskbarListIsMade(t *testing.T) {
	defer apartmentThread()()
	var list *comObject
	if hr, _, _ := coCreateInstance.Call(uintptr(unsafe.Pointer(&clsidTaskbarList)), 0, clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidTaskbarList3)), uintptr(unsafe.Pointer(&list))); failed(hr) {
		t.Fatalf("CoCreateInstance(TaskbarList, ITaskbarList3): HRESULT %#x", uint32(hr))
	}
	list.release()
}

// TestTaskbarIsMadeAgainAfterAFailure checks that a failure to make the taskbar list is logged with its HRESULT and not kept: the next call makes the list.
func TestTaskbarIsMadeAgainAfterAFailure(t *testing.T) {
	defer apartmentThread()()
	var logged bytes.Buffer
	defer log.SetOutput(log.Writer())
	log.SetOutput(&logged)
	documented := iidTaskbarList3
	defer func() { iidTaskbarList3 = documented }()
	// The taskbar list is no drop target.
	iidTaskbarList3 = iidDropTarget
	const eNoInterface = 0x80004002
	if list := taskbar(); list != nil {
		t.Fatal("the taskbar list was made with an interface it does not have")
	}
	if !strings.Contains(logged.String(), "HRESULT 0x80004002") {
		t.Fatalf("the failure logged %q, not E_NOINTERFACE", logged.String())
	}
	iidTaskbarList3 = documented
	list := taskbar()
	defer func() { taskbarList, taskbarFailure = nil, 0 }()
	switch {
	case list != nil:
		list.release()
	case taskbarFailure == eNoInterface:
		t.Fatalf("the taskbar list was not made again after a failure: %s", logged.String())
	default:
		// HrInit answers E_NOTIMPL in a session without a taskbar, such as a service's.
		t.Logf("the taskbar list was made again, and failed further on: %s", logged.String())
	}
}
