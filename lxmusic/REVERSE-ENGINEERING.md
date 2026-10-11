# lx-music-desktop 2.12.6 逆向分析与 Go CLI 实现

分析对象：`lx-music-desktop.exe` v2.12.6（Electron 应用）
产出：`lxmusic` —— 用 Go 重写搜索 / 下载 / 播放 / 歌词核心逻辑的命令行工具

> 这一篇讲**怎么逆向出来的、协议是什么**。用法看 [README.md](README.md)。

---

## 1. 分析方法

`lx-music-desktop.exe` 是 Electron 壳，真正的业务逻辑全在 `resources/app.asar` 里的
`dist/renderer.js`（约 1.7 MB 压缩后的单文件）。asar 解包后逐模块定位：

- 四个内置音源各自是一个 Provider 模块（`kg` / `kw` / `wy` / `tx`），构造函数里集中了
  接口地址、请求头、加密常量；
- 反混淆后所有函数名都是单字母（`e` `t` `n` `o` `i`），所以移植时保留了原名，
  Go 里加 `krc` / `tx` 前缀避免冲突；
- 关键结论：**v2.12.6 把「音频 URL / 高清音质 / 封面补全」这类需要登录态的能力全部
  外包给 User API 插件**（`action: "musicUrl"`）。这就是包里有完整的 `sign` 签名实现、
  却**完全没有** `musicurl.fcgi` 调用点的原因——URL 由用户在设置页配的音源脚本去解。

---

## 2. 各平台协议与加密

### 2.1 酷我（kw）

| 用途 | 接口 |
| --- | --- |
| 搜索 | `http://search.kuwo.cn/r.s` (`pn=0&rn=20&fmt=json`) |
| 歌曲详情 | `http://www.kuwo.cn/api/www/music/musicInfo?rid=..` |
| 歌词 | `http://mlyric.kuwo.cn/mobi.s?f=web&type=lyric&lrcx=1&rid=..&encode=utf8` |
| 流媒体 | `http://antiserver.kuwo.cn/anti.s?...` → 302 到 `*.kwcdn.kuwo.cn` |
| 封面 | `http://artistpicserver.kuwo.cn/pic.web` |
| 官网播放页 | `http://www.kuwo.cn/play_detail/<rid>` |

**歌词传输**（renderer.js 的 decompress 工具）：响应体前两行是文本头，从第一个
`0D 0A 0D 0A` 之后是 zlib 流；解压出来再 base64 解码，最后**用 `"yeelion"` 做
7 字节循环 XOR**。得到的是标准 LRC + `[offset:]` + 翻译行。这是四个音源里唯一
能完全免登录拿到真歌词的。

**音频限制**：匿名访问酷我反 CDN 网关只能拿到一段约 181 KB / 9 秒的预览对象，
四个音质档位返回的是同一个 clip。CLI 里 `MusicURL` 已按原样实现并明确标注
`note: ... a clip rather than the track`，没有伪装成整曲。

**搜索结果里就带体积**：`r.s` 每条结果的 `N_MINFO` 是按逗号的自由文本，
`level:ff,bitrate:2000,format:flac,size:52.83Mb`——各档音质的比特率和体积一次给全，
不用额外请求详情接口就能知道 flac 是多大。注意体积的单位是**两个字母**（`Mb` / `Kb`，
不是 `M`），按单个尾字母剥掉再 `ParseFloat` 会整列失败并静默返回 0，这个坑写表格列
的时候才踩到。比特率到档位的映射：`4000`→flac24bit、`2000`→flac、`320`→320k、
`128`→128k。跨平台交叉验证过：周杰伦《晴天》在 kw / tx 上标 `flac`、kg 上标
`flac24bit`，三个平台报的体积都是 52.83 MB（04:29），《晴天 (Live)》三个平台都是
51.87 MB（04:09）——体积对得上而**档位标签对不上**，说明各家的 `level`/`format` 命名
不统一，但体积是同一个文件的真实大小。加一列体积正好能把这种自报档位的虚标露出来。

### 2.2 酷狗（kg）

| 用途 | 接口 |
| --- | --- |
| 搜索 | `http://songsearch.kugou.com/song_search_v2` |
| 音质矩阵 | `http://gateway.kugou.com/v2/album_audio/audio`（按 hash 查 rendition，只回元数据） |
| 音质 / 权限 / 封面 | `http://media.store.kugou.com/v1/get_res_privilege`（**不含音频地址**） |
| 歌词发现 | `http://lyrics.kugou.com/search` |
| 歌词下载 | `http://lyrics.kugou.com/download` |

