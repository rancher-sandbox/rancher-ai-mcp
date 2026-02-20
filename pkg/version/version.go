package version

var (
	// Version of the MCP server, set via ldflags at build time.
	Version = "dev"
	// GitCommit of the Version, set via ldflags at build time.
	GitCommit = ""
)

// GetVersion returns the version of the MCP server,
// including the git commit if available.
func GetVersion() string {
	version := Version

	if GitCommit != "" {
		version += " (" + GitCommit + ")"
	}

	return version
}
