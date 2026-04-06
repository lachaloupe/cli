package cli

import (
	"os"
	"path/filepath"
	"slices"
)

// PathValidate validates a path argument according to its configured path labels.
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
				return &PathError{Kind: ErrPathAbs, Arg: arg.Name, Path: path, Label: label}
			}
		case "rel":
			if filepath.IsAbs(path) {
				return &PathError{Kind: ErrPathRel, Arg: arg.Name, Path: path, Label: label}
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
						return &PathError{Kind: ErrPathClean, Arg: arg.Name, Path: path, Label: label}
					}
					depth--
				default:
					depth++
				}
			}
		case "glob":
			if _, err := filepath.Match(path, ""); err != nil {
				return &PathError{Kind: ErrPathGlob, Arg: arg.Name, Path: path, Label: label, Err: err}
			}
		default:
			if len(label) != 0 && label[0] == '.' {
				exts = append(exts, label)
			}
		}
	}

	if len(exts) != 0 {
		if got := filepath.Ext(path); !slices.Contains(exts, got) {
			return &PathError{Kind: ErrPathExt, Arg: arg.Name, Path: path, Exts: exts}
		}
	}

	info, statErr := os.Stat(path)
	for _, label := range labels {
		switch label {
		case "exists":
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return &PathError{Kind: ErrPathExists, Arg: arg.Name, Path: path, Label: label}
				}
				return statErr
			}
		case "dir":
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return &PathError{Kind: ErrPathExists, Arg: arg.Name, Path: path, Label: label}
				}
				return statErr
			}
			if !info.IsDir() {
				return &PathError{Kind: ErrPathDir, Arg: arg.Name, Path: path, Label: label}
			}
		case "file":
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return &PathError{Kind: ErrPathExists, Arg: arg.Name, Path: path, Label: label}
				}
				return statErr
			}
			if !info.Mode().IsRegular() {
				return &PathError{Kind: ErrPathFile, Arg: arg.Name, Path: path, Label: label}
			}
		case "empty":
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return &PathError{Kind: ErrPathExists, Arg: arg.Name, Path: path, Label: label}
				}
				return statErr
			}
			if !info.IsDir() {
				return &PathError{Kind: ErrPathEmpty, Arg: arg.Name, Path: path, Label: label}
			}
			entries, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			if len(entries) != 0 {
				return &PathError{Kind: ErrPathEmpty, Arg: arg.Name, Path: path, Label: label}
			}
		case "creatable":
			parent := filepath.Dir(path)
			parentInfo, err := os.Stat(parent)
			if err != nil {
				if os.IsNotExist(err) {
					return &PathError{Kind: ErrPathParentExists, Arg: arg.Name, Path: path, Label: label}
				}
				return err
			}
			if !parentInfo.IsDir() {
				return &PathError{Kind: ErrPathParentDir, Arg: arg.Name, Path: path, Label: label}
			}
			if err := ensureWriteable(parent, parentInfo); err != nil {
				return &PathError{Kind: ErrPathParentWrite, Arg: arg.Name, Path: path, Label: label, Err: err}
			}
		case "readable":
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return &PathError{Kind: ErrPathExists, Arg: arg.Name, Path: path, Label: label}
				}
				return statErr
			}
			if info.IsDir() {
				if _, err := os.ReadDir(path); err != nil {
					return &PathError{Kind: ErrPathReadable, Arg: arg.Name, Path: path, Label: label, Err: err}
				}
			} else {
				f, err := os.Open(path)
				if err != nil {
					return &PathError{Kind: ErrPathReadable, Arg: arg.Name, Path: path, Label: label, Err: err}
				}
				if err := f.Close(); err != nil {
					return err
				}
			}
		case "writeable":
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return &PathError{Kind: ErrPathExists, Arg: arg.Name, Path: path, Label: label}
				}
				return statErr
			}
			if err := ensureWriteable(path, info); err != nil {
				return &PathError{Kind: ErrPathWriteable, Arg: arg.Name, Path: path, Label: label, Err: err}
			}
		case "exec":
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return &PathError{Kind: ErrPathExists, Arg: arg.Name, Path: path, Label: label}
				}
				return statErr
			}
			if info.Mode()&0111 == 0 {
				return &PathError{Kind: ErrPathExecutable, Arg: arg.Name, Path: path, Label: label}
			}
		}
	}

	for _, label := range labels {
		switch label {
		case "not-exists":
			if _, err := os.Lstat(path); err == nil {
				return &PathError{Kind: ErrPathNotExists, Arg: arg.Name, Path: path, Label: label}
			} else if !os.IsNotExist(err) {
				return err
			}
		case "symlink":
			info, err := os.Lstat(path)
			if err != nil {
				if os.IsNotExist(err) {
					return &PathError{Kind: ErrPathExists, Arg: arg.Name, Path: path, Label: label}
				}
				return err
			}
			if info.Mode()&os.ModeSymlink == 0 {
				return &PathError{Kind: ErrPathSymlink, Arg: arg.Name, Path: path, Label: label}
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