**酷狗不解析音频地址，这不是 CLI 的缺陷。** `get_res_privilege` 的 `info` 只有
duration / filesize / bitrate / extname 和封面图，从来没有任何音频地址字段。实测两首歌证实：
周杰伦《晴天》（`privilege=10 pay_type=3 price=200`，付费）和蓝心羽《晴天》
（`privilege=0 pay_type=0 price=0`，免费），两份回包都**完全没有** `downloadurl` 这个键——
免费那首说明缺地址是这个接口的应答形状，不是被付费墙或地域拦住。renderer.js 里的分工很清楚：
这个接口只被用来取 `info.image` 当封面（`M.getPic`），kuGou 源对象上压根没有 `downloadUrl`
方法，`gateway.kugou.com/v2/album_audio/audio` 也只回 `album_info,author_name,audio_info,
ori_audio_name,base,songname` 这些元数据。音频和 wy / tx 一样全部交给 User API 插件去解，
所以 `lxmusic download kg:...` 不加音源必然失败，`MusicURL` 因此改成把权限字段如实报出来，
不再猜"VIP 锁 / 地域封锁"。

搜索用安卓客户端 UA，带 `KG-THash: expand_search_manager.cpp:852736169:451`
之类的固定头。签名是 `md5(key + sort(params) + extra + key)`，安卓 key
`OIlwieks28dk2k092lksi2UIkp`，Web key `NVPh5oo715z5DIWAeQlhMDsWXXQV4hwt`，
`params` 按字符串字典序排序后拼接。

**KRC 歌词格式**（renderer.js 模块 92617）—— 这是本次逆向里最花时间的部分：

传输层：`base64( "krc1" + XOR( zlib(明文), 16 字节固定 key ) )`，key 为
`[64,71,97,119,94,50,116,71,81,54,49,45,206,210,110,105]`。

**解密后的明文不是 LRC**：

```
[ti:晴天]
[ar:周杰伦]
[al:叶惠美]
[by:]
[offset:0]
[0,2250]<0,160,0>晴<160,160,0>天<320,160,0> <480,160,0>-<640,160,0> ...
[2250,2250]<0,450,0>词<450,450,0>：<900,450,0>周<1350,450,0>杰<1800,450,0>伦
```

- `[start,dur]` 是**行首毫秒绝对时间 + 行时长**，两者都是毫秒；
- 每个字前面挂一个**只有开标签**的 `<rel,dur,0>` 标记（相对本行起始），
  **没有闭合标签**；
- 可选的 `[language:base64json]` 行里，`lyricContent` 是**按行索引的数组**，
  `type:0` 是罗马音、`type:1` 是翻译，同类型以后出现的条目覆盖前面的。

转 LRC 时按 bundle 的写法取 `match[2]`（内层组）当毫秒值，把括号里的内容整体替换掉，
再依次 `<a,b,c>`→`<a,b>`→空、HTML 反转义。

> 移植时踩到的三个坑（都已修）：
> 1. Go 里写成了 `\[((\d+),\d+)\]` 却读 `m[2]:m[3]`——那是**外层组** `"29264,3446"`，
>    `Atoi` 直接失败返回 0，所有时间戳塌成 `[00:00.00]`；正确是内层组。
> 2. `krcTag` 本身就带方括号，外面又包了一层，输出 `[[00:00.00]]`。
> 3. 我以为有闭合标签，写了 `krcWordClose`——真实数据里没有，标记只开不闭。
>
> 另外 bundle 的 `id` 头正则 `\[id:\$\w+\]` 匹配不了酷狗真实的
> `[id:$kugou:lyric:hash:abc]`（含冒号，`\w` 不吃冒号），所以这里放宽成
> `\[id:\$[^\]]+\]`。

### 2.3 网易云（wy）

| 用途 | 接口 |
| --- | --- |
| eapi 批量 | `http://interface.music.163.com/eapi/batch` |
| 歌词 | `https://interface3.music.163.com/eapi/song/lyric/v1` |
| 歌曲详情 | `https://music.163.com/weapi/v3/song/detail` |
| 播放地址 | `https://music.163.com/api/cheetah/player/url` |

两套签名并存：

**eapi**（`interface.music.163.com/eapi/batch`）
```
o    = "nobody" + path + "use" + json + "md5forencrypt"
plain= path + "-36cd479b6b5-" + json + "-36cd479b6b5-" + md5(o)
params = hex( AES-128-ECB+PKCS7(plain, "e82ckenh8dichen8") ).toUpperCase()
```

**weapi**（`music.163.com/weapi/...`）
```
n       = 16 字节随机 → 映射到 [a-zA-Z0-9] 的 ASCII
params  = base64( AES-CBC( base64( AES-CBC(json, "0CoJUm6Qyw8W8jud", IV) ), n, IV) )
encSecKey = hex( RSA_NO_PADDING( leftPad(reverse(n), 128) ) )
```
`IV = "0102030405060708"`。Node 的 `RSA_NO_PADDING` 在 Go 标准库里不存在（只有
PKCS1v15），所以 `rsaEncryptRaw` 用手写的 `big.Int.Exp(m, e, n)` 做模幂。

