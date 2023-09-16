// Package vcs provides a function to generate the version number.
package vcs

import "runtime/debug"

const defaultVersion = "0.0.0-dev"

// Version returns the version number using the
// vcs information embedded by the compiler.
// It defaults to 0.0.0-dev when vcs info is unavailable.
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return defaultVersion
	}

	var (
		revision = defaultVersion
		suffix   string
	)

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			if setting.Value == "true" {
				suffix = "-dirty"
			}
		}
	}

	return revision + suffix
}
