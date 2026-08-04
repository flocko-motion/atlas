// package: core / orchestration
// type:    interface
// job:     the response — a lazy byte stream that declares its own content type
// limits:  core frames the body; the endpoint sets status and copies (-> adapters/endpoints)
//
// Every response body — one JSON object, a json-seq or cbor-seq run, a raw blob — is
// bytes with a content type, so a Stream declares its ContentType and frames its body as
// it writes, lazily: neither a million rows nor a gigabyte blob is buffered whole.
//
// Core owns the body, the endpoint the envelope. Per-item rendering is core's own and
// never crosses to the endpoint, which sees only the Stream.
package core

import "io"

// Stream is a response body: framed bytes, lazily produced, plus their content type.
// Close must always run; a mid-stream failure surfaces from WriteTo, after the status.
type Stream interface {
	// ContentType is the MIME type of the bytes WriteTo produces.
	ContentType() string
	// WriteTo writes the framed body to w lazily, reporting bytes and the first error.
	io.WriterTo
	// Close releases the stream's resources.
	Close() error
}
