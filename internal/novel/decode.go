// Package novel splits plain-text novels into chapters and builds EPUBs.
package novel

import (
	"slices"
	"strings"

	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/unicode/utf32"
)

// detectSize is how many leading bytes are sniffed.
const detectSize = 64 << 10

// fallbacks are always offered after the detected encodings.
var fallbacks = []string{"UTF-8", "Big5", "GB18030", "UTF-16LE"}

// Encodings returns candidate encodings of b, the most likely first.
func Encodings(b []byte) []string {
	var out, seen []string
	add := func(name string) {
		key := canonical(name)
		if key != "" && !slices.Contains(seen, key) {
			seen = append(seen, key)
			out = append(out, name)
		}
	}

	if rs, err := chardet.NewTextDetector().DetectAll(b[:min(len(b), detectSize)]); err == nil {
		for _, r := range rs {
			add(r.Charset)
		}
	}
	for _, name := range fallbacks {
		add(name)
	}
	return out
}

// canonical names the encoding name refers to, so aliases such as
// "GB-18030" and "GB18030" match; "" if unknown.
func canonical(name string) string {
	enc := lookup(name)
	if enc == nil {
		return ""
	}
	if n, err := htmlindex.Name(enc); err == nil {
		return n
	}
	return strings.ToUpper(name)
}

func lookup(name string) encoding.Encoding {
	switch strings.ToUpper(name) {
	case "GB-18030":
		name = "gb18030"
	case "UTF-32BE":
		return utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM)
	case "UTF-32LE":
		return utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM)
	}
	enc, err := htmlindex.Get(name)
	if err != nil {
		return nil
	}
	return enc
}

// Decode decodes b as the named encoding, replacing invalid bytes and
// dropping a leading BOM.
func Decode(b []byte, name string) string {
	enc := lookup(name)
	if enc == nil {
		enc = encoding.Nop
	}
	out, err := enc.NewDecoder().Bytes(b)
	if err != nil {
		out = b
	}
	return strings.TrimPrefix(strings.ToValidUTF8(string(out), "�"), "\uFEFF")
}
