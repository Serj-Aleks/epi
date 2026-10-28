// EPI: дополнить образ пакетами, нужными Firefox (и другим программам).
//
// Запускается на машине сборки после сборки образа. Берёт индекс репозитория
// HaikuPorts, проходит по зависимостям от корневых пакетов и сверяет их с тем,
// что реально лежит в собранном образе (а не с чьей-то установленной системой).
// Недостающее скачивается в epi-local/epi-packages.
// Результат: epi-local/epi-packages.jam (подключается из UserBuildConfig).
// Использование: go run . [пакет ...]   (из папки image/fetchdeps)
// По умолчанию корни — набор эталонной машины (image/packages-reference.txt,
// проверено: Firefox на ней работает) за вычетом image/packages-exclude.txt;
// так в образ попадает и то, что программы подгружают на ходу, не объявляя.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const base = "https://eu.hpkg.haiku-os.org/haikuports/master/x86_64/current"

var (
	home, _ = os.UserHomeDir()
	dev     = filepath.Join(home, "haiku-dev")
	gen     = filepath.Join(dev, "haiku", "generated.x86_64")
	local   = filepath.Join(dev, "epi-local")
	pk      = filepath.Join(local, "epi-packages")
	iso     = filepath.Join(gen, "haiku-nightly-anyboot.iso")
	here    = filepath.Join(dev, "epi", "image")
	jamFile = filepath.Join(local, "epi-packages.jam")
)

var (
	nameRe  = regexp.MustCompile(`^([a-z0-9_]+)-[0-9][^-]*-[0-9]+-(?:x86_64|any)\.hpkg$`)
	imageRe = regexp.MustCompile(`([a-z0-9_]+)-[0-9][A-Za-z0-9_.~+]*-[0-9]+-(?:x86_64|any)\.hpkg`)
	splitRe = regexp.MustCompile(`[\s=<>]`)
)

func die(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		die("%v", err)
	}
}

func nameOf(fn string) string {
	if m := nameRe.FindStringSubmatch(fn); m != nil {
		return m[1]
	}
	return ""
}

