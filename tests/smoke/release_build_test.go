package smoke_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"debug/buildinfo"
	"debug/elf"
	"debug/macho"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yzhang1918/superharness/tests/support"
)

func TestBuildReleaseProducesSupportedAlphaArchivesAndVersionedBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	firstOutputDir := newReleaseOutputDir(t, "supported-alpha-a")
	secondOutputDir := newReleaseOutputDir(t, "supported-alpha-b")
	version := "v0.1.0-alpha.1"
	expectedCommit := gitHeadCommit(t, support.RepoRoot(t))

	firstResult := runReleaseBuild(t, version, firstOutputDir)
	secondResult := runReleaseBuild(t, version, secondOutputDir)

	expectedPlatforms := []string{
		"darwin/amd64",
		"darwin/arm64",
		"linux/amd64",
		"linux/arm64",
	}

	firstChecksums := parseChecksums(t, readFile(t, filepath.Join(firstOutputDir, "SHA256SUMS")))
	secondChecksums := parseChecksums(t, readFile(t, filepath.Join(secondOutputDir, "SHA256SUMS")))
	for _, platform := range expectedPlatforms {
		goos, goarch := splitPlatform(t, platform)
		archiveName := "superharness_" + version + "_" + goos + "_" + goarch + ".zip"
		firstArchivePath := filepath.Join(firstOutputDir, archiveName)
		secondArchivePath := filepath.Join(secondOutputDir, archiveName)
		if _, err := os.Stat(firstArchivePath); err != nil {
			t.Fatalf("expected archive %s: %v\n%s", firstArchivePath, err, firstResult)
		}
		if _, err := os.Stat(secondArchivePath); err != nil {
			t.Fatalf("expected archive %s: %v\n%s", secondArchivePath, err, secondResult)
		}
		firstChecksum := checksumFile(t, firstArchivePath)
		secondChecksum := checksumFile(t, secondArchivePath)
		if got := firstChecksums[archiveName]; got != firstChecksum {
			t.Fatalf("expected first checksum for %s to match file contents, got %q want %q", archiveName, got, firstChecksum)
		}
		if got := secondChecksums[archiveName]; got != secondChecksum {
			t.Fatalf("expected second checksum for %s to match file contents, got %q want %q", archiveName, got, secondChecksum)
		}
		if firstChecksum != secondChecksum {
			t.Fatalf("expected deterministic archive checksum for %s, got %q and %q", archiveName, firstChecksum, secondChecksum)
		}
		if !bytes.Equal(readFileBytes(t, firstArchivePath), readFileBytes(t, secondArchivePath)) {
			t.Fatalf("expected deterministic archive bytes for %s across identical builds", archiveName)
		}
		verifyArchiveContents(t, workspace, firstArchivePath, version, goos, goarch, expectedCommit)
	}
}

func TestBuildReleaseCleansReusedOutputDirectory(t *testing.T) {
	outputDir := newReleaseOutputDir(t, "reuse-output")

	hostPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if !isSupportedAlphaPlatform(hostPlatform) {
		t.Skipf("host platform %s is outside the supported alpha target set", hostPlatform)
	}

	version := "v0.1.0-alpha.1"
	runReleaseBuildForPlatforms(t, version, outputDir, hostPlatform)

	staleFile := filepath.Join(outputDir, "stale.txt")
	if err := os.WriteFile(staleFile, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}
	staleArchive := filepath.Join(outputDir, "superharness_stale.zip")
	if err := os.WriteFile(staleArchive, []byte("stale archive"), 0o644); err != nil {
		t.Fatalf("write stale archive: %v", err)
	}

	runReleaseBuildForPlatforms(t, version, outputDir, hostPlatform)

	if _, err := os.Stat(staleFile); !os.IsNotExist(err) {
		t.Fatalf("expected stale file to be removed, got err=%v", err)
	}
	if _, err := os.Stat(staleArchive); !os.IsNotExist(err) {
		t.Fatalf("expected stale archive to be removed, got err=%v", err)
	}

	goos, goarch := splitPlatform(t, hostPlatform)
	wantEntries := map[string]bool{
		"SHA256SUMS": true,
		"superharness_" + version + "_" + goos + "_" + goarch + ".zip": true,
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("read output dir: %v", err)
	}
	if len(entries) != len(wantEntries) {
		t.Fatalf("expected %d release outputs, found %d", len(wantEntries), len(entries))
	}
	for _, entry := range entries {
		if !wantEntries[entry.Name()] {
			t.Fatalf("unexpected leftover release output %q", entry.Name())
		}
	}
}

