package version

import (
	"runtime/debug"
)

// Version is the build version string, set at build time via -ldflags.
// Defaults to "snapshot" if not set.
var Version = "snapshot"

// Get returns the current version string.
// If Version was set via -ldflags (e.g. by ko or CI), it returns it.
// Otherwise, it attempts to read the VCS commit revision from Go runtime build info.
// If unavailable, it returns "snapshot".
func Get() string {
	if Version != "" && Version != "snapshot" {
		return Version
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				return setting.Value
			}
		}
	}

	return "snapshot"
}
