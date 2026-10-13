// Command routing. The cmd* handlers keep their own flag.FlagSet, so every flag
// is defined exactly once; cobra only decides which subcommand runs and where
// the help text comes from.
package main

import (
	"github.com/spf13/cobra"
)

// newCmd wires one existing cmd* handler into the cobra tree.
//
// DisableFlagParsing is load-bearing: cobra resolves the subcommand with its
// own stripFlags, which already skips over flag values, so flags may stand in
// front of the keyword ("lxmusic url --source kw kg:123") and the untouched
// argument vector still reaches the handler. Left enabled, pflag would either
// reject the flags it does not know about or, with the UnknownFlags allowlist,
// silently drop them along with their values — the search keyword would vanish.
func newCmd(use, short, long, example string, run func([]string) error) *cobra.Command {
	return &cobra.Command{
		Use:                use,
		Short:              short,
		Long:               long,
		Example:            example,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				// cmd.Help rather than flag.ErrHelp: with flag parsing disabled
				// cobra never sees --help, so the sentinel would just come back
				// up as "error: flag: help requested".
				return cmd.Help()
			}
			return run(flagsFirst(args))
		},
	}
}

// wantsHelp short-circuits so --help prints the cobra page instead of reaching
// flag.FlagSet, which would print its own usage to stderr first and leave two
// overlapping help blocks on screen.
func wantsHelp(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "-help" || a == "--help" {
			return true
		}
	}
	return false
}

const refHelp = `A song reference is one of:

  <source>:<id>             e.g. kg:20505418B3A52A7A958BF0AED0EBFBA2E9A818B7
                            kw:80196894   wy:347230   tx:<media_mid>
  --search <keyword> ...    look the song up through the search API instead.
                            Required for kugou lyrics, which need the song name.`

