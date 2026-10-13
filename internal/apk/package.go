package apk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// SetPackageInPlace заменяет applicationId (package="...") и связанные
// authorities в raw AndroidManifest.xml (AXML).
//
// Работает in-place: длина строки не меняется (padding нулевыми chars).
// Классы НЕ трогаются — они fully-qualified (com.webtoapp.ui.shell.*) и
// должны совпадать с dex.
//
// Заменяются:
//
//	package="com.webtoapp"                    → package="com.w2a"
//	authorities="com.webtoapp.fileprovider"   → authorities="com.w2a.fileprovider"
//	authorities="com.webtoapp.firebaseinitprovider"
//	authorities="com.webtoapp.androidx-startup"
//
// Ограничение: len(newPkg) ≤ len(oldPkg).
func (a *APK) SetPackageInPlace(oldPkg, newPkg string) (int, error) {
	if oldPkg == newPkg {
		return 0, nil
	}
	if len(newPkg) > len(oldPkg) {
		return 0, fmt.Errorf("apk: newPkg %q longer than %q (in-place)", newPkg, oldPkg)
	}
	e := a.Archive.Get("AndroidManifest.xml")
	if e == nil {
		return 0, fmt.Errorf("apk: no AndroidManifest.xml")
	}
	data := e.Data

	// Определить кодировку pool (flags at offset 8+16=24).
	if len(data) < 26 {
		return 0, fmt.Errorf("apk: manifest too small")
	}
	flags := binary.LittleEndian.Uint16(data[24:26])
	isUTF8 := flags&0x0100 != 0

	targets := []string{
		oldPkg,
		oldPkg + ".fileprovider",
		oldPkg + ".firebaseinitprovider",
		oldPkg + ".androidx-startup",
	}

	count := 0
	for _, old := range targets {
		suffix := old[len(oldPkg):]
		new := newPkg + suffix
		if isUTF8 {
			count += replaceUTF8InPlace(data, old, new)
		} else {
			count += replaceUTF16InPlace(data, old, new)
		}
	}

	if count == 0 {
		return 0, fmt.Errorf("apk: %q not found in AndroidManifest.xml", oldPkg)
	}
	a.Archive.Replace("AndroidManifest.xml", data)
	return count, nil
}

func replaceUTF8InPlace(data []byte, old, new string) int {
	if len(new) > len(old) || len(old) >= 128 {
		return 0
	}
	needle := append([]byte{byte(len(old)), byte(len(old))}, []byte(old)...)
	needle = append(needle, 0x00)
	tail := append([]byte{byte(len(new)), byte(len(new))}, []byte(new)...)
	tail = append(tail, 0x00)
	for len(tail) < len(needle) {
		tail = append(tail, 0x00)
	}
	return replaceInPlace(data, needle, tail)
}

func replaceUTF16InPlace(data []byte, old, new string) int {
	if len(new) > len(old) {
		return 0
	}
	oldU16 := utf16.Encode([]rune(old))
	newU16 := utf16.Encode([]rune(new))

	needle := make([]byte, 0, 2+len(oldU16)*2+2)
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], uint16(len(oldU16)))
	needle = append(needle, b[:]...)
	for _, u := range oldU16 {
		binary.LittleEndian.PutUint16(b[:], u)
		needle = append(needle, b[:]...)
	}
	needle = append(needle, 0x00, 0x00)

	// tail — той же длины, с новыми chars + padding нулями.
	tail := make([]byte, len(needle))
	binary.LittleEndian.PutUint16(tail[0:2], uint16(len(newU16)))
	for i, u := range newU16 {
		binary.LittleEndian.PutUint16(tail[2+i*2:], u)
	}
	// всё остальное — нули (terminator + padding)

	return replaceInPlace(data, needle, tail)
}

func replaceInPlace(data, needle, replacement []byte) int {
	if len(replacement) != len(needle) {
		return 0
	}
	count := 0
	pos := 0
	for {
		idx := bytes.Index(data[pos:], needle)
		if idx < 0 {
			break
		}
		abs := pos + idx
		copy(data[abs:], replacement)
		count++
		pos = abs + len(needle)
	}
	return count
}
