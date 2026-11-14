package common

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestParseBV(t *testing.T) {
	ids, err := ParseBV("BV1CpR1BiEuf，https://www.bilibili.com/video/BV1CpR1BiEuf/?x=1 BV1234567890")
	if err != nil || len(ids) != 2 || ids[0] != "BV1CpR1BiEuf" {
		t.Fatalf("%v %v", ids, err)
	}
	for _, input := range []string{"", "   ", "all", "bv1CpR1BiEuf", "BV1CpR1BiEufEXTRA", "BV1CpR1BiEuf invalid"} {
		if _, err := ParseBV(input); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
}

func fixture(t *testing.T, root, name, bv, status string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"bvid":"` + bv + `","status":"` + status + `","title":"same title","p":1}`)
	for name, data := range map[string][]byte{"videoInfo.json": content, ".videoInfo": content, "v.m4s": []byte("000000000video"), "a.m4s": []byte("000000000audio")} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestDiscoveryAndSelection(t *testing.T) {
	root := t.TempDir()
	one := fixture(t, root, "1", "BV1CpR1BiEuf", "completed")
	fixture(t, root, "2", "BV1CpR1BiEuf", "completed")
	fixture(t, root, "3", "BV1234567890", "downloading")
	fixture(t, root, "output/old", "BV0000000000", "completed")
	custom := filepath.Join(root, "exports")
	fixture(t, custom, "old", "BV0000000000", "completed")
	entries, err := discoverCache(root, custom)
	if err != nil || len(entries) != 3 {
		t.Fatalf("%v %v", entries, err)
	}
	selected, err := selectEntries(entries, []string{"BV1CpR1BiEuf"})
	if err != nil || len(selected) != 2 {
		t.Fatalf("%v %v", selected, err)
	}
	for _, id := range []string{"BV1234567890", "BV0000000000"} {
		if _, err := selectEntries(entries, []string{"BV1CpR1BiEuf", id}); err == nil {
			t.Fatal("missing/incomplete must fail before conversion")
		}
	}
	if err := os.WriteFile(filepath.Join(one, ".videoInfo"), []byte(`{"bvid":"BV1234567890"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := discoverCache(root, custom); err == nil {
		t.Fatal("conflicting metadata accepted")
	}
}

func TestFilterDoesNotTouchUnselectedMedia(t *testing.T) {
	root := t.TempDir()
	chosen := fixture(t, root, "1", "BV1CpR1BiEuf", "completed")
	other := fixture(t, root, "2", "BV1234567890", "completed")
	c := Config{CachePath: root, selected: map[string]cacheEntry{chosen: {BV: "BV1CpR1BiEuf"}}}
	// Passing an unselected media entry to the actual converter callback must be a no-op.
	list, err := os.ReadDir(other)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range list {
		if err := c.FindM4sFiles(filepath.Join(other, e.Name()), e, nil); err != nil {
			t.Fatal(err)
		}
	}
	after, _ := os.ReadDir(other)
	if len(after) != 4 {
		t.Fatal("unselected cache was modified")
	}
	data, _ := os.ReadFile(filepath.Join(other, "v.m4s"))
	if string(data) != "000000000video" {
		t.Fatal("source modified")
	}
}

func TestCopyPreservesTailAndOriginal(t *testing.T) {
	// Regression: the old buffered writer could silently drop the final bytes.
	for _, prefix := range []string{"", "000000000"} {
		dir := t.TempDir()
		src := filepath.Join(dir, "in")
		dst := filepath.Join(dir, "out")
		payload := bytes.Repeat([]byte{1, 2, 3, 4, 5, 6, 7}, 713)
		original := append([]byte(prefix), payload...)
		if err := os.WriteFile(src, original, 0644); err != nil {
			t.Fatal(err)
		}
		c := Config{}
		if err := c.copyFile(src, dst); err != nil {
			t.Fatal(err)
		}
		actual, _ := os.ReadFile(dst)
		source, _ := os.ReadFile(src)
		if !bytes.Equal(actual, payload) || !bytes.Equal(source, original) {
			t.Fatal("truncated copy or changed original")
		}
	}
}
