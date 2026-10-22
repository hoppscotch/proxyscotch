package notifier

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

func GetIcon() string {
	// Backward compatibility: use the legacy icons dir if it contains
	// icon.png
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	legacyFile := filepath.Join(dir, "icons", "icon.png")
	if _, err := os.Stat(legacyFile); err == nil {
		return legacyFile
	}

	// Otherwise, use the file from a standard location, or return an empty
	// string if not found.
	path, _ := xdg.SearchDataFile("proxyscotch/icon.png")
	return path
}
