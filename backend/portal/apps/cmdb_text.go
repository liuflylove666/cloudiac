// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

func normalizeCmdbText(value string) string {
	if value == "" || !mayBeCmdbMojibake(value) {
		return value
	}
	decoded, err := charmap.Windows1252.NewEncoder().Bytes([]byte(value))
	if err != nil || !utf8.Valid(decoded) {
		return value
	}
	fixed := string(decoded)
	if fixed == value || !containsCmdbCJK(fixed) {
		return value
	}
	return fixed
}

func mayBeCmdbMojibake(value string) bool {
	for _, marker := range []string{"Ã", "Â", "Ä", "Å", "Æ", "Ç", "È", "É", "è", "ä", "å", "é", "æ", "ç", "µ", "º", "„", "§"} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func containsCmdbCJK(value string) bool {
	for _, r := range value {
		if (r >= '\u4e00' && r <= '\u9fff') || (r >= '\u3400' && r <= '\u4dbf') {
			return true
		}
	}
	return false
}
