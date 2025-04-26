package main

import (
	"bufio"
	"fmt"
	"github.com/ktr0731/go-fuzzyfinder"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	//configFile  = filepath.Join(os.Getenv("HOME"), ".config")
	//historyFile = filepath.Join(os.Getenv("HOME"), ".recent_github_repos")

	configFile  = filepath.Join(".config")
	historyFile = filepath.Join(".recent_github_repos")
)

type Repo struct {
	Path string
	Name string
}

func main() {
	// Read config
	configPaths, err := readLines(configFile)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	// Read history
	historyRepos, _ := readLines(historyFile)

	// Scan repos
	scannedRepos := scanRepos(configPaths)

	// Merge repos
	allPaths := mergeRepos(historyRepos, scannedRepos)

	// Prepare Repo structs
	var allRepos []Repo
	for _, p := range allPaths {
		allRepos = append(allRepos, Repo{
			Path: p,
			Name: filepath.Base(p),
		})
	}

	if len(allRepos) == 0 {
		log.Fatal("No git repositories found.")
	}

	// Fuzzy find
	idx, err := fuzzyfinder.Find(
		allRepos,
		func(i int) string {
			return allRepos[i].Name // only display Name
		},
		fuzzyfinder.WithPromptString("Select a repository > "),
	)
	if err != nil {
		log.Fatal("Selection canceled")
	}

	selectedRepo := allRepos[idx]
	fmt.Println("Opening:", selectedRepo.Path)

	// Update history
	newHistory := []string{selectedRepo.Path}
	for _, r := range historyRepos {
		if r != selectedRepo.Path {
			newHistory = append(newHistory, r)
		}
	}
	if len(newHistory) > 50 {
		newHistory = newHistory[:50]
	}
	_ = writeLines(newHistory, historyFile)

	// Run lazygit
	cmd := exec.Command("lazygit")
	cmd.Dir = selectedRepo.Path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		log.Fatalf("Failed to run lazygit: %v", err)
	}
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

func writeLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, line := range lines {
		_, _ = file.WriteString(line + "\n")
	}
	return nil
}

func scanRepos(paths []string) []string {
	var repos []string
	seen := make(map[string]bool)

	for _, root := range paths {
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				return nil
			}

			// Check if this directory has a .git folder
			gitPath := filepath.Join(path, ".git")
			info, err := os.Stat(gitPath)
			if err == nil && info.IsDir() {
				if !seen[path] {
					repos = append(repos, path)
					seen[path] = true
				}
				// Skip walking inside this repo
				return filepath.SkipDir
			}

			return nil
		})
	}

	return repos
}

func mergeRepos(history, scanned []string) []string {
	seen := make(map[string]bool)
	var merged []string

	for _, r := range history {
		if !seen[r] {
			merged = append(merged, r)
			seen[r] = true
		}
	}
	for _, r := range scanned {
		if !seen[r] {
			merged = append(merged, r)
			seen[r] = true
		}
	}
	return merged
}
