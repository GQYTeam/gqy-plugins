package archive

import (
	"archive/tar"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type entry struct {
	abs string
	rel string
	dir bool
}

func collect(root string) ([]entry, error) {
	var out []entry
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if skipEntry(rel, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return formatError("不允许符号链接: " + rel)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			out = append(out, entry{abs: p, rel: rel, dir: true})
			return nil
		}
		if info.Mode().IsRegular() == false {
			return formatError("不支持的文件类型: " + rel)
		}
		out = append(out, entry{abs: p, rel: rel})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].rel < out[j].rel })
	return out, nil
}

func skipEntry(rel string, d os.DirEntry) bool {
	name := d.Name()
	if d.IsDir() {
		switch name {
		case ".git", "node_modules", ".gqy", "dist", "coverage", ".idea", ".vscode":
			return true
		}
	}
	if name == ".DS_Store" || name == "Thumbs.db" || strings.HasSuffix(name, ".log") {
		return true
	}
	if name == ".env" || (strings.HasPrefix(name, ".env.") && name != ".env.example") {
		return true
	}
	return rel == "coverage.out"
}

func headerFor(e entry) (*tar.Header, *os.File, error) {
	hdr := &tar.Header{
		Name:    e.rel,
		Mode:    0o644,
		Uid:     0,
		Gid:     0,
		Uname:   "",
		Gname:   "",
		ModTime: timeZero(),
		Format:  tar.FormatPAX,
	}
	if e.dir {
		hdr.Name += "/"
		hdr.Typeflag = tar.TypeDir
		hdr.Mode = 0o755
		return hdr, nil, nil
	}
	info, err := os.Stat(e.abs)
	if err != nil {
		return nil, nil, err
	}
	hdr.Typeflag = tar.TypeReg
	hdr.Size = info.Size()
	if info.Mode().Perm()&0o111 != 0 {
		hdr.Mode = 0o755
	}
	f, err := os.Open(e.abs)
	if err != nil {
		return nil, nil, err
	}
	return hdr, f, nil
}

func formatError(msg string) error {
	return &packError{msg: msg}
}

type packError struct{ msg string }

func (e *packError) Error() string { return "archive: " + e.msg }

func timeZero() time.Time { return time.Unix(0, 0).UTC() }
