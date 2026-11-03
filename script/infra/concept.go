package main

// Файлы концептов (ason-spec 0.23): модель — одна строка в записи скрипта,
// ключей и индексов нет. Системная часть — только то, что не выводится из
// имени, места файла и моделей (имя и путь — из места, число моделей — число
// строк). Путь концепта — из букв имени (как в ПИ): <тема>/<буквы>/<имя>.uic.
//
//	интерпретатор  lexic/preposit/A/R/I/F/M/arifm.uic
//	               Preposit : [ * + - / ]
//	               [ 2 + 3 ] -> 5
//	алиас          lexic/preposit/P/L/U/S/plus.uic
//	               plus : +
//	концепт автора thema/<Тема>/X/x.uic
//	               A : 5
//
// Файлы прежнего вида (пары «l-<ключ> …» / «r-<ключ> …», ПИ) читаются один раз
// и переписываются в новом виде.

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

// letterPath: <корень>/<буквы имени>/<имя>.uic.
func letterPath(root, name string) string {
	parts := []string{root}
	for _, r := range strings.ToUpper(name) {
		parts = append(parts, string(r))
	}
	return filepath.Join(append(parts, strings.ToLower(name)+".uic")...)
}

func (q *Script) frameFile(theme, name string) string {
	return letterPath(filepath.Join(q.dir, "lexic", theme), name)
}

// writeLines: файл целиком через временный файл.
func writeLines(path string, lines []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// isLegacy: файл прежнего вида (ПИ) — строки «l-…» / «r-…».
func isLegacy(b []byte) bool { return strings.HasPrefix(string(b), "l-") }

// ---- прежний вид (только чтение) ----

const gaussABC = "oiztyvslanxedbhfmg"

func isGauss(s string) bool {
	if len(s) != 4 {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune(gaussABC, c) {
			return false
		}
	}
	return true
}

// readLegacy: системная часть (буква → значение) и модели по порядку записи.
func readLegacy(path string) (map[string]string, [][2]string, error) {
	sys := map[string]string{}
	var models [][2]string
	file, err := os.Open(path)
	if err != nil {
		return sys, nil, err
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
			if line[0] == 'r' {
				sys[key] = val
			}
			continue
		}
		if line[0] == 'l' {
			if _, seen := left[key]; !seen {
				order = append(order, key)
			}
			left[key] = val
		} else {
			right[key] = val
		}
	}
	for _, k := range order {
		models = append(models, [2]string{left[k], right[k]})
	}
	return sys, models, sc.Err()
}

// ---- интерпретаторы ----

// loadFrame читает модели фрейма-интерпретатора; прежний вид переписывается.
func (q *Script) loadFrame(f *disp.Frame) error {
	path := q.frameFile("preposit", f.Name)
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if isLegacy(b) {
		_, models, err := readLegacy(path)
		if err != nil {
			return err
		}
		for _, m := range models {
			vs, err := format.Parse(m[1])
			if err != nil || len(vs) != 1 {
				return fmt.Errorf("%s: правый слот «%s» — не одно значение", path, m[1])
			}
			f.Add(disp.Model{Pred: m[0], Value: vs[0]})
		}
		return q.writeFrame(f)
	}
	vs, err := format.Parse(string(b))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	for i := 0; i < len(vs); {
		switch {
		case i+2 < len(vs) && vs[i].Kind == format.Term && vs[i+1].Kind == format.Op && vs[i+1].Text == ":":
			i += 3 // системный слот (Preposit : [ … ]) — ядро задано кодом
		case i+2 < len(vs) && vs[i].Kind == format.Array && vs[i+1].Kind == format.Op && vs[i+1].Text == "->":
			f.Add(disp.Model{Pred: format.Format(vs[i].Items), Value: vs[i+2]})
			i += 3
		default:
			return fmt.Errorf("%s: строка %d: ожидалась модель «[ … ] -> …»", path, vs[i].Line)
		}
	}
	return nil
}

// writeFrame: «Preposit : [ … ]» и модели «[ предикация ] -> интерпретация».
func (q *Script) writeFrame(f *disp.Frame) error {
	preps := make([]string, 0, len(f.Kernel))
	for p := range f.Kernel {
		preps = append(preps, p)
	}
	sort.Strings(preps)
	lines := []string{"Preposit : [ " + strings.Join(preps, " ") + " ]"}
	for _, m := range f.Models {
		lines = append(lines, "[ "+m.Pred+" ] -> "+m.Value.String())
	}
	return writeLines(q.frameFile("preposit", f.Name), lines)
}

// ---- алиасы ----

// loadAliases: файлы препозиций, кроме интерпретаторов, — предикации
// (`plus : +`), они выполняются в общем контексте (9.8).
func (q *Script) loadAliases() error {
	frames := map[string]bool{}
	for _, f := range q.in.Frames {
		frames[strings.ToLower(f.Name)+".uic"] = true
	}
	root := filepath.Join(q.dir, "lexic", "preposit")
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".uic") || frames[d.Name()] {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(b)
		if isLegacy(b) { // прежний вид: r-n имя, r-m алиас
			sys, _, err := readLegacy(path)
			if err != nil {
				return err
			}
			if sys["n"] == "" || sys["m"] == "" {
				return nil
			}
			text = sys["n"] + " : " + sys["m"]
			if err := writeLines(path, []string{text}); err != nil {
				return err
			}
		}
		if _, err := q.in.Run(text, q.env); err != nil {
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