func TestBuildReleaseRejectsUnsafeOutputDirectory(t *testing.T) {
	cases := []struct {
		name      string
		outputDir string
		wantText  string
	}{
		{
			name:      "repo root",
			outputDir: ".",
			wantText:  "refusing to use repository root as the release output directory",
		},
		{
			name:      "relative parent escape",
			outputDir: "../release-out",
			wantText:  "output directory must not contain parent-directory segments",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(
				"scripts/build-release",
				"--version", "v0.1.0-alpha.1",
				"--output-dir", tc.outputDir,
				"--platform", runtime.GOOS+"/"+runtime.GOARCH,
			)
			cmd.Dir = support.RepoRoot(t)
			result, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected build-release to reject output dir %q", tc.outputDir)
			}
			if !strings.Contains(string(result), tc.wantText) {
				t.Fatalf("expected output for %q to contain %q, got:\n%s", tc.outputDir, tc.wantText, result)
			}
		})
	}
}

func runReleaseBuild(t *testing.T, version, outputDir string) string {
	t.Helper()
	return runReleaseBuildForPlatforms(t, version, outputDir)
}

func runReleaseBuildForPlatforms(t *testing.T, version, outputDir string, platforms ...string) string {
	t.Helper()

	args := []string{"--version", version, "--output-dir", outputDir}
	for _, platform := range platforms {
		args = append(args, "--platform", platform)
	}
	cmd := exec.Command("scripts/build-release", args...)
	cmd.Dir = support.RepoRoot(t)
	result, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build release: %v\n%s", err, result)
	}
	return string(result)
}

func verifyArchiveContents(t *testing.T, workspace *support.Workspace, archivePath, version, goos, goarch, expectedCommit string) {
	t.Helper()

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer reader.Close()

	packageRoot := "superharness_" + version + "_" + goos + "_" + goarch + "/"
	binaryName := packageRoot + "harness"
	readmeName := packageRoot + "README.md"
	licenseName := packageRoot + "LICENSE"

	var sawBinary bool
	var sawReadme bool
	var sawLicense bool
	for _, file := range reader.File {
		switch file.Name {
		case binaryName:
			sawBinary = true
		case readmeName:
			sawReadme = true
		case licenseName:
			sawLicense = true
		}
	}
	if !sawBinary {
		t.Fatalf("expected archive to include %s", binaryName)
	}
	if !sawReadme {
		t.Fatalf("expected archive to include %s", readmeName)
	}
	if !sawLicense {
		t.Fatalf("expected archive to include %s", licenseName)
	}

	extractDir := workspace.Path(filepath.Join("extract", goos+"-"+goarch))
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		t.Fatalf("mkdir extract: %v", err)
	}
	if err := unzipArchive(archivePath, extractDir); err != nil {
		t.Fatalf("unzip archive: %v", err)
	}

	binaryPath := filepath.Join(extractDir, binaryName)
	verifyBinaryMetadata(t, binaryPath, version, goos, goarch, expectedCommit)
	if goos == runtime.GOOS && goarch == runtime.GOARCH {
		versionCmd := exec.Command(binaryPath, "--version")
		versionCmd.Dir = extractDir
		versionOutput, err := versionCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("run packaged binary --version: %v\n%s", err, versionOutput)
		}

		output := string(versionOutput)
		if got := requireVersionField(t, output, "version"); got != version {
			t.Fatalf("expected packaged version %q, got %q\noutput:\n%s", version, got, output)
		}
		if got := requireVersionField(t, output, "mode"); got != "release" {
			t.Fatalf("expected packaged mode release, got %q\noutput:\n%s", got, output)
		}
		if got := requireVersionField(t, output, "commit"); got != expectedCommit {
			t.Fatalf("expected packaged commit %q, got %q\noutput:\n%s", expectedCommit, got, output)
		}
		if strings.Contains(output, "path: ") {
			t.Fatalf("expected packaged release output to omit path, got %q", output)
		}
	}
}

