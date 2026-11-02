package main

// Темы автора (ason-spec 9.14): тема — проект, папка thema/<Имя>. В ней
// протокол prtcl (сеансы, ignor.uic) и концепты автора — путь из букв имени,
// мета-файл .uic, как в lexic. Параметр фрейма — модель «l-<индекс> Параметр» /
// «r-<индекс> предикация»; повторное определение параметра перезаписывает его
// модель (индекс сохраняется). Активная тема одна; системный лексикон (lexic) —
// общий для всех тем.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"epi/script/disp"
	"epi/script/format"
)

func (q *Script) themaRoot() string { return filepath.Join(q.dir, "thema") }
func (q *Script) themaDir() string  { return filepath.Join(q.themaRoot(), q.theme) }

// themes: имена тем по алфавиту.
func (q *Script) themes() ([]string, error) {
	es, err := os.ReadDir(q.themaRoot())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ts []string
	for _, e := range es {
		if e.IsDir() {
			ts = append(ts, e.Name())
		}
	}
	sort.Strings(ts)
	return ts, nil
}

// checkName: имя темы — как имя объекта: с прописной буквы (9.9), любой
// алфавит, без пробелов и разделителей пути.
func checkName(name string) error {
	r, _ := utf8.DecodeRuneInString(name)
	switch {
	case name == "":
		return fmt.Errorf("имя темы пусто")
	case !unicode.IsUpper(r):
		return fmt.Errorf("имя темы пишется с прописной буквы (объект)")
	case utf8.RuneCountInString(name) > 64:
		return fmt.Errorf("имя темы длиннее 64 знаков")
	case strings.ContainsAny(name, "/\\:*?\"<>|") || strings.IndexFunc(name, unicode.IsSpace) >= 0:
		return fmt.Errorf("в имени темы — пробел или недопустимый знак")
	}
	return nil
}

// themeExists: имя занято (без учёта регистра, кроме первой буквы — 9.9).
func (q *Script) themeExists(name string) (string, bool) {
	ts, _ := q.themes()
	for _, t := range ts {
		if format.SameTerm(t, name) {
			return t, true
		}
	}
	return "", false
}

