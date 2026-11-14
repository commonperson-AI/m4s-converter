package common

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var bvPattern = regexp.MustCompile(`^BV[0-9A-Za-z]{10}$`)
var bvURLPattern = regexp.MustCompile(`(?:^|/)BV[0-9A-Za-z]{10}(?:[/?#]|$)`)

type cacheEntry struct {
	Dir    string
	BV     string
	Title  string
	Part   int
	Status string
}

// ParseBV accepts exact BV IDs and full video links; BV IDs are case sensitive.
func ParseBV(input string) ([]string, error) {
	tokens := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == ' ' || r == '\n' || r == '\t' || r == '\r'
	})
	seen := map[string]bool{}
	var ids []string
	for _, token := range tokens {
		id := token
		if !bvPattern.MatchString(id) {
			match := bvURLPattern.FindString(token)
			match = strings.Trim(match, "/?#")
			if !bvPattern.MatchString(match) {
				return nil, fmt.Errorf("无效 BV 号或链接：%s", token)
			}
			id = match
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("未输入 BV 号，已取消；不会转换全部视频")
	}
	return ids, nil
}

func within(path, root string) bool {
	if root == "" {
		return false
	}
	p, e1 := filepath.Abs(path)
	r, e2 := filepath.Abs(root)
	if e1 != nil || e2 != nil {
		return false
	}
	rel, err := filepath.Rel(r, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// discoverCache only accepts the Windows layout: metadata and media in one directory.
// It never follows symlinks or scans an output directory.
func discoverCache(root, output string) ([]cacheEntry, error) {
	entries := map[string]cacheEntry{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && (strings.EqualFold(d.Name(), "output") || within(path, output)) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 || (d.Name() != "videoInfo.json" && d.Name() != ".videoInfo") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var m struct {
			BV     string `json:"bvid"`
			Group  string `json:"groupId"`
			Title  string `json:"title"`
			Part   int    `json:"p"`
			Status string `json:"status"`
		}
		if json.Unmarshal(data, &m) != nil {
			fmt.Fprintf(os.Stderr, "跳过损坏的元数据：%s\n", path)
			return nil
		}
		if m.BV == "" {
			m.BV = m.Group
		}
		if !bvPattern.MatchString(m.BV) {
			return nil
		}
		dir := filepath.Dir(path)
		files, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		count := 0
		for _, f := range files {
			if f.Type().IsRegular() && strings.EqualFold(filepath.Ext(f.Name()), ".m4s") {
				count++
			}
		}
		if count < 2 {
			return nil
		}
		entry := cacheEntry{dir, m.BV, m.Title, m.Part, m.Status}
		if previous, ok := entries[dir]; ok && previous.BV != entry.BV {
			return fmt.Errorf("元数据 BV 号冲突：%s", dir)
		}
		// Same preference as the converter: videoInfo.json wins over .videoInfo.
		if _, ok := entries[dir]; !ok || d.Name() == "videoInfo.json" {
			entries[dir] = entry
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := make([]cacheEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Dir < result[j].Dir })
	return result, nil
}

func selectEntries(entries []cacheEntry, ids []string) ([]cacheEntry, error) {
	wanted := map[string]bool{}
	for _, id := range ids {
		wanted[id] = false
	}
	var selected []cacheEntry
	for _, e := range entries {
		if _, ok := wanted[e.BV]; !ok {
			continue
		}
		if e.Status != "" && e.Status != "completed" && e.Status != "视频已缓存完成" {
			continue
		}
		selected = append(selected, e)
		wanted[e.BV] = true
	}
	var missing []string
	for _, id := range ids {
		if !wanted[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) != 0 {
		return nil, fmt.Errorf("以下 BV 没有可转换的完整缓存：%s；本次未开始转换", strings.Join(missing, ", "))
	}
	return selected, nil
}

func (c *Config) prepareSelection() ([]string, error) {
	entries, err := discoverCache(c.CachePath, c.OutputRoot)
	if err != nil {
		return nil, err
	}
	if c.ListOnly || c.SelectBV {
		fmt.Println("可识别的缓存（BV / 分P / 状态 / 标题）：")
		for _, e := range entries {
			fmt.Printf("%s / P%d / %s / %s\n", e.BV, e.Part, e.Status, e.Title)
		}
	}
	if c.ListOnly {
		return nil, nil
	}
	input := c.BVs
	if c.SelectBV && input == "" {
		fmt.Print("输入 BV 号或视频链接，多个用空格分隔；直接回车取消：")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			return nil, fmt.Errorf("未输入 BV 号，已取消")
		}
		input = scanner.Text()
	}
	ids, err := ParseBV(input)
	if err != nil {
		return nil, err
	}
	selected, err := selectEntries(entries, ids)
	if err != nil {
		return nil, err
	}
	c.selected = map[string]cacheEntry{}
	var dirs []string
	fmt.Println("本次仅转换：")
	for _, entry := range selected {
		c.selected[entry.Dir] = entry
		dirs = append(dirs, entry.Dir)
		fmt.Printf("%s / P%d / %s\n", entry.BV, entry.Part, entry.Title)
	}
	return dirs, nil
}
