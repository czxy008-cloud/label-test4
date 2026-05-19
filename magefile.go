//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	appName     = "clinic-appointment"
	mainPackage = "./cmd/server"
	outputDir   = "dist"
)

var (
	version = getVersion()
	commit  = getCommit()
	built   = time.Now().Format(time.RFC3339)
	goos    = runtime.GOOS
	goarch  = runtime.GOARCH
)

func getVersion() string {
	if v := os.Getenv("APP_VERSION"); v != "" {
		return v
	}
	return "dev"
}

func getCommit() string {
	if c := os.Getenv("GIT_COMMIT"); c != "" {
		return c
	}
	c, _ := sh.Output("git", "rev-parse", "--short", "HEAD")
	return c
}

func ldflags() string {
	flags := []string{
		"-s",
		"-w",
		fmt.Sprintf("-X main.version=%s", version),
		fmt.Sprintf("-X main.commit=%s", commit),
		fmt.Sprintf("-X main.built=%s", built),
	}
	return strings.Join(flags, " ")
}

// Clean removes build artifacts
func Clean() error {
	fmt.Println("Cleaning build artifacts...")
	if err := sh.Rm(outputDir); err != nil {
		return err
	}
	return sh.Run("go", "clean", "-cache")
}

// Deps installs dependencies
func Deps() error {
	fmt.Println("Installing dependencies...")
	return sh.Run("go", "mod", "download")
}

// Tidy tidies go modules
func Tidy() error {
	fmt.Println("Tidying go modules...")
	return sh.Run("go", "mod", "tidy")
}

// Build compiles the application
func Build() error {
	mg.Deps(Deps)
	fmt.Printf("Building %s %s/%s...\n", appName, goos, goarch)

	output := filepath.Join(outputDir, fmt.Sprintf("%s-%s-%s", appName, goos, goarch))
	if goos == "windows" {
		output += ".exe"
	}

	env := map[string]string{
		"CGO_ENABLED": "0",
		"GOOS":        goos,
		"GOARCH":      goarch,
	}

	return sh.RunWith(env, "go", "build",
		"-ldflags", ldflags(),
		"-o", output,
		mainPackage,
	)
}

// BuildAll compiles for all supported platforms
func BuildAll() error {
	mg.Deps(Deps)
	platforms := []struct {
		os   string
		arch string
	}{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
	}

	for _, p := range platforms {
		fmt.Printf("Building %s %s/%s...\n", appName, p.os, p.arch)
		output := filepath.Join(outputDir, fmt.Sprintf("%s-%s-%s", appName, p.os, p.arch))
		if p.os == "windows" {
			output += ".exe"
		}

		env := map[string]string{
			"CGO_ENABLED": "0",
			"GOOS":        p.os,
			"GOARCH":      p.arch,
		}

		if err := sh.RunWith(env, "go", "build",
			"-ldflags", ldflags(),
			"-o", output,
			mainPackage,
		); err != nil {
			return err
		}
	}
	return nil
}

// Test runs unit tests
func Test() error {
	mg.Deps(Deps)
	fmt.Println("Running tests...")
	return sh.Run("go", "test", "-v", "-race", "-cover", "./...")
}

// TestCoverage runs tests with coverage report
func TestCoverage() error {
	mg.Deps(Deps)
	fmt.Println("Running tests with coverage...")
	return sh.Run("go", "test", "-v", "-race", "-coverprofile=coverage.out", "./...")
}

// Lint runs go vet and staticcheck
func Lint() error {
	mg.Deps(Deps)
	fmt.Println("Running go vet...")
	if err := sh.Run("go", "vet", "./..."); err != nil {
		return err
	}
	return nil
}

// Run starts the application
func Run() error {
	mg.Deps(Build)
	fmt.Println("Starting application...")
	exe := filepath.Join(outputDir, fmt.Sprintf("%s-%s-%s", appName, goos, goarch))
	if goos == "windows" {
		exe += ".exe"
	}
	return sh.Run(exe, "-config", "config/config.yaml")
}

// Package creates a distribution package
func Package() error {
	mg.Deps(Build)
	fmt.Println("Packaging distribution...")

	pkgDir := filepath.Join(outputDir, "package")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return err
	}

	exe := filepath.Join(outputDir, fmt.Sprintf("%s-%s-%s", appName, goos, goarch))
	if goos == "windows" {
		exe += ".exe"
	}
	baseExe := filepath.Base(exe)
	if err := sh.Copy(filepath.Join(pkgDir, baseExe), exe); err != nil {
		return err
	}

	if err := sh.Copy(filepath.Join(pkgDir, "config.yaml"), "config/config.yaml"); err != nil {
		return err
	}

	if err := sh.Copy(filepath.Join(pkgDir, "001_init.sql"), "migrations/001_init.sql"); err != nil {
		return err
	}

	archive := filepath.Join(outputDir, fmt.Sprintf("%s-%s-%s.tar.gz", appName, goos, goarch))
	if goos == "windows" {
		archive = filepath.Join(outputDir, fmt.Sprintf("%s-%s-%s.zip", appName, goos, goarch))
		return sh.Run("powershell", "-Command",
			fmt.Sprintf("Compress-Archive -Path '%s/*' -DestinationPath '%s'", pkgDir, archive))
	}
	return sh.Run("tar", "-czf", archive, "-C", pkgDir, ".")
}

// All runs the full pipeline: clean, test, build, package
func All() error {
	mg.Deps(Clean)
	mg.Deps(Test)
	mg.Deps(Build)
	mg.Deps(Package)
	fmt.Println("Pipeline completed successfully!")
	return nil
}