### 2.4 QQ 音乐（tx）

**搜索签名**（`u.y.qq.com/cgi-bin/musics.fcg?sign=`）：
```
t = sha1(payload)          // 40 个 hex 字符
sign = "zzc" + t[_e.map] + base64(XOR表20字节)[去掉 /+=] + t[me.map]  ，全部小写
```
其中 `_e = [23,14,6,36,16,40,7,19]`，故意含一个越界的 `40`（`join` 时丢成一个空串），
`he` 是 20 字节的异或表。

**歌词解密 —— 腾讯自研的 64 位块密码**（本次移植的核心工作量）。
`music.musichallSong.PlayLyricInfo` 返回的 `lyric`/`trans`/`roma` 三个字段是
**hex 编码的密文，既不是 base64 也不是 XOR**。整条链路：

```
hex 解码 → 每 8 字节一块做块密码 → zlib 解压 → QrcInfos/KRC 结构 → 降成 LRC
```

密码结构（renderer.js `ParseLyricService`，移植后在 `txcipher.go`）：

| 部件 | 说明 |
| --- | --- |
| `H` | 8 张 64 条目的 4-bit S 盒（`txH [8][64]uint8`） |
| `K` | 16 轮的循环左移量 `[1,1,2,2,2,2,2,2,1,2,2,2,2,2,2,1]` |
| `G` / `W` | 各 28 个位索引，密钥调度的高 / 低半字取位顺序 |
| `Y` | 48 个索引，把两轮旋转结果铺成 16 行 × 6 字节子密钥 |
| 密钥 | 24 字节，分 3 组，按 **`[16:24] → [8:16] → [0:8]`** 顺序使用 |

一轮是 16 轮 Feistel，三个子密钥各过一遍。`X` 的 64 位寻址是
`byte = 4*(t/32) + 3 - (t%32)/8`，位 `7 - t%8`（按最高字节在前的字节序）。

两个容易做错的点：

1. **bundle 里的掩码是十进制写的，间距不均**。`4026531840/4294967280/251658240/
   15728640/983040` 对应 `0xF0000000 / 0xFFFFFFF0 / 0xF000000 / 0xF00000 / 0xF0000`，
   不是看起来最自然的等间距写法。手算是错的，必须用 `node -e` 转。判据：JS 的
   `lo` 半字低位有非零 nibble，`0xFFFFFFE0` 这种掩码不可能产生。
2. **子密钥行镜像由参数决定**：bundle 是 `const s = 0===t ? 15-e : e`，
   参数 **0** 才镜像（`15-r`），参数 1 保持原序。三组密钥分别用 `ne(k[16:24],0)`、
   `ne(k[8:16],1)`、`ne(k[0:8],0)`。

验证方式不是「看最后输出像不像」，而是**逐级对齐**：先用 node 跑通
`X` 的全部 64 个位位置（`TestTxxDump`）→ 密钥调度种子与 16 轮旋转值
（`TestTxSeedDump`，`seed diff: 0`）→ 3×16 个子密钥矩阵（`TestTencentSubkeys`，
`keys diff: 0`）→ 整条块密码后的字节流（`TX_LYRIC_DBG`）→ 最终文本
（`TestTencentDecryptLyric`，9821 字符全等）。

另外验证过 `crypt: 0` 是死路——服务端永远返回密文（响应里 `crypt` 恒为 `1`），
所以本地解密无法绕开。

解密结果不是 LRC，而是裹在 **QrcInfos XML** 里的卡拉 OK 数据，行形如
`[start,duration]字(start,len)字(start,len)…`，还有嵌套词组
`((1440,160)Jay(1600,160))`。`txKRCtoLRC` 把它降成 `[mm:ss.xx]` 纯文本，
嵌套括号折回一对。

> 顺带：`songID` 必须作为 **JSON 数字**发，发字符串会被回 `code 10006`。

### 2.5 User API（自定义音源）

应用在设置页填的音源地址（例如你配的 `pdone/lx-music-source` 的
`flower/latest.js`）会被放进一个隐藏 WebView 执行，宿主通过
`{source, action, info}` 协议和它通信，`action` 取值 `musicUrl` / `playUrl` /
`pic` 等。这套协议是 v2.12.6 把登录态能力外包出去的口子。

CLI 用 `internal/lx/usersource.go` 复刻：下载脚本 → 缓存 → 起 node 子进程 →
按同一套协议问答。支持**任意多个音源**，配置与缓存在 `lxmusic source path`
打印的目录里。音源本身不做搜索：先在平台搜出歌，再把 `kw:228908` 这类 ref 交给音源。
配了之后音源会自动接管对应平台的音频解析（见 3.），不是靠 `--source` 点名。

