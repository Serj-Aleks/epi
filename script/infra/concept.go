package main

// Концепты в lexic (ason-spec 9.13): тема preposit, путь концепта
// из букв имени (как в ПИ), мета-файл .uic — модели парами строк
// «l-<ключ> левый слот» / «r-<ключ> правый слот». Системная часть — буквы
// (n name, q uri, h content, m alias, p preposit, f prefix, v suffix,
// b attrib, w defs, u process, o i-process, x counts), модели интерпретаций —
// индекс Гаусса из 4 знаков.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"epi/script/disp"
	"epi/script/format"
)

const gaussABC = "oiztyvslanxedbhfmg" // 18-ричный алфавит индекса Гаусса (как в ТЕМП)
const gaussLen = 4                    // порядок индекса

var systemSlots = []struct{ letter, name string }{
	{"n", "name"}, {"q", "uri"}, {"h", "content"}, {"m", "alias"}, {"p", "preposit"},
	{"f", "prefix"}, {"v", "suffix"}, {"b", "attrib"}, {"w", "defs"},
	{"u", "process"}, {"o", "i-process"}, {"x", "counts"},
}

func gauss(n int) string {
	b := make([]byte, gaussLen)
	for i := gaussLen - 1; i >= 0; i-- {
		b[i] = gaussABC[n%18]
		n /= 18
	}
	return string(b)
}

func isGauss(s string) bool {
	if len(s) != gaussLen {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune(gaussABC, c) {
			return false
		}
	}
	return true
}

// frameDir: <данные>/lexic/<тема>/<буквы имени>, файл — <имя>.uic (как в ПИ).
func (q *Script) frameFile(theme, name string) string {
	parts := []string{q.dir, "lexic", theme}
	for _, r := range strings.ToUpper(name) {
		parts = append(parts, string(r))
	}
	return filepath.Join(append(parts, strings.ToLower(name)+".uic")...)
}

func frameURI(name string) string {
	var p []string
	for _, r := range strings.ToUpper(name) {
		p = append(p, string(r))
	}
	return strings.Join(p, "/")
}

// loadFrame читает модели фрейма-интерпретатора из .uic.
func (q *Script) loadFrame(f *disp.Frame) error {
	path := q.frameFile("preposit", f.Name)
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	left, right := map[string]string{}, map[string]string{}
	var order []string
	sc := bufio.NewScanner(file)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if len(line) < 3 || line[1] != '-' || (line[0] != 'l' && line[0] != 'r') {
			continue
		}
		key, val, _ := strings.Cut(line[2:], " ")
		if !isGauss(key) {
			continue // системная часть пишется заново из ядра фрейма
		}
		if _, seen := left[key]; !seen && line[0] == 'l' {
			order = append(order, key)
		}
		if line[0] == 'l' {
			left[key] = val
		} else {
			right[key] = val
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	for _, k := range order {
		vs, err := format.Parse(right[k])
		if err != nil || len(vs) != 1 {
			return fmt.Errorf("%s: модель %s: правый слот «%s» — не одно значение", path, k, right[k])
		}
		f.Add(disp.Model{Pred: left[k], Value: vs[0]})
	}
	return nil
}

// writeFrame пишет мета-файл фрейма целиком (через временный файл).
func (q *Script) writeFrame(f *disp.Frame) error {
	path := q.frameFile("preposit", f.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	preps := make([]string, 0, len(f.Kernel))
	for p := range f.Kernel {
		preps = append(preps, p)
	}
	sort.Strings(preps)
	right := map[string]string{"name": f.Name, "uri": frameURI(f.Name),
		"preposit": strings.Join(preps, " "), "counts": gauss(len(f.Models))}
	var sb strings.Builder
	for _, s := range systemSlots {
		fmt.Fprintf(&sb, "l-%s %s\nr-%s %s\n", s.letter, s.name, s.letter, right[s.name])
	}
	for i, m := range f.Models {
		k := gauss(i + 1)
		fmt.Fprintf(&sb, "l-%s %s\nr-%s %s\n", k, m.Pred, k, m.Value.String())
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(sb.String()), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// loadAliases: концепты темы preposit со слотом alias (`r-m +`) становятся
// алиасами препозиций в общем контексте (9.8): `plus : +`.
func (q *Script) loadAliases() error {
	root := filepath.Join(q.dir, "lexic", "preposit")
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".uic") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var name, alias string
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "r-n ") {
				name = strings.TrimSpace(line[4:])
			}
			if strings.HasPrefix(line, "r-m ") {
				alias = strings.TrimSpace(line[4:])
			}
		}
		if name == "" || alias == "" {
			return nil
		}
		if _, err := q.in.Run(name+" : "+alias, q.env); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		return nil
	})
}

// migrateFrame: прежний файл фрейма в корне данных («[ … ] -> …») переносится
// в lexic и удаляется.
func (q *Script) migrateFrame(f *disp.Frame) error {
	old := filepath.Join(q.dir, f.Name)
	b, err := os.ReadFile(old)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	vs, err := format.Parse(string(b))
	if err != nil {
		return fmt.Errorf("%s: %w", old, err)
	}
	for i := 0; i+2 < len(vs); i += 3 {
		if vs[i].Kind == format.Array && vs[i+1].Kind == format.Op && vs[i+1].Text == "->" {
			f.Add(disp.Model{Pred: format.Format(vs[i].Items), Value: vs[i+2]})
		}
	}
	if err := q.writeFrame(f); err != nil {
		return err
	}
	return os.Remove(old)
}
