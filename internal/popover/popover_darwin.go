package popover

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
void rlb_panel_update(const char *json);
void rlb_panel_toggle(void);
void rlb_panel_close(void);
void rlb_settings_show(void);
*/
import "C"

import (
	"encoding/json"
	"unsafe"
)

const Available = true

var Events = make(chan string, 64)

func Update(model Model) {
	data, err := json.Marshal(model)
	if err != nil {
		panic(err)
	}
	text := C.CString(string(data))
	defer C.free(unsafe.Pointer(text))
	C.rlb_panel_update(text)
}

func Toggle()       { C.rlb_panel_toggle() }
func Close()        { C.rlb_panel_close() }
func ShowSettings() { C.rlb_settings_show() }

//export rlbPanelEvent
func rlbPanelEvent(action *C.char) {
	select {
	case Events <- C.GoString(action):
	default:
	}
}
