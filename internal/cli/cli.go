// Package cli implements zburn's command-line subcommands.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"syscall"

	"github.com/zarlcorp/core/pkg/zfilesystem"
	"github.com/zarlcorp/core/pkg/zstore"
	"github.com/zarlcorp/zburn/internal/identity"
	"golang.org/x/term"
)

// DataDir returns the default data directory for zburn.
func DataDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return d + "/zburn"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".zburn"
	}
	return home + "/.local/share/zburn"
}

// ReadPassword prompts for a password on stderr and reads it without echo.
func ReadPassword(prompt string, w io.Writer) (string, error) {
	fmt.Fprint(w, prompt)
	b, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(w)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return string(b), nil
}

// ReadNewPassword prompts for a new password with confirmation.
func ReadNewPassword(w io.Writer) (string, error) {
	pass, err := ReadPassword("master password: ", w)
	if err != nil {
		return "", err
	}
	confirm, err := ReadPassword("confirm password: ", w)
	if err != nil {
		return "", err
	}
	if pass != confirm {
		return "", fmt.Errorf("passwords do not match")
	}
	return pass, nil
}

// IsFirstRun checks whether the store has been initialized.
func IsFirstRun(dir string) bool {
	_, err := os.Stat(dir + "/salt")
	return err != nil
}

// OpenStore prompts for a password and opens the store, returning both the
// store and an identities collection.
func OpenStore(dir string) (*zstore.Store, *zstore.Collection[identity.Identity], error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, nil, fmt.Errorf("create data dir: %w", err)
	}

	var pass string
	var err error
	if IsFirstRun(dir) {
		pass, err = ReadNewPassword(os.Stderr)
	} else {
		pass, err = ReadPassword("master password: ", os.Stderr)
	}
	if err != nil {
		return nil, nil, err
	}

	fsys := zfilesystem.NewOSFileSystem(dir)
	s, err := zstore.Open(fsys, []byte(pass))
	if err != nil {
		return nil, nil, err
	}

	col, err := zstore.NewCollection[identity.Identity](s, "identities")
	if err != nil {
		s.Close()
		return nil, nil, err
	}

	return s, col, nil
}

// CmdEmail generates and prints a random email.
func CmdEmail() {
	g := identity.New()
	fmt.Println(g.Email())
}

// CmdIdentity generates and prints a complete identity.
func CmdIdentity(args []string) {
	asJSON := hasFlag(args, "--json")
	save := hasFlag(args, "--save")

	g := identity.New()
	id := g.Generate()

	if asJSON {
		printJSON(id)
	} else {
		printIdentity(id)
	}

	if save {
		dir := DataDir()
		s, col, err := OpenStore(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "zburn: %v\n", err)
			os.Exit(1)
		}
		defer s.Close()

		if err := col.Put(id.ID, id); err != nil {
			fmt.Fprintf(os.Stderr, "zburn: save: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "saved")
	}
}

// CmdList lists all saved identities.
func CmdList(args []string) {
	asJSON := hasFlag(args, "--json")

	dir := DataDir()
	s, col, err := OpenStore(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "zburn: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	ids, err := col.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "zburn: list: %v\n", err)
		os.Exit(1)
	}

	sort.Slice(ids, func(i, j int) bool {
		return ids[i].CreatedAt.After(ids[j].CreatedAt)
	})

	if len(ids) == 0 {
		fmt.Println("no saved identities")
		return
	}

	if asJSON {
		printJSON(ids)
		return
	}

	for _, id := range ids {
		fmt.Printf("  %-10s %-20s %-30s %s\n",
			id.ID,
			id.FirstName+" "+id.LastName,
			id.Email,
			id.CreatedAt.Format("2006-01-02"),
		)
	}
}

// CmdForget deletes a saved identity by ID.
func CmdForget(id string) {
	dir := DataDir()
	s, col, err := OpenStore(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "zburn: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	if err := col.Delete(id); err != nil {
		fmt.Fprintf(os.Stderr, "zburn: forget: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("deleted %s\n", id)
}

func printIdentity(id identity.Identity) {
	fmt.Printf("  id:       %s\n", id.ID)
	fmt.Printf("  name:     %s %s\n", id.FirstName, id.LastName)
	fmt.Printf("  email:    %s\n", id.Email)
	fmt.Printf("  phone:    %s\n", id.Phone)
	fmt.Printf("  address:  %s, %s, %s %s\n", id.Street, id.City, id.State, id.Zip)
	fmt.Printf("  dob:      %s\n", id.DOB.Format("2006-01-02"))
	fmt.Printf("  password: %s\n", id.Password)
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "zburn: encode json: %v\n", err)
		os.Exit(1)
	}
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if strings.EqualFold(a, flag) {
			return true
		}
	}
	return false
}
