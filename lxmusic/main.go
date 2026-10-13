// Command lxmusic is a CLI port of the core search / download / play / lyric
// logic reverse engineered out of lx-music-desktop 2.12.6.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	lx "lxmusic/internal/lx"
	"lxmusic/internal/tui"
)

// Help text now lives with the commands that use it, so cobra can print it
// per-subcommand. Everything the old usageText said is in commands.go.

func main() {
	restoreConsole := setupConsole()
	if err := newRootCmd().Execute(); err != nil {
		// The error prints first: it is the one line the user is reading, and it
		// is Chinese, so the console is still in UTF-8 mode when it goes out.
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		restoreConsole()
		os.Exit(1)
	}
	// Called twice on purpose, not deferred: os.Exit skips defers, and the
	// function is a real no-op when nothing was changed, so restoring on both
	// ways out is the whole contract.
	restoreConsole()
}

/* --------------------------------------------------------------- helpers */

// flagsFirst reorders the arguments so that every flag comes before the
// positional ones. Go's flag package stops parsing at the first non-flag token,
// which would otherwise make `lxmusic search 晴天 --limit 5` ignore --limit.
func flagsFirst(args []string) []string {
	var flags, rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") || len(a) == 1 {
			rest = append(rest, a)
			continue
		}
		if strings.Contains(a, "=") || i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
			flags = append(flags, a)
			continue
		}
		i++
		// A boolean value is merged into the flag. Go's flag package reads a
		// bool flag as self-contained, so "--tags false kw:1" used to leave
		// "false" behind as the first positional argument: the song reference
		// was lost and the download died with "cannot determine source",
		// handed the key "false". Anything that is not a boolean spelling stays
		// a separate token, so "--verbose kw:123" means a reference and not
		// --verbose=kw:123.
		if boolArg(args[i]) {
			flags = append(flags, a+"="+args[i])
		} else {
			flags = append(flags, a, args[i])
		}
	}
	return append(flags, rest...)
}

// boolArg reports whether s is one of the spellings the flag package accepts
// for a boolean flag, as strconv.ParseBool decides it. Reordering alone cannot
// settle whether the token after a flag is that flag's value or a positional
// argument, so the boolean spellings are the only attachable ones.
func boolArg(s string) bool {
	_, err := strconv.ParseBool(s)
	return err == nil
}

type refs struct {
	key    string
	search string
	index  int
	page   int
	limit  int
}

func (r *refs) register(fs *flag.FlagSet) {
	fs.StringVar(&r.key, "key", "", "song reference source:id")
	fs.StringVar(&r.search, "search", "", "look the song up by keyword instead")
	fs.IntVar(&r.index, "index", 1, "which search result to use")
	fs.IntVar(&r.page, "page", 1, "search page")
	fs.IntVar(&r.limit, "limit", 20, "search page size")
}

func newClient(proxy string, verbose bool) *lx.Client {
	c := lx.NewClient()
	c.Verbose = verbose
	if proxy != "" {
		if u, err := url.Parse(proxy); err == nil {
			c.HTTP.Transport = &http.Transport{
				Proxy:               http.ProxyURL(u),
				TLSHandshakeTimeout: 10 * time.Second,
				IdleConnTimeout:     30 * time.Second,
			}
		} else {
			fmt.Fprintf(os.Stderr, "warning: bad --proxy %q, ignoring\n", proxy)
		}
	}
	return c
}

// resolveRef turns the CLI arguments into a provider plus a song.
func resolveRef(ctx context.Context, key, source, search string, index, page, limit int) (lx.Provider, *lx.Song, error) {
	if source == "" {
		if i := strings.IndexByte(key, ':'); i > 0 {
			source = key[:i]
		} else if search == "" {
			return nil, nil, fmt.Errorf("cannot determine source: use <source>:<id> or --source with --search")
		} else {
			source = "kg"
		}
	}
	p, err := lx.GetProvider(source)
	if err != nil {
		return nil, nil, err
	}
	s, err := lx.SongRef(ctx, p, key, search, index, page, limit)
	if err != nil {
		return nil, nil, err
	}
	if s.Name == "" {
		// A bare --key has no metadata, and user sources need the song name.
		if err := lx.Hydrate(ctx, p, s); err != nil {
			fmt.Fprintf(os.Stderr, "note: could not look up song metadata: %v\n", err)
		}
	}
	if s.PageURL == "" {
		s.PageURL = defaultPageURL(p.ID(), s)
	}
	return p, s, nil
}

// defaultPageURL rebuilds the platform page for a song known only by id. It
// matches the form each adapter puts in Song.PageURL, so a link printed here is
// the same link a search hit would show. An empty result means "cannot be told"
// rather than a guess: a bare "kg:<audioid>" carries no file hash, and kugou's
// page is addressed by hash, so there is no page to build.
func defaultPageURL(source string, s *lx.Song) string {
	switch source {
	case "kw":
		return "http://www.kuwo.cn/play_detail/" + s.ID
	case "wy":
		return "https://music.163.com/#/song?id=" + s.ID
	case "tx":
		if s.MID != "" {
			return "https://y.qq.com/n/yqq/song/" + s.MID + ".html"
		}
	case "kg":
		if s.Hash != "" {
			return "https://www.kugou.com/song/#hash=" + s.Hash + "&album_id=" + s.AlbumID
		}
	}
	return ""
}

func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func openInBrowser(u string) {
	if u == "" {
		return
	}
	var c *exec.Cmd
	switch {
	case os.Getenv("OS") != "": // windows
		c = exec.Command("cmd", "/c", "start", "", u)
	case isDarwin():
		c = exec.Command("open", u)
	default:
		c = exec.Command("xdg-open", u)
	}
	if err := c.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not open browser for %s (%v)\n", u, err)
	}
}

// isDarwin is the compiled target, not the kernel of the machine the shell
// happens to be on. It used to run `uname -s`: that spawned a process on every
// call, and it would have answered Darwin for a Linux binary running under a
// cross-compiled uname - Darwin is a build target, so runtime.GOOS is the whole
// question.
func isDarwin() bool {
	return runtime.GOOS == "darwin"
}

// sanitizeFileName keeps only characters that survive every common shell on both
// sides. The punctuation set is Windows' illegal list - unix accepts all of it,
// but a name written on Windows and moved to a mac must not carry a ":" in it.
func sanitizeFileName(name string) string {
	out := name
	for _, b := range []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"} {
		out = strings.ReplaceAll(out, b, "_")
	}
	// Trailing spaces and dots are rejected by Windows and unreadable in a shell
	// on unix, and they arrive interleaved: "a . " trims to "a " when the trim is
	// done once, so trim until the name stops changing. Leading dots go too, which
	// keeps a mac from treating the file as hidden.
	for {
		trimmed := strings.Trim(out, " .")
		if trimmed == out {
			break
		}
		out = trimmed
	}
	if out == "" {
		return "untitled"
	}
	return out
}

