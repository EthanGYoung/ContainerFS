// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package imgfs

import (
	"syscall"
	"unsafe"

	//"gvisor.googlesource.com/gvisor/pkg/abi/linux"
	"gvisor.googlesource.com/gvisor/pkg/sentry/fs"
	//ktime "gvisor.googlesource.com/gvisor/pkg/sentry/kernel/time"
)

// NulByte is a single NUL byte. It is passed to readlinkat as an empty string.
var NulByte byte = '\x00'

func createLink(fd int, name string, linkName string) error {
	// imgfs doesn't need to createLink
	return nil
}

func readLink(fd int) (string, error) {
	// Buffer sizing copied from os.Readlink.
	for l := 128; ; l *= 2 {
		b := make([]byte, l)
		n, _, errno := syscall.Syscall6(
			syscall.SYS_READLINKAT,
			uintptr(fd),
			uintptr(unsafe.Pointer(&NulByte)), // ""
			uintptr(unsafe.Pointer(&b[0])),
			uintptr(l),
			0, 0)
		if errno != 0 {
			return "", errno
		}
		if n < uintptr(l) {
			return string(b[:n]), nil
		}
	}
}

func unlinkAt(fd int, name string, dir bool) error {
	// imgfs is read-only fs
	return nil
}

func setTimestamps(fd int, ts fs.TimeSpec) error {
	//img fs doesn't need setTimestamps
	return nil
}

func fstatat(fd int, name string, flags int) (syscall.Stat_t, error) {
	var stat syscall.Stat_t
	namePtr, err := syscall.BytePtrFromString(name)
	if err != nil {
		return stat, err
	}
	_, _, errno := syscall.Syscall6(
		syscall.SYS_NEWFSTATAT,
		uintptr(fd),
		uintptr(unsafe.Pointer(namePtr)),
		uintptr(unsafe.Pointer(&stat)),
		uintptr(flags),
		0, 0)
	if errno != 0 {
		return stat, errno
	}
	return stat, nil
}
