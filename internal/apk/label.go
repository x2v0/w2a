package apk

import (
	"bytes"
	"fmt"
)

// SetAppLabelInPlace заменяет строку oldName на newName прямо в raw
// resources.arsc, без parse/write. Работает если len(newName) <= len(oldName):
// в UTF-8 пул пишем новый префикс длины, байты, и добиваем нулями до
// исходной длины. Все offsets и entry остаются валидными.
//
// Применяется при WTA_SKIP_MANIFEST=1 (манифест не трогаем, label — из arsc).
func (a *APK) SetAppLabelInPlace(oldName, newName string) (int, error) {
	if oldName == newName {
		return 0, nil
	}
	if len(newName) > len(oldName) {
		return 0, fmt.Errorf("apk: newName %q longer than oldName %q (in-place)", newName, oldName)
	}
	if len(oldName) >= 128 || len(newName) >= 128 {
		return 0, fmt.Errorf("apk: only short ASCII names supported (len<128)")
	}

	e := a.Archive.Get("resources.arsc")
	if e == nil {
		return 0, fmt.Errorf("apk: no resources.arsc")
	}

	// UTF-8 string pool entry: <u8 len_chars><u8 len_bytes><bytes>\0
	// For ASCII: len_chars == len_bytes < 128, both in one byte.
	oldNeedle := append([]byte{byte(len(oldName)), byte(len(oldName))}, []byte(oldName)...)
	oldNeedle = append(oldNeedle, 0x00)

	newTail := append([]byte{byte(len(newName)), byte(len(newName))}, []byte(newName)...)
	newTail = append(newTail, 0x00)
	// Pad with zeros up to old length so all subsequent offsets stay valid.
	for len(newTail) < len(oldNeedle) {
		newTail = append(newTail, 0x00)
	}

	data := e.Data
	count := 0
	pos := 0
	for {
		idx := bytes.Index(data[pos:], oldNeedle)
		if idx < 0 {
			break
		}
		abs := pos + idx
		copy(data[abs:], newTail)
		count++
		pos = abs + len(oldNeedle)
	}

	if count == 0 {
		return 0, fmt.Errorf("apk: %q not found in resources.arsc", oldName)
	}
	a.Archive.Replace("resources.arsc", data)
	return count, nil
}