// createTheme: новая тема. Первая тема забирает протокол прежнего вида
// (script/prtcl) — он становится её протоколом.
func (q *Script) createTheme(name string) error {
	if err := checkName(name); err != nil {
		return err
	}
	if t, ok := q.themeExists(name); ok {
		return fmt.Errorf("тема %s уже есть", t)
	}
	ts, err := q.themes()
	if err != nil {
		return err
	}
	dir := filepath.Join(q.themaRoot(), name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	legacy := filepath.Join(q.dir, "prtcl")
	if _, err := os.Stat(legacy); len(ts) == 0 && err == nil {
		if err := os.Rename(legacy, filepath.Join(dir, "prtcl")); err != nil {
			return err
		}
	} else if err := os.MkdirAll(filepath.Join(dir, "prtcl"), 0755); err != nil {
		return err
	}
	return q.switchTheme(name)
}

// renameTheme: переименование папки темы; ссылок на имя темы внутри нет.
func (q *Script) renameTheme(old, name string) error {
	if err := checkName(name); err != nil {
		return err
	}
	cur, ok := q.themeExists(old)
	if !ok {
		return fmt.Errorf("темы %s нет", old)
	}
	if t, ok := q.themeExists(name); ok && t != cur {
		return fmt.Errorf("тема %s уже есть", t)
	}
	if err := os.Rename(filepath.Join(q.themaRoot(), cur), filepath.Join(q.themaRoot(), name)); err != nil {
		return err
	}
	if q.theme == cur {
		q.theme = name
	}
	return nil
}

// switchTheme: тема становится активной. Контекст — заново: системные алиасы
// и концепты темы; база нераспознанных — темы; сеанс — последний открытый.
func (q *Script) switchTheme(name string) error {
	q.theme, q.cur, q.closed, q.turns = name, "", false, nil
	q.in.Unknown = nil
	if b, err := os.ReadFile(q.ignor()); err == nil {
		vs, err := format.Parse(string(b))
		if err != nil {
			return fmt.Errorf("ignor.uic: %w", err)
		}
		for _, v := range vs {
			if v.Kind == format.Array {
				q.in.Unknown = append(q.in.Unknown, format.Format(v.Items))
			}
		}
	}
	q.env = disp.NewEnv(nil)
	if err := q.loadAliases(); err != nil {
		return err
	}
	if err := q.loadConcepts(); err != nil {
		return err
	}
	ss, err := q.list()
	if err != nil {
		return err
	}
	for _, s := range ss {
		if !s.Closed {
			return q.switchTo(s.Name)
		}
	}
	return nil
}

// startTheme: при запуске — тема последнего открытого сеанса; нет такого —
// первая по алфавиту; тем нет — ни одной (интерфейс предложит создать).
func (q *Script) startTheme() error {
	ts, err := q.themes()
	if err != nil || len(ts) == 0 {
		return err
	}
	best, last := ts[0], ""
	for _, t := range ts {
		q.theme = t
		ss, err := q.list()
		if err != nil {
			return err
		}
		for _, s := range ss {
			if !s.Closed && s.Name > last {
				best, last = t, s.Name
			}
		}
	}
	return q.switchTheme(best)
}

// conceptFile: <тема>/<буквы имени>/<имя>.uic.
func (q *Script) conceptFile(name string) string {
	parts := []string{q.themaDir()}
	for _, r := range strings.ToUpper(name) {
		parts = append(parts, string(r))
	}
	return filepath.Join(append(parts, strings.ToLower(name)+".uic")...)
}

// readModels: системная часть (буква → значение) и модели по индексам Гаусса в порядке записи.
func readModels(path string) (map[string]string, []string, map[string]string, map[string]string, error) {
	sys, left, right := map[string]string{}, map[string]string{}, map[string]string{}
	var order []string
	file, err := os.Open(path)
	if err != nil {
		return sys, order, left, right, err
	}
	defer file.Close()
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
	return sys, order, left, right, sc.Err()
}

// writeConcept: параметр фрейма автора — в мета-файл концепта темы.
func (q *Script) writeConcept(frame, param string, pred []format.Value) error {
	if q.theme == "" {
		return nil
	}
	path := q.conceptFile(frame)
	_, order, left, right, err := readModels(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	text := strings.ReplaceAll(format.Format(pred), "\n", " . ") // модель — одна строка
	key := ""
	for _, k := range order {
		if format.SameTerm(left[k], param) {
			key = k
		}
	}
	if key == "" {
		key = gauss(len(order) + 1)
		order = append(order, key)
	}
	left[key], right[key] = param, text
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	vals := map[string]string{"name": frame, "uri": frameURI(frame), "counts": gauss(len(order))}
	var sb strings.Builder
	for _, s := range systemSlots {
		fmt.Fprintf(&sb, "l-%s %s\nr-%s %s\n", s.letter, s.name, s.letter, vals[s.name])
	}
	for _, k := range order {
		fmt.Fprintf(&sb, "l-%s %s\nr-%s %s\n", k, left[k], k, right[k])
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(sb.String()), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// loadConcepts: концепты темы — параметры фреймов в контекст темы.
func (q *Script) loadConcepts() error {
	on := q.in.OnParam
	q.in.OnParam = nil
	defer func() { q.in.OnParam = on }()
	return filepath.WalkDir(q.themaDir(), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "prtcl" {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".uic") {
			return nil
		}
		sys, order, left, right, err := readModels(path)
		if err != nil {
			return err
		}
		for _, k := range order {
			if _, err := q.in.Run(sys["n"]+"."+left[k]+" : "+right[k], q.env); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
		}
		return nil
	})
}
