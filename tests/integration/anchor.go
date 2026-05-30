//go:build integration

// Package integration_test.
//
// This non-test file anchors the package name to "integration_test" under
// -tags=integration. The other non-test files (config_helpers.go, doc.go) sort
// alphabetically AFTER concurrent_write_test.go; without an earlier non-test file
// the Go loader processes concurrent_write_test.go first, classifies it as an
// external _test package, and sets the package name to "integration", which then
// collides with config_helpers.go (package integration_test). The "anchor" filename
// sorts first and pins the package name. Verify with:
//
//	go vet -tags=integration ./tests/integration/...
package integration_test
