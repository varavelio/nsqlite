package main

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/crypto/sha3"
)

// To update SQLite version, change the zipURL and zipSHA constants below to point to the new
// version's zip file and its SHA3-256 hash from this page: https://www.sqlite.org/download.html

const (
	zipURL = "https://www.sqlite.org/2026/sqlite-amalgamation-3530100.zip"
	zipSHA = "3c07136e4f6b5dd0c395be86455014039597bc65b6851f7111e88f71b6e06114"
)

func main() {
	fmt.Println("Downloading SQLite amalgamation...")

	resp, err := http.Get(zipURL)
	if err != nil {
		panic(fmt.Errorf("failed to download: %w", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Errorf("failed to read response: %w", err))
	}

	fmt.Println("Verifying SHA3-256...")
	h := sha3.Sum256(body)
	if got := hex.EncodeToString(h[:]); got != zipSHA {
		panic(fmt.Errorf("SHA3-256 mismatch: expected %s, got %s", zipSHA, got))
	}

	fmt.Println("Extracting zip...")
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		panic(fmt.Errorf("failed to open zip: %w", err))
	}

	extract := map[string]bool{
		"sqlite3.c":    true,
		"sqlite3.h":    true,
		"sqlite3ext.h": true,
	}
	destDir := filepath.Join("internal", "sqlitec")

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		panic(fmt.Errorf("failed to create %s: %w", destDir, err))
	}

	for _, f := range zr.File {
		if !extract[filepath.Base(f.Name)] {
			continue
		}
		fmt.Printf("  extracting %s...\n", f.Name)
		rc, err := f.Open()
		if err != nil {
			panic(fmt.Errorf("failed to open %s in zip: %w", f.Name, err))
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			panic(fmt.Errorf("failed to read %s: %w", f.Name, err))
		}
		destPath := filepath.Join(destDir, filepath.Base(f.Name))
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			panic(fmt.Errorf("failed to write %s: %w", destPath, err))
		}
	}

	fmt.Println("Done!")
}
