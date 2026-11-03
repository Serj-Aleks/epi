package main

// Файлы концептов (ason-spec 0.23): модель — одна строка в записи скрипта,
// ключей и индексов нет. Системная часть — только то, что не выводится из
// имени, места файла и моделей (имя и путь — из места, число моделей — число
// строк). Путь концепта — из букв имени (как в ПИ): <тема>/<буквы>/<имя>.uic.
//
//	интерпретатор  lexic/preposit/A/R/I/F/M/arifm.uic
//	               Preposit : [ * + - / ]
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

// syncFrame: файл интерпретатора — только системная часть «Preposit : [ … ]»;
// прецеденты интерпретаций во фрейм не пишутся (решено 03.11.2026: история —
// в протоколе). Существующий файл с прецедентами или прежнего вида
// переписывается; нового файла не создаёт.
func (q *Script) syncFrame(f *disp.Frame) error {
	path := q.frameFile("preposit", f.Name)
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	preps := make([]string, 0, len(f.Kernel))
	for p := range f.Kernel {
		preps = append(preps, p)
	}
	sort.Strings(preps)
	line := "Preposit : [ " + strings.Join(preps, " ") + " ]"
	if strings.TrimSpace(string(b)) == line {
		return nil
	}
	return writeLines(path, []string{line})
}

// ---- алиасы ----

// loadAliases: файлы препозиций, кроме интерпретаторов, — предикации
// (`plus : +`), они выполняются в общем контексте (9.8).
func (q *Script) loadAliases() error {
	on := q.in.OnAlias
	q.in.OnAlias = nil
	defer func() { q.in.OnAlias = on }()
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