> **WebView 和 node 子进程的差别** —— 接入 huibq 音源时发现的两处，是 node 复刻里
> 特有的坑，应用本身不会遇到：
> 1. **console 输出会污染协议响应**。桥接的 JSON 回包写在 `stdout`，而不少脚本用
>    `console.group` / `console.log` 打调试日志，同样落在 `stdout`。应用里这些输出
>    进 DevTools，所以从来没人发现；CLI 里 Go 把 `stdout` 当 JSON 解析，拿到的是
>    `Handle Action(musicUrl)\n source kw\n quality 320k ...`。修法：把脚本的
>    `console.*` 全部改道到 `stderr`，`stdout` 只留给协议回包。
> 2. **`??` 不吃空字符串**。脚本用 `musicInfo.hash ?? musicInfo.songmid` 取 id，
>    但酷我的 `hash` 是 `""`（不是 `undefined`），`??` 不会继续往下取，于是请求
>    变成 `/url/kw//320k` 直接 404。修法：`hash` 为空时**不写这个键**，让它落到
>    `songmid`（酷我的数字 id 就在这里）。用 `||` 的脚本两种写法行为一致，所以这个
>    改动对它们是纯收益。
>
> 另外 `source test` 需要音源带 `--source` 才能测，因为探测平台时要一个 ref 前缀。
> 多平台音源建议不加 `--source`，用 `lxmusic url <ref> --source <name>` 直接指定即可——
> ref 自带平台前缀，`--source` 只是缺省值。
>
> 3. **`url.Parse` 把盘符当成 scheme**。本地音源路径 `C:\Users\..\x.js` 会被
>    `url.Parse` 解析成 `scheme = "c"`（Go 1.25 实测，且不报错），于是走 HTTP
>    分支得到 `unsupported protocol scheme "c"`。修法：先问 `filepath.IsAbs`。
> 4. **文件后缀曾经写死 `.mp3`**。音源可能回一个和请求档位不同的格式，所以
>    `AudioExt` 以 **URL 路径里的扩展名**为准（酷写的 flac 在 `trackmedia/F000000B...flac`），
>    档位只做兜底。只看路径不看 query——酷写的 query 用 `$` 分隔，里面还埋了个
>    伪装的 `kwplayer...apk` User-Agent。

---

## 3. CLI 实现

```
lxmusic platforms                     列出可用音源
lxmusic search   <kw> [--source kg|kw|wy|tx|all] [--limit N] [--page N]
                 [--quality ...] [--out DIR] [--plain] [--json]
lxmusic url      <ref>  [--quality 128k|320k|flac|flac24bit|auto] [--json]
lxmusic download <ref>  [--quality ...] [--out DIR]
lxmusic lyric    <ref>  [--show-trans] [--index N] [--json]
lxmusic pic      <ref>
lxmusic play     <ref>  [--quality ...]
lxmusic share    <kw>   [--source ...] [--quality ...]
lxmusic detail   <ref>  [--json]

lxmusic source list|add|rm|info|test|path    自定义音源
```

**界面**：子命令路由用 `spf13/cobra`，搜索结果的表格用 `olekukonko/tablewriter`
（`序号 / 平台名称 / 音乐名称 / 时长 / 作者 / 音质 / 文件大小`，每行只显示最高音质，
体积取该档的声明值、单位 MB 写在单元格里）。表格先按音质倒序、同档内按体积倒序，最多
保留 50 行——先排序后截断，所以活下来的是最高的 50 首；`--limit` 是每个平台的页大小，
不会放宽这个上限，`--json` 走原始响应，不经截断、排序和过滤这三样处理。列宽按内容动态
测量，富余宽度给「音乐名称」和「作者」，太窄时按 自由文本 → 平台 → 量出来的字段 的顺序
让位，任何一列都不窄于自己的表头，数值列另有底线：`04:29` 不能变成 `04..`，`flac24bit`
不能变成 `flac..`（那样和 `flac` 分不开，而列表正是按音质排的）。截断记号用两个半角点
`..`，因为 tablewriter 量不出 `…` 的宽度。stdout 是终端时表格变成交互选择器（带 `+---+`
/ `|` 边框，选中的那一行整体反显；↑↓ 每次只动一行、PgUp/PgDn 整页跳、Enter 先确认再下
并显示保存路径、下载时滚动进度条、q/Esc 退出），被管道或重定向时自动退回同款的纯文本
ASCII 表，可以直接 `| less` 或存文件。
`lxmusic download` 在真终端里也显示同一条进度条，管道时保持安静。协议与加密层不碰
这两个库。

