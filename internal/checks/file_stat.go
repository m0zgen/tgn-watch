package checks

import "os"

func osStat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}
