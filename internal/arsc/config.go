package arsc

import (
	"encoding/binary"
	"fmt"
)

// ResTableConfig mirrors the AOSP struct. We only decode the fields we need
// for matching (density, locale, sdkVersion, etc.); the rest is kept as raw
// bytes for re-serialisation.
type ResTableConfig struct {
	Raw []byte // full config bytes

	Size         uint32
	IMSI         uint32
	Locale       uint32
	ScreenType   uint32
	Input        uint32
	ScreenSize   uint32
	Version      uint32
	ScreenConfig uint32
	ScreenSizeDP uint32
	LocaleScript [4]byte
	LocaleVariant [8]byte
	ScreenConfig2 uint32
}

// ParseConfig decodes a ResTable_config of `size` bytes.
func ParseConfig(buf []byte) *ResTableConfig {
	if len(buf) < 4 {
		return nil
	}
	size := binary.LittleEndian.Uint32(buf)
	if int(size) > len(buf) {
		size = uint32(len(buf))
	}
	c := &ResTableConfig{Size: size}
	c.Raw = append([]byte(nil), buf[:size]...)

	read := func(off int) uint32 {
		if off+4 > len(c.Raw) {
			return 0
		}
		return binary.LittleEndian.Uint32(c.Raw[off:])
	}

	c.IMSI = read(4)
	c.Locale = read(8)
	c.ScreenType = read(12)
	c.Input = read(16)
	c.ScreenSize = read(20)
	c.Version = read(24)
	c.ScreenConfig = read(28)
	c.ScreenSizeDP = read(32)

	if size >= 44 {
		copy(c.LocaleScript[:], c.Raw[36:40])
		copy(c.LocaleVariant[:], c.Raw[40:48])
	}
	if size >= 52 {
		c.ScreenConfig2 = read(48)
	}
	return c
}

// Density returns the density DPI (0 = default).
func (c *ResTableConfig) Density() uint16 {
	if c == nil {
		return 0
	}
	return uint16(c.ScreenType >> 16)
}

// SDkVersion returns the minimum SDK version this config applies to.
func (c *ResTableConfig) SdkVersion() uint16 {
	if c == nil {
		return 0
	}
	return uint16(c.Version & 0xFFFF)
}

// Language returns the ISO-639-1 language code, or "" for default.
func (c *ResTableConfig) Language() string {
	if c == nil || c.Locale&0xFFFF == 0 {
		return ""
	}
	l := c.Locale & 0xFFFF
	return fmt.Sprintf("%c%c", byte(l), byte(l>>8))
}

// Country returns the ISO-3166 country code, or "" for default.
func (c *ResTableConfig) Country() string {
	if c == nil || c.Locale>>16 == 0 {
		return ""
	}
	l := c.Locale >> 16
	return fmt.Sprintf("%c%c", byte(l), byte(l>>8))
}

// IsDefault reports whether this config is the "default" (no qualifiers).
func (c *ResTableConfig) IsDefault() bool {
	if c == nil {
		return true
	}
	return c.IMSI == 0 && c.Locale == 0 && c.ScreenType == 0 &&
		c.Input == 0 && c.ScreenSize == 0 && c.Version == 0 &&
		c.ScreenConfig == 0 && c.ScreenSizeDP == 0 && c.ScreenConfig2 == 0
}

// String returns a human-readable config summary.
func (c *ResTableConfig) String() string {
	if c == nil || c.IsDefault() {
		return "default"
	}
	out := ""
	if lang := c.Language(); lang != "" {
		out += lang
		if cc := c.Country(); cc != "" {
			out += "-" + cc
		}
		out += " "
	}
	if d := c.Density(); d != 0 {
		out += fmt.Sprintf("%ddpi ", d)
	}
	if sdk := c.SdkVersion(); sdk != 0 {
		out += fmt.Sprintf("v%d ", sdk)
	}
	return out
}
