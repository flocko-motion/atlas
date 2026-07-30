// package: core / orchestration
// type:    interface
// job:     the response — a lazy byte stream that declares its own content type
// limits:  core frames the body; the endpoint sets status and copies (-> adapters/endpoints)
//
// A response is a Stream: every HTTP body is ultimately a byte stream — a single
// JSON object, a json-seq run, a cbor-seq run, or a raw blob are all just bytes
// with a content type. So the Stream declares its ContentType and writes its body
// with WriteTo, framing and rendering as it goes (RS/LF for json-seq, concat for
// cbor-seq, the bare value for a single object, a copy for a blob) — lazily, so
// neither a million-row query nor a multi-gigabyte blob is buffered whole. The
// endpoint owns only the HTTP envelope: it sets the status and headers, then
// copies. Core owns the body; the endpoint owns the wrapper.
//
// Internally a concrete stream renders per-item — each item can write itself as
// JSON, CBOR, or raw bytes — but that rendering unit is core's own; it lands with
// the execute-stage engines that produce items, and never crosses to the endpoint,
// which sees only the Stream.
package core

import "io"

// Stream is a response body: a lazy producer of framed bytes plus the content type
// that describes them. Close releases resources (a storage cursor, an open blob)
// and must always run; a mid-stream failure surfaces as the error from WriteTo,
// after the status line is already on the wire.
type Stream interface {
	// ContentType is the MIME type of the bytes WriteTo produces (e.g.
	// application/json, application/json-seq, application/octet-stream).
	ContentType() string
	// WriteTo writes the framed, rendered body to w, pulling and rendering items
	// lazily. It reports bytes written and the first error that stopped it.
	io.WriterTo
	// Close releases the stream's resources.
	Close() error
}
