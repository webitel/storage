package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// See: http://en.wikipedia.org/wiki/Binary_prefix
const (
	// Decimal

	KB = 1000
	MB = 1000 * KB
	GB = 1000 * MB
	TB = 1000 * GB
	PB = 1000 * TB

	// Binary

	KiB = 1024
	MiB = 1024 * KiB
	GiB = 1024 * MiB
	TiB = 1024 * GiB
	PiB = 1024 * TiB
)

type unitMap map[byte]int64

var (
	binaryMap   = unitMap{'k': KiB, 'm': MiB, 'g': GiB, 't': TiB, 'p': PiB}
	binaryAbbrs = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"}
)

func FromHumanSize(size string) (int64, error) {
	return parseSize(size, binaryMap)
}

func getSizeAndUnit(size float64, base float64, _map []string) (float64, string) {
	i := 0
	unitsLimit := len(_map) - 1
	for size >= base && i < unitsLimit {
		size = size / base
		i++
	}
	return size, _map[i]
}

// CustomSize returns a human-readable approximation of a size
// using custom format.
func CustomSize(format string, size float64, base float64, _map []string) string {
	size, unit := getSizeAndUnit(size, base, _map)
	return fmt.Sprintf(format, size, unit)
}

func BytesSize(size float64) string {
	return CustomSize("%.4g%s", size, 1024.0, binaryAbbrs)
}

// Parses the human-readable size string into the amount it represents.
func parseSize(sizeStr string, uMap unitMap) (int64, error) {
	// TODO: rewrite to use strings.Cut if there's a space
	// once Go < 1.18 is deprecated.
	sep := strings.LastIndexAny(sizeStr, "01234567890. ")
	if sep == -1 {
		// There should be at least a digit.
		return -1, fmt.Errorf("invalid size: '%s'", sizeStr)
	}
	var num, sfx string
	if sizeStr[sep] != ' ' {
		num = sizeStr[:sep+1]
		sfx = sizeStr[sep+1:]
	} else {
		// Omit the space separator.
		num = sizeStr[:sep]
		sfx = sizeStr[sep+1:]
	}

	size, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return -1, err
	}
	// Backward compatibility: reject negative sizes.
	if size < 0 {
		return -1, fmt.Errorf("invalid size: '%s'", sizeStr)
	}

	if len(sfx) == 0 {
		return int64(size), nil
	}

	// Process the suffix.

	if len(sfx) > 3 { // Too long.
		goto badSuffix
	}
	sfx = strings.ToLower(sfx)
	// Trivial case: b suffix.
	if sfx[0] == 'b' {
		if len(sfx) > 1 { // no extra characters allowed after b.
			goto badSuffix
		}
		return int64(size), nil
	}
	// A suffix from the map.
	if mul, ok := uMap[sfx[0]]; ok {
		size *= float64(mul)
	} else {
		goto badSuffix
	}

	// The suffix may have extra "b" or "ib" (e.g. KiB or MB).
	switch {
	case len(sfx) == 2 && sfx[1] != 'b':
		goto badSuffix
	case len(sfx) == 3 && sfx[1:] != "ib":
		goto badSuffix
	}

	return int64(size), nil

badSuffix:
	return -1, fmt.Errorf("invalid suffix: '%s'", sfx)
}
