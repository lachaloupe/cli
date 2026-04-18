package cli

import (
	"errors"
	"fmt"
	"os"
)

var (
	ErrHelp = errors.New("help")

	ErrMissingRequired = errors.New("missing required argument")
	ErrUnknownFlag     = errors.New("unknown flag")
	ErrMissingValue    = errors.New("missing flag value")
	ErrUnexpectedArg   = errors.New("unexpected argument")

	ErrPathAbs          = errors.New("path must be absolute")
	ErrPathRel          = errors.New("path must be relative")
	ErrPathClean        = errors.New("path must not escape its root")
	ErrPathGlob         = errors.New("path must be a valid glob")
	ErrPathExt          = errors.New("path must use an allowed extension")
	ErrPathExists       = errors.New("path must exist")
	ErrPathDir          = errors.New("path must be a directory")
	ErrPathFile         = errors.New("path must be a file")
	ErrPathEmpty        = errors.New("path must be an empty directory")
	ErrPathParentExists = errors.New("path parent must exist")
	ErrPathParentDir    = errors.New("path parent must be a directory")
	ErrPathParentWrite  = errors.New("path parent must be writeable")
	ErrPathReadable     = errors.New("path must be readable")
	ErrPathWriteable    = errors.New("path must be writeable")
	ErrPathExecutable   = errors.New("path must be executable")
	ErrPathNotExists    = errors.New("path must not exist")
	ErrPathSymlink      = errors.New("path must be a symlink")
)

// ParseError reports why parsing failed and, when available, which input caused it.
type ParseError struct {
	Kind  error
	Name  string
	Value string
}

func (err *ParseError) Error() string {
	switch err.Kind {
	case ErrMissingRequired:
		return fmt.Sprintf("missing required argument %q", err.Name)
	case ErrUnknownFlag:
		return fmt.Sprintf("unknown flag %q", err.Name)
	case ErrMissingValue:
		return fmt.Sprintf("missing value for flag %q", err.Name)
	case ErrUnexpectedArg:
		return fmt.Sprintf("unexpected argument %q", err.Value)
	default:
		return "parse error"
	}
}

func (err *ParseError) Unwrap() error {
	return err.Kind
}

// ArgError reports that a specific argument value could not be parsed or validated.
type ArgError struct {
	Arg   string
	Value string
	Err   error
}

func (err *ArgError) Error() string {
	return fmt.Sprintf("invalid value for argument %q: %v", err.Arg, err.Err)
}

func (err *ArgError) Unwrap() error {
	return err.Err
}

// PathError reports that a path argument violates one of its configured path constraints.
type PathError struct {
	Kind  error
	Arg   string
	Path  string
	Exts  []string
	Label string
	Err   error
}

func (err *PathError) Error() string {
	detail := func() string {
		if err.Err == nil {
			return ""
		}

		if errors.Is(err.Err, os.ErrPermission) {
			return "permission denied"
		}

		return err.Err.Error()
	}

	switch err.Kind {
	case ErrPathAbs:
		return fmt.Sprintf("%q must reference an absolute path", err.Arg)
	case ErrPathRel:
		return fmt.Sprintf("%q must reference a relative path", err.Arg)
	case ErrPathClean:
		return fmt.Sprintf("%q must not backtrack outside its root", err.Arg)
	case ErrPathGlob:
		return fmt.Sprintf("%q must be a valid glob pattern: %v", err.Arg, err.Err)
	case ErrPathExt:
		return fmt.Sprintf("%q must use one of these extensions: %v", err.Arg, err.Exts)
	case ErrPathExists:
		return fmt.Sprintf("%q must reference an existing path", err.Arg)
	case ErrPathDir:
		return fmt.Sprintf("%q must reference a directory", err.Arg)
	case ErrPathFile:
		return fmt.Sprintf("%q must reference a file", err.Arg)
	case ErrPathEmpty:
		return fmt.Sprintf("%q must reference an empty directory", err.Arg)
	case ErrPathParentExists:
		return fmt.Sprintf("%q must have an existing parent directory", err.Arg)
	case ErrPathParentDir:
		return fmt.Sprintf("%q must have a directory parent", err.Arg)
	case ErrPathParentWrite:
		return fmt.Sprintf("%q must have a writeable parent directory: %s", err.Arg, detail())
	case ErrPathReadable:
		return fmt.Sprintf("%q must reference a readable path: %s", err.Arg, detail())
	case ErrPathWriteable:
		return fmt.Sprintf("%q must reference a writeable path: %s", err.Arg, detail())
	case ErrPathExecutable:
		return fmt.Sprintf("%q must reference an executable path", err.Arg)
	case ErrPathNotExists:
		return fmt.Sprintf("%q must not reference an existing path", err.Arg)
	case ErrPathSymlink:
		return fmt.Sprintf("%q must reference a symlink", err.Arg)
	default:
		return fmt.Sprintf("%q failed path validation", err.Arg)
	}
}

func (err *PathError) Unwrap() []error {
	if err.Err == nil {
		return []error{err.Kind}
	}

	return []error{err.Kind, err.Err}
}
