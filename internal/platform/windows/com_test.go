//go:build windows

package windows

import (
	"runtime"
	"testing"
	"unsafe"
)

var clsidFromString = ole32.NewProc("CLSIDFromString")

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
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	const coinitApartmentThreaded = 2
	coInitializeEx.Call(0, coinitApartmentThreaded)
	var list *comObject
	if hr, _, _ := coCreateInstance.Call(uintptr(unsafe.Pointer(&clsidTaskbarList)), 0, clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidTaskbarList3)), uintptr(unsafe.Pointer(&list))); failed(hr) {
		t.Fatalf("CoCreateInstance(TaskbarList, ITaskbarList3): HRESULT %#x", uint32(hr))
	}
	list.release()
}
