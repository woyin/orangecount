// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
//go:build linux && cgo

package sourceexport

/*
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <fcntl.h>
#include <unistd.h>
static int oc_publish_tree(int fd,const char *from,const char *to) {
 if(renameat2(fd,from,fd,to,RENAME_NOREPLACE)==0)return 0;
 return errno;
}
static int oc_publish_file(int fd,const char *from,const char *to,int exclusive) {
 if(exclusive) {if(linkat(fd,from,fd,to,0)==0)return 0;}
 else {if(renameat(fd,from,fd,to)==0)return 0;}
 return errno;
}
*/
import "C"
import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func publishNewTree(parent *os.File, from, to string) error {
	a, b := C.CString(from), C.CString(to)
	defer C.free(unsafe.Pointer(a))
	defer C.free(unsafe.Pointer(b))
	if code := C.oc_publish_tree(C.int(parent.Fd()), a, b); code != 0 {
		return syscall.Errno(code)
	}
	return nil
}
func publishFile(parent *os.File, from, to string, exclusive bool) error {
	a, b := C.CString(from), C.CString(to)
	defer C.free(unsafe.Pointer(a))
	defer C.free(unsafe.Pointer(b))
	flag := C.int(0)
	if exclusive {
		flag = 1
	}
	if code := C.oc_publish_file(C.int(parent.Fd()), a, b, flag); code != 0 {
		return syscall.Errno(code)
	}
	return nil
}
func pathForDirectory(parent *os.File, initial string) (string, error) {
	return os.Readlink(fmt.Sprintf("/proc/self/fd/%d", parent.Fd()))
}
func checkAlias(path string) error { return nil }
