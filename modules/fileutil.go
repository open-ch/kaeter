package modules

import "os"

func fileExists(targetPath string) bool {
	_, err := os.Stat(targetPath)
	return !os.IsNotExist(err)
}
