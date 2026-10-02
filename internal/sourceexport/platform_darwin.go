// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
//go:build darwin && cgo

package sourceexport

/*
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <sys/xattr.h>
#include <fcntl.h>
#include <unistd.h>
static int oc_publish_tree(int fd,const char *from, const char *to) {
 if (renameatx_np(fd,from,fd,to,RENAME_EXCL)==0) return 0;
 return errno;
}
static int oc_publish_file(int fd,const char *from,const char *to,int exclusive) {
 if(exclusive) {if(linkat(fd,from,fd,to,0)==0)return 0;}
 else {if(renameat(fd,from,fd,to)==0)return 0;}
 return errno;
}
static int oc_directory_path(int fd,char *path) {return fcntl(fd,F_GETPATH,path);}
static int oc_alias(const char *path) {
 unsigned char info[32];
 ssize_t n=getxattr(path,"com.apple.FinderInfo",info,sizeof(info),0,XATTR_NOFOLLOW);
 if(n<0) return errno==ENOATTR ? 0 : -errno;
 if(n!=32) return -EINVAL;
 return (info[8]&0x80)!=0;
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
	var path [4096]C.char
	if C.oc_directory_path(C.int(parent.Fd()), &path[0]) != 0 {
		return "", fmt.Errorf("cannot locate owned staging directory")
	}
	return C.GoString(&path[0]), nil
}
func checkAlias(path string) error {
	p := C.CString(path)
	defer C.free(unsafe.Pointer(p))
	n := C.oc_alias(p)
	if n != 0 {
		return fmt.Errorf("E-EXPORT-TARGET: Finder alias or unreadable Finder metadata (%d)", n)
	}
	return nil
}
