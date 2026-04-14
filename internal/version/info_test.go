package version

import (
	"runtime/debug"
	"testing"
)

func TestCurrentUsesBuildInfoMetadataInReleaseMode(t *testing.T) {
	t.Cleanup(func() {
		BuildCommit = ""
		BuildMode = ""
		BuildVersion = ""
	})

	info := current(
		func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{
				GoVersion: "go1.25.0",
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc123"},
					{Key: "vcs.time", Value: "2026-04-14T12:34:56Z"},
					{Key: "vcs.modified", Value: "true"},
				},
			}, true
		},
		func() (string, error) {
			t.Fatal("release mode should not need executable path")
			return "", nil
		},
	)

	if info.Commit != "abc123" {
		t.Fatalf("expected build-info commit, got %#v", info)
	}
	if info.Mode != "release" {
		t.Fatalf("expected release mode by default, got %#v", info)
	}
	if info.GoVersion != "go1.25.0" {
		t.Fatalf("expected Go version in release mode, got %#v", info)
	}
	if info.BuildTime != "2026-04-14T12:34:56Z" {
		t.Fatalf("expected build time in release mode, got %#v", info)
	}
	if info.Path != "" {
		t.Fatalf("expected release mode to omit path, got %#v", info)
	}
	if info.Modified != nil {
		t.Fatalf("expected release mode to omit modified, got %#v", info)
	}
}

func TestCurrentUsesExplicitDevMetadata(t *testing.T) {
	t.Cleanup(func() {
		BuildCommit = ""
		BuildMode = ""
		BuildVersion = ""
	})

	BuildCommit = "deadbeef"
	BuildMode = "dev"
	BuildVersion = "v0.0.0-dev"

	info := current(
		func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{
				GoVersion: "go1.25.0",
				Settings: []debug.BuildSetting{
					{Key: "vcs.time", Value: "2026-04-14T12:34:56Z"},
					{Key: "vcs.modified", Value: "true"},
				},
			}, true
		},
		func() (string, error) {
			return "/tmp/dev-harness", nil
		},
	)

	if info.Commit != "deadbeef" {
		t.Fatalf("expected explicit build commit, got %#v", info)
	}
	if info.Mode != "dev" {
		t.Fatalf("expected dev mode, got %#v", info)
	}
	if info.Path != "/tmp/dev-harness" {
		t.Fatalf("expected dev path, got %#v", info)
	}
	if info.Version != "v0.0.0-dev" {
		t.Fatalf("expected explicit build version, got %#v", info)
	}
	if info.GoVersion != "go1.25.0" {
		t.Fatalf("expected Go version in dev mode, got %#v", info)
	}
	if info.BuildTime != "2026-04-14T12:34:56Z" {
		t.Fatalf("expected build time in dev mode, got %#v", info)
	}
	if info.Modified == nil || !*info.Modified {
		t.Fatalf("expected modified=true in dev mode, got %#v", info)
	}
}

func TestCurrentFallsBackToUnknownCommit(t *testing.T) {
	t.Cleanup(func() {
		BuildCommit = ""
		BuildMode = ""
		BuildVersion = ""
	})

	info := current(
		func() (*debug.BuildInfo, bool) {
			return nil, false
		},
		func() (string, error) {
			return "", nil
		},
	)

	if info.Commit != "unknown" {
		t.Fatalf("expected unknown commit fallback, got %#v", info)
	}
	if info.Version != "" {
		t.Fatalf("expected unknown-commit fallback to omit version, got %#v", info)
	}
	if info.Modified != nil {
		t.Fatalf("expected unknown-commit fallback to omit modified, got %#v", info)
	}
}

func TestCurrentUsesBuildInfoVersionWhenExplicitVersionIsMissing(t *testing.T) {
	t.Cleanup(func() {
		BuildCommit = ""
		BuildMode = ""
		BuildVersion = ""
	})

	info := current(
		func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{
				Main: debug.Module{
					Version: "v1.2.3",
				},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc123"},
				},
			}, true
		},
		func() (string, error) {
			return "", nil
		},
	)

	if info.Version != "v1.2.3" {
		t.Fatalf("expected build-info version, got %#v", info)
	}
}

func TestCurrentOmitsInvalidModifiedFlag(t *testing.T) {
	t.Cleanup(func() {
		BuildCommit = ""
		BuildMode = ""
		BuildVersion = ""
	})

	BuildMode = "dev"

	info := current(
		func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.modified", Value: "not-a-bool"},
				},
			}, true
		},
		func() (string, error) {
			return "/tmp/dev-harness", nil
		},
	)

	if info.Modified != nil {
		t.Fatalf("expected invalid modified value to be omitted, got %#v", info)
	}
}
