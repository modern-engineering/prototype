// Package sef encodes and decodes solution artifacts. JSON files use the suffix
// .json.sef; gob files use .gob.sef. DecodeJSON and EncodeJSON read and write JSON;
// DecodeGob and EncodeGob read and write gob. Both carry the same information.
//
// The logical representation is solution.Artifact. Encoding checks establish
// the supported schema and binding forms, leaving semantic checks to solution
// tooling. Successful decoding does not establish that symbols or fields exist.
//
// The wire schema is experimental. Codecs accept exactly Version and reject
// other identifiers; no compatibility across experimental versions is promised.
package sef