func verifyBinaryMetadata(t *testing.T, binaryPath, version, goos, goarch, expectedCommit string) {
	t.Helper()

	info, err := buildinfo.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("read Go build info for %s: %v", binaryPath, err)
	}
	if info.GoVersion == "" {
		t.Fatalf("expected Go build info in %s", binaryPath)
	}

	binaryData := readFileBytes(t, binaryPath)
	if !bytes.Contains(binaryData, []byte(version)) {
		t.Fatalf("expected binary %s to contain release version %q", binaryPath, version)
	}
	if !bytes.Contains(binaryData, []byte(expectedCommit)) {
		t.Fatalf("expected binary %s to contain build commit %q", binaryPath, expectedCommit)
	}

	switch goos {
	case "darwin":
		file, err := macho.Open(binaryPath)
		if err != nil {
			t.Fatalf("open Mach-O binary %s: %v", binaryPath, err)
		}
		defer file.Close()

		wantCPU := macho.CpuAmd64
		if goarch == "arm64" {
			wantCPU = macho.CpuArm64
		}
		if file.Cpu != wantCPU {
			t.Fatalf("expected Mach-O CPU %v for %s, got %v", wantCPU, binaryPath, file.Cpu)
		}
	case "linux":
		file, err := elf.Open(binaryPath)
		if err != nil {
			t.Fatalf("open ELF binary %s: %v", binaryPath, err)
		}
		defer file.Close()

		wantMachine := elf.EM_X86_64
		if goarch == "arm64" {
			wantMachine = elf.EM_AARCH64
		}
		if file.FileHeader.Machine != wantMachine {
			t.Fatalf("expected ELF machine %v for %s, got %v", wantMachine, binaryPath, file.FileHeader.Machine)
		}
	default:
		t.Fatalf("unsupported target OS %q", goos)
	}
}

func parseChecksums(t *testing.T, data string) map[string]string {
	t.Helper()

	checksums := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			t.Fatalf("malformed checksum line %q", line)
		}
		checksums[fields[len(fields)-1]] = fields[0]
	}
	return checksums
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	return string(readFileBytes(t, path))
}

func readFileBytes(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func checksumFile(t *testing.T, path string) string {
	t.Helper()

	data := readFileBytes(t, path)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func splitPlatform(t *testing.T, platform string) (string, string) {
	t.Helper()

	parts := strings.Split(platform, "/")
	if len(parts) != 2 {
		t.Fatalf("invalid platform %q", platform)
	}
	return parts[0], parts[1]
}

func isSupportedAlphaPlatform(platform string) bool {
	switch platform {
	case "darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64":
		return true
	default:
		return false
	}
}

func newReleaseOutputDir(t *testing.T, prefix string) string {
	t.Helper()

	baseDir := filepath.Join(support.RepoRoot(t), ".local", "release-smoke")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatalf("mkdir release smoke base dir: %v", err)
	}
	outputDir, err := os.MkdirTemp(baseDir, prefix+"-*")
	if err != nil {
		t.Fatalf("mktemp release output dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(outputDir)
	})
	return outputDir
}

func unzipArchive(archivePath, destDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		targetPath := filepath.Join(destDir, filepath.FromSlash(file.Name))
		if !strings.HasPrefix(targetPath, destDir+string(os.PathSeparator)) && targetPath != destDir {
			return os.ErrPermission
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, data, file.Mode()); err != nil {
			return err
		}
	}

	return nil
}
