package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/GQYTeam/gqy-plugins/internal/install"
	"github.com/GQYTeam/gqy-plugins/internal/registry"
)

type verifyOutput struct {
	OK       bool                   `json:"ok"`
	Plugins  int                    `json:"plugins"`
	Versions int                    `json:"versions"`
	Results  []install.VerifyResult `json:"artifacts,omitempty"`
}

func runVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	source := fs.String("registry", defaultRegistry(), "registry.json URL 或本地路径")
	strict := fs.Bool("strict", false, "发布口径：要求 https 绝对产物地址")
	checkArtifacts := fs.Bool("artifacts", false, "下载每个产物并校验 sha256、大小与 manifest")
	asJSON := fs.Bool("json", false, "输出 JSON")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return fmt.Errorf("verify 不接受位置参数")
	}
	doc, err := registry.Load(context.Background(), *source)
	if err != nil {
		return err
	}
	if *strict {
		if err := doc.Validate(true); err != nil {
			return err
		}
	}
	out := verifyOutput{OK: true, Plugins: len(doc.Plugins)}
	for i := range doc.Plugins {
		out.Versions += len(doc.Plugins[i].Versions)
	}
	if *checkArtifacts {
		for i := range doc.Plugins {
			p := &doc.Plugins[i]
			for j := range p.Versions {
				v := &p.Versions[j]
				res, err := install.VerifyArtifact(context.Background(), doc, p, v)
				if err != nil {
					out.OK = false
					if *asJSON {
						_ = writeJSON(out)
					}
					return err
				}
				out.Results = append(out.Results, *res)
			}
		}
	}
	if *asJSON {
		return writeJSON(out)
	}
	fmt.Printf("目录校验通过：%d 个插件，%d 个版本", out.Plugins, out.Versions)
	if *checkArtifacts {
		fmt.Printf("，%d 个产物逐一下载校验通过", len(out.Results))
	}
	fmt.Println()
	return nil
}
