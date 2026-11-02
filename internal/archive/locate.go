package archive

import (
	"errors"
	"os"
	"path/filepath"
)

// LocatePluginRoot 找到解包目录里的插件根：包根直接有 manifest.json，
// 或只有一个顶层目录且该目录里有 manifest.json。
func LocatePluginRoot(dir string) (string, error) {
	if isFile(filepath.Join(dir, "manifest.json")) {
		return dir, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(dir, e.Name()))
		}
	}
	if len(dirs) == 1 && isFile(filepath.Join(dirs[0], "manifest.json")) {
		return dirs[0], nil
	}
	return "", errors.New("archive: 找不到唯一的插件根 manifest.json")
}

func isFile(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	return true
}
