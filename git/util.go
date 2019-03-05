package git

import (
	"os"
	"path/filepath"
)

func chmodRecursively(path string, mode os.FileMode) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chmod(path, mode)
	})
}
