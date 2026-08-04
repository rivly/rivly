package buildinfo

import "runtime/debug"

var version = "dev"

type Info struct {
	Version string
	Commit  string
	BuiltAt string
	Dirty   bool
	Go      string
}

func Read() Info {
	info := Info{Version: version}

	build, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}
	info.Go = build.GoVersion

	for _, setting := range build.Settings {
		switch setting.Key {
		case "vcs.revision":
			info.Commit = setting.Value
		case "vcs.time":
			info.BuiltAt = setting.Value
		case "vcs.modified":
			info.Dirty = setting.Value == "true"
		}
	}
	return info
}