**表格只列当前配置下能下下来的行。** 两条依据：一是每个 Provider 自己声明的
`AnonymousAudio()`——它是协议层面的事实，写在适配器旁边，加新平台时不会漏标。酷我算能下
（给的是约 9 秒预览片段，但确实是落在磁盘上的文件）；酷狗、网易云、QQ 算不能下——音频直链
正是 v2.12.6 外包给 User API 的那部分能力。实测网易云 6 首**免费**曲的 `cheetah/player/url`
全部返回空地址，说明缺地址是这个接口的应答形状，不是付费墙拦的。二是 `UserSourceFor`：配了
音源、而这个音源覆盖该平台，那这个平台就算能下——音源存在的意义就是补这块，配了还把行过滤掉
就自相矛盾了。两条都不满足才隐藏，并在 stderr 打一行 `note:` 说明，想看完整列表用 `--json`。

**配了音源就自动接管音频。** 这是照桌面应用抄的行为：给某个平台配了音源，音源就接管该平台的
`musicUrl`，内置接口只当兜底。CLI 原来不是这样——`--source` 是选一个具体 provider，内置的 kw
从不回头问音源，于是选择器里按回车永远拿到 9 秒预览片段。规则在 `AudioProvider` 里：点名了音源
就不再转接（自己就是音源）；钉在某平台的音源优先于「不限平台」的那个；平台为空时不路由。
音源答不出来就退回内置，不会把一个下不动的歌当成失败。

**`auto` 的判定分三级，最后一级几乎不该触发。** 音质矩阵（搜索带来的）→ 音源自己声明的档
（`bestDeclared`，只有没有矩阵时才去问一次，代价是一次脚本 init，结果记在 `Cap` 里）→ 兜底
128k。裸 ref `kw:228908` 没有矩阵，原来直接落到 128k 兜底，`auto` 下出来是 2.79MB 的 ogg，
而搜索表格里同一首写的是 flac 64.58MB。补上中间那一级的判定后，同一条命令现在给 flac。
`bestOf` 按 `flac24bit > flac > 320k > 128k` 选，不认识的档名（`ogg` / `ape` / `auto`）一律忽略，
不会拿一个没标过的档当最高档。

**下载超时和接口超时是分开的两件事。** 接口客户端带着 25 秒的请求上限（挂住的接口不该拖满十分钟），
但 `http.Client.Timeout` 覆盖的是**整个请求，包括读 body**——所以 200MB 的无损在慢链路上会被它
在第 25 秒掐掉，调用方的 10 分钟 context 根本轮不到。下载走一份把 `Timeout` 归零的副本，
deadline 交还给 context；接口调用仍用原来的客户端。连接池和跳转由指针共享，行为不变。

超时按**每次下载**给，不是按会话给。选择器原来直接拿搜索的 context，搜索自己占 2 分钟、
再花掉几十秒，下载到开始就只剩不到 90 秒的预算，而这张表有 50 行——用户翻完它再按回车，
真正能下多久早就花光了。现在会话用不带 deadline 的 context 跑（退出时取消），
每次下载各自套一个 10 分钟。

**一个音源只起一个 node，跑完整次会话。** 每首歌下载前都要问音源要地址，原来每次一个新进程，
脚本的 init 跟着每首歌重跑一遍。桌面应用不是这样：音源跑在一个比所有歌都活得长的 WebView 里，
`lx.send(inited)` 只跑一次。改成常驻进程后桥接的 stdin/stdout 变成**换行分隔的 JSON**（双向）——
`JSON.stringify` 不会在字符串里冒出裸换行，所以回包永远是一行，管道读取器就够了，
不用长度前缀。脚本加载走 `loadOnce` 记忆化，且**失败不缓存**（缓存住一次拒绝就会把上游的
瞬时故障变成整个会话不答话）；进程自己退用的是 `process.exitCode` 不是 `process.exit`，
后者会截断 stdout，把这一次调用刚写完的回包一起吃掉。脚本内部真的 `process.exit` 则由 Go 兜底：
管道断、`do` 失败、会话丢弃、起一个新的，两次不行再退回一次性进程。

收益和那次测量的大头要分开。收益是省下每首歌的一次进程创建加一次完整 init，本机实测约 400ms
（同一进程里连续两次解析 551ms → 130ms）。但「下载前要等很久」量到的是另一回事：当时 init
单独跑就是 25.2s、整次解析 50.4s，现在 init 只要 177ms、整次解析 450ms——脚本自己的 init
不再耗时，是它上游的密钥服务当时慢。那部分不是代码改掉的，是上游自己好了；但常驻进程把
「每首歌重付一次 init」这件事从设计上拿掉了，上游一慢就不会再被放大成每首歌一次。

进度条的文案跟着改了：字节还没到齐时那条线写的是「正在解析」而不是「正在下载」，那段时间确实
在解析地址、没有任何传输。`Report` 在第一次 `done` 增长时自己把动词翻成「正在下载」，
不需要额外的回调。


