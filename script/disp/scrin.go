// epi/script/disp/scrin.go
// scrin: интерпретатор файлов — lectu, scrib, catal, migra; путь — от папки данных Root, писать и перемещать — только в thema/
package disp
import ("fmt"; "os"; "path/filepath"; "strings"; "epi/script/format")
// scrinPath: путь внутри Root; абсолютный, с «..» или вне thema/ при записи — отказ с причиной
func (in *Interp) scrinPath(v *format.Value, write bool) (string, bool) {
p, ok := text(v)
if !ok {return "", false}
fail := func(why string) (string, bool) {in.RawErr = fmt.Errorf("scrin: «%s» — %s", p, why); return "", false}
switch {
case in.Root == "": return fail("файлы недоступны: папка данных не задана")
case p == "" || filepath.IsAbs(p) || strings.HasPrefix(p, "/"): return fail("путь пишется от папки данных, без «/» в начале")
case strings.Contains("/"+p+"/", "/../"): return fail("«..» в пути не допускается")
case write && !strings.HasPrefix(p+"/", "thema/"): return fail("писать и перемещать можно только в темах — путь начинается с thema/")}; return filepath.Join(in.Root, filepath.FromSlash(p)), true }
// scrinFrame: фрейм scrin; lectu — текст файла (нет файла — @none), scrib — записать (папки создаются), catal — имена в папке
// (у папок «/» в конце; нет папки — @none), migra — переместить (занятое место — отказ)
func (in *Interp) scrinFrame() *Frame {
str := func(s string) format.Value {return format.Value{Kind: format.String, Text: s}}
k := map[string]Prep{
	"lectu": func(a, b *format.Value) (format.Value, bool) {
		if a != nil {return none, false}
		p, ok := in.scrinPath(b, false)
		if !ok {return none, false}
		t, err := os.ReadFile(p)
		if os.IsNotExist(err) {return none, true}
		if err != nil {in.RawErr = err; return none, false}; return str(string(t)), true},
	"scrib": func(a, b *format.Value) (format.Value, bool) {
		t, ok := text(b)
		if !ok {return none, false}
		p, ok := in.scrinPath(a, true)
		if !ok {return none, false}
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {in.RawErr = err; return none, false}
		if err := os.WriteFile(p+".tmp", []byte(t), 0644); err != nil {in.RawErr = err; return none, false}
		if err := os.Rename(p+".tmp", p); err != nil {in.RawErr = err; return none, false}; return format.Value{Kind: format.Literal, Text: "@true"}, true},
	"catal": func(a, b *format.Value) (format.Value, bool) {
		if a != nil {return none, false}
		p, ok := in.scrinPath(b, false)
		if !ok {return none, false}
		es, err := os.ReadDir(p)
		if os.IsNotExist(err) {return none, true}
		if err != nil {in.RawErr = err; return none, false}
		out := format.Value{Kind: format.Array, Items: []format.Value{}}
		for _, e := range es {n := e.Name(); if e.IsDir() {n += "/"}; out.Items = append(out.Items, str(n))}; return out, true},
	"migra": func(a, b *format.Value) (format.Value, bool) {
		from, ok := in.scrinPath(a, true)
		if !ok {return none, false}
		to, ok := in.scrinPath(b, true)
		if !ok {return none, false}
		if _, err := os.Stat(from); err != nil {in.RawErr = fmt.Errorf("scrin: «%s» — такого файла или папки нет", a.Text); return none, false}
		if _, err := os.Stat(to); err == nil {in.RawErr = fmt.Errorf("scrin: «%s» уже есть", b.Text); return none, false}
		if err := os.MkdirAll(filepath.Dir(to), 0755); err != nil {in.RawErr = err; return none, false}
		if err := os.Rename(from, to); err != nil {in.RawErr = err; return none, false}; return format.Value{Kind: format.Literal, Text: "@true"}, true},}
return &Frame{Name: "scrin", Kernel: k} }