func lines(p string) []string {
	b, err := os.ReadFile(p)
	check(err)
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

var excl []string

func excluded(n string) bool {
	for _, e := range excl {
		if ok, _ := path.Match(e, n); ok {
			return true
		}
	}
	return false
}

// инструмент чтения индекса репозитория: версия для машины сборки (Linux),
// а не для Haiku
var errFound = errors.New("found")

func findTool() string {
	found := ""
	filepath.WalkDir(filepath.Join(gen, "objects", "linux"), func(p string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if !d.IsDir() && d.Name() == "package_repo" {
			if st, e := os.Stat(p); e == nil && st.Mode()&0111 != 0 {
				found = p
				return errFound
			}
		}
		return nil
	})
	return found
}

func download(url, dst string) {
	r, err := http.Get(url)
	check(err)
	defer r.Body.Close()
	if r.StatusCode != 200 {
		die("%s: %s", url, r.Status)
	}
	f, err := os.Create(dst)
	check(err)
	_, err = io.Copy(f, r.Body)
	check(err)
	check(f.Close())
}

// имена пакетов в собранном образе. Образ читается кусками; кусок обрезается
// по байту, который не может входить в имя файла пакета, чтобы ни одно имя
// не разрезать.
func imageNames() map[string]bool {
	f, err := os.Open(iso)
	check(err)
	defer f.Close()
	inName := func(c byte) bool {
		return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
			strings.IndexByte("_.~+-", c) >= 0
	}
	set := map[string]bool{}
	buf := make([]byte, 64<<20)
	var rest []byte
	for {
		n, err := io.ReadFull(f, buf)
		chunk := append(rest, buf[:n]...)
		eof := err == io.EOF || err == io.ErrUnexpectedEOF
		if err != nil && !eof {
			check(err)
		}
		cut := len(chunk)
		if !eof {
			for cut > 0 && inName(chunk[cut-1]) {
				cut--
			}
			if cut == 0 {
				cut = len(chunk)
			}
		}
		// совпадение лежит внутри сплошного отрезка байтов имени и содержит
		// ".hpkg": регулярное выражение применяется только к таким отрезкам
		part, done := chunk[:cut], 0
		for {
			k := bytes.Index(part[done:], []byte(".hpkg"))
			if k < 0 {
				break
			}
			a, b := done+k, done+k
			for a > done && inName(part[a-1]) {
				a--
			}
			for b < len(part) && inName(part[b]) {
				b++
			}
			for _, m := range imageRe.FindAllSubmatch(part[a:b], -1) {
				set[string(m[1])] = true
			}
			done = b
		}
		if eof {
			return set
		}
		rest = append([]byte(nil), chunk[cut:]...)
	}
}

func main() {
	for _, l := range lines(filepath.Join(here, "packages-exclude.txt")) {
		if strings.TrimSpace(l) != "" && !strings.HasPrefix(l, "#") {
			excl = append(excl, strings.TrimSpace(l))
		}
	}

	tool := findTool()
	if tool == "" {
		c := exec.Command("jam", "-q", "<build>package_repo")
		c.Dir, c.Stdout, c.Stderr = gen, os.Stdout, os.Stderr
		check(c.Run())
		tool = findTool()
	}
	if tool == "" {
		die("не найден package_repo для Linux в generated.x86_64/objects/linux")
	}

	check(os.MkdirAll(pk, 0755))
	repo := filepath.Join(local, "haikuports.repo")
	download(base+"/repo", repo)
	// инструменты сборки для Linux используют свои библиотеки (libbe_build.so и др.)
	var libdirs []string
	filepath.WalkDir(filepath.Join(gen, "objects", "linux"), func(p string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			es, _ := os.ReadDir(p)
			for _, e := range es {
				if strings.HasSuffix(e.Name(), "_build.so") {
					libdirs = append(libdirs, p)
					break
				}
			}
		}
		return nil
	})
	sort.Strings(libdirs)
	c := exec.Command(tool, "list", "-v", "-f", repo)
	c.Env = append(os.Environ(), "LD_LIBRARY_PATH="+strings.Join(append(libdirs, os.Getenv("LD_LIBRARY_PATH")), ":"))
	c.Stderr = os.Stderr
	out, err := c.Output()
	check(err)

	// индекс: пакет -> файл, требования; возможность -> пакет
	files := map[string]string{}
	requires := map[string][]string{}
	provider := map[string][]string{}
	cur := ""
	for _, line := range strings.Split(string(out), "\n") {
		s := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(s, "name:"):
			if f := strings.Fields(s); len(f) > 1 {
				cur = f[1]
				requires[cur] = []string{}
			}
		case cur != "" && strings.HasPrefix(s, "provides:"):
			k := splitRe.Split(strings.TrimSpace(s[9:]), 2)[0]
			provider[k] = append(provider[k], cur)
		case cur != "" && strings.HasPrefix(s, "requires:"):
			requires[cur] = append(requires[cur], splitRe.Split(strings.TrimSpace(s[9:]), 2)[0])
		case cur != "" && strings.HasPrefix(s, "filename:"):
			if f := strings.Fields(s); len(f) > 1 {
				files[cur] = f[1]
			}
		}
	}

	inImage := imageNames()
	// что добавили мы сами (не считается базой): наши файлы и всё, что прошлый
	// запуск записал в epi-packages.jam (иначе база «плавает» от сборки к сборке)
	ours := map[string]bool{}
	es, err := os.ReadDir(pk)
	check(err)
	for _, e := range es {
		if n := nameOf(e.Name()); n != "" {
			ours[n] = true
		}
	}
	if _, err := os.Stat(jamFile); err == nil {
		for _, l := range lines(jamFile) {
			if strings.HasPrefix(l, "# names:") {
				for _, n := range strings.Fields(l)[2:] {
					ours[n] = true
				}
			}
		}
	}
	baseSet := map[string]bool{}
	for n := range inImage {
		if !ours[n] {
			baseSet[n] = true
		}
	}

	// из нескольких пакетов, дающих одно и то же, берём уже имеющийся:
	// сначала из образа, потом из наших пакетов (как на эталонной машине)
	pick := func(req string) string {
		cands := provider[req]
		for _, group := range []map[string]bool{baseSet, ours} {
			for _, c := range cands {
				if group[c] {
					return c
				}
			}
		}
		if len(cands) > 0 {
			return cands[0]
		}
		return ""
	}

	var roots []string
	if len(os.Args) > 1 {
		roots = os.Args[1:]
	} else {
		for _, l := range lines(filepath.Join(here, "packages-reference.txt")) {
			if n := nameOf(strings.TrimSpace(l)); n != "" && !excluded(n) {
				roots = append(roots, n)
			}
		}
	}
	// корни, которые и так есть в базовом образе, не добавляем (иначе — дубли)
	var r2 []string
	isRoot := map[string]bool{}
	for _, r := range roots {
		if !baseSet[r] {
			r2 = append(r2, r)
			isRoot[r] = true
		}
	}
	var need []string
	inNeed := map[string]bool{}
	todo := append([]string(nil), r2...)
	for len(todo) > 0 {
		p := todo[0]
		todo = todo[1:]
		if inNeed[p] || (baseSet[p] && !isRoot[p]) {
			continue
		}
		reqs, ok := requires[p]
		if !ok {
			fmt.Println("нет в репозитории:", p)
			continue
		}
		need = append(need, p)
		inNeed[p] = true
		for _, r := range reqs {
			q := pick(r)
			// исключённое не попадает в образ и как зависимость
			if q != "" && !inNeed[q] && !baseSet[q] && !excluded(q) {
				todo = append(todo, q)
			}
		}
	}

	// всё недостающее кладём файлами из репозитория HaikuPorts: на сервере
	// пакетов сборки есть не всё, что указано в её списке (file_data — 404)
	var byFile []string
	for _, p := range need {
		var have []string
		es, _ := os.ReadDir(pk)
		for _, e := range es {
			if nameOf(e.Name()) == p {
				have = append(have, e.Name())
			}
		}
		if len(have) > 0 {
			sort.Strings(have)
			byFile = append(byFile, filepath.Join(pk, have[len(have)-1]))
			continue
		}
		fn := files[p]
		fmt.Println("скачиваю", fn)
		download(base+"/packages/"+fn, filepath.Join(pk, fn))
		byFile = append(byFile, filepath.Join(pk, fn))
	}

	sortedNeed := append([]string(nil), need...)
	sort.Strings(sortedNeed)
	sort.Strings(byFile)
	var w bytes.Buffer
	w.WriteString("# создано image/fetchdeps\n")
	for i := 0; i < len(sortedNeed); i += 10 {
		j := i + 10
		if j > len(sortedNeed) {
			j = len(sortedNeed)
		}
		w.WriteString("# names: " + strings.Join(sortedNeed[i:j], " ") + "\n")
	}
	// по одному имени в строке: jam не принимает слишком длинные строки
	if len(byFile) > 0 {
		w.WriteString("AddFilesToHaikuImage system packages :\n\t" + strings.Join(byFile, "\n\t") + " ;\n")
	}
	check(os.WriteFile(jamFile, w.Bytes(), 0644))

	var base []string
	for _, x := range byFile {
		base = append(base, filepath.Base(x))
	}
	sort.Strings(base)
	fmt.Printf("по имени (0): \n")
	fmt.Printf("файлами (%d): %s\n", len(byFile), strings.Join(base, " "))
	var missing []string
	for _, p := range need {
		if !inImage[p] {
			missing = append(missing, p)
		}
	}
	m := strings.Join(missing, " ")
	if m == "" {
		m = "-"
	}
	fmt.Printf("не хватало в образе (%d): %s\n", len(missing), m)
}
