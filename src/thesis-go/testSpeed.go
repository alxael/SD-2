package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// findCargo locates the cargo executable. It first checks PATH, then falls back
// to the standard rustup install location at ~/.cargo/bin/cargo.
func findCargo() (string, error) {
	if path, err := exec.LookPath("cargo"); err == nil {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err == nil {
		candidate := filepath.Join(home, ".cargo", "bin", "cargo")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("cargo not found in PATH or ~/.cargo/bin")
}

// generateSpeedReport runs the Rust implementation and writes the resulting
// CSV into thesis-go/reports/test-speed.csv. The Rust binary is responsible
// for measuring across input sizes and core counts.
func generateSpeedReport(outputSize int, sectionCount int) {
	_ = outputSize
	_ = sectionCount

	if err := os.MkdirAll("reports", 0755); err != nil {
		fmt.Println("Could not create reports directory:", err)
		return
	}

	outputPath, err := filepath.Abs("reports/test-speed.csv")
	if err != nil {
		fmt.Println("Could not resolve report path:", err)
		return
	}

	manifestPath, err := filepath.Abs("../thesis-rust/Cargo.toml")
	if err != nil {
		fmt.Println("Could not resolve Rust manifest path:", err)
		return
	}

	rustDir := filepath.Dir(manifestPath)
	binaryPath := filepath.Join(rustDir, "target", "release", "thesis_rust")

	cargo, err := findCargo()
	if err != nil {
		fmt.Println("Could not locate cargo:", err)
		return
	}

	// Build the release binary (no-op if already up to date).
	build := exec.Command(cargo, "build", "--release", "--quiet", "--manifest-path", manifestPath)
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Println("Rust build failed:", err)
		return
	}

	// Invoke the prebuilt binary directly so cargo does not need to be on PATH
	// at run time.
	cmd := exec.Command(binaryPath, outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Rust speed report failed:", err)
		return
	}
}
