package main

import (
	"flag"
	"testing"
)

func TestParseInterspersed(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	force := fs.Bool("force", false, "force")
	name := fs.String("name", "", "name")
	pos, err := parseInterspersed(fs, []string{"demo", "--force", "--name", "x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(pos) != 1 || pos[0] != "demo" {
		t.Fatalf("pos = %v", pos)
	}
	if *force == false || *name != "x" {
		t.Fatalf("flags force=%v name=%q", *force, *name)
	}
}

func TestSplitIDVersion(t *testing.T) {
	id, ver := splitIDVersion("demo@1.2.3")
	if id != "demo" || ver != "1.2.3" {
		t.Fatalf("split = %q %q", id, ver)
	}
	id, ver = splitIDVersion("demo")
	if id != "demo" || ver != "" {
		t.Fatalf("split = %q %q", id, ver)
	}
}
