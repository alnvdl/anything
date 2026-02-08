// Package version provides functionality to retrieve the application version.
package version

import (
	"runtime/debug"
)

// Version returns the current version of the application, which is determined
// by the VCS revision if available.
func Version() string {
	info, _ := debug.ReadBuildInfo()
	if info != nil {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}

	return "unknown"
}
