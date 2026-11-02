package main

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/GQYTeam/gqy-plugins/internal/registry"
)

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	source := fs.String("registry", defaultRegistry(), "registry.json URL 或本地路径")
	asJSON := fs.Bool("json", false, "输出 JSON")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return fmt.Errorf("list 不接受位置参数")
	}
	doc, err := registry.Load(context.Background(), *source)
	if err != nil {
		return err
	}
	if *asJSON {
		return writeJSON(doc)
	}
	sort.SliceStable(doc.Plugins, func(i, j int) bool { return doc.Plugins[i].ID < doc.Plugins[j].ID })
	renderPlugins(doc.Plugins)
	return nil
}

func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	source := fs.String("registry", defaultRegistry(), "registry.json URL 或本地路径")
	asJSON := fs.Bool("json", false, "输出 JSON")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	if len(pos) == 0 || strings.TrimSpace(pos[0]) == "" {
		return fmt.Errorf("search 需要关键词")
	}
	query := strings.TrimSpace(pos[0])
	doc, err := registry.Load(context.Background(), *source)
	if err != nil {
		return err
	}
	var out []registry.Plugin
	for _, p := range doc.Plugins {
		if matches(p, query) {
			out = append(out, p)
		}
	}
	if *asJSON {
		return writeJSON(out)
	}
	renderPlugins(out)
	return nil
}

func matches(p registry.Plugin, query string) bool {
	if containsFold(p.ID, query) || containsFold(p.Name, query) ||
		containsFold(p.Summary, query) || containsFold(p.Description, query) {
		return true
	}
	for _, tag := range p.Tags {
		if containsFold(tag, query) {
			return true
		}
	}
	return false
}

func runInfo(args []string) error {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	source := fs.String("registry", defaultRegistry(), "registry.json URL 或本地路径")
	asJSON := fs.Bool("json", false, "输出 JSON")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	if len(pos) == 0 || strings.TrimSpace(pos[0]) == "" {
		return fmt.Errorf("info 需要插件 id")
	}
	doc, err := registry.Load(context.Background(), *source)
	if err != nil {
		return err
	}
	p, err := doc.Find(strings.TrimSpace(pos[0]))
	if err != nil {
		return err
	}
	if *asJSON {
		return writeJSON(p)
	}
	fmt.Printf("%s (%s)\n%s\n", p.Name, p.ID, p.Summary)
	if p.Homepage != "" {
		fmt.Printf("主页: %s\n", p.Homepage)
	}
	if len(p.Tags) > 0 {
		fmt.Printf("标签: %s\n", strings.Join(p.Tags, ", "))
	}
	if p.Description != "" {
		fmt.Printf("说明: %s\n", p.Description)
	}
	fmt.Println("版本:")
	for i := range p.Versions {
		v := &p.Versions[i]
		state := "可安装"
		if v.Yanked {
			state = "已撤回"
		}
		fmt.Printf("  %s\t%s\t%s\t%d bytes\t%s\n", v.Version, v.Mode, v.Protocol, v.Size, state)
	}
	return nil
}