顺带修了选择器下载的文件名：原来它把行的 `Ref` 重新解一遍再查一次元数据，查不到时 `Song.Name`
是空的，落盘的文件就只剩下扩展名——`.mp3`。现在行直接带着它当初搜索结果里的 `Song`，不再重查，
顺带省了一次网络往返。

顺带把「不可搜索」和「请求失败」分开了：`UserSource.Search` 返回的是 `ErrNoSearch` 这个
sentinel，扫全部音源时静默跳过（用户音源本来就不做搜索，每次调用都警告三行纯属噪声），
但用户显式指定 `--source flower` 时仍然照实解释。

**网关 `data` 有两种形状。** 酷狗网关对**认识的** hash 回 `[[{...}]]`，对**不认识的**
回 `[{}]`——一层数组里塞一个空对象。后者就是服务端说"这里没有"的方式。原来按单一形状
解码，截断或写错的 id 会把 `json: cannot unmarshal object into Go struct field` 直接漏到
输出里。现在两种形状依次尝试，再丢掉没有音频 hash 也没有歌名的行；只有两种形状都读不出
的应答才算错误。

**Windows 键盘输入的两个坑**（和上面的选择器直接相关）：一是 `INPUT_RECORD` 各字段的
偏移量在文档里查不到——联合体补了多少 padding 取决于各 SDK 的布局，是靠
`WriteConsoleInputW` 写入合成按键记录再读回来、逐字节比对候选布局定下来的，测试把偏移量钉死
防漂移。二是 Windows 在每次 key-DOWN 之后会再发一条携带**同一个虚拟键码**的 key-UP 记录，
只按事件类型过滤就等于把每次按压算两遍：方向键每次跳两行；更要命的是回车自己应答了自己，确认
提示根本来不及显示就自动确认了——也就是"按了回车既没提示也没下载"。

这个 key-UP 最终是靠**时间窗**滤掉的，不是靠 `bKeyDown`。写代码时本来以为读 `bKeyDown` 就行，
但那条判断从来没生效过：它读偏移 8，而在所有合理的 `KEY_EVENT_RECORD` 布局里，偏移 8 装的是
`wRepeatCount` 或 `wVirtualKeyCode`，这两个值对 key-down 和 key-up **都是非零**——判断永远为真，
等于没判，这正是每次按压被算两遍的直接证据，也说明偏移 8 不可能是 `bKeyDown`。整条记录里根本
没有哪一位标示"键按下了"，所以改比时间：同一个虚拟键码加同一个字符在 100ms 内再来一次，就是
上一次按压的回声，直接丢掉。这个窗口只要求大于 key-DOWN 到 key-UP 的间隔（几百微秒），不需要
小到能放过长按自动重复。50 行的列表确实需要滚，但不是靠长按：连续两次同一个虚拟键码的间隔大约
几十毫秒，正好落进 100ms 窗口被当成回声丢掉，长按没有效果——整页跳靠 PgUp/PgDn，这是把上限
从 20 提到 50 之后新加的。有回归测试拿固定时钟把窗口两边都钉住了。

**tablewriter 的四个坑**（全宽中文 + 固定列宽，一个都躲不开）：一是开启
`WithEastAsian` 之后 ASCII 标点也按东亚宽度算，`(` 和 `)` 各占 2 格，所以 `文件大小(MB)`
这个表头在 12 格的列里装不下，被截成 `文件大小( MB`——单位只好挪进每个单元格；二是
StyleASCII 的边框是两头各一根竖线再加每对列之间一根，竖线数是 `列数+1`，把边框宽度写死
成 7 在 6 列时正好、加一列就撑出 80 列；三是 tablewriter 对溢出的单元格是**换行**而不是
省略，一行内容变两行会让整张表的行号映射全乱，所以每个单元格都先按列宽截断再喂进去；
四是**它量不出省略号**。`WithColumnWidths` 是底线不是定宽，tablewriter 按它自己算出的
内容宽度定列宽，而 U+2026 在它那里时而被丢掉时而被算成 0 格：单元格内容刚好填满声明宽度
时 `04:..` 会被整个吃掉（渲染出来只剩 `04:`），单元格留了余量时那一列又比声明窄一格。
两种情况下表头和数据行都错位，而且表头和数据行是各自量的、互不相同步。所以每个单元格
都自己补齐到声明宽度（`padCell`，按列对齐方向补），让 tablewriter 没有可裁量的空间；截断
记号也换成两个半角点 `..`，它和内部宽度算法口径一致。`…` 只用在进度条那类直接画到终端、
不经过表格的文本里。

