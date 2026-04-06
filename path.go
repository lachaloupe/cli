package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

func PathValidate(arg *Arg, path string) error {
	labels := arg.Labels["path"]
	if len(labels) == 0 {
		return nil
	}

	exts := []string{}

	for _, label := range labels {
		if label == "mkdir" {
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
			continue
		}

		switch label {
		case "abs":
			if !filepath.IsAbs(path) {
				return fmt.Errorf("%q must reference an absolute path", arg.Name)
			}
		case "rel":
			if filepath.IsAbs(path) {
				return fmt.Errorf("%q must reference a relative path", arg.Name)
			}
		case "clean":
			volume := filepath.VolumeName(path)

			rest := path[len(volume):]
			depth := 0
			if filepath.IsAbs(path) {
				depth = 1
			}

			start := 0
			for i := 0; i <= len(rest); i++ {
				if i != len(rest) && rest[i] != filepath.Separator {
					continue
				}

				part := rest[start:i]
				start = i + 1

				switch part {
				case "", ".":
				case "..":
					if depth <= 1 {
						return fmt.Errorf("%q must not backtrack outside its root", arg.Name)
					}
					depth--
				default:
					depth++
				}
			}
		case "glob":
			if _, err := filepath.Match(path, ""); err != nil {
				return fmt.Errorf("%q must be a valid glob pattern: %w", arg.Name, err)
			}
		default:
			if len(label) != 0 && label[0] == '.' {
				exts = append(exts, label)
			}
		}
	}

	if len(exts) != 0 {
		if got := filepath.Ext(path); !slices.Contains(exts, got) {
			return fmt.Errorf("%q must use one of these extensions: %v", arg.Name, exts)
		}
	}

	info, statErr := os.Stat(path)
	for _, label := range labels {
		switch label {
		case "exists":
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return fmt.Errorf("%q must reference an existing path", arg.Name)
				}
				return statErr
			}
		case "dir":
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return fmt.Errorf("%q must reference an existing path", arg.Name)
				}
				return statErr
			}
			if !info.IsDir() {
				return fmt.Errorf("%q must reference a directory", arg.Name)
			}
		case "file":
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return fmt.Errorf("%q must reference an existing path", arg.Name)
				}
				return statErr
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("%q must reference a file", arg.Name)
			}
		case "empty":
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return fmt.Errorf("%q must reference an existing path", arg.Name)
				}
				return statErr
			}
			if !info.IsDir() {
				return fmt.Errorf("%q must reference an empty directory", arg.Name)
			}
			entries, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			if len(entries) != 0 {
				return fmt.Errorf("%q must reference an empty directory", arg.Name)
			}
		case "creatable":
			parent := filepath.Dir(path)
			parentInfo, err := os.Stat(parent)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("%q must have an existing parent directory", arg.Name)
				}
				return err
			}
			if !parentInfo.IsDir() {
				return fmt.Errorf("%q must have a directory parent", arg.Name)
			}
			if err := ensureWriteable(parent, parentInfo); err != nil {
				return fmt.Errorf("%q must have a writeable parent directory: %w", arg.Name, err)
			}
		case "readable":
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return fmt.Errorf("%q must reference an existing path", arg.Name)
				}
				return statErr
			}
			if info.IsDir() {
				if _, err := os.ReadDir(path); err != nil {
					return fmt.Errorf("%q must reference a readable path: %w", arg.Name, err)
				}
			} else {
				f, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("%q must reference a readable path: %w", arg.Name, err)
				}
				if err := f.Close(); err != nil {
					return err
				}
			}
		case "writeable":
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return fmt.Errorf("%q must reference an existing path", arg.Name)
				}
				return statErr
			}
			if err := ensureWriteable(path, info); err != nil {
				return fmt.Errorf("%q must reference a writeable path: %w", arg.Name, err)
			}
		case "exec":
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return fmt.Errorf("%q must reference an existing path", arg.Name)
				}
				return statErr
			}
			if info.Mode()&0111 == 0 {
				return fmt.Errorf("%q must reference an executable path", arg.Name)
			}
		}
	}

	for _, label := range labels {
		switch label {
		case "not-exists":
			if _, err := os.Lstat(path); err == nil {
				return fmt.Errorf("%q must not reference an existing path", arg.Name)
			} else if !os.IsNotExist(err) {
				return err
			}
		case "symlink":
			info, err := os.Lstat(path)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("%q must reference an existing path", arg.Name)
				}
				return err
			}
			if info.Mode()&os.ModeSymlink == 0 {
				return fmt.Errorf("%q must reference a symlink", arg.Name)
			}
		}
	}

	return nil
}

func isCleanPath(path string) bool {
	volume := filepath.VolumeName(path)
	rest := path[len(volume):]
	depth := 0
	if filepath.IsAbs(path) {
		depth = 1
	}

	start := 0
	for i := 0; i <= len(rest); i++ {
		if i != len(rest) && rest[i] != filepath.Separator {
			continue
		}

		part := rest[start:i]
		start = i + 1

		switch part {
		case "", ".":
		case "..":
			if depth <= 1 {
				return false
			}
			depth--
		default:
			depth++
		}
	}

	return true
}

func ensureWriteable(path string, info os.FileInfo) error {
	if info.IsDir() {
		f, err := os.CreateTemp(path, ".cli-writeable-*")
		if err != nil {
			return err
		}

		name := f.Name()
		if err := f.Close(); err != nil {
			_ = os.Remove(name)
			return err
		}

		return os.Remove(name)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}

	return f.Close()
}
