# lxmusic

`lx-music-desktop 2.12.6` 的搜索 / 下载 / 播放 / 歌词核心逻辑的 Go 命令行实现。

协议是从 `resources/app.asar` 里的 `dist/renderer.js` 逆向出来的（酷我、酷狗、网易云、QQ 四个平台的接口与加密各写一份），不是调第三方库。

**依赖**只有两个，都用在界面层，协议与加密那部分仍是纯标准库：

| 库 | 用途 |
| --- | --- |
| `github.com/spf13/cobra v1.10.2` | 子命令路由与 `--help` 生成 |
| `github.com/olekukonko/tablewriter v1.1.5` | 搜索结果表格的列宽、对齐、CJK 宽度 |

逆向过程与协议细节见 [REVERSE-ENGINEERING.md](REVERSE-ENGINEERING.md)。

---

## 安装

**要求**

- Go 1.25+
- Node.js —— **只有用自定义音源时才需要**，并且要在 PATH 上

**构建**

```bash
cd lxmusic
go build -o lxmusic.exe .     # Windows
go build -o lxmusic .         # Linux / macOS
```

产物是单文件，没有运行时依赖（Node 例外，见下文）。可以直接扔进 PATH 里。

---

## 快速开始

```bash
lxmusic platforms                        # 看有哪些音源
lxmusic search 晴天                      # 全平台搜
lxmusic search 晴天 --source kw --limit 5
lxmusic download kw:228908 --out ~/Downloads
lxmusic lyric kw:228908
lxmusic share 晴天 --source kw
```

五条命令覆盖了主要用法：搜、下、看歌词、分享。

---

## 歌曲引用（song reference）

CLI 用两种方式指认一首歌，**所有命令通用**：

### 1. 直接给 id

```
<音源>:<id>
```

```
kg:20505418B3A52A7A958BF0AED0EBFBA2E9A818B7
kw:228908
wy:347230
tx:<media_mid>
```

前缀就是 `lxmusic platforms` 里那个 id（`kg` / `kw` / `wy` / `tx`，加上你配的自定义音源名）。id 用 `search --json` 查，拼成 `<source>:<id>` 即可：

```bash
lxmusic search 晴天 --source kw --json --limit 1
#   "source": "kw", "id": "228908"      →  kw:228908
```

表格输出里没有 id 列（表格给人看，id 给机器抄），所以脚本化取 id 请用 `--json`。

### 2. 按关键词搜

```
--search <关键词> --index <第几个>
```

```bash
lxmusic lyric --search 晴天 --source kw --index 1
lxmusic download --search 晴天 --source kw --index 1 --quality flac
```

`--index` 从 1 开始，默认 1。

> **酷狗歌词必须用第二种方式。** 酷狗的歌词接口要歌名做参数，光有 id 拿不到；直接给 `kg:<id>` 会报
> `this source needs a song name for lyrics: re-run with --search <keyword> --index N`。

`--key` 是 `--search` 的等价写法（`--key kw:228908` 等价于位置参数 `kw:228908`），写脚本时更好区分。

---

## 命令

### `platforms`

列出可用音源：四个内置平台加所有已配置的自定义音源。

```
kg   酷狗音乐
kw   酷我音乐
wy   网易云音乐
tx   QQ音乐
hyw
```

### `search`

```bash
lxmusic search <关键词> [--source ...] [--limit N] [--page N]
                   [--quality ...] [--out DIR] [--plain] [--json]
```

默认 `--source all`，逐个平台搜，某个平台失败只给一行 warning 不中断。**每行只显示该曲当前能拿到的最高音质**：

```
搜索 "晴天" — 6 条结果

+------+------------+----------------+--------+---------+-----------+----------+
| 序号 | 平台名称   | 音乐名称       |   时长 | 作者    | 音质      | 文件大小 |
+------+------------+----------------+--------+---------+-----------+----------+
|    1 | 酷我音乐   | 晴天           |  04:29 | 周杰伦  | flac      |  52.83MB |
|    2 | 酷我音乐   | 晴天 (Live)    |  04:09 | 周杰伦  | flac      |  51.87MB |
|    3 | 酷我音乐   | 晴天 (女声治.. |  04:33 | 何老希  | flac      |  47.94MB |
|    4 | 酷我音乐   | 晴天 (女声独.. |  04:28 | 梅菜..  | flac      |  32.68MB |
|    5 | 酷我音乐   | 晴天+稻香 (L.. |  07:42 | 周杰伦  | 128k      |   7.05MB |
|    6 | 酷我音乐   | 晴天           |  02:56 | 蓝心羽  | 128k      |   2.70MB |
+------+------------+----------------+--------+---------+-----------+----------+
```

