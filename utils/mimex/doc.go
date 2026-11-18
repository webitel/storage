// Package mimex provides utilities for MIME type detection, sniffing, and validation
// built on top of the github.com/gabriel-vasile/mimetype library.
//
// It handles media type normalization, parameter stripping (via mime.ParseMediaType),
// and identification of generic or fallback binary streams (such as application/octet-stream).
//
// # Matching and Hierarchy
//
// The package introduces hierarchical MIME type matching through [Matches], which traverses
// the parent-child relationships and aliases of registered MIME types. This ensures that
// specialized subtypes (e.g., audio/wave or audio/x-wav) correctly match their canonical counterparts
// or parent categories.
//
// # Custom Extensions
//
// Custom MIME types and extensions can be registered during initialization, allowing the
// underlying detector to recognize specialized formats (such as ETSI ASiC-E container files).
package mimex
