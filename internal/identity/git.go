// Package identity discovers canonical repository identity from local Git metadata.
package identity

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// RepositoryInfo describes the repository metadata discovered for a path.
type RepositoryInfo struct {
	Root   string
	Remote string
	Branch string
	HasGit bool
}

// DiscoverRepository inspects start and its parents for Git repository metadata.
func DiscoverRepository(start string) (RepositoryInfo, error) {
	absStart, err := filepath.Abs(start)
	if err != nil {
		return RepositoryInfo{}, fmt.Errorf("resolve start path: %w", err)
	}

	root, gitDir, found, err := findGitDir(absStart)
	if err != nil {
		return RepositoryInfo{}, err
	}
	if !found {
		return RepositoryInfo{Root: absStart}, nil
	}

	remote, err := readRemote(filepath.Join(gitDir, "config"))
	if err != nil && !os.IsNotExist(err) {
		return RepositoryInfo{}, fmt.Errorf("read git remote: %w", err)
	}
	branch, err := readBranch(filepath.Join(gitDir, "HEAD"))
	if err != nil && !os.IsNotExist(err) {
		return RepositoryInfo{}, fmt.Errorf("read git branch: %w", err)
	}

	return RepositoryInfo{
		Root:   NormalizePath(root),
		Remote: remote,
		Branch: branch,
		HasGit: true,
	}, nil
}

// NormalizePath canonicalizes a filesystem path for identity comparisons.
func NormalizePath(path string) string {
	clean := filepath.Clean(path)
	clean = filepath.ToSlash(clean)
	if runtime.GOOS == "windows" {
		clean = strings.ToLower(clean)
	}
	return clean
}

// NormalizeRemote canonicalizes a Git remote into host/path form.
func NormalizeRemote(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimSuffix(trimmed, ".git")
	if trimmed == "" {
		return ""
	}

	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err == nil {
			host := strings.ToLower(parsed.Host)
			path := strings.Trim(strings.ToLower(parsed.Path), "/")
			return strings.Trim(host+"/"+path, "/")
		}
	}

	if at := strings.Index(trimmed, "@"); at >= 0 && strings.Contains(trimmed[at+1:], ":") {
		remainder := trimmed[at+1:]
		parts := strings.SplitN(remainder, ":", 2)
		host := strings.ToLower(parts[0])
		path := strings.Trim(strings.ToLower(parts[1]), "/")
		return strings.Trim(host+"/"+path, "/")
	}

	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	return strings.Trim(strings.ToLower(trimmed), "/")
}

// NameFromRemote returns the repository name segment from a normalized remote.
func NameFromRemote(remote string) string {
	remote = strings.Trim(remote, "/")
	if remote == "" {
		return ""
	}
	parts := strings.Split(remote, "/")
	return parts[len(parts)-1]
}

func findGitDir(start string) (root string, gitDir string, found bool, err error) {
	current := start
	for {
		gitPath := filepath.Join(current, ".git")
		info, statErr := os.Stat(gitPath)
		if statErr == nil {
			if info.IsDir() {
				return current, gitPath, true, nil
			}

			content, readErr := os.ReadFile(gitPath)
			if readErr != nil {
				return "", "", false, fmt.Errorf("read .git file: %w", readErr)
			}
			text := strings.TrimSpace(string(content))
			const prefix = "gitdir:"
			if strings.HasPrefix(strings.ToLower(text), prefix) {
				target := strings.TrimSpace(text[len(prefix):])
				if !filepath.IsAbs(target) {
					target = filepath.Join(current, target)
				}
				return current, filepath.Clean(target), true, nil
			}
			return current, current, true, nil
		}
		if !os.IsNotExist(statErr) {
			return "", "", false, fmt.Errorf("inspect .git: %w", statErr)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", "", false, nil
		}
		current = parent
	}
}

func readRemote(configPath string) (string, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	currentSection := ""
	firstRemoteURL := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		if !strings.Contains(currentSection, `remote "`) || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if !strings.EqualFold(key, "url") {
			continue
		}
		if strings.EqualFold(currentSection, `remote "origin"`) {
			return value, nil
		}
		if firstRemoteURL == "" {
			firstRemoteURL = value
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return firstRemoteURL, nil
}

func readBranch(headPath string) (string, error) {
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(data))
	const prefix = "ref:"
	if strings.HasPrefix(strings.ToLower(line), prefix) {
		ref := strings.TrimSpace(line[len(prefix):])
		ref = strings.TrimPrefix(ref, "refs/heads/")
		return ref, nil
	}
	return "", nil
}