几个点：

- 「平台名称」是中文名而不是 `kw` 这种 id，方便人看；`--json` 里仍然是 id。
- 「文件大小」是该曲**最高音质**在平台上声明的体积，单位 MB，写在每个单元格里；表头不带单位是因为 `(`、`)` 在全角宽度算法下按 2 格计算，塞进表头会把列撑破。
- 平台没声明体积时显示 `-`，这类行排在最后——显示 `-` 而不是 `0.00MB`，免得被误读成"免费小文件"。
- **先按音质倒序（flac24bit > flac > 320k > 128k），同一音质内再按文件大小从大到小**，只留前 **50 条**。排序在前截断在后，所以活下来的是最高的 50 首，序号连续 1..N。音质是关键、大小是打平用的：酷狗把晴天标成 `flac24bit`，酷我和 QQ 把同一个 52.83MB 的文件标成 `flac`，两档音质可能装的是同一份文件。没有音质也没有体积的行排在最后。
- **默认只显示当前配置下能下下来的行**。表格是下载列表，一行确定会在回车时失败就不是结果而是死路。判断依据是两条：音源自己声明的 `AnonymousAudio()`——酷我算能下（给的是约 9 秒预览片段，但确实是落在磁盘上的文件），酷狗、网易云、QQ 算不能下；**或者**你配了音源、而这个音源覆盖该平台——配了音源之后这些行就都回来了，因为音源正是补这块的。本机实测网易云 6 首免费曲的 `cheetah/player/url` 全部返回空地址，说明缺地址不是付费墙拦的，是这个接口的应答形状。被过滤掉的音源会在 stderr 打一行 `note:` 说明，想看完整列表用 `--json`。
- `--limit` 是每个平台请求的**页大小**，不会放宽 50 条上限。`--json` 输出未经截断、排序和过滤的原始响应，方便脚本用。

### 交互选择器

stdout 是终端时，表格自动变成可交互的：**↑↓（或 j/k）每次只移动一行**，**PgUp/PgDn 整页跳**（50 行的列表在常见终端里已经装不下了，翻页才用得上；Home / End 落在首行和末行），**Enter 先确认再下载**，q / Esc 退出。

按下 Enter 之后不会立刻开始下载，而是先在表格下方提示：

```
下载 晴天 - 周杰伦?
保存到 C:\Users\jmrx\Downloads\晴天.flac    [y] 下载  [n] 取消
```

- **y / Enter** —— 开始下载，表格下方滚动进度条。地址解析阶段先显示「正在解析」，第一个字节到达后才变成「正在下载」：

  ```
  正在解析 kw:228908 …
  正在下载 kw:228908  [████████████░░]  60%  31.70MB / 52.83MB  3.10MB/s
  ```

  配了音源时，要地址这一步常常比传输本身还久，所以把这段时间也叫「下载」是不准的——那时候没有任何字节在传。服务端没报总体积时只显示已下字节数，不显示百分比——那种情况下百分比是假的。完成后表格不关，可以继续选下一首；每首歌开始前进度计数会清零，上一首的已下载字节数、总体积和计时起点都不会带到下一首上。
- **n** —— 取消，回到表格。
- **↑↓** —— 取消并移动选择行，改主意是按键的事，不用先 n 再翻回去。
- **q / Esc / Ctrl+C** —— 随时退出。

保存路径是按当前行的数据**推算**的，不做任何网络请求：它在回车和提示出现之间运行，里面有任何一次 HTTP 往返都会把界面卡住。所以**目录一定是对的，只有文件名是预测**——文件名按扩展名规则拼出来，下载完成后会打印真实落盘的路径。推算不出时显示 `--out` 目录。下载目录用 `--out` 指定，默认是**执行命令时的当前目录下的 `music/`**，不存在会自动创建。

