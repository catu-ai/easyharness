package version

import (
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

var (
	BuildCommit  = ""
	BuildMode    = ""
	BuildVersion = ""
)

type Info struct {
	Version   string `json:"version,omitempty"`
	Mode      string `json:"mode"`
	Commit    string `json:"commit"`
	GoVersion string `json:"go_version,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
	Modified  *bool  `json:"modified,omitempty"`
	Path      string `json:"path,omitempty"`
}

func Current() Info {
	return current(debug.ReadBuildInfo, os.Executable)
}

func current(readBuildInfo func() (*debug.BuildInfo, bool), executablePath func() (string, error)) Info {
	var buildInfo *debug.BuildInfo
	var ok bool
	if readBuildInfo != nil {
		buildInfo, ok = readBuildInfo()
	}

	info := Info{
		Version:   resolveVersion(buildInfo, ok),
		Mode:      resolveMode(),
		Commit:    resolveCommit(buildInfo, ok),
		GoVersion: resolveGoVersion(buildInfo, ok),
		BuildTime: resolveBuildTime(buildInfo, ok),
	}
	if info.Mode == "dev" {
		info.Modified = resolveModified(buildInfo, ok)
		if path, err := executablePath(); err == nil {
			info.Path = strings.TrimSpace(path)
		}
	}
	return info
}

func resolveVersion(buildInfo *debug.BuildInfo, ok bool) string {
	if version := strings.TrimSpace(BuildVersion); version != "" {
		return version
	}
	if ok && buildInfo != nil {
		if version := strings.TrimSpace(buildInfo.Main.Version); version != "" && version != "(devel)" {
			return version
		}
	}
	return ""
}

func resolveCommit(buildInfo *debug.BuildInfo, ok bool) string {
	if commit := strings.TrimSpace(BuildCommit); commit != "" {
		return commit
	}
	if commit := resolveBuildSetting(buildInfo, ok, "vcs.revision"); commit != "" {
		return commit
	}
	return "unknown"
}

func resolveGoVersion(buildInfo *debug.BuildInfo, ok bool) string {
	if ok && buildInfo != nil {
		return strings.TrimSpace(buildInfo.GoVersion)
	}
	return ""
}

func resolveBuildTime(buildInfo *debug.BuildInfo, ok bool) string {
	return resolveBuildSetting(buildInfo, ok, "vcs.time")
}

func resolveModified(buildInfo *debug.BuildInfo, ok bool) *bool {
	modifiedValue := resolveBuildSetting(buildInfo, ok, "vcs.modified")
	if modifiedValue == "" {
		return nil
	}
	modified, err := strconv.ParseBool(modifiedValue)
	if err != nil {
		return nil
	}
	return &modified
}

func resolveMode() string {
	if mode := strings.TrimSpace(BuildMode); mode != "" {
		return mode
	}
	return "release"
}

func resolveBuildSetting(buildInfo *debug.BuildInfo, ok bool, key string) string {
	if !ok || buildInfo == nil {
		return ""
	}
	for _, setting := range buildInfo.Settings {
		if setting.Key == key {
			return strings.TrimSpace(setting.Value)
		}
	}
	return ""
}
