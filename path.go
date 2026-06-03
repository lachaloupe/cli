package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// PathValidate validates a path argument according to its configured path labels.
func PathValidate(arg *Arg, path string) error {
	labels := arg.Labels["path"]
	if len(labels) == 0 {
		return nil
	}

	exts := []string{}
	mode := os.FileMode(0755)
	mkdir := false

	for _, label := range labels {
		if label == "mkdir" {
			mkdir = true
			continue
		}

		if after, ok := strings.CutPrefix(label, "mkdir:"); ok {
			mkdir = true
			m, err := strconv.ParseUint(after, 8, 32)
			if err != nil {
				panic("invalid mkdir mode: " + after)
			}

			mode = os.FileMode(m)
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
			limit := 0
			if filepath.IsAbs(path) {
				depth = 1
				limit = 1
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
					if depth == limit {
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

	if mkdir {
		target := path

		if !slices.Contains(labels, "dir") {
			target = filepath.Dir(path)
		}

		if err := os.MkdirAll(target, mode); err != nil {
			return err
		}
	}

	info, err := os.Stat(path)

	for _, label := range labels {
		switch label {
		case "exists":
			if err != nil {
				if os.IsNotExist(err) {
					return &PathError{Kind: ErrPathExists, Arg: arg.Name, Path: path, Label: label}
				}

				return err
			}
		case "dir":
			if err != nil {
				if os.IsNotExist(err) {
					return &PathError{Kind: ErrPathExists, Arg: arg.Name, Path: path, Label: label}
				}

				return err
			}

			if !info.IsDir() {
				return &PathError{Kind: ErrPathDir, Arg: arg.Name, Path: path, Label: label}
			}
		case "file":
			if err != nil {
				if os.IsNotExist(err) {
					return &PathError{Kind: ErrPathExists, Arg: arg.Name, Path: path, Label: label}
				}

				return err
			}

			if !info.Mode().IsRegular() {
				return &PathError{Kind: ErrPathFile, Arg: arg.Name, Path: path, Label: label}
			}
		case "empty":
			if err != nil {
				if os.IsNotExist(err) {
					return &PathError{Kind: ErrPathExists, Arg: arg.Name, Path: path, Label: label}
				}

				return err
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
		case "exec":
			if err != nil {
				if os.IsNotExist(err) {
					return &PathError{Kind: ErrPathExists, Arg: arg.Name, Path: path, Label: label}
				}

				return err
			}

			if info.Mode()&0111 == 0 {
				return &PathError{Kind: ErrPathExecutable, Arg: arg.Name, Path: path, Label: label}
			}
		}
	}

	for _, label := range labels {
		switch label {
		case "not-exists":
			if _, err := os.Stat(path); err == nil {
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
