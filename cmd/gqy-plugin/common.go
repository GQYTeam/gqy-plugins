package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/GQYTeam/gqy-plugins/internal/registry"
)

func defaultRegistry() string {
	if v := strings.TrimSpace(os.Getenv("GQY_PLUGIN_REGISTRY")); v != "" {
		return v
	}
	return registry.DefaultURL
}

func writeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func latestVersion(p *registry.Plugin) string {
	v, err := p.ResolveVersion("")
	if err != nil {
		return "-"
	}
	return v.Version
}

func renderPlugins(plugins []registry.Plugin) {
	if len(plugins) == 0 {
		fmt.Println("目录为空。")
		return
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tVERSION\tMODE\tSUMMARY")
	for i := range plugins {
		p := &plugins[i]
		mode := "resident"
		if v, err := p.ResolveVersion(""); err == nil && v.Mode != "" {
			mode = v.Mode
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", p.ID, latestVersion(p), mode, p.Summary)
	}
	_ = tw.Flush()
}

func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func splitIDVersion(raw string) (string, string) {
	if i := strings.LastIndex(raw, "@"); i > 0 {
		return raw[:i], raw[i+1:]
	}
	return raw, ""
}

// parseInterspersed 允许 flag 出现在位置参数之后（标准 flag 包只解析到第一个位置参数）。
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var flags []string
	var positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if len(a) > 0 && strings.HasPrefix(a, "-") && a != "-" {
			flags = append(flags, a)
			name := strings.TrimLeft(strings.SplitN(a, "=", 2)[0], "-")
			f := fs.Lookup(name)
			if f == nil || strings.Contains(a, "=") {
				continue
			}
			if boolFlag, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && boolFlag.IsBoolFlag() {
				continue
			}
			if i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		positional = append(positional, a)
	}
	if err := fs.Parse(flags); err != nil {
		return nil, err
	}
	return positional, nil
}