const userSourceHelp = `User sources are the desktop app's 自定义音源, as many as you like:

  lxmusic source add --name flower --url https://raw.githubusercontent.com/pdone/lx-music-source/main/flower/latest.js

A source script is downloaded and cached, then run in a node subprocess that
answers lx-music's {source, action, info} requests — the same contract the
desktop app uses inside a hidden Electron window. One node per source lives for
the whole run, so the script's init runs once instead of once per song, which is
what the app gets from its long-lived WebView. If it dies it is restarted.
node must be on PATH.
A source does not search; look the song up on a platform and pass it along:
"lxmusic url kw:228908 --source flower". --quality settles on the highest
quality the source announced, so "auto" never asks for a rendition it cannot
deliver. Config and cache live in the directory shown by "lxmusic source path".`

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "lxmusic",
		Short: "CLI port of lx-music-desktop's search / download / play / lyric logic",
		Long: `lxmusic is a CLI port of the core search / download / play / lyric logic
reverse engineered out of lx-music-desktop 2.12.6.

Search results print as a table — 序号 · 平台名称 · 音乐名称 · 时长 · 作者 · 音质 ·
文件大小, with the size in MB on each row, showing the highest quality each hit
can serve. Rows are ordered best fidelity first, then largest file: 酷狗 may call
晴天 flac24bit where 酷我 and QQ call that same 52.83 MB file flac, so two tiers
can hold one file and the size breaks the tie. At most 50 rows are shown. Column
widths are measured from the content rather than fixed, so nothing is hidden —
only the title and the artist give ground, and never below their own caption.

On a terminal the table is interactive: up/down (or j/k) moves the selection one
row, Enter asks whether to download and shows where the file will go, y downloads
it into --out showing a progress bar, n goes back, q or Esc quits. "lxmusic
download" shows the same bar. --plain forces the plain table for piping; --json
keeps emitting JSON.

` + refHelp + "\n\n" + userSourceHelp,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newCmd("platforms", "List the built-in platforms",
			"Prints every built-in platform id and its display name.\n\n"+userSourceHelp,
			"  lxmusic platforms", cmdPlatforms),
		newCmd("search <keyword>", "Search for songs and print a table",
			"Searches every platform (or one with --source), showing the highest quality each hit serves.\n\n"+
				"音质 and 文件大小 are the rendition the gateway will actually hand out, not the tier the\n"+
				"platform's catalogue advertises: each row's rendition is asked for once before the table\n"+
				"is built, and the sort follows what came back. The check takes a couple of seconds and\n"+
				"runs on a budget of its own, so a platform that will not answer is left at the advertised\n"+
				"tier and named in a note on stderr. migu is the common case - it advertises flac24bit\n"+
				"and serves a 128k MP3 to an unauthenticated client.\n\n"+
				"At most 50 rows are kept, best fidelity first then largest file, so capping happens\n"+
				"after the sort and the fifty that survive are the highest quality. --limit is the\n"+
				"page size asked of each platform; it does not lift the 50-row cap. --json emits the\n"+
				"raw search responses untouched, so it is neither capped nor sorted.\n\n"+
				"  --source   kg|kw|wy|tx|mg|<user source>|all  default all\n"+
				"  --limit    page size asked per source  default 20\n"+
				"  --page     page number                default 1\n"+
				"  --out      download dir for the picker  default ./music (cwd)\n"+
				"  --quality  flac24bit|flac|wav|ape|320k|192k|128k|auto  default auto (best available)\n"+
				"  --force    overwrite a file already in --out\n"+
				"  --tags     embed title/artist/album/cover/lyric  default on\n"+
				"  --plain    print the table even on a terminal\n"+
				"  --json     emit JSON instead of the table\n"+
				"  --proxy    http proxy URL\n"+
				"  --verbose  log every HTTP exchange",
			"  lxmusic search 晴天\n"+
				"  lxmusic search 晴天 --source kw --limit 10\n"+
				"  lxmusic search 晴天 --out ~/Downloads --quality flac\n"+
				"  lxmusic search 晴天 --plain   # pipeable\n"+
				"  lxmusic search 晴天 --json", cmdSearch),
		newCmd("url <song-ref>", "Resolve a song to a direct audio URL",
			"Resolves the requested rendition and prints the URL plus the headers the player must send.\n\n"+refHelp+"\n\n"+
				"  --source   source id for a bare reference\n"+
				"  --quality  flac24bit|flac|wav|ape|320k|192k|128k|auto  default auto (best available)\n"+
				"  --json     emit JSON\n"+
				"  --proxy    http proxy URL\n"+
				"  --verbose  log every HTTP exchange",
			"  lxmusic url kw:228908\n"+
				"  lxmusic url kg:20505418B3A52A7A958BF0AED0EBFBA2E9A818B7 --quality flac\n"+
				"  lxmusic url kw:228908 --source flower   # through a user source", cmdURL),
		newCmd("download <song-ref>", "Download a song to disk",
			"Downloads the requested rendition into --out, overwriting nothing: a file that is already\n"+
				"there is left exactly as it is and the command reports that instead. --force opts out.\n"+
				"A zero-byte file left by an earlier failed run is treated as not-there and retried.\n\n"+
				"On a terminal a progress bar tracks the transfer; when stdout is piped the command\n"+
				"stays silent until the file is saved.\n\n"+
				"After the transfer the tag is written into the file itself: title, artist and album as\n"+
				"text, plus the cover and the synced lyric, read back off the song just downloaded. MP3\n"+
				"gets ID3v2.4 (TIT2/TPE1/TALB/TRCK/TDRC/TCON/USLT/APIC) and FLAC gets a VORBIS_COMMENT\n"+
				"and a PICTURE block with STREAMINFO left in place. Any ID3 tag the stream already\n"+
				"carried is stripped first, so a re-download does not stack two tags - migu ships its mp3\n"+
				"with an ID3v2.3 tag holding only TLEN and TSSE, which is why a file saved straight from\n"+
				"that stream shows as untitled in a music library. --tags off skips the step. Everything\n"+
				"else is a no-op: ape, ogg, m4a, wav and wma are not written, and the lyric and cover\n"+
				"requests are best effort, so a host that goes down never turns a saved file into an\n"+
				"error.\n\n"+refHelp+"\n\n"+
				"  --source   source id for a bare reference\n"+
				"  --quality  flac24bit|flac|wav|ape|320k|192k|128k|auto  default auto (best available)\n"+
				"  --out      download directory           default ./music (cwd)\n"+
				"  --force    overwrite a file already in --out\n"+
				"  --tags     embed title/artist/album/cover/lyric  default on\n"+
				"  --proxy    http proxy URL\n"+
				"  --verbose  log every HTTP exchange",
			"  lxmusic download kw:228908 --out ~/Downloads\n"+
				"  lxmusic download --search 龙战骑士 --index 1 --quality flac --out ~/Downloads", cmdDownload),
		newCmd("lyric <song-ref>", "Print a song's lyrics",
			"Prints the synced lyrics as plain text.\n\n"+refHelp+"\n\n"+
				"  --source     source id for a bare reference\n"+
				"  --show-trans also print the translation\n"+
				"  --json       emit JSON\n"+
				"  --proxy      http proxy URL\n"+
				"  --verbose    log every HTTP exchange",
			"  lxmusic lyric kw:228908\n"+
				"  lxmusic lyric --search 晴天 --index 1 --source kg --show-trans", cmdLyric),
		newCmd("pic <song-ref>", "Print the cover image URL",
			"Prints the album artwork URL, or {url, song} with --json.\n\n"+refHelp+"\n\n"+
				"  --source   source id for a bare reference\n"+
				"  --json     emit JSON\n"+
				"  --proxy    http proxy URL\n"+
				"  --verbose  log every HTTP exchange",
			"  lxmusic pic kw:228908", cmdPic),
		newCmd("play <song-ref>", "Resolve a stream and open the platform page",
			"Resolves the rendition the way the desktop player does, prints the stream and the headers it "+
				"needs, then opens the platform's own page in the default browser.\n\n"+refHelp+"\n\n"+
				"  --source   source id for a bare reference\n"+
				"  --quality  flac24bit|flac|wav|ape|320k|192k|128k|auto  default auto (best available)\n"+
				"  --proxy    http proxy URL\n"+
				"  --verbose  log every HTTP exchange",
			"  lxmusic play kw:228908", cmdPlay),
		newCmd("share <song-ref | keyword>", "Print a shareable link for a song",
			"Prints two kinds of link, which live for very different lengths of time:\n\n"+
				"  page    the platform's own page, stable for years — this is what you share\n"+
				"  stream  a direct audio URL with an embedded vkey/token the platform expires in\n"+
				"          minutes to hours; useful for piping into ffplay, useless as a share\n\n"+
				"Pass a <source>:<id> reference to share a song you already know, or a bare keyword\n"+
				"to look it up. With a keyword and --source all the sources are tried in turn and the\n"+
				"first hit wins.\n\n"+refHelp+"\n\n"+
				"  --source   kg|kw|wy|tx|mg|<user source>|all  default all\n"+
				"  --quality  rendition to resolve            default auto\n"+
				"  --index    which search result to use      default 1\n"+
				"  --limit    search page size                default 10\n"+
				"  --page     search page                     default 1\n"+
				"  --json     emit JSON\n"+
				"  --proxy    http proxy URL\n"+
				"  --verbose  log every HTTP exchange",
			"  lxmusic share 晴天 --source kw\n"+
				"  lxmusic share 晴天 --quality flac\n"+
				"  lxmusic share kw:228908        # a known song, no search", cmdShare),
		newCmd("detail <song-ref>", "Refresh and print full song metadata",
			"Re-fetches the song record so album, cover and duration are populated, and prints it.\n\n"+refHelp+"\n\n"+
				"  --source   source id for a bare reference\n"+
				"  --json     emit JSON\n"+
				"  --proxy    http proxy URL\n"+
				"  --verbose  log every HTTP exchange",
			"  lxmusic detail kw:228908", cmdDetail),
		newCmd("source <sub>", "Manage user sources (自定义音源)",
			userSourceHelp+"\n\nSubcommands — their own flags:\n\n"+
				"  list                     show config dir, node, and every source\n"+
				"  add    --name <id> --url <script url or path> [--source kg|kw|wy|tx|mg]\n"+
				"         [--node <path>] [--timeout 25] [--ttl 24]\n"+
				"  rm     <name>\n"+
				"  info   <name>            what a source announced it can serve\n"+
				"  test   <name> <song-ref> [--quality ...]\n"+
				"  path                       print the config and cache directory",
			"  lxmusic source list\n"+
				"  lxmusic source add --name flower --url https://raw.githubusercontent.com/pdone/lx-music-source/main/flower/latest.js\n"+
				"  lxmusic source info flower\n"+
				"  lxmusic source test flower kw:228908\n"+
				"  lxmusic source rm flower\n"+
				"  lxmusic source path", cmdSource),
	)
	return root
}
