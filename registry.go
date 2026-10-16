//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

const (
	hkcu = syscall.Handle(syscall.HKEY_CURRENT_USER)
	hklm = syscall.Handle(syscall.HKEY_LOCAL_MACHINE)

	keyRead       = 0x20019
	keyAllAccess  = 0xF003F
	keyWow64_64   = 0x0100
	keyWow64_32   = 0x0200
	regSZ         = 1
	regExpandSZ   = 2
	regDWORDType  = 4
	errNoMoreItem = 259
)

func regOpen(root syscall.Handle, path string, access uint32) (syscall.Handle, bool) {
	var h syscall.Handle
	if err := syscall.RegOpenKeyEx(root, u16(path), 0, access, &h); err != nil {
		return 0, false
	}
	return h, true
}

// regString reads a REG_SZ / REG_EXPAND_SZ value ("" name = default value).
func regString(root syscall.Handle, path, name string, view uint32) (string, bool) {
	h, ok := regOpen(root, path, keyRead|view)
	if !ok {
		return "", false
	}
	defer syscall.RegCloseKey(h)
	return regStringH(h, name)
}

func regStringH(h syscall.Handle, name string) (string, bool) {
	var typ, n uint32
	var np *uint16
	if name != "" {
		np = u16(name)
	}
	if err := syscall.RegQueryValueEx(h, np, nil, &typ, nil, &n); err != nil {
		return "", false
	}
	if typ != regSZ && typ != regExpandSZ {
		return "", false
	}
	if n == 0 {
		return "", true
	}
	buf := make([]uint16, n/2+1)
	if err := syscall.RegQueryValueEx(h, np, nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &n); err != nil {
		return "", false
	}
	s := syscall.UTF16ToString(buf)
	if typ == regExpandSZ {
		s = expandEnv(s)
	}
	return s, true
}

func regDWORD(root syscall.Handle, path, name string) (uint32, bool) {
	h, ok := regOpen(root, path, keyRead)
	if !ok {
		return 0, false
	}
	defer syscall.RegCloseKey(h)
	var typ, v uint32
	n := uint32(4)
	if err := syscall.RegQueryValueEx(h, u16(name), nil, &typ, (*byte)(unsafe.Pointer(&v)), &n); err != nil || typ != regDWORDType {
		return 0, false
	}
	return v, true
}

func regSubkeys(root syscall.Handle, path string, view uint32) []string {
	h, ok := regOpen(root, path, keyRead|view)
	if !ok {
		return nil
	}
	defer syscall.RegCloseKey(h)
	var out []string
	for i := uint32(0); ; i++ {
		buf := make([]uint16, 256)
		n := uint32(len(buf))
		if err := syscall.RegEnumKeyEx(h, i, &buf[0], &n, nil, nil, nil, nil); err != nil {
			break
		}
		out = append(out, syscall.UTF16ToString(buf[:n]))
	}
	return out
}

// regValues lists the string values of a key as name -> data.
func regValues(root syscall.Handle, path string) map[string]string {
	out := map[string]string{}
	h, ok := regOpen(root, path, keyRead)
	if !ok {
		return out
	}
	defer syscall.RegCloseKey(h)
	for i := uint32(0); ; i++ {
		name := make([]uint16, 16384)
		nameLen := uint32(len(name))
		data := make([]uint16, 1024)
		dataLen := uint32(len(data) * 2)
		var typ uint32
		r, _, _ := pRegEnumValueW.Call(uintptr(h), uintptr(i), uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&nameLen)),
			0, uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&dataLen)))
		if r == errNoMoreItem {
			break
		}
		if r != 0 {
			continue
		}
		if typ == regSZ || typ == regExpandSZ {
			out[syscall.UTF16ToString(name[:nameLen])] = syscall.UTF16ToString(data[:dataLen/2])
		}
	}
	return out
}

func regCreate(root syscall.Handle, path string) (syscall.Handle, bool) {
	var h syscall.Handle
	r, _, _ := pRegCreateKeyExW.Call(uintptr(root), u16p(path), 0, 0, 0, keyAllAccess, 0, uintptr(unsafe.Pointer(&h)), 0)
	return h, r == 0
}

func regSetString(root syscall.Handle, path, name, value string) bool {
	h, ok := regCreate(root, path)
	if !ok {
		return false
	}
	defer syscall.RegCloseKey(h)
	data, _ := syscall.UTF16FromString(value)
	var np uintptr
	if name != "" {
		np = u16p(name)
	}
	r, _, _ := pRegSetValueExW.Call(uintptr(h), np, 0, regSZ, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)*2))
	return r == 0
}

func regDeleteTree(root syscall.Handle, path string) {
	pRegDeleteTreeW.Call(uintptr(root), u16p(path))
	// RegDeleteTree empties the key; delete the key itself through its parent.
	parent, leaf := splitKey(path)
	if h, ok := regOpen(root, parent, keyAllAccess); ok {
		pRegDeleteTreeW.Call(uintptr(h), u16p(leaf))
		syscall.RegCloseKey(h)
	}
}

func regDeleteValue(root syscall.Handle, path, name string) {
	if h, ok := regOpen(root, path, keyAllAccess); ok {
		pRegDeleteValueW.Call(uintptr(h), u16p(name))
		syscall.RegCloseKey(h)
	}
}

func splitKey(path string) (string, string) {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '\\' {
			return path[:i], path[i+1:]
		}
	}
	return "", path
}

// regValueNames lists the value names of a key, whatever their type.
func regValueNames(root syscall.Handle, path string) []string {
	h, ok := regOpen(root, path, keyRead)
	if !ok {
		return nil
	}
	defer syscall.RegCloseKey(h)
	var out []string
	for i := uint32(0); ; i++ {
		name := make([]uint16, 16384)
		n := uint32(len(name))
		r, _, _ := pRegEnumValueW.Call(uintptr(h), uintptr(i), uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&n)), 0, 0, 0, 0)
		if r == errNoMoreItem {
			break
		}
		if r == 0 {
			out = append(out, syscall.UTF16ToString(name[:n]))
		}
	}
	return out
}
