//go:build integration

// Package integration_test — anchor file ensures p.Name is integration_test before
// concurrent_write_test.go is loaded (Go classifies the first *_test.go in this
// directory as an external test for package "integration" when p.Name is still empty).
package integration_test
