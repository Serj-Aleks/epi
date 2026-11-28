// epi/script/infra/sdict.go
package main
import ("archive/tar"; "compress/gzip"; "fmt"; "io"; "net/http"; "os"; "path/filepath"; "sort"; "strconv"; "strings"; "unicode/utf8"; "epi/script/disp"; "epi/script/format")
// sdict: системные фреймы — папка sdict рядом с кодом (апдейт заменяет её целиком), на машине не меняется
// sdictFile: файл системного фрейма — sdict/<первая буква>/<имя>.uic; нет — пусто
func (q *Medium) sdictFile(name string) string {
r, n := utf8.DecodeRuneInString(name)
if n == 0 {return ""}
if p := filepath.Join(q.dir, "sdict", strings.ToUpper(string(r)), strings.ToLower(name)+".uic"); exists(p) {return p}; return "" }
// inSdict: файл лежит в системной папке sdict (не меняется в script)
func (q *Medium) inSdict(p string) bool { return p != "" && strings.HasPrefix(p, filepath.Join(q.dir, "sdict")+string(filepath.Separator)) }
// sdictList: системные фреймы для карточки темы — имя, картинка, число параметров, спутники; в алфавите темы
func (q *Medium) sdictList() []map[string]string {
out := []map[string]string{}
filepath.WalkDir(filepath.Join(q.dir, "sdict"), func(p string, d os.DirEntry, err error) error {
	if err != nil || d.IsDir() || !strings.HasSuffix(p, ".uic") {return nil}
	name, np, sats := conceptName(p, ""), 0, []string{}
	if b, err := os.ReadFile(p); err == nil {ls, _ := conceptLines(b); for _, l := range ls {if isParam(l) {np++}}}
	for _, t := range satTypes {if exists(strings.TrimSuffix(p, ".uic") + "." + t) {sats = append(sats, t)}}
	r := map[string]string{"name": name, "params": strconv.Itoa(np), "sats": strings.Join(sats, " ")}
	if q.iconFile(name) != "" {r["icon"] = "/api/icon?name=" + name}
	out = append(out, r); return nil})
sort.Slice(out, func(i, j int) bool {return aLess(out[i]["name"], out[j]["name"])}); return out }
// sdictUpdate: sdict из репозитория — архив ветки, из него только script/sdict во временную папку; проверка: загрузка и Sdir;
// не прошла — sdict прежний, ошибка; прошла — папка заменяется целиком, контекст ядра и темы заново; ответ — число фреймов
func (q *Medium) sdictUpdate() (int, error) {
resp, err := http.Get(sdictSource)
if err != nil {return 0, fmt.Errorf("репозиторий недоступен: %v — проверьте сеть", err)}
defer resp.Body.Close()
if resp.StatusCode != http.StatusOK {return 0, fmt.Errorf("репозиторий ответил %s", resp.Status)}
gz, err := gzip.NewReader(resp.Body)
if err != nil {return 0, fmt.Errorf("архив не читается: %v", err)}
dir, tmp := filepath.Join(q.dir, "sdict"), filepath.Join(q.dir, "sdict.new")
os.RemoveAll(tmp)
defer os.RemoveAll(tmp)
tr, n := tar.NewReader(gz), 0
for {
	h, err := tr.Next()
	if err == io.EOF {break}
	if err != nil {return 0, fmt.Errorf("архив не читается: %v", err)}
	_, rel, _ := strings.Cut(h.Name, "/")
	rel, ok := strings.CutPrefix(rel, "script/sdict/")
	if !ok || h.Typeflag != tar.TypeReg || rel == "" || strings.Contains(rel, "..") {continue}
	dst := filepath.Join(tmp, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {return 0, err}
	b, err := io.ReadAll(tr)
	if err != nil {return 0, err}
	if err := os.WriteFile(dst, b, 0644); err != nil {return 0, err}
	if strings.HasSuffix(rel, ".uic") {n++}}
if n == 0 {return 0, fmt.Errorf("в репозитории нет фреймов sdict — sdict не изменён")}
core, cenv := disp.New(limit, nil), disp.NewEnv(nil)
if err := q.loadConceptsIn(core, cenv, tmp); err != nil {return 0, fmt.Errorf("новый sdict не загрузился: %v — sdict не изменён", err)}
if _, err := core.Invoke("Sdir", []format.Value{{Kind: format.String, Text: "A"}}, cenv); err != nil {return 0, fmt.Errorf("в новом sdict не работает Sdir: %v — sdict не изменён", err)}
old := filepath.Join(q.dir, "sdict.old")
os.RemoveAll(old)
if err := os.Rename(dir, old); err != nil && !os.IsNotExist(err) {return 0, err}
if err := os.Rename(tmp, dir); err != nil {os.Rename(old, dir); return 0, err}
os.RemoveAll(old)
q.core, q.cenv, q.env = core, cenv, disp.NewEnv(nil)
if err := q.loadAliases(); err != nil {return n, err}
if q.theme != "" {if err := q.loadConcepts(); err != nil {return n, err}}; return n, nil }
