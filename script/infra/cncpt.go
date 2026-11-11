// epi/script/infra/cncpt.go
package main
import ("fmt"; "os"; "path/filepath"; "strings"; "unicode"; "unicode/utf8"; "epi/script/disp"; "epi/script/format")
// файлы концептов: модель — одна строка, ключей и индексов нет; путь — из букв имени: <тема>/<буквы>/<имя>.uic
//
//	интерпретатор  lexic/preposit/A/R/I/F/M/arifm.uic
//	               Preposit : [ * + - / ]
//	алиас          lexic/preposit/P/L/U/S/plus.uic
//	               plus : +
//	концепт автора thema/<Тема>/X/x.uic
//	               A : 5
// letterPath: <корень>/<буквы имени>/<имя>.uic
func letterPath(root, name string) string {
parts := []string{root}
for _, r := range strings.ToUpper(name) {parts = append(parts, string(r))}; return filepath.Join(append(parts, strings.ToLower(name)+".uic")...) }
// writeLines: файл целиком через временный файл
func writeLines(path string, lines []string) error {
if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {return err}
tmp := path + ".tmp"
if err := os.WriteFile(tmp, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {return err}; return os.Rename(tmp, path) }
// conceptFile: файл концепта темы — по указателю (клон лежит по пути другого концепта), иначе по буквам имени
func (q *Script) conceptFile(name string) string {
if p, ok := q.cidx[strings.ToLower(name)]; ok {return p}; return letterPath(q.themaDir(), name) }
// conceptLines: строки концепта «Параметр : предикация»
func conceptLines(b []byte) ([]string, error) {return disp.Split(string(b)) }
// lineKey: параметр строки «Параметр : …»
func lineKey(line string) string {
k, _, _ := strings.Cut(line, " "); return k }
// writeConcept: параметр фрейма автора строкой в файле концепта, повторное определение заменяет строку; синонимы — группа
func (q *Script) writeConcept(frame, param string, pred []format.Value) error {
if q.theme == "" {return nil}
if root, ok := q.linkRoot(param); ok {return q.writeLink(frame, param, pred, root, false)}
if format.SameTerm(param, "Rank") {return q.writeRank(frame, pred)}
if !format.SameTerm(param, "Alias") {q.known(q.conceptFile(frame)); return writeParam(q.conceptFile(frame), param, pred)}
path := q.conceptFile(frame)
old, _ := readLink(path, "Alias")
if err := writeParam(path, "Alias", pred); err != nil {return err}
q.cidx[strings.ToLower(frame)] = path; return q.syncAlias(frame, old, names(pred)) }
// writeRank: ранг концепта; пустой ранг (Rank : []) — концепт больше не таксон, он убирается из Taxon всех дочерних
func (q *Script) writeRank(frame string, pred []format.Value) error {
path := q.conceptFile(frame); q.known(path)
was := setting(path, "Rank") != ""
if err := writeParam(path, "Rank", pred); err != nil || !was || setting(path, "Rank") != "" {return err}
_, kids := readLink(path, "Taxon")
root, _ := q.linkRoot("Taxon")
for _, k := range kids {
	kp := q.conceptPath(k); if kp == "" {continue}
	d, _ := readLink(kp, "Taxon")
	var ns []format.Value
	for _, n := range d {if !format.SameTerm(n, frame) {ns = append(ns, format.Value{Kind: format.Term, Text: n})}}
	if err := q.writeLink(conceptName(kp, ""), "Taxon", ns, root, false); err != nil {return err}}; return nil }
// writeParam: строка «Параметр : предикация» в файле концепта path (пустое значение — «Параметр :»); новый параметр — перед телом фрейма; повтор строки параметра убирается
func writeParam(path, param string, pred []format.Value) error {
var lines []string
if b, err := os.ReadFile(path); err == nil {if lines, err = conceptLines(b); err != nil {return fmt.Errorf("%s: %w", path, err)}} else if !os.IsNotExist(err) {return err}
text := strings.TrimRight(param + " : " + strings.ReplaceAll(format.Format(pred), "\n", " . "), " ")
found := false
out := lines[:0]
for _, l := range lines {if isParam(l) && format.SameTerm(lineKey(l), param) {if found {continue}; l, found = text, true}; out = append(out, l)}
lines = out
if !found {lines = append(lines, "")
	k := len(lines) - 1
	for k > 0 && !isParam(lines[k-1]) {lines[k] = lines[k-1]; k--}
	lines[k] = text}; return writeLines(path, lines) }
// isParam: строка концепта — параметр «Имя : значение» или «Имя :» (имя с прописной, без точки), иначе тело фрейма
func isParam(l string) bool {
f := strings.Fields(l)
if len(f) < 2 || f[1] != ":" || strings.Contains(f[0], ".") {return false}
r, _ := utf8.DecodeRuneInString(f[0]); return unicode.IsUpper(r) }
// writeBody: тело фрейма «Фрейм : [ … ]» строками после параметров концепта
func (q *Script) writeBody(frame string, body []format.Value) error {
if q.theme == "" {return nil}
path := q.conceptFile(frame)
var lines []string
if b, err := os.ReadFile(path); err == nil {if lines, err = conceptLines(b); err != nil {return fmt.Errorf("%s: %w", path, err)}} else if !os.IsNotExist(err) {return err}
var out []string
for _, l := range lines {if isParam(l) {out = append(out, l)}}
preds, err := disp.Split(format.Format(body))
if err != nil {return err}
q.known(path); return writeLines(path, append(out, preds...)) }
// loadConcepts: системные концепты (lexic/constans) и концепты темы в контекст; концепт темы дополняет и переопределяет системный
func (q *Script) loadConcepts() error {
on, of := q.in.OnParam, q.in.OnFrame
q.in.OnParam, q.in.OnFrame = nil, nil
defer func() {q.in.OnParam, q.in.OnFrame = on, of}()
if err := q.loadConceptsIn(filepath.Join(q.dir, "lexic", "constans")); err != nil {return err}; return q.loadConceptsIn(q.themaDir()) }
// loadConceptsIn: концепты в папке root — имя из пути; у связи в контекст — только прямой список
func (q *Script) loadConceptsIn(root string) error {return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
	if os.IsNotExist(err) && path == root {return nil}
	if err != nil {return err}
	if d.IsDir() && d.Name() == "prtcl" {return filepath.SkipDir}
	if d.IsDir() || !strings.HasSuffix(path, ".uic") {return nil}
	b, err := os.ReadFile(path)
	if err != nil {return err}
	name := conceptName(path, root)
	lines, err := conceptLines(b)
	if err != nil {return fmt.Errorf("%s: %w", path, err)}
	var body []string
	for _, l := range lines {
		if !isParam(l) {body = append(body, l); continue}
		if _, ok := q.linkRoot(lineKey(l)); ok {l = directOnly(l)}
		if _, err := q.in.Run(name+"."+l, q.env); err != nil {return fmt.Errorf("%s: %w", path, err)}}
	if len(body) > 0 {if _, err := q.in.Run(name+" : [ "+strings.Join(body, " . ")+" ]", q.env); err != nil {return fmt.Errorf("%s: %w", path, err)}}; return nil}) }