**cobra 的一个坑**：子命令的 flag 定义在 Go 标准库的 `flag.FlagSet` 上（不想把整套
handler 迁到 pflag），所以命令开 `DisableFlagParsing: true`，让原始参数原样到达 handler。
不能用 `FParseErrWhitelist.UnknownFlags` 代替——pflag 遇到不认识的 flag 会走
`stripUnknownFlagValue`，把 **flag 和它的取值一起丢掉**，`--source kw` 会连搜索关键词一起被吞掉。
同理 `RunE` 里不能 `return flag.ErrHelp`：禁用了解析后 cobra 根本看不到 `--help`，
那个哨兵只会以 `error: flag: help requested` 冒回来，要用 `cmd.Help()`。

**Unix 终端：ioctl 号别自己拼，用 syscall 的。** BSD 把一个 ioctl 编成
`方向 | 长度<<16 | 组<<8 | 号`。长度用 `unsafe.Sizeof(syscall.Termios{})` 是对的，
但**号**当初写死成了 `0x68`——那是 `TIOCGWINSZ` 的号，不是 termios 的（termios 是 `0x13`）。
于是 darwin 上的 `tcgets` 算出 `0x40487468`：`TIOCGWINSZ` 带 72 字节长度，
内核里没有注册这个请求，返回 `ENOTTY`。`isTerminal()` 把失败当「不是终端」，
`search` 于是打表格、直接退出，交互选择器一次都没进去。

这个坑只在编译期看不见——`go build` 48 个目标能过 43 个，全绿，问题一个都暴露不出来；
而且 Linux 走的是另一个文件（`0x5401` / `0x5402` / `0x5413` 直接写死，恰好是对的），
所以本地和 CI 都发现不了，只有真在 macOS 上跑才暴露。现在改成
`syscall.TIOCGETA` / `TIOCSETA` / `TIOCGWINSZ`：Go 从平台头文件生成这些常量，
号和长度都和内核核对过，而且长度字段天然和 `syscall.Termios` 同步
（darwin 是 72 字节 = `0x48`，其余 BSD 是 44 字节 = `0x2c`）。
Linux 那边 stdlib 不导出 `TCGETS`，仍然写死，值不变。

有回归测试钉住：长度字段必须等于 `unsafe.Sizeof(syscall.Termios{})`，
`tcgets` 和 `winsize` 的号不许相同。顺带把 `isTerminal()` 补全——文档一直写「两个流都
是终端」，但代码只查了 stdout，`</dev/null` 之类会把交互模式打开再因拿不到终端而退回，
现在两个流都查。

**一个还没查清的地方**：`Cc` 槽位。本机实测（`golang.org/x/sys/unix` 从平台头生成的常量）
Linux 是 `VMIN=0x6 / VTIME=0x5`，darwin 报的是 `VMIN=0x10 / VTIME=0x11`。代码按 Linux 的
6/5 设 VMIN=0、VTIME=1（100ms 读超时，用来分辨单独一个 ESC 和某个序列的第一个字节）。
如果 darwin 的真值真是 0x10/0x11，那这两个写到了别的槽上，读会一直阻塞到真有按键，
**单独按 Esc 不会生效，得再按一个键才动**。也可能是那组常量取的是 4.2BSD 遗留的
`term.h` 而不是 `termios.h`（`VEOF=0x0` 也指向这个可能），那样 6/5 就是对的。
本机跑不了 BSD，判不出来——修完上面的 ioctl 之后在 Mac 上按一下 Esc 就知道答案。

**音质选择**：搜索结果里每首歌带 `Qualities` 列表（各档 type + size）。
`--quality auto`（默认）取 `Song.BestQuality()` = 可选档里最高的那档；
显式指定的档位若音源不给，会**逐级降级**并在输出里写明
`note: kuwo refused flac anonymously; 320k was served instead`，不会静默换档。
自定义音源场景下取的是脚本自己声明的最高档，所以 `auto` 绝不会要一个它给不了的版本。

**分享链接**（`lxmusic share 晴天`）分两层：

| 类型 | 特点 |
| --- | --- |
| `page`（官网播放页） | **永不过期**，这是真正该分享出去的那个 |
| `stream`（直链） | 带平台 vkey/token，几分钟到几小时就失效 |

输出里明确提示这一点，避免把会过期的直链当成分享链接发出去。

---

## 4. 已知限制

