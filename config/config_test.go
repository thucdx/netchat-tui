package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// redirectConfigHome overrides the HOME (or USERPROFILE on Windows) environment
// variable to tmpDir so that os.UserConfigDir() resolves inside the temp
// directory for the duration of the test.  It returns a restore function that
// the caller must defer.
func redirectConfigHome(t *testing.T, tmpDir string) {
	t.Helper()

	switch runtime.GOOS {
	case "darwin", "linux", "freebsd", "openbsd", "netbsd":
		// os.UserConfigDir() on Darwin: $HOME/Library/Application Support
		// On Unix: $XDG_CONFIG_HOME if set, else $HOME/.config
		origHome := os.Getenv("HOME")
		origXDG := os.Getenv("XDG_CONFIG_HOME")

		if runtime.GOOS == "darwin" {
			// Darwin ignores XDG_CONFIG_HOME; redirect HOME.
			if err := os.Setenv("HOME", tmpDir); err != nil {
				t.Fatalf("could not set HOME: %v", err)
			}
		} else {
			// Linux/BSD: set XDG_CONFIG_HOME directly — no need to redirect HOME.
			if err := os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config")); err != nil {
				t.Fatalf("could not set XDG_CONFIG_HOME: %v", err)
			}
		}

		t.Cleanup(func() {
			_ = os.Setenv("HOME", origHome)
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		})

	case "windows":
		orig := os.Getenv("APPDATA")
		appData := filepath.Join(tmpDir, "AppData", "Roaming")
		if err := os.MkdirAll(appData, 0700); err != nil {
			t.Fatalf("could not create AppData dir: %v", err)
		}
		if err := os.Setenv("APPDATA", appData); err != nil {
			t.Fatalf("could not set APPDATA: %v", err)
		}
		t.Cleanup(func() { _ = os.Setenv("APPDATA", orig) })

	default:
		t.Skipf("unsupported OS: %s", runtime.GOOS)
	}
}

// expectedAuthJSONPath derives the expected absolute path that configPath()
// will resolve to given the current OS after redirectConfigHome was called.
func expectedAuthJSONPath(t *testing.T, tmpDir string) string {
	t.Helper()
	path, err := configPath()
	if err != nil {
		t.Fatalf("configPath() returned error: %v", err)
	}
	return path
}

// ----------------------------------------------------------------------------
// Tests
// ----------------------------------------------------------------------------

// TestSaveAndLoad verifies that saving an AuthConfig and loading it back
// produces the identical Token and UserID values.
func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	redirectConfigHome(t, tmpDir)

	want := &AuthConfig{
		Token:  "test-token-abc123",
		UserID: "user-xyz789",
	}

	if err := Save(want); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if got.Token != want.Token {
		t.Errorf("Token mismatch: got %q, want %q", got.Token, want.Token)
	}
	if got.UserID != want.UserID {
		t.Errorf("UserID mismatch: got %q, want %q", got.UserID, want.UserID)
	}
}

// TestLoadMissingFile verifies that Load() returns an empty AuthConfig (not an
// error) when no config file exists yet.
func TestLoadMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	redirectConfigHome(t, tmpDir)

	// Deliberately do NOT call Save — the file must not exist.
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error for missing file: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil AuthConfig for missing file")
	}
	if cfg.Token != "" {
		t.Errorf("expected empty Token, got %q", cfg.Token)
	}
	if cfg.UserID != "" {
		t.Errorf("expected empty UserID, got %q", cfg.UserID)
	}
}

// TestFilePermissions verifies that the auth.json file is written with mode
// 0600 (owner read/write only).
func TestFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permission bits are not applicable on Windows")
	}

	tmpDir := t.TempDir()
	redirectConfigHome(t, tmpDir)

	cfg := &AuthConfig{Token: "tok", UserID: "uid"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	authPath := expectedAuthJSONPath(t, tmpDir)

	info, err := os.Stat(authPath)
	if err != nil {
		t.Fatalf("stat(%q) failed: %v", authPath, err)
	}

	const wantMode = os.FileMode(0600)
	gotMode := info.Mode().Perm()
	if gotMode != wantMode {
		t.Errorf("auth.json permissions: got %04o, want %04o", gotMode, wantMode)
	}
}

// TestDirPermissions verifies that the netchat-tui config directory is created
// with mode 0700 (owner read/write/execute only).
func TestDirPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permission bits are not applicable on Windows")
	}

	tmpDir := t.TempDir()
	redirectConfigHome(t, tmpDir)

	cfg := &AuthConfig{Token: "tok", UserID: "uid"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	authPath := expectedAuthJSONPath(t, tmpDir)
	dirPath := filepath.Dir(authPath)

	info, err := os.Stat(dirPath)
	if err != nil {
		t.Fatalf("stat(%q) failed: %v", dirPath, err)
	}

	const wantMode = os.FileMode(0700)
	gotMode := info.Mode().Perm()
	if gotMode != wantMode {
		t.Errorf("config dir permissions: got %04o, want %04o", gotMode, wantMode)
	}
}
