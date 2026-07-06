// Package trust implements a direnv-style content-hash trust store for
// project-local .gwrc hook overrides. A project's non-empty hook values are
// only executed once the user has explicitly approved the exact file
// content at its exact absolute path; any change to either re-triggers
// approval.
package trust

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// permTrustDir is the permission for the trust store directory (~/.gw/trust).
const permTrustDir = 0o755

// permTrustFile is the permission for individual trust marker files: owner
// read/write only, since their names (content hashes) are not secret but the
// directory should not be world-writable.
const permTrustFile = 0o600

// Compute returns the trust hash for a project config file: sha256 of the
// absolute path followed by a newline followed by the file's raw content.
// Including the path means two different clones (or a renamed directory)
// with identical content require independent approval, matching direnv's
// per-path trust model.
func Compute(absPath string, content []byte) string {
	h := sha256.New()
	h.Write([]byte(absPath))
	h.Write([]byte("\n"))
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// trustFilePath returns the path of the trust marker file for hash, rooted
// at the user's home directory (~/.gw/trust/<hash>). It never uses a literal
// "~" — Go does not expand it — and instead resolves the home directory via
// os.UserHomeDir().
func trustFilePath(hash string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory: %w", err)
	}
	return filepath.Join(home, ".gw", "trust", hash), nil
}

// IsApproved reports whether hash has a corresponding trust marker file.
// Any error resolving the home directory or statting the file is treated as
// "not approved" (fail-closed) rather than surfaced to the caller, since
// this function's contract is a plain bool.
func IsApproved(hash string) bool {
	path, err := trustFilePath(hash)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Approve records hash as trusted by creating its marker file. It is
// idempotent: approving an already-approved hash is a no-op. Any failure to
// create the trust directory or file is returned as an error so callers can
// fail closed (treat the hook value as not approved) rather than assume
// success.
func Approve(hash string) error {
	path, err := trustFilePath(hash)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, permTrustDir); err != nil {
		return fmt.Errorf("failed to create trust directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, permTrustFile)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("failed to create trust marker: %w", err)
	}
	return file.Close()
}
