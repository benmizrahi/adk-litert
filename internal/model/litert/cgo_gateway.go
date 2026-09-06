package litert

/*
#include "csrc/litert_lm_c_api.h"
*/
import "C"

import (
	"sync"
	"unsafe"
)

type tokenHandlerFunc func(token string, isDone bool) bool

var (
	callbackMu sync.RWMutex
	callbacks  = make(map[uintptr]tokenHandlerFunc)
	nextID     uintptr
)

func registerCallback(fn tokenHandlerFunc) uintptr {
	callbackMu.Lock()
	defer callbackMu.Unlock()
	nextID++
	id := nextID
	callbacks[id] = fn
	return id
}

func unregisterCallback(id uintptr) {
	callbackMu.Lock()
	defer callbackMu.Unlock()
	delete(callbacks, id)
}

//export goTokenCallbackGatewayLitertInternal
func goTokenCallbackGatewayLitertInternal(token *C.char, isDone C.bool, userData unsafe.Pointer) C.bool {
	if userData == nil {
		return C.bool(false)
	}
	id := *(*uintptr)(userData)
	callbackMu.RLock()
	fn, ok := callbacks[id]
	callbackMu.RUnlock()

	if !ok || fn == nil {
		return C.bool(false)
	}

	goStr := C.GoString(token)
	cont := fn(goStr, bool(isDone))
	return C.bool(cont)
}
