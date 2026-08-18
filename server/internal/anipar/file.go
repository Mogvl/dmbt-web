package anipar

import (
	"regexp"
	"strings"
)

// FileExtensions mirrors the FileExtensions set in file.ts.
var FileExtensions = map[string]bool{
	"3GP":  true,
	"AVI":  true,
	"DIVX": true,
	"FLV":  true,
	"M2TS": true,
	"MKV":  true,
	"MOV":  true,
	"MP4":  true,
	"MPG":  true,
	"OGM":  true,
	"RM":   true,
	"RMVB": true,
	"TS":   true,
	"WEBM": true,
	"WMV":  true,
}

var fileExtensionRE = regexp.MustCompile(`\.([^./\s]+)\s*$`)

// parseFileExtension extracts a known video file extension from the tail of
// the title, mirroring parseFileExtension in file.ts. It returns the title
// without the extension and the uppercased extension ("" when absent).
func parseFileExtension(title string) (string, string) {
	m := fileExtensionRE.FindStringSubmatchIndex(title)
	if m == nil {
		return title, ""
	}
	extension := strings.ToUpper(title[m[2]:m[3]])
	if !FileExtensions[extension] {
		return title, ""
	}
	return jsTrimRight(title[:m[0]]), extension
}
