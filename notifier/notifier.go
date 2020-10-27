package notifier

import (
	"os"
	"path/filepath"
)

func GetIcon() string {
	return GetIconPath() + "/icon.png"
}

func GetIconPath() string {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return dir + "/icons"
}
