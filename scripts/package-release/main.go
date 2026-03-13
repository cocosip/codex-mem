// Command package-release builds per-platform release archives from the local source tree.
package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type target struct {
	GOOS   string
	GOARCH string
}

const goosWindows = "windows"

var defaultTargets = []target{
	{GOOS: goosWindows, GOARCH: "amd64"},
	{GOOS: goosWindows, GOARCH: "arm64"},
	{GOOS: "linux", GOARCH: "amd64"},
	{GOOS: "linux", GOARCH: "arm64"},
	{GOOS: "darwin", GOARCH: "amd64"},
	{GOOS: "darwin", GOARCH: "arm64"},
}
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	repoRoot, err := os.Getwd()
	if err != nil {
		failf("resolve working directory: %v", err)
	}

	version := firstNonEmpty(os.Getenv("CODEX_MEM_VERSION"), os.Getenv("GITHUB_REF_NAME"), "dev")
	commit := firstNonEmpty(os.Getenv("CODEX_MEM_COMMIT"), os.Getenv("GITHUB_SHA"), "unknown")
	date := firstNonEmpty(os.Getenv("CODEX_MEM_BUILD_DATE"), time.Now().UTC().Format(time.RFC3339))
	distDir := filepath.Join(repoRoot, "dist")
	if err := os.RemoveAll(distDir); err != nil {
		failf("reset dist directory: %v", err)
	}
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		failf("create dist directory: %v", err)
	}

	for _, target := range defaultTargets {
		buildTarget(ctx, repoRoot, distDir, version, commit, date, target)
	}
	writeChecksums(distDir)

	fmt.Printf("release packaging complete\n")
	fmt.Printf("version=%s\n", version)
	fmt.Printf("targets=%d\n", len(defaultTargets))
	fmt.Printf("dist=%s\n", distDir)
}

func buildTarget(ctx context.Context, repoRoot, distDir, version, commit, date string, target target) {
	baseName := fmt.Sprintf("codex-mem_%s_%s_%s", sanitizeVersion(version), target.GOOS, target.GOARCH)
	stageDir := filepath.Join(distDir, baseName)
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		failf("create stage dir for %s/%s: %v", target.GOOS, target.GOARCH, err)
	}

	binaryName := "codex-mem"
	if target.GOOS == goosWindows {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(stageDir, binaryName)
	buildBinary(ctx, repoRoot, binaryPath, version, commit, date, target)
	copyFile(filepath.Join(repoRoot, "README.md"), filepath.Join(stageDir, "README.md"))
	copyFile(filepath.Join(repoRoot, "configs", "codex-mem.example.json"), filepath.Join(stageDir, "codex-mem.example.json"))

	if target.GOOS == goosWindows {
		archivePath := filepath.Join(distDir, baseName+".zip")
		writeZip(archivePath, stageDir)
	} else {
		archivePath := filepath.Join(distDir, baseName+".tar.gz")
		writeTarGz(archivePath, stageDir)
	}
	if err := os.RemoveAll(stageDir); err != nil {
		failf("remove stage dir %s: %v", stageDir, err)
	}
}

func buildBinary(ctx context.Context, repoRoot, outputPath, version, commit, date string, target target) {
	ldflags := fmt.Sprintf("-X codex-mem/internal/buildinfo.Version=%s -X codex-mem/internal/buildinfo.Commit=%s -X codex-mem/internal/buildinfo.Date=%s", version, commit, date)
	cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-ldflags", ldflags, "-o", outputPath, "./cmd/codex-mem")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS="+target.GOOS,
		"GOARCH="+target.GOARCH,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		failf("build %s/%s: %v\n%s", target.GOOS, target.GOARCH, err, string(output))
	}
}

func writeZip(archivePath, sourceDir string) {
	file, err := os.Create(archivePath)
	if err != nil {
		failf("create zip %s: %v", archivePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	zipWriter := zip.NewWriter(file)
	defer func() {
		_ = zipWriter.Close()
	}()

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(filepath.Dir(sourceDir), path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		header.Method = zip.Deflate
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		return copyIntoWriter(path, writer)
	})
	if err != nil {
		failf("write zip %s: %v", archivePath, err)
	}
}

func writeTarGz(archivePath, sourceDir string) {
	file, err := os.Create(archivePath)
	if err != nil {
		failf("create tar.gz %s: %v", archivePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	gzipWriter := gzip.NewWriter(file)
	defer func() {
		_ = gzipWriter.Close()
	}()
	tarWriter := tar.NewWriter(gzipWriter)
	defer func() {
		_ = tarWriter.Close()
	}()

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(filepath.Dir(sourceDir), path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = rel
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		return copyIntoWriter(path, tarWriter)
	})
	if err != nil {
		failf("write tar.gz %s: %v", archivePath, err)
	}
}

func copyIntoWriter(path string, writer io.Writer) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	_, err = io.Copy(writer, file)
	return err
}

func copyFile(src, dst string) {
	input, err := os.ReadFile(src)
	if err != nil {
		failf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, input, 0o644); err != nil {
		failf("write %s: %v", dst, err)
	}
}

func writeChecksums(distDir string) {
	entries, err := os.ReadDir(distDir)
	if err != nil {
		failf("read dist directory %s: %v", distDir, err)
	}

	type checksumEntry struct {
		name string
		sum  string
	}

	checksums := make([]checksumEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "SHA256SUMS" {
			continue
		}
		checksums = append(checksums, checksumEntry{
			name: name,
			sum:  fileSHA256(filepath.Join(distDir, name)),
		})
	}
	sort.Slice(checksums, func(i, j int) bool {
		return checksums[i].name < checksums[j].name
	})

	var builder strings.Builder
	for _, entry := range checksums {
		fmt.Fprintf(&builder, "%s  %s\n", entry.sum, entry.name)
	}

	outputPath := filepath.Join(distDir, "SHA256SUMS")
	if err := os.WriteFile(outputPath, []byte(builder.String()), 0o644); err != nil {
		failf("write %s: %v", outputPath, err)
	}
}

func fileSHA256(path string) string {
	file, err := os.Open(path)
	if err != nil {
		failf("open %s: %v", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		failf("hash %s: %v", path, err)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func sanitizeVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "dev"
	}
	value = strings.TrimPrefix(value, "refs/tags/")
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-")
	return replacer.Replace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func failf(format string, args ...any) {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, format+"\n", args...)
	if runtime.GOOS == goosWindows {
		buffer.WriteString("")
	}
	fmt.Fprint(os.Stderr, buffer.String())
	os.Exit(1)
}











