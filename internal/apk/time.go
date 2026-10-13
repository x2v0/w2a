package apk

import "time"

// zipMsDosToTime converts an MS-DOS packed date/time back to time.Time.
func zipMsDosToTime(dt uint32) time.Time {
	year := int(dt>>25) + 1980
	month := time.Month((dt >> 21) & 0x0F)
	day := int((dt >> 16) & 0x1F)
	hour := int((dt >> 11) & 0x1F)
	min := int((dt >> 5) & 0x3F)
	sec := int(dt&0x1F) * 2
	if month < 1 || month > 12 {
		month = 1
	}
	if day < 1 {
		day = 1
	}
	return time.Date(year, month, day, hour, min, sec, 0, time.UTC)
}