| 项 | 现状 |
| --- | --- |
| 酷我音频 | 匿名直连只有 ~9 秒预览 clip（`content-length` 181521），已标注未伪装。走 huibq 音源能拿整曲 320k（10.8MB）；走 **HYWmusic** 音源能拿 **flac** |
| flac | 需要带会员账号池的音源（HYWmusic 类）。URL 形如 `trackmedia/F000000B...flac?format$flac&...&convert_url_with_sign&user$<uid>`——`F000000B` 前缀是酷我的 flac trackmedia（320k 是 `M800000B`）。签名每次轮换，必须即取即下 |
| kg 音频 | 匿名拿不到，而且**不是付费墙的问题**。`get_res_privilege` 从不返回音频地址，免费曲和付费曲一样都没有 `downloadurl`——实测 `privilege=0 pay_type=0` 的免费曲回包同样缺这个键。和 wy / tx 一样要交给音源脚本解 |
| wy / tx 音频直链 | 匿名拿不到（需要登录 uin/vkey），这正是 v2.12.6 用 User API 外包出去的能力 |
| HYWmusic 音源 | `HYWmusic_公益版 v1.2.0`，MIT，[Macrohard0001/HYWmusic_source](https://github.com/Macrohard0001/HYWmusic_source)。声明 kw/tx/wy/mg/qs/bili/ytb/xm/kg，kw/tx/wy/kg 声称 flac/flac24bit。**实测 kw flac 可用**（32.4MB，44100Hz/stereo/s16，STREAMINFO MD5 校验通过）。只声明 `actions:['musicUrl']`，歌词/封面走平台源 |
| huibq 音源 | `Huibq_lxmusic源 v1.2.0`。`kw` 可用（整曲、128k/320k，无 flac）；`kg`/`tx`/`wy` 后端只有一条写死的 `panspace.kuwo.cn` 兜底地址，换歌换音质返回同一个 158KB 文件，等于坏掉 |
| flower 音源后端 | 该脚本声明的接口当前返回 404，协议链路本身通，是远端服务问题 |
| 本机 tx 环境 | QQ 域名被 DNS 黑洞，歌词/搜索需换网络 |
| 网易云搜索排序 | `晴天` 排第一的是同人翻唱版，官方录音在后面——排序问题，不是解码 bug |
| 自定义音源 | 执行第三方脚本属于**供应链风险**；应用本身也这么用，CLI 只是继承了同样的信任前提。用共享 key 的公益源还额外承担了账号池被回收的风险 |

---

## 5. 文件

```
main.go                    各命令 handler，flag 定义与业务逻辑
commands.go                cobra 命令树：路由与 --help 文案
internal/lx/               协议层，纯标准库
  http.go                  带代理 / UA / 日志的 HTTP 客户端
  model.go                 Song / Quality / Lyric / PlayerURL
  flex.go                  FStr/FInt：JSON 字段在字符串和数字间漂移
  registry.go              音源注册表
  crypto.go                eapi / weapi / tencent sign / kugou token
  kugou.go  kuwo.go  netease.go  tencent.go
  lyric.go                 KRC → LRC（酷狗）+ KRC 结构解析
  txcipher.go              腾讯 64 位块密码
  usersource.go            User API 协议 / 多音源
internal/tui/              表格与交互选择器
  table.go                 tablewriter 封装：列宽、对齐、单元格截断
  tui.go                   屏幕绘制、键盘事件循环、Enter 确认与下载
  scan.go                  方向键 / PgUp / Enter / y / n / Ctrl+C 扫描
  term_*.go                build tag 拆出的平台适配层
```

**跨平台**：平台相关代码全走 build tag——unix 用 termios（`term_unix.go` 加
`linux` / `bsd` / `sv4` / `bsd42` 各一份 ioctl），windows 用 `ReadConsoleInputW`，
两者都没有的架构落到 `term_nosys.go`。协议层没有任何平台分支。所有 unix 平台共用
`scan.go` 里的同一个字节扫描器，加新按键只需要改 `scan.go` 和 `term_windows.go`。

构建矩阵（`CGO_ENABLED=0`，`go tool dist list` 的 48 个 GOOS/GOARCH）通过 43 个；
失败的是 `android/{386,amd64,arm}` 和 `ios/{amd64,arm64}`，报
`requires external (cgo) linking, but cgo is not enabled`。用零依赖程序测同样这 5 个
目标报错完全一致，属本机缺 C 交叉链接器，不是本项目或它依赖的问题。

测试：`go test -count=1 ./...`
- `txcipher_test.go` —— 用 node 参考实现做交叉验证，`seed diff: 0` / `keys diff: 0` /
  全文本逐字节相等；
- `lyric_test.go` —— KRC 编解码往返 + 双括号 / 字标记泄漏 / 时间戳解析回归用例；
- `tui_test.go` —— 表格合同：任意宽度下**一行对一行**且不超终端宽度；CJK 宽度；
  布局边界。

> 交互键盘路径在这个自动化环境里**没跑过**：没有 PTY
> （`LoadLibrary(api-ms-win-console-pty-l1-1-0.dll)` 失败，`AllocConsole` 也失败，
> `GetConsoleMode` 不可用），所以只有单元测试和纯文本输出被真正执行过。
> 方向键 / Enter 的行为请在真终端里确认。
