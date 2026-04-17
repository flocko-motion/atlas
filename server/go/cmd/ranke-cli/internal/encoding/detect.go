// Package encoding maps filenames to (encoding_class, encoding_format, content_type)
// triples for L0 ingest. Shared by the ingest command so detection stays consistent
// across single-file and directory walks.
//
// The content_type (bulk | conversation | contact | media | record | data) is only
// filled in when the extension makes it unambiguous. For generic text formats
// (.txt, .md, .log) the type is left empty, forcing the caller to pass --type.
package encoding

import (
	"path/filepath"
	"strings"
)

// Detect returns (class, format, typ, true) for a filename based on its extension,
// or ("", "", "", false) if the extension is not recognized. `typ` may be empty
// even when ok is true — the extension is known but the source type is ambiguous.
func Detect(name string) (class, format, typ string, ok bool) {
	lower := strings.ToLower(name)
	// Multi-suffix cases first.
	if strings.HasSuffix(lower, ".tar.gz") {
		return "application", "tar+gzip", "bulk", true
	}
	ext := strings.ToLower(filepath.Ext(name))
	if e, found := table[ext]; found {
		return e.class, e.format, e.typ, true
	}
	return "", "", "", false
}

type entry struct{ class, format, typ string }

var table = map[string]entry{
	// text — communicative acts
	".eml":      {"text", "eml", "conversation"},
	".whatsapp": {"text", "whatsapp", "conversation"},
	".html":     {"text", "html", "conversation"},
	".htm":      {"text", "html", "conversation"},
	// text — contact
	".vcf": {"text", "vcf", "contact"},
	// text — event (calendar entries)
	".ics": {"text", "ics", "event"},
	// text — data (structured)
	".json": {"text", "json", "data"},
	".yaml": {"text", "yaml", "data"},
	".yml":  {"text", "yaml", "data"},
	".xml":  {"text", "xml", "data"},
	".csv":  {"text", "csv", "data"},
	".tsv":  {"text", "tsv", "data"},
	".srt":  {"text", "srt", "data"},
	".vtt":  {"text", "vtt", "data"},
	// text — ambiguous (type left empty, --type required)
	".txt": {"text", "plain", ""},
	".md":  {"text", "markdown", ""},
	".log": {"text", "plain", ""},
	// image — media
	".jpg":  {"image", "jpeg", "media"},
	".jpeg": {"image", "jpeg", "media"},
	".png":  {"image", "png", "media"},
	".gif":  {"image", "gif", "media"},
	".webp": {"image", "webp", "media"},
	".heic": {"image", "heic", "media"},
	".tif":  {"image", "tiff", "media"},
	".tiff": {"image", "tiff", "media"},
	".bmp":  {"image", "bmp", "media"},
	".svg":  {"image", "svg", "media"},
	// audio — media
	".mp3":  {"audio", "mpeg", "media"},
	".wav":  {"audio", "wav", "media"},
	".m4a":  {"audio", "m4a", "media"},
	".aac":  {"audio", "aac", "media"},
	".ogg":  {"audio", "ogg", "media"},
	".flac": {"audio", "flac", "media"},
	".opus": {"audio", "opus", "media"},
	// video — media
	".mp4":  {"video", "mp4", "media"},
	".mov":  {"video", "mov", "media"},
	".mkv":  {"video", "mkv", "media"},
	".avi":  {"video", "avi", "media"},
	".webm": {"video", "webm", "media"},
	// application — bulk (archives to unpack)
	".zip": {"application", "zip", "bulk"},
	".tar": {"application", "tar", "bulk"},
	".tgz": {"application", "tar+gzip", "bulk"},
	".gz":  {"application", "gzip", "bulk"},
	".7z":  {"application", "7z", "bulk"},
	".rar": {"application", "rar", "bulk"},
	// application — data (documents)
	".pdf":  {"application", "pdf", "data"},
	".doc":  {"application", "msword", "data"},
	".docx": {"application", "docx", "data"},
	".xls":  {"application", "msexcel", "data"},
	".xlsx": {"application", "xlsx", "data"},
	".ppt":  {"application", "mspowerpoint", "data"},
	".pptx": {"application", "pptx", "data"},
}
