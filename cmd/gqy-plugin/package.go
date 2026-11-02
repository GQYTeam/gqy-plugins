package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GQYTeam/gqy-plugins/internal/archive"
	"github.com/GQYTeam/gqy-plugins/internal/manifest"
)

func runPackage(args []string) error {
	fs := flag.NewFlagSet("package", flag.ContinueOnError)
	output := fs.String("output", "", "输出 tar.gz 路径")
	asJSON := fs.Bool("json", false, "输出 JSON")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	if len(pos) == 0 || strings.TrimSpace(pos[0]) == "" {
		return fmt.Errorf("package 需要插件目录")
	}
	root := strings.TrimSpace(pos[0])
	outPath := strings.TrimSpace(*output)
	if outPath == "" {
		data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
		if err != nil {
			return err
		}
		m, err := manifest.Parse(data)
		if err != nil {
			return err
		}
		outPath = filepath.Join("dist", m.ID+"-"+m.Version+".tar.gz")
	}
	res, err := archive.Pack(root, outPath)
	if err != nil {
		return err
	}
	if *asJSON {
		return writeJSON(res)
	}
	fmt.Printf("已打包 %s@%s -> %s\n", res.ID, res.Version, res.Path)
	fmt.Printf("size: %d\nsha256: %s\n", res.Size, res.SHA256)
	return nil
}
