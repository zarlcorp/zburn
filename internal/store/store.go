// Package store provides an encrypted identity store backed by a filesystem.
// identities are stored as individual AES-256-GCM encrypted files.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zarlcorp/core/pkg/zcrypto"
	"github.com/zarlcorp/core/pkg/zfilesystem"
)

const (
	saltFile       = "salt"
	verifyFile     = "verify"
	identitiesDir  = "identities"
	verifyToken    = "zburn-identity-store-ok"
)

// ErrNotFound is returned when an identity does not exist.
var ErrNotFound = errors.New("identity not found")

// Identity represents a generated identity.
type Identity struct {
	ID        string    `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Street    string    `json:"street"`
	City      string    `json:"city"`
	State     string    `json:"state"`
	Zip       string    `json:"zip"`
	DOB       time.Time `json:"dob"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
}

// Store manages encrypted identity files on a filesystem.
type Store struct {
	fs  zfilesystem.ReadWriteFileFS
	key []byte
}

// Open opens or initializes an encrypted identity store.
// On first run, it creates the salt and verification token.
// On subsequent runs, it verifies the password by decrypting the token.
func Open(fsys zfilesystem.ReadWriteFileFS, password string) (*Store, error) {
	salt, err := readOrCreateSalt(fsys)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	key, _, err := zcrypto.DeriveKey([]byte(password), salt)
	if err != nil {
		return nil, fmt.Errorf("open store: derive key: %w", err)
	}

	if err := verifyOrCreateToken(fsys, key); err != nil {
		zcrypto.Erase(key)
		return nil, fmt.Errorf("open store: %w", err)
	}

	if err := fsys.MkdirAll(identitiesDir, 0o700); err != nil {
		zcrypto.Erase(key)
		return nil, fmt.Errorf("open store: create identities dir: %w", err)
	}

	return &Store{fs: fsys, key: key}, nil
}

// Save encrypts and writes an identity to disk.
func (s *Store) Save(id Identity) error {
	data, err := json.Marshal(id)
	if err != nil {
		return fmt.Errorf("save identity: marshal: %w", err)
	}

	ct, err := zcrypto.Encrypt(s.key, data)
	if err != nil {
		return fmt.Errorf("save identity: encrypt: %w", err)
	}

	path := identityPath(id.ID)
	if err := s.fs.WriteFile(path, ct, 0o600); err != nil {
		return fmt.Errorf("save identity: write %s: %w", id.ID, err)
	}

	return nil
}

// Get decrypts and returns a single identity by ID.
func (s *Store) Get(id string) (Identity, error) {
	path := identityPath(id)

	ct, err := s.fs.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Identity{}, ErrNotFound
		}
		return Identity{}, fmt.Errorf("get identity: read %s: %w", id, err)
	}

	return s.decryptIdentity(ct)
}

// List returns all stored identities sorted by CreatedAt descending.
func (s *Store) List() ([]Identity, error) {
	var ids []Identity

	err := s.fs.WalkDir(identitiesDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasPrefix(path, identitiesDir+"/") && !strings.HasPrefix(path, identitiesDir+"\\") {
			return nil
		}
		if filepath.Ext(path) != ".enc" {
			return nil
		}

		ct, err := s.fs.ReadFile(path)
		if err != nil {
			return fmt.Errorf("list identities: read %s: %w", path, err)
		}

		identity, err := s.decryptIdentity(ct)
		if err != nil {
			return fmt.Errorf("list identities: %w", err)
		}

		ids = append(ids, identity)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list identities: %w", err)
	}

	sort.Slice(ids, func(i, j int) bool {
		return ids[i].CreatedAt.After(ids[j].CreatedAt)
	})

	return ids, nil
}

// Delete removes an identity file by ID.
func (s *Store) Delete(id string) error {
	path := identityPath(id)

	if err := s.fs.Remove(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrNotFound
		}
		return fmt.Errorf("delete identity: remove %s: %w", id, err)
	}

	return nil
}

// Close erases the encryption key from memory.
func (s *Store) Close() error {
	zcrypto.Erase(s.key)
	s.key = nil
	return nil
}

func (s *Store) decryptIdentity(ct []byte) (Identity, error) {
	data, err := zcrypto.Decrypt(s.key, ct)
	if err != nil {
		return Identity{}, fmt.Errorf("decrypt identity: %w", err)
	}

	var id Identity
	if err := json.Unmarshal(data, &id); err != nil {
		return Identity{}, fmt.Errorf("unmarshal identity: %w", err)
	}

	return id, nil
}

func readOrCreateSalt(fsys zfilesystem.ReadWriteFileFS) ([]byte, error) {
	salt, err := fsys.ReadFile(saltFile)
	if err == nil {
		return salt, nil
	}

	salt, err = zcrypto.RandBytes(zcrypto.SaltSize)
	if err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	if err := fsys.WriteFile(saltFile, salt, 0o600); err != nil {
		return nil, fmt.Errorf("write salt: %w", err)
	}

	return salt, nil
}

func verifyOrCreateToken(fsys zfilesystem.ReadWriteFileFS, key []byte) error {
	ct, err := fsys.ReadFile(verifyFile)
	if err != nil {
		// first run — create the verification token
		ct, err = zcrypto.Encrypt(key, []byte(verifyToken))
		if err != nil {
			return fmt.Errorf("encrypt verify token: %w", err)
		}

		if err := fsys.WriteFile(verifyFile, ct, 0o600); err != nil {
			return fmt.Errorf("write verify token: %w", err)
		}

		return nil
	}

	// subsequent run — verify the password
	plain, err := zcrypto.Decrypt(key, ct)
	if err != nil {
		return errors.New("wrong password")
	}

	if string(plain) != verifyToken {
		return errors.New("wrong password")
	}

	return nil
}

func identityPath(id string) string {
	return identitiesDir + "/" + id + ".enc"
}
