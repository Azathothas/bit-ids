// selfcorroboration - whether the second connector of a capture may be the
// target it is about to corroborate. `CLIENT-05`.
//
// ⛔ aria2 IS A TARGET AND A CANDIDATE CONNECTOR, and aria2 reading aria2's
// bytes would agree with itself on every field of every capture. `E-COR-20`
// refuses such a record in `crates/bit-ids/src/validate.rs`; this refuses the
// capture while the host still exists, before the second reading runs, so a
// circular pairing never reaches a bundle at all.
//
// ⭐ THE IDENTITY TEST IS THE RECORD'S: either name holding the other anywhere
// is one implementation, so `aria2` is refused beside `aria2`, `aria2c`,
// `libaria2` and `aria2-next`, in both directions. ⚠ It is written twice, in
// Rust and here, because the record's rule cannot run before a record exists;
// `capture-client --selftest` and `profile_schema.rs` hold the same cases.
//
// ⚠ A connector names itself with `--describe`. Its `id` is tested, and so is
// its `runtime` when it declares one: a connector built on aria2 under another
// identifier would still say what it runs on.
package main

import "strings"

// sameImplementation is `E-COR-20`'s identity test.
func sameImplementation(connector, target string) bool {
	return connector != "" && target != "" &&
		(strings.Contains(connector, target) || strings.Contains(target, connector))
}

// connectorIsTarget answers the described value that names the target, or ""
// when the connector's `--describe` names nothing the target is.
func connectorIsTarget(describe []byte, target string) string {
	for _, key := range []string{"id", "runtime"} {
		if v := keyField(describe, key); sameImplementation(v, target) {
			return key + "=" + v
		}
	}
	return ""
}