// expandHome turns a leading "~/" or "~\" into the user's home directory. The
// shells already do this on unix, but only for an unquoted value, and Windows has
// no equivalent at all, so `--out "~/Downloads"` created a literal directory
// called "~" there. A "~" anywhere else in the path is left alone: it may be a
// real directory name.
func expandHome(p string) string {
	h, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	switch {
	case p == "~":
		return h
	case strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`):
		return filepath.Join(h, p[2:])
	}
	return p
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func qualityList(s *lx.Song) string {
	if len(s.Qualities) == 0 {
		return "-"
	}
	out := make([]string, 0, len(s.Qualities))
	for _, q := range s.Qualities {
		if q.Size > 0 {
			out = append(out, q.Type+" "+lx.FormatBytes(q.Size))
		} else {
			out = append(out, q.Type)
		}
	}
	sort.Strings(out)
	return strings.Join(out, " ")
}

// printSongBlock renders one song the way the app's list item reads.
func printSongBlock(i int, s *lx.Song) {
	fmt.Printf("%2d. %s - %s  (%s)  %s\n", i, s.Name, s.Singer, lx.FormatDuration(s.Duration), qualityList(s))
	if s.Album != "" {
		fmt.Printf("    album : %s\n", s.Album)
	}
	if s.Pic != "" {
		fmt.Printf("    pic   : %s\n", s.Pic)
	}
	if s.PageURL != "" {
		fmt.Printf("    page  : %s\n", s.PageURL)
	}
	fmt.Printf("    ref   : %s\n", s.KeyString())
}

// platformName shows the app's display name for a source id when it is known.
func platformName(id string) string {
	if p, err := lx.GetProvider(id); err == nil {
		return p.Name()
	}
	return id
}

// bestQualityLabel is the single rendition shown in the table: the highest
// quality the provider announced, or "-" when none could be determined.
func bestQualityLabel(s *lx.Song) string {
	if q := s.BestQuality(); q != "" {
		return q
	}
	for _, q := range s.Qualities {
		if q.Type != "" {
			return q.Type
		}
	}
	return "-"
}

// maxResults is the most rows the search table ever shows, across all sources.
// The picker is a scrolling list, so fifty leaves room to find a specific
// version or a specific singer without paging; the table is sorted by quality,
// so capping happens after the sort and the fifty that survive are the
// highest-fidelity ones.
const maxResults = 50

// downloadTimeout is the deadline for one download, measured from when it
// starts. It is deliberately a per-download budget rather than a session
// budget: the picker runs for as long as the user sits at the table, and
// charging the wait time against the transfer would leave a slow download
// finishing the job with seconds left on the clock.
const downloadTimeout = 10 * time.Minute

// qualityRank orders the labels the table shows, best first. The order itself
// lives in lx.qualityOrder so the table cannot sort wav/ape/192k rows below
// real files just because the built-in adapters never advertise them. Anything
// unrecognised falls to zero, including "-", the marker for a row with no
// quality at all: an unknown label is not a reason to bury a file.
func qualityRank(q string) int {
	if r := lx.QualityRank(q); r >= 0 {
		return r
	}
	return 0
}

// toRows flattens one search page into table rows. Indexes are not assigned
// here because the rows are sorted and capped afterwards; cmdSearch numbers
// them once the final set is settled.
func toRows(res *lx.SearchResult) []tui.Row {
	if res == nil || len(res.List) == 0 {
		return nil
	}
	out := make([]tui.Row, 0, len(res.List))
	for _, s := range res.List {
		out = append(out, tui.Row{
			Platform: platformName(res.Source),
			Name:     s.Name,
			Artist:   s.Singer,
			Duration: lx.FormatDuration(s.Duration),
			Quality:  bestQualityLabel(s),
			Size:     s.BestQualitySize(),
			Ref:      s.KeyString(),
			Song:     s,
		})
	}
	return out
}

// probeConcurrency caps how many renditions are resolved at once while the
// table is built. One worker per row is a rate limit on the platforms; eight is
// enough that a page of fifty does not read as a stall.
const probeConcurrency = 8

// probeBudget caps the whole rendition check, and is deliberately far shorter
// than the search deadline. The check is a correction to a display column, not
// part of the answer: letting it run to the search's 120 s used to exhaust the
// deadline, the sort then ran on rows nobody had measured, and the first row of
// the table still advertised the tier the download would not give. A short
// budget fails the other way - a few rows keep the catalogue number and the
// note says so - which is readable instead of silent. A var so a test can
// exercise the bound itself.
var probeBudget = 8 * time.Second

// providerFor resolves a row's source to its adapter, indirection so a test can
// stand in a provider that never answers - which is what the budget has to bound.
var providerFor = lx.GetProvider

// servedRows replaces the tier a row advertises with the tier it will actually
// serve, so the 音质 and 文件大小 columns and the sort order all read as the
// file that will land in --out.
//
// The search answer is a catalogue: it says what a platform holds, not what its
// gateway will hand out. migu advertises 星月神话 as flac24bit / 52.6MB and serves
// a 128k mp3 of 4MB, so a table sorted on the catalogue put the worst file in
// the list on top.
//
// The probe asks the built-in gateway and stops there. The catalogue tier on a
// row is the built-in's own claim, so the built-in is the only thing worth
// holding it to; going through resolveAudio would spend the user source's 25 s
// timeout on every row first, and a script that has stopped answering is
// precisely what the table never claimed. A user source that does answer is
// named on the row it serves.
//
// Rows the probe cannot answer keep the tier the search announced: a platform
// that is down is no reason to print a number worse than the truth, and those
// rows are counted separately rather than quietly passed off as checked.
func servedRows(ctx context.Context, rows []tui.Row, quality string) (measured, below, unmeasured int, unmeasuredOf []string) {
	probeCtx, cancel := context.WithTimeout(ctx, probeBudget)
	defer cancel()
	sem := make(chan struct{}, probeConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	missed := map[string]int{}
	for i := range rows {
		r := &rows[i]
		if r.Song == nil {
			continue
		}
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			p, err := providerFor(r.Song.Source)
			if err != nil {
				return
			}
			q := r.Song.QualityByType(quality)
			pl, err := p.MusicURL(probeCtx, r.Song, q)
			if err != nil || pl == nil {
				mu.Lock()
				defer mu.Unlock()
				unmeasured++
				missed[p.Name()]++
				return
			}
			mu.Lock()
			defer mu.Unlock()
			measured++
			if _, d := applyServed(r, pl); d {
				below++
			}
		}()
	}
	wg.Wait()
	for name, n := range missed {
		unmeasuredOf = append(unmeasuredOf, fmt.Sprintf("%s%d行", name, n))
	}
	sort.Strings(unmeasuredOf)
	return measured, below, unmeasured, unmeasuredOf
}

// applyServed records one resolved rendition on its row, reporting whether the
// row changed and whether the served tier is below the one advertised.
//
// The size follows the served tier. The gateway never reports a byte count, so
// it has to come from the catalogue entry for the tier that was actually
// served; leaving the advertised number in place would show a row reading 128k
// alongside a 52MB flac24bit size.
func applyServed(r *tui.Row, pl *lx.PlayerURL) (changed, down bool) {
	if r.Song == nil || pl == nil || pl.Quality == "" || pl.Quality == r.Quality {
		return false, false
	}
	if qualityRank(pl.Quality) < qualityRank(r.Quality) {
		down = true
	}
	r.Quality = pl.Quality
	switch sz := r.Song.QualitySize(pl.Quality); {
	case sz > 0:
		r.Size = sz
	case pl.Size > 0:
		r.Size = pl.Size
	default:
		r.Size = 0
	}
	return true, down
}

// sortRows orders the results the way a user reads them: the hit that best
// answers the query first, then the best fidelity, then the largest file.
//
// Relevance leads because quality is a property of the recording and not of the
// answer — a cover in lossless is still the wrong song, and sorting it above the
// original is the failure mode the README documented. Within equal relevance the
// version markers break the tie the same way, so "晴天(现场版)" does not outrank
// "晴天" for a keyword of "晴天". Only then does quality rank, and only then the
// size, because tiers are announced inconsistently — kugou calls 晴天 flac24bit
// where kuwo and QQ call the same 52.83 MB file flac, so two tiers may hold the
// same file and the size breaks that tie. Stable, so each platform's own
// ordering survives a full tie.
func sortRows(rows []tui.Row, keyword string) {
	// The scores are keyed by index, so they have to travel with the rows.
	// Sorting rows in place while leaving a parallel score slice alone makes the
	// very first swap lie about what is being compared.
	idx := make([]int, len(rows))
	rel := make([]int, len(rows))
	agree := make([]bool, len(rows))
	for i := range idx {
		idx[i] = i
		rel[i] = lx.RelevanceScore(rows[i].Name, rows[i].Artist, keyword)
		agree[i] = lx.VersionAgree(rows[i].Name, keyword)
	}
	sort.SliceStable(idx, func(i, j int) bool {
		a, b := idx[i], idx[j]
		if rel[a] != rel[b] {
			return rel[a] > rel[b]
		}
		if agree[a] != agree[b] {
			return agree[a] && !agree[b]
		}
		if qa, qb := qualityRank(rows[a].Quality), qualityRank(rows[b].Quality); qa != qb {
			return qa > qb
		}
		return rows[a].Size > rows[b].Size
	})
	// Gather into a fresh slice and copy back. Writing straight into rows would
	// read rows[n] after an earlier assignment overwrote it whenever the
	// permutation sends an element backwards.
	sorted := make([]tui.Row, len(rows))
	for i, n := range idx {
		sorted[i] = rows[n]
	}
	copy(rows, sorted)
}

// downloadDir is where downloads land unless --out says otherwise: a music
// subdirectory of the directory the command was run from. It is relative to the
// caller's own working directory, so `lxmusic download` in a repo lands in
// ./music there rather than in a fixed home folder, and it is created on demand
// by whoever writes the file.
func downloadDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "music"
	}
	return filepath.Join(dir, "music")
}

/* ------------------------------------------------------------ platforms */

func cmdPlatforms(args []string) error {
	lx.Register(newClient("", false))
	for _, id := range lx.Sources() {
		p, err := lx.GetProvider(id)
		if err != nil {
			return err
		}
		fmt.Printf("%-3s  %s\n", id, p.Name())
	}
	return nil
}

/* -------------------------------------------------------------- search */

func cmdSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	var source, proxy, outDir, quality string
	var limit, page int
	var jsonOut, verbose, plain, force, tags bool
	fs.StringVar(&source, "source", "all", "kg|kw|wy|tx|mg|<user source>|all")
	fs.IntVar(&limit, "limit", 20, "page size")
	fs.IntVar(&page, "page", 1, "page number")
	fs.StringVar(&outDir, "out", downloadDir(), "directory the picker downloads into (default ./music)")
	fs.StringVar(&quality, "quality", lx.QAuto, qualityFlagHelp())
	fs.BoolVar(&force, "force", false, "overwrite a file that is already in --out")
	fs.BoolVar(&tags, "tags", true, "embed title/artist/album/cover/lyric into the file")
	fs.BoolVar(&jsonOut, "json", false, "emit JSON")
	fs.BoolVar(&plain, "plain", false, "print the table even on a terminal")
	fs.StringVar(&proxy, "proxy", "", "http proxy URL")
	fs.BoolVar(&verbose, "verbose", false, "log every HTTP exchange")
	if err := fs.Parse(args); err != nil {
		return err
	}
	outDir = expandHome(outDir)
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: lxmusic search <keyword> [--source kg|kw|wy|tx|mg|all]")
	}
	keyword := fs.Arg(0)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	c := newClient(proxy, verbose)
	lx.Register(c)

	// One page per source; a source that fails is reported and skipped, so a
	// flaky endpoint never hides the hits the others returned.
	ids := lx.Sources()
	if source != "" && source != "all" {
		ids = []string{source}
	}
	var results []*lx.SearchResult
	var rows []tui.Row
	var hidden []string
	for _, id := range ids {
		p, err := lx.GetProvider(id)
		if err != nil {
			return err
		}
		res, err := p.Search(ctx, keyword, page, limit)
		if err != nil {
			// A source that does not search by design is not a warning when the
			// sweep covers everything. Asked for by name it still explains itself.
			if errors.Is(err, lx.ErrNoSearch) && source == "all" {
				continue
			}
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", id, err)
			continue
		}
		results = append(results, res)
		// The table is a download list, so a row only earns a place if it can be
		// downloaded with what is configured. Anonymous audio is enough on its
		// own; otherwise a user source that serves the platform is enough - that
		// is exactly the capability a user source exists to add, and once one is
		// configured these rows are real results again. --json above still shows
		// everything, untouched.
		if p.AnonymousAudio() || lx.UserSourceFor(id) != nil {
			rows = append(rows, toRows(res)...)
		} else if len(res.List) > 0 {
			hidden = append(hidden, id)
		}
	}
	// --json dumps the raw search responses untouched, so the filter, the cap
	// and the sort are applied only to the table, which is a presentation
	// concern.
	if jsonOut {
		if len(results) == 1 {
			return printJSON(results[0])
		}
		return printJSON(results)
	}
	if len(hidden) > 0 {
		fmt.Fprintf(os.Stderr, "note: 已过滤掉 %s 的结果：这些音源匿名拿不到音频直链，也没有配置能供它们的音源（--json 可看完整搜索结果）\n",
			strings.Join(hidden, " "))
	}

	// Highest fidelity first, then keep the top maxResults. The slice is cut
	// before numbering, so the table reads 1..N with no gaps.
	//
	// The tiers are checked against the gateways before sorting, so the fidelity
	// key is what will actually be downloaded: sorting on the catalogue would cap
	// the table with rows that serve a lower tier and hide the real lossless hits.
	measured, below, unmeasured, missedOf := servedRows(ctx, rows, quality)
	if measured == 0 {
		fmt.Fprintf(os.Stderr, "note: 未能核对实际可下载档位，音质/大小列显示的是各平台目录标称\n")
	} else {
		fmt.Fprintf(os.Stderr, "note: 已核对 %d 行的实际可下载档位，%d 行低于目录标称 — 音质/大小列显示的是实际能拿到的",
			measured, below)
		if unmeasured > 0 {
			// Naming the platforms, not just the count, tells the user which rows
			// to distrust. The rows that miss are the ones whose built-in gateway
			// no longer serves anonymous audio, so the same two platforms are
			// always the ones.
			fmt.Fprintf(os.Stderr, "；另有 %d 行未能核对仍显示标称（%s）", unmeasured,
				strings.Join(missedOf, "、"))
		}
		fmt.Fprintln(os.Stderr)
	}
	sortRows(rows, keyword)
	if len(rows) > maxResults {
		rows = rows[:maxResults]
	}
	for i := range rows {
		rows[i].Index = i + 1
	}

	title := fmt.Sprintf("搜索 %q — %d 条结果", keyword, len(rows))
	if len(rows) == 0 {
		fmt.Printf("搜索 %q: 无结果\n", keyword)
		return nil
	}
	if plain || !tui.Terminal() {
		tui.PrintTable(os.Stdout, 0, title, rows)
		return nil
	}

	prog := &tui.DownloadProgress{}
	// The search deadline belongs to the search. This one carries the whole
	// interactive session - how long the user spends reading the list and
	// choosing - and each download is given its own downloadTimeout below.
	// Deriving the picker from ctx would cap a download at whatever time the
	// search left over, so a 2 minute search left the transfer with about 90
	// seconds and the table was the bottleneck the user never saw.
	session, sessionCancel := context.WithCancel(context.Background())
	defer sessionCancel()
	status, err := tui.Pick(session, tui.Prompt{
		Title: title,
		Rows:  rows,
		Foot:  "保存到 " + outDir,
		Download: func(r tui.Row, dctx context.Context) (string, error) {
			d, cancel := context.WithTimeout(dctx, downloadTimeout)
			defer cancel()
			return downloadSelected(c, r, quality, outDir, force, tags, d, prog.Report)
		},
		SavePath: func(r tui.Row) string {
			return savePathFor(r, outDir)
		},
		Progress: prog,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "note: %v; 改用表格输出\n", err)
		tui.PrintTable(os.Stdout, 0, title, rows)
	} else if status != "" {
		fmt.Println(status)
	}
	return nil
}

/* ------------------------------------------------------- url / download */

func cmdURL(args []string) error {
	fs := flag.NewFlagSet("url", flag.ContinueOnError)
	ref := &refs{}
	ref.register(fs)
	var source, quality, proxy string
	var jsonOut, verbose bool
	fs.StringVar(&source, "source", "", "source id for a bare reference")
	fs.StringVar(&quality, "quality", lx.QAuto, qualityFlagHelp())
	fs.BoolVar(&jsonOut, "json", false, "emit JSON")
	fs.StringVar(&proxy, "proxy", "", "http proxy URL")
	fs.BoolVar(&verbose, "verbose", false, "log every HTTP exchange")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if ref.key == "" && fs.NArg() > 0 {
		ref.key = fs.Arg(0)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := newClient(proxy, verbose)
	lx.Register(c)

	p, s, err := resolveRef(ctx, ref.key, source, ref.search, ref.index, ref.page, ref.limit)
	if err != nil {
		return err
	}
	pl, _, err := resolveAudio(ctx, p, s, quality)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(pl)
	}
	fmt.Printf("song    : %s\n", s.KeyString())
	fmt.Printf("quality : %s\n", pl.Quality)
	fmt.Printf("url     : %s\n", pl.URL)
	for _, k := range sortedKeys(pl.Headers) {
		fmt.Printf("header  : %s: %s\n", k, pl.Headers[k])
	}
	if pl.Size > 0 {
		fmt.Printf("size    : %s\n", lx.FormatBytes(pl.Size))
	}
	if pl.Note != "" {
		fmt.Printf("note    : %s\n", pl.Note)
	}
	return nil
}

func cmdDownload(args []string) error {
	fs := flag.NewFlagSet("download", flag.ContinueOnError)
	ref := &refs{}
	ref.register(fs)
	var source, quality, outDir, proxy string
	var verbose, force, tags bool
	fs.StringVar(&source, "source", "", "source id for a bare reference")
	fs.StringVar(&quality, "quality", lx.QAuto, qualityFlagHelp())
	fs.StringVar(&outDir, "out", downloadDir(), "download directory (default ./music)")
	fs.BoolVar(&force, "force", false, "overwrite a file that is already in --out")
	fs.BoolVar(&tags, "tags", true, "embed title/artist/album/cover/lyric into the file")
	fs.StringVar(&proxy, "proxy", "", "http proxy URL")
	fs.BoolVar(&verbose, "verbose", false, "log every HTTP exchange")
	if err := fs.Parse(args); err != nil {
		return err
	}
	outDir = expandHome(outDir)
	if ref.key == "" && fs.NArg() > 0 {
		ref.key = fs.Arg(0)
	}
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	c := newClient(proxy, verbose)
	lx.Register(c)

	p, s, err := resolveRef(ctx, ref.key, source, ref.search, ref.index, ref.page, ref.limit)
	if err != nil {
		return err
	}
	var on func(done, total int64)
	stop := func() {}
	if tui.Terminal() {
		prog := &tui.DownloadProgress{}
		on = prog.Report
		stop = pumpProgress(os.Stdout, prog, s.KeyString())
	}
	// With no table above, the promise is the tier the reference was asked for -
	// for --quality auto, the best rendition the platform announced.
	advertised := s.QualityByType(quality).Type
	status, err := downloadOne(ctx, c, p, s, quality, advertised, outDir, force, tags, on)
	stop()
	if err != nil {
		return err
	}
	fmt.Println(status)
	return nil
}

// qualityFlagHelp is the --quality vocabulary, generated from the same table
// the ranking uses, so the help text cannot drift out of step with what the CLI
// actually accepts.
func qualityFlagHelp() string {
	return strings.Join(lx.QualityList(), "|") + "  default auto (best available)"
}

// progressTick is how often the bar is redrawn. A variable, not a constant, so
// a test can shorten it rather than sleeping on a real cadence.
var progressTick = 100 * time.Millisecond

// pumpProgress redraws one progress line while a download runs, so the plain
// "download" command reports the transfer the way the picker does. The caller
// gates it on tui.Terminal, so piped output stays byte-clean.
//
// The bar goes to stdout rather than stderr: the preview and note warnings are
// emitted to stderr partway through the transfer, and a \r bar on the same
// stream would overwrite them. stop returns only after the final line has been
// printed and cleared, so the saved line lands on its own line; calling it more
// than once is a no-op.
func pumpProgress(w io.Writer, p *tui.DownloadProgress, label string) (stop func()) {
	stopCh := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		tick := time.NewTicker(progressTick) // the picker polls at the same rate
		defer tick.Stop()
		for {
			select {
			case <-stopCh:
				_, _ = fmt.Fprintf(w, "\r%s\n", p.Line(label))
				return
			case <-tick.C:
				_, _ = fmt.Fprintf(w, "\r%s", p.Line(label))
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(stopCh)
			<-finished
		})
	}
}

// resolveAudio asks for one rendition, preferring a configured user source, and
// reports the id of whoever answered - "" when the named provider did. When a
// user source takes over and answers with no note of its own, one is added: the
// built-in's "this is a short preview" note would describe the other request,
// not the bytes that were actually served.
//
// A user source that fails does not fail the request: the built-in answers next.
// The user source is a script hosted somewhere, and it goes stale without
// warning - a 500 is the common shape. Failing closed turns a lower-fidelity
// download into no download, and for mg in particular the user source is the
// whole reason audio is routed away from the built-in at all.
// userSourceBreaks is how many times in a row a user source has to fail before
// the session stops asking it. A source is a script hosted somewhere and it
// times out at 25 s per call, so two consecutive misses cost half a minute
// before the built-in is asked at all.
const userSourceBreaks = 2

var (
	userSourceFailsMu sync.Mutex
	userSourceFails   = map[string]int{}
)

// userSourceBlocked reports whether src has failed often enough to stop asking
// it. Scoped per platform, not per song: a script that has stopped answering
// does not answer for one song only, and remembering it per song would pay the
// timeout again on every row.
func userSourceBlocked(src string) bool {
	userSourceFailsMu.Lock()
	defer userSourceFailsMu.Unlock()
	return userSourceFails[src] >= userSourceBreaks
}

// noteUserSource folds one outcome in. A success clears the count, so a script
// that comes back to life starts serving again instead of staying skipped.
func noteUserSource(src string, ok bool) {
	userSourceFailsMu.Lock()
	defer userSourceFailsMu.Unlock()
	if ok {
		delete(userSourceFails, src)
		return
	}
	userSourceFails[src]++
}

func resolveAudio(ctx context.Context, p lx.Provider, s *lx.Song, quality string) (*lx.PlayerURL, string, error) {
	q := s.QualityByType(quality)
	by := lx.AudioProvider(p, s)
	if by != p && userSourceBlocked(by.ID()) {
		by = p
	}
	pl, err := by.MusicURL(ctx, s, q)
	if pl != nil {
		if by != p {
			noteUserSource(by.ID(), true)
			if pl.Note == "" {
				pl.Note = "served by user source " + by.ID()
			}
		}
		return pl, by.ID(), nil
	}
	if by == p {
		if err == nil {
			err = fmt.Errorf("%s: no rendition", p.ID())
		}
		return nil, "", err
	}
	noteUserSource(by.ID(), false)
	fb, fbErr := p.MusicURL(ctx, s, q)
	if fb != nil {
		if fb.Note == "" {
			fb.Note = fmt.Sprintf("user source %q failed, the built-in answered instead", by.ID())
		}
		return fb, "", nil
	}
	if err == nil {
		err = fmt.Errorf("%s: no rendition", by.ID())
	}
	if fbErr == nil {
		fbErr = fmt.Errorf("%s: no rendition", p.ID())
	}
	return nil, "", fmt.Errorf("%v; built-in also failed: %v", err, fbErr)
}

// downloadOne resolves the rendition and writes the file, returning a one-line
// status. Shared by "download" and the interactive picker. advertised is the tier
// the user was shown - the row's quality in the picker, the tier the reference
// was asked for on the command line - and the file that lands is held to it: a
// transfer that comes back in a different family fails rather than reporting a
// save. on receives the bytes written as they happen, or nil when nothing is
// watching. When force is false a file already present is left exactly as it is,
// which is the promise the help text has always made; --force opts back into
// overwriting.
func downloadOne(ctx context.Context, c *lx.Client, p lx.Provider, s *lx.Song,
	quality, advertised, outDir string, force, tags bool, on func(done, total int64)) (string, error) {
	pl, user, err := resolveAudio(ctx, p, s, quality)
	if err != nil {
		return "", err
	}
	name := pl.FileName
	if name == "" {
		// s.Name can be empty when a song is known only by id and the metadata
		// lookup failed, so fall back to the reference: an empty base is what
		// produced a file called ".mp3".
		base := s.Name
		if base == "" {
			base = s.KeyString()
		}
		name = base + lx.AudioExt(pl.Quality, pl.URL)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(outDir, sanitizeFileName(name))
	if !force {
		if st, err := os.Stat(path); err == nil && !st.IsDir() && st.Size() > 0 {
			return fmt.Sprintf("＝ 已存在,未下载 %s (%s) — --force 可覆盖", path, lx.FormatBytes(st.Size())), nil
		}
	}
	n, err := c.DownloadFileOn(ctx, pl.URL, path, pl.Headers, on)
	if err != nil {
		return "", err
	}
	// Held against the tier the list showed, not against pl.Quality: the gateway
	// already reported its own compromise in pl.Note. The file is left on disk and
	// the path named, so the user can keep it or clear it - deleting a file this
	// far into the call would be an unannounced destruction.
	if why := formatMismatch(advertised, storedExt(pl, path)); why != "" {
		return "", fmt.Errorf("下载失败：%s；文件已写入 %s (%s)", why, path, lx.FormatBytes(n))
	}
	// The notes are returned, not printed: the picker erases its screen after
	// every repaint, so a stderr note was visible for one frame and then gone.
	// "已保存" is the only line that survived, which is how a 11 second clip got
	// saved as if it were the track. Warnings lead the status, because the
	// status block is capped and truncates from the end - the save line is the
	// one that can afford to be cut.
	notes := []string{}
	if m := previewNote(s.Duration, n); m != "" {
		notes = append(notes, m)
	}
	// Tagging is best effort and reported through the status: the audio is already
	// on disk, so a cover host that goes down must not turn a finished download
	// into an error, and the picker wipes its screen after every repaint.
	if tags {
		if got, err := tagDownload(ctx, c, p, s, path); err != nil {
			notes = append(notes, fmt.Sprintf("音频已保存,标签未写入: %v", err))
		} else if got != "" {
			notes = append(notes, got)
		}
	}
	if user != "" {
		notes = append(notes, fmt.Sprintf("音频由用户音源 %q 提供,不是内置 %q", user, p.ID()))
	}
	if pl.Note != "" {
		notes = append(notes, pl.Note)
	}
	saved := fmt.Sprintf("✓ 已保存 %s (%s)", path, lx.FormatBytes(n))
	for i := len(notes) - 1; i >= 0; i-- {
		saved = notes[i] + "\n" + saved
	}
	return saved, nil
}

// tagDownload embeds title/artist/album/cover/lyric into the file that was just
// written, and returns what landed. It never fails the download — the audio is
// on disk, and a cover or lyric host going down must not turn a finished
// download into an error. It checks the extension and the name first: the writer
// knows only mp3 and flac, and a song with no name has nothing to put in there.
func tagDownload(ctx context.Context, c *lx.Client, p lx.Provider, s *lx.Song, path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".mp3" && ext != ".flac" {
		return "", nil
	}
	if strings.TrimSpace(s.Name) == "" {
		return "", nil
	}
	m := &lx.Metadata{Title: s.Name, Artist: s.Singer, Album: s.Album}
	got := "信息"
	if strings.TrimSpace(s.Pic) != "" {
		if b, err := c.GetBytes(ctx, s.Pic, nil); err == nil && len(b) > 0 {
			m.Picture, m.PictureMime = b, lx.DetectImageMime(b)
			got += "/封面"
		}
	}
	if ly, err := p.Lyric(ctx, s); err == nil && strings.TrimSpace(ly.Plain) != "" {
		m.Lyric = strings.TrimSpace(ly.Plain)
		got += "/歌词"
	}
	if err := lx.WriteTags(path, m); err != nil {
		return "", err
	}
	return "已写入标签 " + got, nil
}

// downloadSelected saves one table row. The row already carries the song it was
// drawn from, so that is used: re-resolving the reference would be a network
// round trip for nothing, and when the lookup fails the song arrives without a
// name, which is how a file came out called ".mp3".
func downloadSelected(c *lx.Client, r tui.Row, quality, outDir string, force, tags bool,
	ctx context.Context, on func(done, total int64)) (string, error) {
	s := r.Song
	if s == nil {
		// A row that only ever had a reference in it.
		_, q, err := resolveRef(ctx, r.Ref, "", "", 0, 0, 0)
		if err != nil {
			return "", err
		}
		s = q
	}
	p, err := lx.GetProvider(s.Source)
	if err != nil {
		return "", err
	}
	return downloadOne(ctx, c, p, s, quality, r.Quality, outDir, force, tags, on)
}

// savePathFor predicts where a row will be written, without a network round
// trip. The confirm prompt appears between the Enter keypress and this
// returning, so anything blocking in here reads as the UI freezing. The download
// settles the real name afterwards and reports the exact path it wrote; this is
// the same fallback downloadOne uses when the platform supplies no file name.
// The directory is always the truth - only the file name is a prediction.
func savePathFor(r tui.Row, outDir string) string {
	name := r.Name
	if name == "" {
		name = r.Ref
	}
	return filepath.Join(outDir, sanitizeFileName(name+lx.AudioExt(r.Quality, "")))
}

// previewNote warns when the platform handed back a fragment of the song
// rather than the track. kuwo's anonymous convert_url serves the same ~181 KB
// preview object for every song whatever the bitrate, so the size alone is a
// reliable tell: a real mp3 runs at least about 16 KB per second of runtime.
// A bare "source:id" reference carries no runtime, so a fixed floor stands in
// for it - no track longer than a minute is ever that small. It returns the
// line rather than printing it: the picker wipes its screen after every repaint,
// so a stderr note was gone before anyone could read it.
func previewNote(durationSec int, got int64) string {
	if got <= 0 {
		return ""
	}
	if durationSec > 0 {
		if got >= int64(durationSec)*16000*3/4 {
			return ""
		}
	} else if got >= 400*1024 {
		return ""
	}
	if durationSec > 0 {
		return fmt.Sprintf("⚠ 只是 %s 试听,不是 %s 整首;完整曲目要用户音源或登录",
			lx.FormatBytes(got), lx.FormatDuration(durationSec))
	}
	return fmt.Sprintf("⚠ 只有 %s,疑似试听片段;完整曲目要用户音源或登录", lx.FormatBytes(got))
}

// tierFamily and extFamily reduce a quality label and a file extension to the
// one distinction that matters: lossless or lossy. Comparing extensions
// directly would fail a row that promises 128k and lands as a .ogg or .aac,
// which is the same rendition in another container, and "" is returned when
// neither side can be classified so an unknown label cannot fail a transfer that
// otherwise succeeded.
func tierFamily(t string) string {
	switch t {
	case lx.QFlac, lx.QFlac24, lx.QWav, lx.QApe:
		return "无损"
	case lx.Q320, lx.Q192, lx.Q128:
		return "有损"
	}
	return ""
}

func extFamily(ext string) string {
	switch ext {
	case ".flac", ".wav", ".ape":
		return "无损"
	case ".mp3", ".ogg", ".m4a", ".aac", ".wma", ".opus", ".mka":
		return "有损"
	}
	return ""
}

// storedExt is the format the bytes on disk actually are. The URL wins over the
// file name: a platform can hand back a file name with a different extension than
// the payload it serves, which is how a 3.8MB mp3 once came out named .flac.
func storedExt(pl *lx.PlayerURL, path string) string {
	if e := lx.PathExt(pl.URL); e != "" {
		return e
	}
	return strings.ToLower(filepath.Ext(path))
}

// formatMismatch reports that the file that landed is a worse family than the
// list promised - lossless promised, lossy stored. An upgrade is not flagged: the
// user asked for one thing and got a better rendition, which is not a failure.
// "" also covers a match and anything that cannot be classified, since an unknown
// tier or an unrecognized extension must not fail a transfer that otherwise
// succeeded.
func formatMismatch(advertised, stored string) string {
	want, got := tierFamily(advertised), extFamily(stored)
	if want == "" || got == "" || familyRank(got) >= familyRank(want) {
		return ""
	}
	return fmt.Sprintf("列表标注 %s，实际写入的是%s %s 文件", advertised, got, stored)
}

// familyRank orders the families worst first, so a comparison reads as "is what
// landed at least as good as what was promised".
func familyRank(f string) int {
	if f == "无损" {
		return 1
	}
	return 0
}

/* --------------------------------------------------------------- lyric */

func cmdLyric(args []string) error {
	fs := flag.NewFlagSet("lyric", flag.ContinueOnError)
	ref := &refs{}
	ref.register(fs)
	var source, proxy string
	var showTrans, jsonOut, verbose bool
	fs.StringVar(&source, "source", "", "source id for a bare reference")
	fs.BoolVar(&showTrans, "show-trans", false, "also print the translation")
	fs.BoolVar(&jsonOut, "json", false, "emit JSON")
	fs.StringVar(&proxy, "proxy", "", "http proxy URL")
	fs.BoolVar(&verbose, "verbose", false, "log every HTTP exchange")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if ref.key == "" && fs.NArg() > 0 {
		ref.key = fs.Arg(0)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := newClient(proxy, verbose)
	lx.Register(c)

	p, s, err := resolveRef(ctx, ref.key, source, ref.search, ref.index, ref.page, ref.limit)
	if err != nil {
		return err
	}
	if s.Name == "" {
		return fmt.Errorf("this source needs a song name for lyrics: re-run with --search <keyword> --index N")
	}
	ly, err := p.Lyric(ctx, s)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(ly)
	}
	if ly.Format == "krc" {
		fmt.Fprintln(os.Stderr, "note: kugou returned KRC karaoke data (decoded)")
	}
	fmt.Println(ly.Plain)
	if showTrans && strings.TrimSpace(ly.Trans) != "" {
		fmt.Println("\n--- translation ---")
		fmt.Println(ly.Trans)
	}
	if strings.TrimSpace(ly.Roman) != "" {
		fmt.Println("\n--- romanization ---")
		fmt.Println(ly.Roman)
	}
	return nil
}

/* ------------------------------------------------------------- pic/play */

func cmdPic(args []string) error {
	fs := flag.NewFlagSet("pic", flag.ContinueOnError)
	ref := &refs{}
	ref.register(fs)
	var source, proxy string
	var jsonOut, verbose bool
	fs.StringVar(&source, "source", "", "source id for a bare reference")
	fs.BoolVar(&jsonOut, "json", false, "emit JSON")
	fs.StringVar(&proxy, "proxy", "", "http proxy URL")
	fs.BoolVar(&verbose, "verbose", false, "log every HTTP exchange")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if ref.key == "" && fs.NArg() > 0 {
		ref.key = fs.Arg(0)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := newClient(proxy, verbose)
	lx.Register(c)

	p, s, err := resolveRef(ctx, ref.key, source, ref.search, ref.index, ref.page, ref.limit)
	if err != nil {
		return err
	}
	u, err := p.Pic(ctx, s)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(map[string]string{"url": u, "song": s.KeyString()})
	}
	fmt.Println(u)
	return nil
}

func cmdPlay(args []string) error {
	fs := flag.NewFlagSet("play", flag.ContinueOnError)
	ref := &refs{}
	ref.register(fs)
	var source, quality, proxy string
	var verbose bool
	fs.StringVar(&source, "source", "", "source id for a bare reference")
	fs.StringVar(&quality, "quality", lx.QAuto, qualityFlagHelp())
	fs.StringVar(&proxy, "proxy", "", "http proxy URL")
	fs.BoolVar(&verbose, "verbose", false, "log every HTTP exchange")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if ref.key == "" && fs.NArg() > 0 {
		ref.key = fs.Arg(0)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := newClient(proxy, verbose)
	lx.Register(c)

	p, s, err := resolveRef(ctx, ref.key, source, ref.search, ref.index, ref.page, ref.limit)
	if err != nil {
		return err
	}
	fmt.Printf("playing: %s - %s\n", s.Singer, s.Name)
	pl, user, err := resolveAudio(ctx, p, s, quality)
	if err != nil {
		fmt.Fprintf(os.Stderr, "note: no direct stream (%v)\n", err)
	} else {
		fmt.Printf("quality: %s\nstream : %s\n", pl.Quality, pl.URL)
		if user != "" {
			fmt.Printf("note   : stream served by user source %q\n", user)
		}
		if pl.Note != "" {
			fmt.Printf("note   : %s\n", pl.Note)
		}
		if len(pl.Headers) > 0 {
			for _, k := range sortedKeys(pl.Headers) {
				fmt.Printf("header : %s: %s\n", k, pl.Headers[k])
			}
		}
	}
	openInBrowser(s.PageURL)
	_ = c
	return nil
}

/* -------------------------------------------------------------- detail */

func cmdDetail(args []string) error {
	fs := flag.NewFlagSet("detail", flag.ContinueOnError)
	ref := &refs{}
	ref.register(fs)
	var source, proxy string
	var jsonOut, verbose bool
	fs.StringVar(&source, "source", "", "source id for a bare reference")
	fs.BoolVar(&jsonOut, "json", false, "emit JSON")
	fs.StringVar(&proxy, "proxy", "", "http proxy URL")
	fs.BoolVar(&verbose, "verbose", false, "log every HTTP exchange")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if ref.key == "" && fs.NArg() > 0 {
		ref.key = fs.Arg(0)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := newClient(proxy, verbose)
	lx.Register(c)

	p, s, err := resolveRef(ctx, ref.key, source, ref.search, ref.index, ref.page, ref.limit)
	if err != nil {
		return err
	}
	// A song with no name was already looked up by resolveRef, which reported
	// why, and no source can give more for it: Pic re-enters Detail, so calling
	// it here would ask the same question and print the same failure.
	if s.Name != "" {
		if err := p.Detail(ctx, s); err != nil {
			fmt.Fprintf(os.Stderr, "note: %v\n", err)
		}
		if _, err := p.Pic(ctx, s); err != nil && s.Pic == "" {
			fmt.Fprintf(os.Stderr, "note: %v\n", err)
		}
	}
	if jsonOut {
		return printJSON(s)
	}
	if s.Name == "" {
		return nil
	}
	printSongBlock(1, s)
	return nil
}

/* --------------------------------------------------------------- share */

// cmdShare answers "give me a link for this song name I can hand to someone".
// Two kinds of link have very different lifetimes, so both are printed:
//
//	page   — the platform's own page, stable for years. This is what you share.
//	stream — a direct audio URL with an embedded vkey/token that the platform
//	         expires on the order of minutes to hours. Useful for piping into
//	         ffplay/ffmpeg, useless as a permanent share.
func cmdShare(args []string) error {
	fs := flag.NewFlagSet("share", flag.ContinueOnError)
	ref := &refs{}
	ref.register(fs)
	var source, quality, proxy string
	var jsonOut, verbose bool
	fs.StringVar(&source, "source", "all", "kg|kw|wy|tx|mg|<user source>|all")
	fs.StringVar(&quality, "quality", lx.QAuto, qualityFlagHelp())
	fs.StringVar(&proxy, "proxy", "", "http proxy URL")
	fs.BoolVar(&jsonOut, "json", false, "emit JSON")
	fs.BoolVar(&verbose, "verbose", false, "log every HTTP exchange")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if ref.key == "" && fs.NArg() > 0 {
		ref.key = fs.Arg(0)
	}
	if ref.key == "" && ref.search == "" {
		return fmt.Errorf("usage: lxmusic share <song-ref|keyword> [--source kg|kw|wy|tx|mg|all]")
	}
	// "all" is the keyword-sweep default, but a key already names its own source,
	// and passing "all" through would look it up as one. The sweep below keeps
	// "all" untouched, since it is the thing that makes the sweep happen.
	if ref.key != "" && (source == "all" || source == "") {
		if src, _, ok := strings.Cut(ref.key, ":"); ok && src != "" {
			source = src
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	c := newClient(proxy, verbose)
	lx.Register(c)

	var p lx.Provider
	var s *lx.Song
	if ref.key != "" {
		// A reference: one song, the same path every other command uses.
		q, song, err := resolveRef(ctx, ref.key, source, ref.search, ref.index, ref.page, ref.limit)
		if err != nil {
			return err
		}
		p, s = q, song
	} else if source == "all" || source == "" {
		// A keyword with no source pinned: try every source in turn, first hit wins.
		for _, id := range lx.Sources() {
			q, err := lx.GetProvider(id)
			if err != nil {
				continue
			}
			res, err := q.Search(ctx, ref.search, ref.page, ref.limit)
			if err != nil || len(res.List) == 0 {
				continue
			}
			if hit, err2 := lx.SongFromSearchResult(res, ref.index); err2 == nil {
				p, s = q, hit
				break
			}
		}
		if p == nil {
			return fmt.Errorf("no hits for %q on any source", ref.search)
		}
	} else {
		q, err := lx.GetProvider(source)
		if err != nil {
			return err
		}
		res, err := q.Search(ctx, ref.search, ref.page, ref.limit)
		if err != nil {
			return err
		}
		s, err = lx.SongFromSearchResult(res, ref.index)
		if err != nil {
			return err
		}
		p = q
	}
	if s.PageURL == "" {
		s.PageURL = defaultPageURL(p.ID(), s)
	}

	stream, streamQuality, streamNote, streamErr := "", "", "", error(nil)
	if pl, _, err := resolveAudio(ctx, p, s, quality); err == nil {
		stream, streamQuality, streamNote = pl.URL, pl.Quality, pl.Note
	} else {
		streamErr = err
	}

	out := map[string]any{
		"name":    s.Name + " - " + s.Singer,
		"source":  p.ID(),
		"ref":     s.KeyString(),
		"page":    s.PageURL,
		"quality": streamQuality,
		"stream":  stream,
	}
	if streamNote != "" {
		out["note"] = streamNote
	}
	if streamErr != nil {
		out["streamError"] = streamErr.Error()
	}
	if jsonOut {
		return printJSON(out)
	}
	fmt.Printf("song    : %s - %s\n", s.Name, s.Singer)
	fmt.Printf("source  : %s   ref: %s\n", p.ID(), s.KeyString())
	fmt.Printf("page    : %s\n", s.PageURL)
	if stream != "" {
		fmt.Printf("quality : %s\n", streamQuality)
		fmt.Printf("stream  : %s\n", stream)
		if streamNote != "" {
			fmt.Printf("note    : %s\n", streamNote)
		}
	} else {
		fmt.Fprintf(os.Stderr, "note: no direct stream (%v)\n", streamErr)
	}
	fmt.Println("\nThe page link is the one to share — it never expires. The stream link")
	fmt.Println("carries a platform vkey/token and stops working within minutes to hours.")
	return nil
}

/* -------------------------------------------------------- source config */

func cmdSource(args []string) error {
	// flagsFirst hoists every flag ahead of the subcommand, so find the first
	// bare token, skipping the value of each flag that takes one, then drop it
	// from the list the subcommand handlers parse.
	sub, rest := "list", args
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
			}
			continue
		}
		sub = a
		rest = append(append([]string{}, args[:i]...), args[i+1:]...)
		break
	}
	switch sub {
	case "list", "ls":
		cfg, err := lx.LoadConfig()
		if err != nil {
			return err
		}
		dir, _ := lx.ConfigDir()
		fmt.Printf("config : %s\n", dir)
		node := cfg.Node
		if node == "" {
			if found, err := exec.LookPath("node"); err == nil {
				node = found
			} else {
				node = "(not found — user sources need node on PATH)"
			}
		}
		fmt.Printf("node   : %s\n", node)
		if len(cfg.Sources) == 0 {
			fmt.Println("\nno user sources configured. Add one with:")
			fmt.Println("  lxmusic source add --name flower --url https://raw.githubusercontent.com/pdone/lx-music-source/main/flower/latest.js")
			return nil
		}
		fmt.Println("\nname        source  url")
		for _, s := range cfg.Sources {
			src := s.Source
			if src == "" {
				src = "(any)"
			}
			fmt.Printf("%-10s %-8s %s\n", s.Name, src, s.URL)
		}
		return nil
	case "add":
		return cmdSourceAdd(rest)
	case "rm", "remove", "del":
		return cmdSourceRemove(rest)
	case "path":
		dir, err := lx.ConfigDir()
		if err != nil {
			return err
		}
		fmt.Println(dir)
		return nil
	case "test":
		return cmdSourceType(rest)
	case "info", "describe":
		return cmdSourceInfo(rest)
	default:
		return fmt.Errorf("unknown source subcommand %q (list|add|rm|path|info|test)", sub)
	}
}

// cmdSourceInfo runs the script once and prints what it announced. The quality
// column is what --quality can actually be honoured with.
func cmdSourceInfo(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: lxmusic source info <name>")
	}
	c := newClient("", false)
	lx.Register(c)
	p, err := lx.GetProvider(args[0])
	if err != nil {
		return err
	}
	u, ok := p.(*lx.UserSource)
	if !ok {
		return fmt.Errorf("%q is not a user source (lxmusic source list)", args[0])
	}
	if err := u.Describe(context.Background()); err != nil {
		return err
	}
	fmt.Printf("name    : %s\n", u.Spec.Name)
	if u.Info.Name != "" {
		fmt.Printf("script  : %s", u.Info.Name)
		if u.Info.Version != "" {
			fmt.Printf(" v%s", u.Info.Version)
		}
		fmt.Println()
	}
	if u.Info.Description != "" {
		fmt.Printf("desc    : %s\n", u.Info.Description)
	}
	src := u.Spec.Source
	if src == "" {
		src = "(any)"
	}
	fmt.Printf("serves  : %s\n", src)
	if len(u.Cap) == 0 {
		fmt.Println("\ndeclared : nothing — the script never called lx.send(inited)")
		return nil
	}
	fmt.Println("\nplatform  actions    qualities")
	names := make([]string, 0, len(u.Cap))
	for k := range u.Cap {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		cap := u.Cap[k]
		fmt.Printf("%-9s %-11s %s\n", k, strings.Join(cap.Actions, ","), strings.Join(cap.Qualitys, ","))
	}
	return nil
}

func cmdSourceAdd(args []string) error {
	fs := flag.NewFlagSet("source add", flag.ContinueOnError)
	var name, u, source, node string
	var timeout, ttl int
	fs.StringVar(&name, "name", "", "short id, e.g. flower")
	fs.StringVar(&u, "url", "", "script url or local file path")
	fs.StringVar(&source, "source", "", "platform this source serves (kg|kw|wy|tx|mg)")
	fs.StringVar(&node, "node", "", "node executable, defaults to PATH")
	fs.IntVar(&timeout, "timeout", 25, "per-request seconds")
	fs.IntVar(&ttl, "ttl", 24, "script cache ttl hours")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if name == "" || u == "" {
		return fmt.Errorf("usage: lxmusic source add --name <id> --url <script url or path>")
	}
	cfg, err := lx.LoadConfig()
	if err != nil {
		return err
	}
	spec := lx.UserSourceSpec{Name: name, URL: u, Source: source, Node: node, TimeoutSec: timeout, CacheTTLHrs: ttl}
	if err := cfg.Set(spec); err != nil {
		return err
	}
	fmt.Printf("added source %q -> %s\n", name, u)
	return nil
}

func cmdSourceRemove(args []string) error {
	fs := flag.NewFlagSet("source rm", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: lxmusic source rm <name>")
	}
	cfg, err := lx.LoadConfig()
	if err != nil {
		return err
	}
	if err := cfg.Remove(fs.Arg(0)); err != nil {
		return err
	}
	fmt.Printf("removed source %q\n", fs.Arg(0))
	return nil
}

// cmdSourceType downloads the script and runs its `musicUrl` action once to
// prove the node bridge works end to end before the user relies on it.
func cmdSourceType(args []string) error {
	fs := flag.NewFlagSet("source test", flag.ContinueOnError)
	var quality string
	fs.StringVar(&quality, "quality", lx.QAuto, "rendition to resolve")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: lxmusic source test <name> <song-ref> [--quality ...]")
	}
	c := newClient("", false)
	lx.Register(c)
	p, err := lx.GetProvider(fs.Arg(0))
	if err != nil {
		return err
	}
	u, ok := p.(*lx.UserSource)
	if !ok {
		return fmt.Errorf("%q is not a user source (lxmusic source list)", fs.Arg(0))
	}
	lookup, err := lx.GetProvider(u.Spec.Source)
	if err != nil {
		return err
	}
	s, err := lx.SongRef(context.Background(), lookup, fs.Arg(1), "", 1, 1, 20)
	if err != nil {
		return err
	}
	if s.Name == "" {
		if err := lx.Hydrate(context.Background(), lookup, s); err != nil {
			fmt.Fprintf(os.Stderr, "note: %v\n", err)
		}
	}
	if s.PageURL == "" {
		s.PageURL = defaultPageURL(lookup.ID(), s)
	}
	fmt.Printf("source  : %s (%s)\n", u.Spec.Name, u.Spec.URL)
	fmt.Printf("song    : %s - %s   [%s]\n", s.Name, s.Singer, s.KeyString())
	pl, err := u.MusicURL(context.Background(), s, s.QualityByType(quality))
	if err != nil {
		return err
	}
	fmt.Printf("quality : %s\nurl     : %s\n", pl.Quality, pl.URL)
	if ly, err := u.Lyric(context.Background(), s); err == nil {
		fmt.Printf("\nlyric   : OK (%s, %d bytes)\n", ly.Format, len(ly.Plain))
	} else {
		fmt.Printf("lyric  : unavailable (%v)\n", err)
	}
	return nil
}
