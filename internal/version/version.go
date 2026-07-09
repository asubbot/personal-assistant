package version

// Commit and BuildTime are set at link time via -ldflags -X (see Makefile and Dockerfile).
// Defaults apply when built without those flags (plain go build).
var (
	Commit    = "unknown"
	BuildTime = "unknown"
)

// String returns a single-line build identity for logs and -version output.
func String() string {
	return "commit=" + Commit + " built=" + BuildTime
}
