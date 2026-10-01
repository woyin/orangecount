// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

/*
#include <stdlib.h>
*/
import "C"
import "unsafe"

// OC_FreeString releases exactly one owned response returned by the bridge.
//
//export OC_FreeString
func OC_FreeString(ptr *C.char) {
	if ptr != nil {
		C.free(unsafe.Pointer(ptr))
	}
}

func main() {}
