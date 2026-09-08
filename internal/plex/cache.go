package plex

import (
	"fmt"
	"os"
	"path/filepath"
)

// ClearCache empties Plex's photo transcoder cache and returns how many
// top-level entries went. Plex keeps every downloaded original and every
// resized copy there forever, under names that cannot be derived from the
// URL, so a changed portrait only shows once the whole directory is cleared.
// Plex rebuilds it on demand. The directory must be named PhotoTranscoder,
// so a mistyped setting cannot empty something else.
func ClearCache(dir string) (int, error) {
	if filepath.Base(filepath.Clean(dir)) != "PhotoTranscoder" {
		return 0, fmt.Errorf("%s is not a PhotoTranscoder directory; refusing to clear it", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
