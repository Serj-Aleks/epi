// epi/script/infra/cncpt.go
package main
// файлы концептов: модель — одна строка, ключей и индексов нет; путь — из букв имени: <тема>/<буквы>/<имя>.uic
//
//	интерпретатор  lexic/preposit/A/R/I/F/M/arifm.uic
//	               Preposit : [ * + - / ]
//	алиас          lexic/preposit/P/L/U/S/plus.uic
//	               plus : +
//	концепт автора thema/<Тема>/X/x.uic
//	               A : 5
import ("fmt"; "os"; "path/filepath"; "sort"; "strings"; "epi/script/disp")
// letterPath: <корень>/<буквы имени>/<имя>.uic
func letterPath(root, name string) string {
parts := []string{root}
for _, r := range strings.ToUpper(name) {parts = append(parts, string(r))}; return filepath.Join(append(parts, strings.ToLower(name)+".uic")...) }
// frameFile: файл фрейма name в классе lexic/<theme>
func (q *Script) frameFile(theme, name string) string {return letterPath(filepath.Join(q.dir, "lexic", theme), name) }
// writeLines: файл целиком через временный файл
func writeLines(path string, lines []string) error {
if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {return err}
tmp := path + ".tmp"
if err := os.WriteFile(tmp, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {return err}; return os.Rename(tmp, path) }
// syncFrame: файл интерпретатора — только «Preposit : [ … ]» (история — в протоколе); нет файла — создаётся
func (q *Script) syncFrame(f *disp.Frame) error {
path := q.frameFile("preposit", f.Name)
b, err := os.ReadFile(path)
if err != nil && !os.IsNotExist(err) {return err}
preps := make([]string, 0, len(f.Kernel))
for p := range f.Kernel {preps = append(preps, p)}
sort.Strings(preps)
line := "Preposit : [ " + strings.Join(preps, " ") + " ]"
if strings.TrimSpace(string(b)) == line {return nil}; return writeLines(path, []string{line}) }
// loadAliases: файлы препозиций, кроме интерпретаторов, — предикации (plus : +) в общем контексте
func (q *Script) loadAliases() error {
on := q.in.OnAlias
q.in.OnAlias = nil
defer func() { q.in.OnAlias = on }()
frames := map[string]bool{}
for _, f := range q.in.Frames {frames[strings.ToLower(f.Name)+".uic"] = true}
root := filepath.Join(q.dir, "lexic", "preposit"); return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
	if os.IsNotExist(err) {return nil}
	if err != nil || d.IsDir() || !strings.HasSuffix(path, ".uic") || frames[d.Name()] {return err}
	b, err := os.ReadFile(path)
	if err != nil {return err}
	text := string(b)
	if _, err := q.in.Run(text, q.env); err != nil {return fmt.Errorf("%s: %w", path, err)}; return nil}) }
