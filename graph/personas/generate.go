// Package personas holds the embedded persona markdown files and the
// generated HiveStatus map (status_gen.go). Run `go generate ./graph/personas/...`
// after bumping the third_party/hive submodule.
package personas

//go:generate go run ../../cmd/gen-persona-status