// conceptName: имя концепта из имени файла с прописной первой буквой (К/В/А/Д/Р/А/Т/квадрат.uic → Квадрат)
func conceptName(path, root string) string { return capital(strings.TrimSuffix(filepath.Base(path), ".uic")) }
// known: файл концепта темы в указатель (новый концепт виден сразу), файлы лексикона — нет
func (q *Script) known(path string) {
if q.theme != "" && strings.HasPrefix(path, q.themaDir()+string(filepath.Separator)) {q.cidx[strings.ToLower(strings.TrimSuffix(filepath.Base(path), ".uic"))] = path} }
// setting: значение настройки концепта (Nord : Links) без кавычек
func setting(path, key string) string {
b, err := os.ReadFile(path)
if err != nil {return ""}
lines, _ := conceptLines(b)
for _, l := range lines {
	if !isParam(l) || !format.SameTerm(lineKey(l), key) {continue}
	vs, err := format.Parse(strings.TrimSpace(l[strings.Index(l, ":")+1:]))
	if err == nil && len(vs) == 1 {return vs[0].Text}}; return "" }
// conceptPath: файл существующего концепта в теме или в лексиконе (prfix, sufix, radix); "" — концепта нет
func (q *Script) conceptPath(name string) string {
if p := q.conceptFile(name); exists(p) {return p}
for _, c := range []string{"prfix", "sufix", "radix"} {if p := letterPath(filepath.Join(q.dir, "lexic", c), name); exists(p) {return p}}; return "" }
// iconFile: картинка концепта — <папка>/<имя>.svg или <папка>/img/<имя>.png|jpg|jpeg|webp|gif|ico
func (q *Script) iconFile(name string) string {
if name == "" || strings.ContainsAny(name, "/\\") {return ""}
roots := []string{q.themaDir()}
for _, c := range []string{"prfix", "sufix", "radix", "preposit", "atrbt", "constans"} {roots = append(roots, filepath.Join(q.dir, "lexic", c))}
base := strings.ToLower(name)
for _, r := range roots {
	dir := filepath.Dir(letterPath(r, name))
	if r == q.themaDir() {dir = filepath.Dir(q.conceptFile(name))}
	if p := filepath.Join(dir, base+".svg"); exists(p) {return p}
	for _, ext := range []string{"png", "jpg", "jpeg", "webp", "gif", "ico"} {if p := filepath.Join(dir, "img", base+"."+ext); exists(p) {return p}}}; return "" }
// settingList: имена в настройке-блоке концепта (Graph : [ nr ], List : [ Taxon ])
func settingList(path, key string) []string {
b, err := os.ReadFile(path)
if err != nil {return nil}
lines, _ := conceptLines(b)
for _, l := range lines {
	if !isParam(l) || !format.SameTerm(lineKey(l), key) {continue}
	vs, err := format.Parse(strings.TrimSpace(l[strings.Index(l, ":")+1:]))
	if err == nil {return names(vs)}}; return nil }
// exists: файл есть
func exists(path string) bool {_, err := os.Stat(path); return err == nil }
// capital: имя объекта с прописной
func capital(w string) string {r, n := utf8.DecodeRuneInString(w); return string(unicode.ToUpper(r)) + w[n:] }