列宽是**按内容动态计算**的，不是写死的：先看标题、作者、平台、时长、音质、体积各自需要多宽，再把富余的宽给「音乐名称」和「作者」，所以够宽的时候一个字都不会被截断。终端太窄装不下时按优先级依次让位——先让「音乐名称」「作者」，再让「平台名称」，最后才动「时长」「音质」「文件大小」「序号」；让到各自下限之后，剩下的空位再还回给「音乐名称」和「作者」，还到它们实际需要的宽度为止。不还就是浪费：一张 80 列的终端会把表画成 68 列，白白丢 12 列，还把歌名截成「晴天 (..」和「告..」。

有两层底线：

- 任何一列都不会窄到自己的表头，否则数据还在、标题没了。
- 「时长」「音质」「文件大小」这几列是**数值**，还有更严格的底线：`04:29` 永远要能显示成 `04:29` 而不是 `04..`，`flac24bit` 不能显示成 `flac..`——那和 `flac` 就分不开了，而列表正是按音质排的。被截断的歌名还能看出是哪首歌，被截断的数字就是错的答案。

截断记号是 `..`（两个半角点）而不是 `…`，不是审美问题：tablewriter 量不出省略号的宽度，单元格刚好填满时它会把 `…` 整个丢掉，单元格留了余量时又把列算窄一格，两种情况表头和数据都对不上。两个点是半角，宽度和内部算法一致，列宽才不会漂。进度条那类直接画到终端、不进表格的文本仍然用 `…`。

交互选择器窄到 60 列以下会回退到最小布局，带边框的表格要装下全部表头和数值底线需要 68 列（60 列内容 + 8 列竖线）。列宽按全部结果计算而不是按当前页，所以翻页时列不会左右抖动。

**stdout 被管道或重定向时自动退回纯文本表格**，不会产生转义乱码，可以直接 `| less` 或存文件。`--plain` 强制纯文本，`--json` 输出 JSON。

> 交互选择器依赖真实终端，在未交互的环境（CI、脚本、`make`）下自动降级为纯文本表格；纯文本模式下没有 Enter 确认这一步。

### `url`

解析播放地址，只给链接不下载。

```bash
lxmusic url kw:228908 [--quality ...] [--json]
```

```
song    : kw:228908
quality : 320k
url     : https://kw-bj.kuwo.cn/.../69/32/588957081.mp3
note    : kuwo serves one fixed short preview anonymously, so this is a clip rather than the track
```

`note` 那行是诚实提示：平台给的其实是短预览而不是整曲时，会明确说出来，不会伪装成整曲。

### `download`

```bash
lxmusic download <ref> [--quality ...] [--out DIR]
```

```bash
lxmusic download kw:228908 --out ~/Downloads
lxmusic download kw:440618 --source hyw --quality flac --out ~/Downloads
# ✓ 已保存 C:\Users\jmrx\Downloads\龙战骑士 (《熊猫人》电视剧插曲).flac (32.43MB)
```

在真终端里传输过程会滚动一条进度条，和选择器里是同一条：

```
正在下载 kw:228908  [████████████░░]  60%  31.70MB / 52.83MB  3.10MB/s
```

stdout 被管道或重定向时不打进度条，一直安静到文件存完，所以 `lxmusic download ... | tee log.txt` 的输出是干净的。进度条写在 stdout 而不是 stderr——预览提示和 note 走的是 stderr，同一个流上 `\r` 会把它们覆盖掉。

文件名取服务端实际给的格式扩展名，不会请求 flac 却存成 `.mp3`。超时 10 分钟，足够下无损。

如果下到的是预览片段，会额外提示：

```
✓ 已保存 C:\...\晴天.mp3 (177.27KB)
note  : this is a preview, not the song (177.27KB of a 04:29 track).
        a user source, or a logged-in session, is what gets the full track.
```

判断依据是码率下限：按歌曲时长算，低于约 12KB/秒就认定为片段（未知时长时按 400KB 兜底），不是猜的。

### `lyric`

```bash
lxmusic lyric <ref> [--show-trans]
```

输出标准 LRC。如果平台给了罗马音会自动追加一段：

```
[ver:v1.0]
[ti:晴天]
[ar:周杰伦]
...
[00:20.099]词：方文山

--- translation ---
...
```

`--show-trans` 决定是否打印翻译段。酷狗返回的是 KRC 卡拉 OK 数据，已解码成 LRC，并提示 `note: kugou returned KRC karaoke data (decoded)`。

### `pic`

打印封面图 URL。

```bash
lxmusic pic kw:228908
```

### `play`

打印流地址并**在浏览器打开官网播放页**。

```bash
lxmusic play kw:228908
```

```
playing: 周杰伦 - 晴天
quality: 320k
stream : https://kw-bj.kuwo.cn/...
```

之所以同时打开网页：匿名拿不到整曲直链时，播放页是当前唯一能听到完整内容的途径。

### `share`

按**歌名**给出可分享的链接，两种都打出来，因为寿命差别很大：

```bash
lxmusic share 晴天 --source kw
```

```
song    : 晴天 - 周杰伦
source  : kw   ref: kw:228908
page    : http://www.kuwo.cn/play_detail/228908
quality : 320k
stream  : https://kw-bj.kuwo.cn/...
note    : kuwo serves one fixed short preview anonymously, so this is a clip rather than the track

The page link is the one to share — it never expires. The stream link
carries a platform vkey/token and stops working within minutes to hours.
```

| 链接 | 寿命 | 用途 |
| --- | --- | --- |
| `page` | **永不过期** | 真正该发出去的那个 |
| `stream` | 几分钟到几小时 | 喂给 ffplay / ffmpeg，别当分享链接 |

### `detail`

补齐元数据并解析封面。

```bash
lxmusic detail kw:440618
```

### `--json`

`search` / `url` / `lyric` / `pic` / `detail` / `share` 都支持 `--json`，输出机器可读的 JSON：

```json
{
  "url": "http://car-er.kuwo.cn/.../trackmedia/F000000B3Ekk1I79hc_1.flac?...&format$flac&...",
  "quality": "flac",
  "fileName": "龙战骑士 (《熊猫人》电视剧插曲).flac",
  "source": "hyw"
}
```

---

## 音质选择

`--quality` 取值：`128k` / `320k` / `flac` / `flac24bit` / `auto`。**默认 `auto`，取当前歌曲可选档里最高的那档。**

```bash
lxmusic download kw:228908                    # auto，配音源时 → 音源声明的最高档
lxmusic download kw:228908 --quality flac     # 要 flac
lxmusic download kw:228908 --quality 128k     # 明确要低档
```

三条规则：

1. **`auto` 只取能拿到的最高档**，不会去要一个注定拿不到的版本。
2. **指定了但拿不到时逐级降级**，并在输出里写明降到了哪档：
   `note: kuwo refused flac anonymously; 320k was served instead`。不会静默换档。
3. **`auto` 的判定依据按情况分三级**：搜索结果带来的音质矩阵 → 音源自己声明的档（脚本是唯一能看到自己声明表的东西，只有它没有矩阵时才去问一次）→ 最后兜底 128k。
   最后这一级几乎不该触发：兜底之前那个"问一次"就是把 `auto` 从 128k 提到 flac 的那一步。裸 ref（`kw:228908`）没有矩阵，如果直接落到 128k 兜底，`auto` 会下出 2.79MB 的 ogg，而表格里同一首写的是 flac 64.58MB。

---

## 自定义音源

这是 `lx-music-desktop` 设置页里那个「自定义音源」，CLI 里叫 user source。**可以配任意多个。**

自定义音源**不做搜索**——先在平台搜出歌，再把 `kw:228908` 这种 ref 交给音源去解析地址。这和应用里的行为一致。

### 添加

```bash
lxmusic source add --name flower --url https://raw.githubusercontent.com/pdone/lx-music-source/main/flower/latest.js
lxmusic source add --name hyw  --url 'C:\Users\me\Downloads\某音源.js'
```

`--url` 接受 **URL 或本地文件路径**（绝对/相对路径都行）。

可选参数：

| 参数 | 默认 | 含义 |
| --- | --- | --- |
| `--source` | 空（不限平台） | 这个音源只服务某个平台时指定 |
| `--node` | PATH 查找 | node 可执行文件路径 |
| `--timeout` | 25 | 单次请求超时秒数 |
| `--ttl` | 24 | 脚本缓存有效期小时数 |

### 查看 / 删除

```bash
lxmusic source list
lxmusic source rm <name>
```

```
config : C:\Users\jmrx\AppData\Roaming\lxmusic-cli
node   : C:\Users\jmrx\scoop\apps\nodejs\current\node.exe

name        source  url
flower      kw      https://raw.githubusercontent.com/.../flower/latest.js
hyw         (any)   C:\Users\jmrx\Downloads\某音源.js
```

### 音源能力探查

```bash
lxmusic source info <name>
```

跑一次脚本，打印它自己声明能干什么。**`qualities` 那一列就是 `--quality` 真正能实现到多少：**

```
name    : hyw
script  : HYWmusic_公益版 v1.2.0
serves  : (any)

platform  actions    qualities
kw        musicUrl    128k,320k,flac,flac24bit
tx        musicUrl    128k,320k,flac,flac24bit
kg        musicUrl    128k,320k,flac,flac24bit
```

### 联通性测试

```bash
lxmusic source test <name> <song-ref> [--quality ...]
```

端到端跑一次 `musicUrl`，确认 node 桥接和上游都通：

```
source  : hyw (https://.../latest.js)
song    : 龙战骑士 - 周杰伦   [kw:440618]
quality : flac
url     : http://car-er.kuwo.cn/...
lyric   : unavailable (source "hyw" does not serve "lyric" (declared actions: musicUrl))
```

> **多平台音源不用配 `--source`。** `source test` 要用 `--source`，因为它得借一个平台去查歌；
> 但日常用直接指定 ref 就行，前缀自带平台信息：
>
> ```bash
> lxmusic url kw:228908 --source flower
> lxmusic download kw:440618 --source hyw --quality flac
> ```

### 工作机制

脚本被下载后缓存到本地，再起一个 node 子进程执行，按 lx-music 的 `{source, action, info}` 协议问答——和桌面应用在隐藏 Electron 窗口里跑它用的是同一套契约。

**一次运行只起一个 node，跑完整次会话。** 桌面应用是把音源跑在一个比所有歌都活得长的 WebView 里，`lx.send(inited)` 只执行一次；CLI 原来每首歌都新起一个 node，脚本的 init 跟着每首歌重跑一遍。现在整个命令生命周期内每个音源共用一个常驻进程，stdin/stdout 走换行分隔的 JSON。进程崩了就丢掉重起，两次不行再退回一次性进程——不会把下载搞挂。

**配好之后就会自动用上，不用每次打 `--source`。** 桌面应用就是这种做法：给某个平台配了音源，音源就接管该平台的音频解析，内置接口只当兜底。CLI 现在一样：

```bash
lxmusic download kw:228908 --quality flac
note  : audio served by user source "hyw", not the built-in "kw"
✓ 已保存 ...\晴天.flac (52.83MB)
```

不配音源时同样是这条命令，但拿到的是酷自带的 9 秒预览片段（约 177KB 的 `.mp3`）——所以这一行 `note` 也是提示：你没意识到自己在用音源。`url` / `play` / `share` 走同一条规则，`download` 失败时会退回内置接口，不会把一个下不动的文件当成失败。

**node 必须在 PATH 上**，或者用 `--node` 指定路径。

> 执行第三方脚本属于**供应链风险**。桌面应用本身也是这么用的，CLI 只是继承了同样的信任前提。用共享 key 的公益源还额外承担账号池被回收的风险。

---

## 参数

参数是**每个命令各自的**，不是全局的，直接看 `--help`：

```bash
lxmusic search --help
```

常见取值：

| 参数 | 出现于 | 默认 |
| --- | --- | --- |
| `--source` | 大多数 | `all` |
| `--quality` | search / url / download / play / share | `auto` = 最高可用 |
| `--limit` | search（20）/ share（10） | 每个平台的页大小，不放宽 50 条上限 |
| `--page` | search / share | 1 |
| `--out` | search / download | 当前目录下的 `music/` |
| `--proxy` | 需要联网的 | 无 |
| `--verbose` | 需要联网的 | 关 |
| `--json` | search / url / lyric / pic / detail / share | 关 |
| `--plain` | search | 关 |

flag 写在位置参数后面也可以，会自动重排：

```bash
lxmusic search 晴天 --limit 5          # ✅
lxmusic search --limit 5 晴天          # ✅ 等价
```

`lxmusic <命令> --help` 会打出该命令的参数表和示例；`lxmusic --help` 有总览；shell 补全用 `lxmusic completion bash|zsh|fish`。

### 退出码

| 码 | 含义 |
| --- | --- |
| 0 | 成功，或请求了 `--help` |
| 1 | 出错——网络失败、音源报错、参数不对、命令不存在，信息都在 stderr |

### 超时

`download` **10 分钟**，`search` / `share` 2 分钟，其余 90 秒。

下载的 10 分钟算的是**整个传输**（连上到最后一个字节写完），不是只有握手和响应头。接口调用（搜索、详情、歌词）另有 25 秒的请求上限，两者互不影响——一个挂住的接口不该拖满 10 分钟，一个 200MB 的无损也不该在第 25 秒被掐断。

这个 10 分钟是**每次下载**给的，不是每次会话：从选择器里下载时，搜索用的 2 分钟不会扣到下载头上，你翻完 50 行表再按回车，下载照样有完整的 10 分钟。

---

## 配置目录

```bash
lxmusic source path
```

```
%APPDATA%\lxmusic-cli\                    # Windows: C:\Users\<you>\AppData\Roaming\lxmusic-cli
~/Library/Application Support/lxmusic-cli # macOS
$XDG_CONFIG_HOME/lxmusic-cli              # Linux
```

```
lxmusic.json    音源配置
cache/          脚本缓存 + node 桥接
```

---

## 已知限制

这些不是 bug，是各平台匿名访问的真实边界。

| 项 | 现状 |
| --- | --- |
| 酷我音频 | 匿名只有约 9 秒预览片段（固定 181521 字节），CLI 会明确标注，**不会伪装成整曲**。整曲 320k 要靠自定义音源 |
| flac | 需要带会员账号池的自定义音源。匿名通道给不了无损 |
| 酷狗音频 | 匿名拿不到，**而且不是付费墙的问题**：`get_res_privilege` 从不返回音频地址，免费曲（`privilege=0 pay_type=0`）和付费曲的回包同样没有 `downloadurl`。要下酷狗音频得配一个能服务 kg 的音源，比如 hyw |
| 网易云 / QQ 音频直链 | 匿名拿不到（需要登录 uin / vkey）——这正是桌面应用把音频地址外包给音源脚本的原因 |
| 搜索排序 | 各平台搜索结果顺序不同，且可能把翻唱/翻版排在原版前面。这不是解码问题 |
| 自定义音源 | 第三方脚本，**供应链风险**；共享 key 的公益源还可能因账号池失效而突然不可用 |

要区分开的是：上表是**内置接口**（不配音源时）的边界，不是这个 CLI 的边界。配了一个音源之后，
酷我整曲、酷狗音频、网易云 / QQ 直链、flac 全部解锁，命令行上完全不用改——`lxmusic download kw:228908`
就是同一条命令，音源自动接管，下载完会打一行 `note: audio served by user source "hyw"` 告诉你
是音源在供，不是内置接口在供。唯一还要留意的就是最后一行：音源是第三方脚本，账号池随时可能失效。

想要 flac / 无损 / 整曲，实际可行的路径只有两条：配一个能用的自定义音源，或者走正版渠道购买。

---

## 项目结构

```
main.go                    各命令的 handler：flag、排序截断、保存路径推算、进度泵
commands.go                cobra 命令树：路由与 --help 文案
internal/lx/               逆向出来的协议层（纯标准库）
  http.go                  带代理 / UA / 日志的 HTTP 客户端与下载进度上报
  model.go                 Song / Quality / Lyric / PlayerURL / 扩展名推断
  flex.go                  FStr/FInt：JSON 字段在字符串和数字间漂移时兼容
  registry.go              音源注册表
  crypto.go                eapi / weapi / tencent sign / kugou token
  kuwo.go  kugou.go  netease.go  tencent.go
  lyric.go                 KRC → LRC（酷狗）
  txcipher.go              腾讯自研 64 位块密码
  usersource.go            User API 协议 / 多音源 / node 桥接
internal/tui/              表格与交互选择器
  table.go                 tablewriter 封装：列宽动态测量与让位、对齐、截断
  tui.go                   绘制、键盘事件循环、Enter 确认、进度条与下载
  scan.go                  方向键 / PgUp / Enter / y / n / Ctrl+C 扫描
  term_*.go                按 build tag 拆的平台适配层
```

**跨平台**：不写死任何单个平台。平台相关代码全部走 build tag——unix 用 termios，windows 用 `ReadConsoleInputW`，没有 syscall 的架构有 `term_nosys.go` 兜底。协议层一个平台分支都没有。

**构建验证**（`CGO_ENABLED=0`，纯交叉编译）：`go tool dist list` 里的 48 个 GOOS/GOARCH 组合通过 43 个；剩下的 5 个（`android/{386,amd64,arm}`、`ios/{amd64,arm64}`）报 `requires external (cgo) linking`——用一个零依赖的程序测同样的 5 个目标，报错完全一致，属于本机没有 C 交叉链接器，与本项目和它的依赖无关。

**交叉编译能过不等于能跑。** darwin 的 ioctl 号曾经拼错，48 个目标一个报错都没有，macOS 上 `search` 却是打表格就退。这类错误只在本机能跑的那个平台（Windows）上看不到，所以下面的行为验证只覆盖 Windows 和纯文本输出，Unix 终端的实际表现请在你的机器上确认。

**测试**

```bash
go test -count=1 ./...
```

覆盖腾讯 64 位块密码（与 node 参考实现逐级交叉验证，密钥调度和子密钥矩阵 `diff: 0`）、KRC 歌词编解码往返、文件扩展名推断、本地路径识别，以及：

- 表格合同：任意终端宽度下**一行对应一行**，不超出终端宽度，而且**表头和每一行数据的竖线落在同一批列上**。这条有专门断言——早期用无框样式渲染时表头行比数据行宽两格，每列的标题都坐在它该标的值右边。tablewriter 遇到超长单元格是折行而不是截断，折行会多出一行并把后面所有列挤偏，所以每个单元格都预先截断好。
- 单元格宽度：每个单元格自己补齐到声明宽度，不依赖 tablewriter 的去对齐。它是按内容算列宽的，声明宽度只当底线，而且量不出省略号——单元格填满时 `…` 被整个丢掉，留了余量时列又窄一格。补齐之后 tablewriter 没有可裁量空间。
- CJK 宽度：中日韩全角字符 2 格、半角片假名 1 格，保证列对齐。
- 布局边界：在 `pickerMin..200` 列、`11..40` 行的组合下，屏幕内容不越界、`layout` 返回的行数恰好等于终端行数；确认提示这种两行状态栏也要占对行数。`11` 是底线——家具 8 行 + 一行数据 + 两行状态，低于这个数任何布局都装不下。
- **动态列宽**：给够宽度时一个字都不截断、也不浪费终端；压到最窄时总宽度仍然装得下、且每列都不窄于自己的表头**也不窄于数值底线**（`04:29` 不显示成 `04..`，`flac24bit` 不显示成 `flac..`）；自由文本先让位，量出来的字段（时长／音质／体积／序号）后让位。
- 文件大小列：`0` 和负数显示 `-`（平台没报体积，不是零字节文件），字节数按 MB 两位小数。
- 排序与截断：音质倒序优先、同档内体积倒序，无音质无体积排最后；先排序后截断，活下来的是最高的 50 首，每档内部单调不增。比较器方向专门有个测试钉着——写反过一次，会把列表变成升序。
- 音质档位取值（`bestOf`）：按 `flac24bit > flac > 320k > 128k` 取最高，声明顺序不影响结果，重复无害；不认识的档名（`ogg` / `ape` / `auto`）一律忽略，不会拿一个没标过的档当最高档；大小写不折叠（`FLAC` 不算 `flac`）。这是 `auto` 三级判定的中间那一级，判断写错就是 2.79MB 的 ogg 顶替 flac。
- 音频路由：配了音源时它接管该平台，没配就用内置的；点名的音源不会被二次转接；钉在某平台的音源优先级高于"不限平台"的那个，且不会去答别的平台；平台名大小写与空格不敏感；平台为空或 `nil` 时不路由也不 panic。
- 导航边界：`moveSelection` 扫了 `1..60` 行 × `1..60` 可见行 × 每个光标位置 × 4 个翻页键，断言结果永远落在 `[0, n)`——越界在布局里是 panic，不是坏帧。
- 进度条：`█`／`░` 数量、固定 27 格总宽、三位百分比；无总体积时只显示字节数不出百分比；迟到的 `Report` 不能让进度倒退。
- 进度计数跨首歌清零：选择器整个会话共用一个计数对象，而 `Report` 只升不降，所以第一首下载完留在 100% 的字节数、总体积和计时起点必须显式重置，否则第二首开头就是满格且不再动；另有一条并发测试钉住 `Reset` 和 `Report`／`Line` 走同一把锁，`go test -race` 跑过。
- 进度条两阶段文案：没有字节进来时说「正在解析」，第一个字节到达时由 `Report` 自己翻成「正在下载」；计数对象为 `nil` 时按零值渲染，不会和空对象走出两条不同的文案。
- 音源会话常驻（node 桥接）：用带 1500ms 模拟 init 延迟的桩音源断言 init 只付一次——第一次 1.6s，第二次必须远小于三分之一；另有一条让第一个进程在 init 前直接退出的用例，断言它被丢弃、换新进程后能答上来，而且那个活下来的会话不会在下次请求时又被扔掉。测试把 `APPDATA` 指向临时目录，不碰真实配置。
- 下载超时：接口调用保留 25 秒请求上限，下载路径把它归零改由 context 决定——`http.Client.Timeout` 覆盖整个请求包括读 body，不归零的话大文件会被它在 25 秒掐掉，调用方设的 10 分钟根本轮不到。
- 进度泵：`stop` 返回后不再写任何字节（goroutine 真的退出了）、输出以换行结尾（`✓ 已保存` 独占一行）、重复调用 `stop` 不阻塞不 panic。`go test -race` 跑过。
- 下载字节上报：累计、单调、最终等于整个 body；`Content-Length` 缺省时原样透传 `-1`。
- Windows `INPUT_RECORD` 偏移量与 **key-up 回声过滤**：方向键、PgUp/PgDn、Home/End、Enter（CR 和 LF）、y/n、Ctrl+C/Ctrl+D、字符事件、全零记录。key-up 有专门回归测试——之前每次按方向键跳两行，而且回车被自己的 key-up 应答，导致确认提示永远不出现。过滤是按时间窗做的，不是读 `bKeyDown`：那条判断读偏移 8，而在任何合理的布局里偏移 8 装的都是 `wRepeatCount` 或 `wVirtualKeyCode`，对 key-down 和 key-up 都非零，所以那条判断从来没生效过，也正是每次按压被算两遍的直接证据。测试用固定时钟把 100ms 窗口的两边都钉住了。
- Unix 终端 ioctl 编码：BSD 把 ioctl 编成 `方向 | 长度<<16 | 组<<8 | 号`，测试断言长度字段等于 `unsafe.Sizeof(syscall.Termios{})`、且 `tcgets` 和 `winsize` 的号不相等。这条是从真 bug 里来的——号的写死值 `0x68` 是 `TIOCGWINSZ` 的号不是 termios 的，darwin 上 `tcgets` 算出一个内核没注册的请求，返回 `ENOTTY`，`isTerminal()` 当「不是终端」，`search` 打完表格就直接退、选择器一次都没进去。Linux 走另一个文件、值恰好写对，所以交叉编译全绿也查不出来，只有真在 macOS 上跑才暴露。
- 按键扫描：tmux 前缀序列，以及跨两次 read 被截断的转义序列。
- 酷我体积解析：`size:52.83Mb` 这种**两个字母**单位的容错。

> 交互键盘路径和进度条渲染**无法在本项目所在的自动化环境里验证**——这里没有 PTY（`LoadLibrary(api-ms-win-console-pty-l1-1-0.dll)` 返回 false，`AllocConsole` 也失败，`GetConsoleMode` 不可用，`tui.Terminal()` 恒为 false），所以只有单元测试、合成的控制台输入记录和纯文本输出被真正跑过。方向键 / Enter / 进度条的实际显示请在你的真终端里确认。

---

## 免责声明

本工具的用途是协议研究与客户端实现练习。所涉及的接口、加密算法与音源脚本均属于第三方，**使用者需自行承担使用责任**。请勿用于商业分发，勿批量下载，尊重版权。
