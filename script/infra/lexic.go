// epi/script/infra/lexic.go
package main
import ("fmt"; "os"; "path/filepath"; "sort"; "strings"; "epi/script/disp")
// лексикон: файлы фреймов-интерпретаторов lexic/preposit и алиасы препозиций
// frameFile: файл фрейма name в классе lexic/<theme>
func (q *Medium) frameFile(theme, name string) string {return letterPath(filepath.Join(q.dir, "lexic", theme), name) }
// syncFrame: файл интерпретатора — только «Preposit : [ … ]» (история — в протоколе); нет файла — создаётся
func (q *Medium) syncFrame(f *disp.Frame) error {
path := q.frameFile("preposit", f.Name)
b, err := os.ReadFile(path)
if err != nil && !os.IsNotExist(err) {return err}
preps := make([]string, 0, len(f.Kernel))
for p := range f.Kernel {preps = append(preps, p)}
sort.Strings(preps)
line := "Preposit : [ " + strings.Join(preps, " ") + " ]"
if strings.TrimSpace(string(b)) == line {return nil}; return writeLines(path, []string{line}) }
// loadAliases: файлы препозиций, кроме интерпретаторов, — предикации (plus : +) в общем контексте
func (q *Medium) loadAliases() error {
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
