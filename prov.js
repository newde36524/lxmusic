// slice [1298000,1425000) of C:\lx-rl\extracted\dist\renderer.js
ric: () = > {
  (0,
  s.fu)(!s.qS["desktopLyric.isLock"])
}
}
};
(Object.getOwnPropertyDescriptor(a,
"name")|| {
  
}).writable||Object.defineProperty(a,
"name",
 {
  value: "default",
  configurable: !0
})
},
74536(e,
t,
n) {
  "use strict";
  n.d(t,
   {
    O: () = >r,
    t: () = >l
  });
  var o = n(17247),
  i = n(47957);
  const s =  {
    
  },
  a =  {
    
  },
  l =  {
    
  };
  for(const e of o.A) {
    l[e.id] = e.supportQualitys;
    for(const t of Object.keys(e.supportQualitys))a[`${e.id}_api_${t}`] = s[`${e.id}_${t}`]
  }const r = e = > {
    if(/^user_api/.test(i.s_.value))return i.i$.apis[e];
    let t = (e = >a[`${i.s_.value}_api_${e}`])(e);
    if(t)return t;
    throw new Error("Api is not found")
  }
},
9444(e,
t,
n) {
  "use strict";
  n.d(t,
   {
    A: () = >At
  });
  var o = n(91199);
  const i =  {
    regExps:  {
      relWord: /RELWORD=(.+)/
    },
    requestObj: null,
    async tipSearchBySong(e) {
      return this.cancelTipSearch(),
      this.requestObj = (0,
      o.tX)(`https://tips.kuwo.cn/t.s?corp=kuwo&newver=3&p2p=1&notrace=0&c=mbox&w=${encodeURIComponent(e)}&encoding=utf8&rformat=json`,
       {
        Referer: "http://www.kuwo.cn/"
      }),
      this.requestObj.promise.then(( {
        body: e,
        statusCode: t
      }) = >200 =  = t&&e.WORDITEMS?e.WORDITEMS: Promise.reject(new Error("请求失败")))
    },
    handleResult: e = >e.map(e = >e.RELWORD),
    cancelTipSearch() {
      this.requestObj&&this.requestObj.cancelHttp&&this.requestObj.cancelHttp()
    },
    async search(e) {
      return this.tipSearchBySong(e).then(e = >this.handleResult(e))
    }
  };
  var s = n(31878),
  a = n(76982),
  l = n.n(a);
  require("dns");
  const r = e = >l().createHash("md5").update(e).digest("hex"),
  c = (new Map,
  (e,
  t = "name",
  n = "、") = > {
    if(Array.isArray(e)) {
      const o = [];
      return e.forEach(e = > {
        let n = e[t];
        n&&o.push(n)
      }),
      (0,
      s.e4)(o.join(n))
    }return(0,
    s.e4)(String(e??""))
  });
  var d = n(43106);
  const u = Buffer.from("yeelion"),
  p = u.length,
  _ = async(e,
  t) = > {
    const n = Buffer.isBuffer(e)?e: Buffer.from(e);
    if("tp=content"! =  = n.toString("utf8",
    0,
    10).toLowerCase())return"";
    const o = await(async e = >new Promise((t,
    n) = > {
      (0,
      d.inflate)(e,
      (e,
      o) = > {
        e?n(e): t(o)
      })
    }))(n.subarray(n.indexOf("\r\n\r\n")+4));
    if(!t)return o.toString("utf8");
    const i = Buffer.from(o.toString(),
    "base64"),
    s = i.length,
    a = new Uint8Array(s);
    let l = 0;
    for(;
    l<s;
    ) {
      let e = 0;
      for(;
      e<p&&l<s;
      )a[l] = i[l]^u[e],
      l++,
      e++
    }return Buffer.from(a).toString("utf8")
  },
  m = e = >JSON.parse(e.replace(/('(?=(,\s*')))|('(?=:))|((?<=([:,]\s*))')|((?<={)')|('(?=}))/g,
  '"')),
  h = e = >e.replace(/&/g,
  "、"),
  g =  {
    rxps:  {
      wordLine: /^(\[\d{1,2}:.*\d{1,4}])\s*(\S+(?:\s+\S+)*)?\s*/,
      tagLine: /\[(ver|ti|ar|al|offset|by|kuwo):\s*(\S+(?:\s+\S+)*)\s*]/,
      wordTimeAll: /<(-?\d+),(-?\d+)(?:,-?\d+)?>/g,
      wordTime: /<(-?\d+),(-?\d+)(?:,-?\d+)?>/
    },
    offset: 1,
    offset2: 1,
    isOK: !1,
    lines: [],
    tags: [],
    getWordInfo(e,
    t,
    n) {
      const o = parseInt(e),
      i = parseInt(t);
      let s = Math.trunc(Math.abs((o+i)/(2*this.offset))),
      a = Math.trunc(Math.abs((o-i)/(2*this.offset2))+s);
      return n&&s<n.endTime&&(n.endTime = s,
      n.startTime>n.endTime&&(n.startTime = n.endTime),
      n.newTimeStr = `<${n.startTime},${Math.trunc(n.endTime-n.startTime)}>`),
       {
        startTime: s,
        endTime: a,
        timeStr: `<${s},${Math.trunc(a-s)}>`
      }
    },
    parseLine(e) {
      if(e.length<6)return;
      let t = this.rxps.wordLine.exec(e);
      if(t) {
        const e = t[1];
        let n = t[2];
        null =  = n&&(n = "");
        const o = n.match(this.rxps.wordTimeAll);
        if(!o)return;
        let i;
        for(const e of o) {
          const t = this.rxps.wordTime.exec(e),
          o = this.getWordInfo(t[1],
          t[2],
          i);
          n = n.replace(e,
          o.timeStr),
          i?.newTimeStr&&(n = n.replace(i.timeStr,
          i.newTimeStr)),
          i = o
        }return void this.lines.push(e+n)
      }if(t = this.rxps.tagLine.exec(e),
      t)if("kuwo" =  = t[1]) {
        let e = t[2];
        null! = e&&e.includes("][")&&(e = e.substring(0,
        e.indexOf("][")));
        const n = parseInt(e,
        8);
        this.offset = Math.trunc(n/10),
        this.offset2 = Math.trunc(n%10),
        (0 =  = this.offset||Number.isNaN(this.offset)||0 =  = this.offset2||Number.isNaN(this.offset2))&&(this.isOK = !1)
      }else this.tags.push(e)
    },
    parse(e) {
      const t = e.split(/\r\n|\r|\n/),
      n = Object.create(this);
      n.isOK = !0,
      n.offset = 1,
      n.offset2 = 1,
      n.lines = [],
      n.tags = [];
      for(const e of t) {
        if(!n.isOK)throw new Error("failed");
        n.parseLine(e)
      }if(!n.lines.length)return"";
      let o = n.lines.join("\n");
      return n.tags.length&&(o = `${n.tags.join("\n")}\n${o}`),
      o
    }
  },
  f =  {
    aesMode: "aes-128-ecb",
    aesKey: Buffer.from([112,
    87,
    39,
    61,
    199,
    250,
    41,
    191,
    57,
    68,
    45,
    114,
    221,
    94,
    140,
    228],
    "binary"),
    aesIv: "",
    appId: "y67sprxhhpws",
    decodeData(e) {
      const t = Buffer.from(decodeURIComponent(e),
      "base64");
      return JSON.parse(((e,
      t,
      n,
      o) = > {
        const i = (0,
        a.createDecipheriv)(t,
        n,
        o);
        return Buffer.concat([i.update(e),
        i.final()])
      })(t,
      this.aesMode,
      this.aesKey,
      this.aesIv).toString())
    },
    createSign(e,
    t) {
      const n = `${this.appId}${e}${t}`;
      return r(n).toUpperCase()
    },
    buildParam(e) {
      const t = Buffer.from(JSON.stringify(e)),
      n = Date.now(),
      o = ((e,
      t,
      n,
      o) = > {
        const i = (0,
        a.createCipheriv)(t,
        n,
        o);
        return Buffer.concat([i.update(e),
        i.final()])
      })(t,
      this.aesMode,
      this.aesKey,
      this.aesIv).toString("base64"),
      i = this.createSign(o,
      n);
      return`data=${encodeURIComponent(o)}&time=${n}&appId=${this.appId}&sign=${i}`
    }
  },
  y =  {
    regExps:  {
      mInfo: /level:(\w+),bitrate:(\d+),format:(\w+),size:([\w.]+)/
    },
    limit: 30,
    total: 0,
    page: 0,
    allPage: 1,
    musicSearch: (e,
    t,
    n) = >(0,
    o.tX)(`http://search.kuwo.cn/r.s?client=kt&all=${encodeURIComponent(e)}&pn=${t-1}&rn=${n}&uid=794762570&ver=kwplayer_ar_9.2.2.1&vipver=1&show_copyright_off=1&newver=1&ft=music&cluster=0&strategy=2012&encoding=utf8&rformat=json&vermerge=1&mobi=1&issubtitle=1`).promise,
    handleResult(e) {
      const t = [];
      if(!e)return t;
      for(let n = 0;
      n<e.length;
      n++) {
        const o = e[n];
        let i = o.MUSICRID.replace("MUSIC_",
        "");
        if(!o.N_MINFO)return console.log("N_MINFO is undefined"),
        null;
        const a = [],
        l =  {
          
        };
        let r = o.N_MINFO.split(";");
        for(let e of r)if(e = e.match(this.regExps.mInfo),
        e)switch(e[2]) {
          case"4000": a.push( {
            type: "flac24bit",
            size: e[4]
          }),
          l.flac24bit =  {
            size: e[4].toLocaleUpperCase()
          };
          break;
          case"2000": a.push( {
            type: "flac",
            size: e[4]
          }),
          l.flac =  {
            size: e[4].toLocaleUpperCase()
          };
          break;
          case"320": a.push( {
            type: "320k",
            size: e[4]
          }),
          l["320k"] =  {
            size: e[4].toLocaleUpperCase()
          };
          break;
          case"128": a.push( {
            type: "128k",
            size: e[4]
          }),
          l["128k"] =  {
            size: e[4].toLocaleUpperCase()
          }
        }a.reverse();
        let c = parseInt(o.DURATION);
        t.push( {
          name: (0,
          s.e4)(o.SONGNAME),
          singer: h((0,
          s.e4)(o.ARTIST)),
          source: "kw",
          songmid: i,
          albumId: (0,
          s.e4)(o.ALBUMID||""),
          interval: Number.isNaN(c)?0: (0,
          s.Zg)(c),
          albumName: o.ALBUM?(0,
          s.e4)(o.ALBUM): "",
          lrc: null,
          img: null,
          otherSource: null,
          types: a,
          _types: l,
          typeUrl:  {
            
          }
        })
      }return t
    },
    search(e,
    t = 1,
    n,
    o = 0) {
      return o>2?Promise.reject(new Error("try max num")): (null =  = n&&(n = this.limit),
      this.musicSearch(e,
      t,
      n).then(( {
        body: i
      }) = > {
        if(!i||"0"! =  = i.TOTAL&&"0" =  =  = i.SHOW)return this.search(e,
        t,
        n,
        ++o);
        let s = this.handleResult(i.abslist);
        return null =  = s?this.search(e,
        t,
        n,
        ++o): (this.total = parseInt(i.TOTAL),
        this.page = t,
        this.allPage = Math.ceil(this.total/n),
        Promise.resolve( {
          list: s,
          allPage: this.allPage,
          total: this.total,
          limit: n,
          source: "kw"
        }))
      }))
    }
  },
  v = [ {
    id: "kw__93",
    name: "飙升榜",
    bangid: "93"
  },
   {
    id: "kw__17",
    name: "新歌榜",
    bangid: "17"
  },
   {
    id: "kw__16",
    name: "热歌榜",
    bangid: "16"
  },
   {
    id: "kw__158",
    name: "抖音热歌榜",
    bangid: "158"
  },
   {
    id: "kw__292",
    name: "铃声榜",
    bangid: "292"
  },
   {
    id: "kw__284",
    name: "热评榜",
    bangid: "284"
  },
   {
    id: "kw__290",
    name: "ACG新歌榜",
    bangid: "290"
  },
   {
    id: "kw__286",
    name: "台湾KKBOX榜",
    bangid: "286"
  },
   {
    id: "kw__279",
    name: "冬日暖心榜",
    bangid: "279"
  },
   {
    id: "kw__281",
    name: "巴士随身听榜",
    bangid: "281"
  },
   {
    id: "kw__255",
    name: "KTV点唱榜",
    bangid: "255"
  },
   {
    id: "kw__280",
    name: "家务进行曲榜",
    bangid: "280"
  },
   {
    id: "kw__282",
    name: "熬夜修仙榜",
    bangid: "282"
  },
   {
    id: "kw__283",
    name: "枕边轻音乐榜",
    bangid: "283"
  },
   {
    id: "kw__278",
    name: "古风音乐榜",
    bangid: "278"
  },
   {
    id: "kw__264",
    name: "Vlog音乐榜",
    bangid: "264"
  },
   {
    id: "kw__242",
    name: "电音榜",
    bangid: "242"
  },
   {
    id: "kw__187",
    name: "流行趋势榜",
    bangid: "187"
  },
   {
    id: "kw__204",
    name: "现场音乐榜",
    bangid: "204"
  },
   {
    id: "kw__186",
    name: "ACG神曲榜",
    bangid: "186"
  },
   {
    id: "kw__185",
    name: "最强翻唱榜",
    bangid: "185"
  },
   {
    id: "kw__26",
    name: "经典怀旧榜",
    bangid: "26"
  },
   {
    id: "kw__104",
    name: "华语榜",
    bangid: "104"
  },
   {
    id: "kw__182",
    name: "粤语榜",
    bangid: "182"
  },
   {
    id: "kw__22",
    name: "欧美榜",
    bangid: "22"
  },
   {
    id: "kw__184",
    name: "韩语榜",
    bangid: "184"
  },
   {
    id: "kw__183",
    name: "日语榜",
    bangid: "183"
  },
   {
    id: "kw__145",
    name: "会员畅听榜",
    bangid: "145"
  },
   {
    id: "kw__153",
    name: "网红新歌榜",
    bangid: "153"
  },
   {
    id: "kw__64",
    name: "影视金曲榜",
    bangid: "64"
  },
   {
    id: "kw__176",
    name: "DJ嗨歌榜",
    bangid: "176"
  },
   {
    id: "kw__106",
    name: "真声音",
    bangid: "106"
  },
   {
    id: "kw__12",
    name: "Billboard榜",
    bangid: "12"
  },
   {
    id: "kw__49",
    name: "iTunes音乐榜",
    bangid: "49"
  },
   {
    id: "kw__180",
    name: "beatport电音榜",
    bangid: "180"
  },
   {
    id: "kw__13",
    name: "英国UK榜",
    bangid: "13"
  },
   {
    id: "kw__164",
    name: "百大DJ榜",
    bangid: "164"
  },
   {
    id: "kw__246",
    name: "YouTube音乐排行榜",
    bangid: "246"
  },
   {
    id: "kw__265",
    name: "韩国Genie榜",
    bangid: "265"
  },
   {
    id: "kw__14",
    name: "韩国M-net榜",
    bangid: "14"
  },
   {
    id: "kw__8",
    name: "香港电台榜",
    bangid: "8"
  },
   {
    id: "kw__15",
    name: "日本公信榜",
    bangid: "15"
  },
   {
    id: "kw__151",
    name: "腾讯音乐人原创榜",
    bangid: "151"
  }],
  w =  {
    list: [ {
      id: "kwbiaosb",
      name: "飙升榜",
      bangid: 93
    },
     {
      id: "kwregb",
      name: "热歌榜",
      bangid: 16
    },
     {
      id: "kwhuiyb",
      name: "会员榜",
      bangid: 145
    },
     {
      id: "kwdouyb",
      name: "抖音榜",
      bangid: 158
    },
     {
      id: "kwqsb",
      name: "趋势榜",
      bangid: 187
    },
     {
      id: "kwhuaijb",
      name: "怀旧榜",
      bangid: 26
    },
     {
      id: "kwhuayb",
      name: "华语榜",
      bangid: 104
    },
     {
      id: "kwyueyb",
      name: "粤语榜",
      bangid: 182
    },
     {
      id: "kwoumb",
      name: "欧美榜",
      bangid: 22
    },
     {
      id: "kwhanyb",
      name: "韩语榜",
      bangid: 184
    },
     {
      id: "kwriyb",
      name: "日语榜",
      bangid: 183
    }],
    regExps:  {
      mInfo: /level:(\w+),bitrate:(\d+),format:(\w+),size:([\w.]+)/
    },
    limit: 100,
    _requestBoardsObj: null,
    getBoardsData() {
      return this._requestBoardsObj&&this._requestBoardsObj.cancelHttp(),
      this._requestBoardsObj = (0,
      o.tX)("http://qukudata.kuwo.cn/q.k?op=query&cont=tree&node=2&pn=0&rn=1000&fmt=json&level=2"),
      this._requestBoardsObj.promise
    },
    getData: e = >(0,
    o.tX)(e).promise,
    filterData(e) {
      return e.map(e = > {
        let t = [];
        const n =  {
          
        },
        o = new Set;
        return e.n_minfo.split(";").forEach(e = > {
          const i = e.match(this.regExps.mInfo);
          if(!i)return;
          const s = i[2],
          a = i[4].toLocaleUpperCase();
          if(!o.has(s))switch(o.add(s),
          s) {
            case"4000": t.push( {
              type: "flac24bit",
              size: a
            }),
            n.flac24bit =  {
              size: a
            };
            break;
            case"2000": t.push( {
              type: "flac",
              size: a
            }),
            n.flac =  {
              size: a
            };
            break;
            case"320": t.push( {
              type: "320k",
              size: a
            }),
            n["320k"] =  {
              size: a
            };
            break;
            case"128": t.push( {
              type: "128k",
              size: a
            }),
            n["128k"] =  {
              size: a
            }
          }
        }),
        t = (e = > {
          const t =  {
            flac24bit: 4,
            flac: 3,
            "320k": 2,
            "128k": 1
          },
          n = [],
          o = [];
          return e.forEach((e,
          o) = > {
            const i = t[e.type];
            i&&n.push( {
              type: i,
              index: o
            })
          }),
          n.sort((e,
          t) = >e.type-t.type),
          n.forEach(t = > {
            o.push(e[t.index])
          }),
          o
        })(t),
         {
          singer: h((0,
          s.e4)(e.artist)),
          name: (0,
          s.e4)(e.name),
          albumName: (0,
          s.e4)(e.album),
          albumId: e.albumId,
          songmid: e.id,
          source: "kw",
          interval: (0,
          s.Zg)(parseInt(e.duration)),
          img: e.pic,
          lrc: null,
          otherSource: null,
          types: t,
          _types: n,
          typeUrl:  {
            
          }
        }
      })
    },
    filterBoardsData(e) {
      let t = [];
      for(const n of e)"1" =  = n.source&&t.push( {
        id: "kw__"+n.sourceid,
        name: n.name,
        bangid: String(n.sourceid)
      });
      return t
    },
    async getBoards(e = 0) {
      return this.list = v,
       {
        list: v,
        source: "kw"
      }
    },
    getList(e,
    t,
    n = 0) {
      if(++n>3)return Promise.reject(new Error("try max num"));
      const i =  {
        uid: "",
        devId: "",
        sFrom: "kuwo_sdk",
        user_type: "AP",
        carSource: "kwplayercar_ar_6.0.1.0_apk_keluze.apk",
        id: e,
        pn: t-1,
        rn: this.limit
      },
      s = `https://wbd.kuwo.cn/api/bd/bang/bang_info?${f.buildParam(i)}`;
      return(0,
      o.tX)(s).promise.then(( {
        statusCode: o,
        body: i
      }) = > {
        const s = f.decodeData(i),
        a = s.data;
        if(200! =  = o||200! = s.code||!a.musiclist)return this.getList(e,
        t,
        n);
        return {
          total: parseInt(a.total),
          list: this.filterData(a.musiclist),
          limit: this.limit,
          page: t,
          source: "kw"
        }
      })
    }
  },
  b = /^\[([\d:.]*)\]{1}/g,
  k = /\[\d{1,2}:.*\d{1,4}]/,
  A = /^<-?\d+,-?\d+>/,
  x =  {
    sortLrcArr(e) {
      const t = new Set;
      let n = [],
      o = [],
      i = !1;
      for(const s of e) {
        if(t.has(s.time)) {
          if(n.length<2)continue;
          const e = n.pop();
          e.time = n[n.length-1].time,
          o.push(e),
          n.push(s)
        }else n.push(s),
        t.add(s.time);
        !i&&A.test(s.text)&&(i = !0)
      }if(!i&&o.length>.3*n.length&&n.length-o.length>6)throw new Error("failed");
      return {
        lrc: n,
        lrcT: o
      }
    },
    transformLrc: (e,
    t) = >`${e.join("\n")}\n${t?t.map(e=>`[$ {
      e.time
    }]$ {
      e.text
    }\n`).join(""):"暂无歌词"}`,
    parseLrc(e) {
      const t = e.split(/\r\n|\r|\n/);
      let n = [],
      o = [];
      for(let e = 0;
      e<t.length;
      e++) {
        const i = t[e].trim();
        if(b.exec(i)) {
          const e = i.replace(b,
          "").trim();
          let t = RegExp.$1;
          /\.\d\d$/.test(t)&&(t+ = "0"),
          o.push( {
            time: t,
            text: e
          })
        }else g.rxps.tagLine.test(i)&&n.push(i)
      }const i = this.sortLrcArr(o);
      return {
        lyric: (0,
        s.e4)(this.transformLrc(n,
        i.lrc)),
        tlyric: i.lrcT.length?(0,
        s.e4)(this.transformLrc(n,
        i.lrcT)): ""
      }
    },
    getLyric(e,
    t = !0) {
      const n = (0,
      o.tX)(`http://mlyric.kuwo.cn/mobi.s?f=web&type=lyric&lrcx=${t?1:0}&rid=${e.songmid}&encode=utf8`);
      return n.promise = n.promise.then(async( {
        statusCode: e,
        body: n,
        raw: o
      }) = > {
        if(200! = e)return Promise.reject(new Error(JSON.stringify(n)));
        const i = await _(o,
        t);
        let s;
        try {
          s = this.parseLrc(i)
        }catch(e) {
          return Promise.reject(new Error("Get lyric failed"))
        }s.tlyric&&(s.tlyric = s.tlyric.replace(g.rxps.wordTimeAll,
        ""));
        try {
          s.lxlyric = g.parse(s.lyric)
        }catch {
          s.lxlyric = ""
        }return s.lyric = s.lyric.replace(g.rxps.wordTimeAll,
        ""),
        k.test(s.lyric)?s: Promise.reject(new Error("Get lyric failed"))
      }),
      n
    }
  },
  S =  {
    getPic( {
      songmid: e
    }) {
      const t = (0,
      o.tX)(`http://artistpicserver.kuwo.cn/pic.web?corp=kuwo&type=rid_pic&pictype=500&size=500&rid=${e}`);
      return t.promise = t.promise.then(( {
        body: e
      }) = >/^http/.test(e)?e: null),
      t.promise
    }
  };
  var C = n(74536);
  const E =  {
    limit_list: 36,
    limit_song: 1e3,
    filterListDetail: (e,
    t,
    n) = >e.map((e,
    o) = > {
      let i = e.formats.split("|"),
      a = [],
      l =  {
        
      };
      return i.includes("MP3128")&&(a.push( {
        type: "128k",
        size: null
      }),
      l["128k"] =  {
        size: null
      }),
      i.includes("MP3H")&&(a.push( {
        type: "320k",
        size: null
      }),
      l["320k"] =  {
        size: null
      }),
      i.includes("ALFLAC")&&(a.push( {
        type: "flac",
        size: null
      }),
      l.flac =  {
        size: null
      }),
      i.includes("HIRFLAC")&&(a.push( {
        type: "flac24bit",
        size: null
      }),
      l.flac24bit =  {
        size: null
      }),
       {
        singer: h((0,
        s.e4)(e.artist)),
        name: (0,
        s.e4)(e.name),
        albumName: t,
        albumId: n,
        songmid: e.id,
        source: "kw",
        interval: null,
        img: e.pic,
        lrc: null,
        otherSource: null,
        types: a,
        _types: l,
        typeUrl:  {
          
        }
      }
    }),
    formatPlayCount: e = >e>1e8?parseInt(e/1e7)/10+"亿": e>1e4?parseInt(e/1e3)/10+"万": e,
    getAlbumListDetail(e,
    t,
    n = 0) {
      if(n>2)return Promise.reject(new Error("try max num"));
      return(0,
      o.tX)(`http://search.kuwo.cn/r.s?pn=${t-1}&rn=${this.limit_song}&stype=albuminfo&albumid=${e}&show_copyright_off=0&encoding=utf&vipver=MUSIC_9.1.0`).promise.then(( {
        statusCode: o,
        body: i
      }) = >200! =  = o?this.getAlbumListDetail(e,
      t,
      ++n): (i = m(i)).musiclist?(i.name = (0,
      s.e4)(i.name),
       {
        list: this.filterListDetail(i.musiclist,
        i.name,
        i.albumid),
        page: t,
        limit: this.limit_song,
        total: parseInt(i.songnum),
        source: "kw",
        info:  {
          name: i.name,
          img: i.img||i.hts_img,
          desc: (0,
          s.e4)(i.info),
          author: (0,
          s.e4)(i.artist)
        }
      }): this.getAlbumListDetail(e,
      t,
      ++n))
    }
  },
  N =  {
    _musicInfoRequestObj: null,
    _musicInfoPromiseCancelFn: null,
    _musicPicRequestObj: null,
    _musicPicPromiseCancelFn: null,
    tipSearch: i,
    musicSearch: y,
    leaderboard: w,
    songList:  {
      _requestObj_tags: null,
      _requestObj_hotTags: null,
      _requestObj_list: null,
      limit_list: 36,
      limit_song: 1e3,
      successCode: 200,
      sortList: [ {
        name: "最新",
        id: "new"
      },
       {
        name: "最热",
        id: "hot"
      }],
      regExps:  {
        mInfo: /level:(\w+),bitrate:(\d+),format:(\w+),size:([\w.]+)/,
        listDetailLink: /^.+\/playlist(?:_detail)?\/(\d+)(?:\?.*|&.*$|#.*$|$)/
      },
      tagsUrl: "http://wapi.kuwo.cn/api/pc/classify/playlist/getTagList?cmd=rcm_keyword_playlist&user=0&prod=kwplayer_pc_9.0.5.0&vipver=9.0.5.0&source=kwplayer_pc_9.0.5.0&loginUid=0&loginSid=0&appUid=76039576",
      hotTagUrl: "http://wapi.kuwo.cn/api/pc/classify/playlist/getRcmTagList?loginUid=0&loginSid=0&appUid=76039576",
      getListUrl( {
        sortId: e,
        id: t,
        type: n,
        page: o
      }) {
        if(!t)return`http://wapi.kuwo.cn/api/pc/classify/playlist/getRcmPlayList?loginUid=0&loginSid=0&appUid=76039576&&pn=${o}&rn=${this.limit_list}&order=${e}`;
        switch(n) {
          case"10000": return`http://wapi.kuwo.cn/api/pc/classify/playlist/getTagPlayList?loginUid=0&loginSid=0&appUid=76039576&pn=${o}&id=${t}&rn=${this.limit_list}`;
          case"43": return`http://mobileinterfaces.kuwo.cn/er.s?type=get_pc_qz_data&f=web&id=${t}&prod=pc`
        }
      },
      getListDetailUrl(e,
      t) {
        return`http://nplserver.kuwo.cn/pl.svc?op=getlistinfo&pid=${e}&pn=${t-1}&rn=${this.limit_song}&encode=utf8&keyset=pl2012&identity=kuwo&pcmp4=1&vipver=MUSIC_9.0.5.0_W1&newver=1`
      },
      getTag(e = 0) {
        return this._requestObj_tags&&this._requestObj_tags.cancelHttp(),
        e>2?Promise.reject(new Error("try max num")): (this._requestObj_tags = (0,
        o.tX)(this.tagsUrl),
        this._requestObj_tags.promise.then(( {
          body: t
        }) = >t.code! =  = this.successCode?this.getTag(++e): this.filterTagInfo(t.data)))
      },
      getHotTag(e = 0) {
        return this._requestObj_hotTags&&this._requestObj_hotTags.cancelHttp(),
        e>2?Promise.reject(new Error("try max num")): (this._requestObj_hotTags = (0,
        o.tX)(this.hotTagUrl),
        this._requestObj_hotTags.promise.then(( {
          body: t
        }) = >t.code! =  = this.successCode?this.getHotTag(++e): this.filterInfoHotTag(t.data[0].data)))
      },
      filterInfoHotTag: e = >e.map(e = >( {
        id: `${e.id}-${e.digest}`,
        name: e.name,
        source: "kw"
      })),
      filterTagInfo: e = >e.map(e = >( {
        name: e.name,
        list: e.data.map(t = >( {
          parent_id: e.id,
          parent_name: e.name,
          id: `${t.id}-${t.digest}`,
          name: t.name,
          source: "kw"
        }))
      })),
      getList(e,
      t,
      n,
      i = 0) {
        if(this._requestObj_list&&this._requestObj_list.cancelHttp(),
        i>2)return Promise.reject(new Error("try max num"));
        let s,
        a;
        if(t) {
          let e = t.split("-");
          s = e[0],
          a = e[1]
        }else s = null;
        return this._requestObj_list = (0,
        o.tX)(this.getListUrl( {
          sortId: e,
          id: s,
          type: a,
          page: n
        })),
        this._requestObj_list.promise.then(( {
          body: o
        }) = >s&&"10000"! = a?o.length? {
          list: this.filterList2(o),
          total: 1e3,
          page: n,
          limit: 1e3,
          source: "kw"
        }: this.getList(e,
        t,
        n,
        ++i): o.code! =  = this.successCode?this.getList(e,
        t,
        n,
        ++i):  {
          list: this.filterList(o.data.data),
          total: o.data.total,
          page: o.data.pn,
          limit: o.data.rn,
          source: "kw"
        })
      },
      formatPlayCount: e = >e>1e8?parseInt(e/1e7)/10+"亿": e>1e4?parseInt(e/1e3)/10+"万": e,
      filterList(e) {
        return e.map(e = >( {
          play_count: this.formatPlayCount(e.listencnt),
          id: `digest-${e.digest}__${e.id}`,
          author: e.uname,
          name: e.name,
          total: e.total,
          img: e.img,
          grade: e.favorcnt/10,
          desc: e.desc,
          source: "kw"
        }))
      },
      filterList2(e) {
        const t = [],
        n = ["songlist",
        "list",
        "album"];
        return e.forEach(e = > {
          e.list.forEach(e = > {
            n.includes(e.type)&&t.push( {
              play_count: e.play_count&&this.formatPlayCount(e.listencnt),
              id: `digest-${e.digest}__${e.id}`,
              author: e.uname,
              name: e.name,
              total: e.total,
              img: e.img,
              grade: e.favorcnt&&e.favorcnt/10,
              desc: e.desc,
              source: "kw"
            })
          })
        }),
        t
      },
      getListDetailDigest8(e,
      t,
      n = 0) {
        if(n>2)return Promise.reject(new Error("try max num"));
        return(0,
        o.tX)(this.getListDetailUrl(e,
        t)).promise.then(( {
          body: o
        }) = >"ok"! =  = o.result?this.getListDetail(e,
        t,
        ++n):  {
          list: this.filterListDetail(o.musiclist),
          page: t,
          limit: o.rn,
          total: o.total,
          source: "kw",
          info:  {
            name: o.title,
            img: o.pic,
            desc: o.info,
            author: o.uname,
            play_count: this.formatPlayCount(o.playnum)
          }
        })
      },
      getListDetailDigest5Info(e,
      t = 0) {
        if(t>2)return Promise.reject(new Error("try max num"));
        return(0,
        o.tX)(`http://qukudata.kuwo.cn/q.k?op=query&cont=ninfo&node=${e}&pn=0&rn=1&fmt=json&src=mbox&level=2`).promise.then(( {
          statusCode: n,
          body: o
        }) = >200 =  = n&&o.child?o.child.length?o.child[0].sourceid: null: this.getListDetail(e,
        ++t))
      },
      getListDetailDigest5Music(e,
      t,
      n = 0) {
        if(n>2)return Promise.reject(new Error("try max num"));
        return(0,
        o.tX)(`http://nplserver.kuwo.cn/pl.svc?op=getlistinfo&pid=${e}&pn=${t-1}}&rn=${this.limit_song}&encode=utf-8&keyset=pl2012&identity=kuwo&pcmp4=1`).promise.then(( {
          body: o
        }) = >"ok"! =  = o.result?this.getListDetail(e,
        t,
        ++n):  {
          list: this.filterListDetail(o.musiclist),
          page: t,
          limit: o.rn,
          total: o.total,
          source: "kw",
          info:  {
            name: o.title,
            img: o.pic,
            desc: o.info,
            author: o.uname,
            play_count: this.formatPlayCount(o.playnum)
          }
        })
      },
      async getListDetailDigest5(e,
      t,
      n) {
        const o = await this.getListDetailDigest5Info(e,
        n);
        return this.getListDetailDigest5Music(o,
        t,
        n)
      },
      filterBDListDetail: e = >e.map(e = > {
        let t = [],
        n =  {
          
        };
        for(let o of e.audios)switch(o.size = o.size?.toLocaleUpperCase(),
        o.bitrate) {
          case"4000": t.push( {
            type: "flac24bit",
            size: o.size
          }),
          n.flac24bit =  {
            size: o.size
          };
          break;
          case"2000": t.push( {
            type: "flac",
            size: o.size
          }),
          n.flac =  {
            size: o.size
          };
          break;
          case"320": t.push( {
            type: "320k",
            size: o.size
          }),
          n["320k"] =  {
            size: o.size
          };
          break;
          case"128": t.push( {
            type: "128k",
            size: o.size
          }),
          n["128k"] =  {
            size: o.size
          }
        }return t.reverse(),
         {
          singer: e.artists.map(e = >e.name).join("、"),
          name: e.name,
          albumName: e.album,
          albumId: e.albumId,
          songmid: e.id,
          source: "kw",
          interval: (0,
          s.Zg)(e.duration),
          img: e.albumPic,
          releaseDate: e.releaseDate,
          lrc: null,
          otherSource: null,
          types: t,
          _types: n,
          typeUrl:  {
            
          }
        }
      }),
      getReqId() {
        function e() {
          return(65536*(1+Math.random())|0).toString(16).substring(1)
        }return e()+e()+e()+e()+e()+e()+e()+e()
      },
      async getListDetailMusicListByBDListInfo(e,
      t) {
        const {
          body: n
        } = await(0,
        o.tX)(`https://bd-api.kuwo.cn/api/service/playlist/info/${e}?reqId=${this.getReqId()}&source=${t}`,
         {
          headers:  {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36",
            plat: "h5"
          }
        }).promise.catch(() = >( {
          code: 0
        }));
        return 200! = n.code?null:  {
          name: n.data.name,
          img: n.data.pic,
          desc: n.data.description,
          author: n.data.creatorName,
          play_count: n.data.playNum
        }
      },
      async getListDetailMusicListByBDUserPub(e) {
        const {
          body: t
        } = await(0,
        o.tX)(`https://bd-api.kuwo.cn/api/ucenter/users/pub/${e}?reqId=${this.getReqId()}`,
         {
          headers:  {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36",
            plat: "h5"
          }
        }).promise.catch(() = >( {
          code: 0
        }));
        return 200! = t.code?null:  {
          name: t.data.userInfo.nickname+"喜欢的音乐",
          img: t.data.userInfo.headImg,
          desc: "",
          author: t.data.userInfo.nickname,
          play_count: ""
        }
      },
      async getListDetailMusicListByBDList(e,
      t,
      n,
      i = 0) {
        const {
          body: s
        } = await(0,
        o.tX)(`https://bd-api.kuwo.cn/api/service/playlist/${e}/musicList?reqId=${this.getReqId()}&source=${t}&pn=${n}&rn=${this.limit_song}`,
         {
          headers:  {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36",
            plat: "h5"
          }
        }).promise.catch(() = >i>2?Promise.reject(new Error("try max num")): this.getListDetailMusicListByBDList(e,
        t,
        n,
        ++i));
        return 200! =  = s.code?Promise.reject(new Error("failed")):  {
          list: this.filterBDListDetail(s.data.list),
          page: n,
          limit: s.data.pageSize,
          total: s.data.total,
          source: "kw"
        }
      },
      async getListDetailMusicListByBD(e,
      t) {
        const n = /uid=(\d+)/.exec(e)?.[1],
        o = /playlistId=(\d+)/.exec(e)?.[1],
        i = /source=(\d+)/.exec(e)?.[1];
        if(!o)return Promise.reject(new Error("failed"));
        const s = [this.getListDetailMusicListByBDList(o,
        i,
        t)];
        switch(i) {
          case"4": s.push(this.getListDetailMusicListByBDListInfo(o,
          i));
          break;
          case"5": s.push(this.getListDetailMusicListByBDUserPub(n??o))
        }const[a,
        l] = await Promise.all(s);
        return a.info = l?? {
          name: "",
          img: "",
          desc: "",
          author: "",
          play_count: ""
        },
        a
      },
      getListDetail(e,
      t,
      n = 0) {
        if(/\/bodian\//.test(e))return this.getListDetailMusicListByBD(e,
        t);
        if(/[?&:/]/.test(e))e = e.replace(this.regExps.listDetailLink,
        "$1");
        else if(/^digest-/.test(e)) {
          let[o,
          i] = e.split("__");
          switch(o = o.replace("digest-",
          ""),
          e = i,
          o) {
            case"8": break;
            case"13": return E.getAlbumListDetail(e,
            t,
            n);
            default: return this.getListDetailDigest5(e,
            t,
            n)
          }
        }return this.getListDetailDigest8(e,
        t,
        n)
      },
      filterListDetail(e) {
        return e.map(e = > {
          let t = e.N_MINFO.split(";"),
          n = [],
          o =  {
            
          };
          for(let e of t)if(e = e.match(this.regExps.mInfo),
          e)switch(e[2]) {
            case"4000": n.push( {
              type: "flac24bit",
              size: e[4]
            }),
            o.flac24bit =  {
              size: e[4].toLocaleUpperCase()
            };
            break;
            case"2000": n.push( {
              type: "flac",
              size: e[4]
            }),
            o.flac =  {
              size: e[4].toLocaleUpperCase()
            };
            break;
            case"320": n.push( {
              type: "320k",
              size: e[4]
            }),
            o["320k"] =  {
              size: e[4].toLocaleUpperCase()
            };
            break;
            case"128": n.push( {
              type: "128k",
              size: e[4]
            }),
            o["128k"] =  {
              size: e[4].toLocaleUpperCase()
            }
          }return n.reverse(),
           {
            singer: h((0,
            s.e4)(e.artist)),
            name: (0,
            s.e4)(e.name),
            albumName: (0,
            s.e4)(e.album),
            albumId: e.albumid,
            songmid: e.id,
            source: "kw",
            interval: (0,
            s.Zg)(parseInt(e.duration)),
            img: null,
            lrc: null,
            otherSource: null,
            types: n,
            _types: o,
            typeUrl:  {
              
            }
          }
        })
      },
      getTags() {
        return Promise.all([this.getTag(),
        this.getHotTag()]).then(([e,
        t]) = >( {
          tags: e,
          hotTag: t,
          source: "kw"
        }))
      },
      getDetailPageUrl(e) {
        if(/[?&:/]/.test(e))e = e.replace(this.regExps.listDetailLink,
        "$1");
        else if(/^digest-/.test(e)) {
          let t = e.split("__");
          e = t[1]
        }return`http://www.kuwo.cn/playlist_detail/${e}`
      },
      search(e,
      t,
      n = 20) {
        return(0,
        o.tX)(`http://search.kuwo.cn/r.s?all=${encodeURIComponent(e)}&pn=${t-1}&rn=${n}&rformat=json&encoding=utf8&ver=mbox&vipver=MUSIC_8.7.7.0_BCS37&plat=pc&devid=28156413&ft=playlist&pay=0&needliveshow=0`).promise.then(( {
          body: e
        }) = >( {
          list: (e = m(e)).abslist.map(e = >( {
            play_count: this.formatPlayCount(e.playcnt),
            id: String(e.playlistid),
            author: (0,
            s.e4)(e.nickname),
            name: (0,
            s.e4)(e.name),
            total: e.songnum,
            img: e.pic,
            desc: (0,
            s.e4)(e.intro),
            source: "kw"
          })),
          limit: n,
          total: parseInt(e.TOTAL),
          source: "kw"
        }))
      }
    },
    hotSearch:  {
      _requestObj: null,
      async getList(e = 0) {
        if(this._requestObj&&this._requestObj.cancelHttp(),
        e>2)return Promise.reject(new Error("try max num"));
        const t = (0,
        o.tX)("http://hotword.kuwo.cn/hotword.s?prod=kwplayer_ar_9.3.0.1&corp=kuwo&newver=2&vipver=9.3.0.1&source=kwplayer_ar_9.3.0.1_40.apk&p2p=1&notrace=0&uid=0&plat=kwplayer_ar&rformat=json&encoding=utf8&tabid=1",
         {
          headers:  {
            "User-Agent": "Dalvik/2.1.0 (Linux; U; Android 9;)"
          }
        }),
         {
          body: n,
          statusCode: i
        } = await t.promise;
        if(200! = i||"ok"! =  = n.status)throw new Error("获取热搜词失败");
        return {
          source: "kw",
          list: this.filterList(n.tagvalue)
        }
      },
      filterList: e = >e.map(e = >e.key)
    },
    comment:  {
      _requestObj: null,
      _requestObj2: null,
      async getComment( {
        songmid: e
      },
      t = 1,
      n = 20) {
        this._requestObj&&this._requestObj.cancelHttp();
        const i = (0,
        o.tX)(`http://ncomment.kuwo.cn/com.s?f=web&type=get_comment&aapiver=1&prod=kwplayer_ar_10.5.2.0&digest=15&sid=${e}&start=${n*(t-1)}&msgflag=1&count=${n}&newver=3&uid=0`,
         {
          headers:  {
            "User-Agent": "Dalvik/2.1.0 (Linux; U; Android 9;)"
          }
        }),
         {
          body: s,
          statusCode: a
        } = await i.promise;
        if(200! = a||"200"! = s.code)throw new Error("获取评论失败");
        const l = s.comments_counts;
        return {
          source: "kw",
          comments: this.filterComment(s.comments),
          total: l,
          page: t,
          limit: n,
          maxPage: Math.ceil(l/n)||1
        }
      },
      async getHotComment( {
        songmid: e
      },
      t = 1,
      n = 100) {
        this._requestObj2&&this._requestObj2.cancelHttp();
        const i = (0,
        o.tX)(`http://ncomment.kuwo.cn/com.s?f=web&type=get_rec_comment&aapiver=1&prod=kwplayer_ar_10.5.2.0&digest=15&sid=${e}&start=${n*(t-1)}&msgflag=1&count=${n}&newver=3&uid=0`,
         {
          headers:  {
            "User-Agent": "Dalvik/2.1.0 (Linux; U; Android 9;)"
          }
        }),
         {
          body: s,
          statusCode: a
        } = await i.promise;
        if(200! = a||"200"! = s.code)throw new Error("获取热门评论失败");
        const l = s.hot_comments_counts;
        return {
          source: "kw",
          comments: this.filterComment(s.hot_comments),
          total: l,
          page: t,
          limit: n,
          maxPage: Math.ceil(l/n)||1
        }
      },
      filterComment: e = >e?e.map(e = >( {
        id: e.id,
        text: e.msg,
        time: e.time,
        timeStr: (0,
        s.Uj)(1e3*Number(e.time)),
        userName: e.u_name,
        avatar: e.u_pic,
        userId: e.u_id,
        likedCount: e.like_num,
        images: e.mpic?[decodeURIComponent(e.mpic)]: [],
        reply: e.child_comments?e.child_comments.map(e = >( {
          id: e.id,
          text: e.msg,
          time: e.time,
          timeStr: (0,
          s.Uj)(1e3*Number(e.time)),
          userName: e.u_name,
          avatar: e.u_pic,
          userId: e.u_id,
          likedCount: e.like_num,
          images: e.mpic?[e.mpic]: []
        })): []
      })): []
    },
    getLyric: (e,
    t) = >x.getLyric(e,
    t),
    handleMusicInfo(e) {
      return this.getMusicInfo(e).then(t = >(e.name = t.name,
      e.singer = h(t.artist),
      e.img = t.pic,
      e.albumName = t.album,
      e))
    },
    getMusicUrl: (e,
    t) = >(0,
    C.O)("kw").getMusicUrl(e,
    t),
    getMusicInfo(e) {
      return this._musicInfoRequestObj&&this._musicInfoRequestObj.cancelHttp(),
      this._musicInfoRequestObj = (0,
      o.tX)(`http://www.kuwo.cn/api/www/music/musicInfo?mid=${e.songmid}`),
      this._musicInfoRequestObj.promise.then(( {
        body: e
      }) = >200 =  =  = e.code?e.data: Promise.reject(new Error(e.msg)))
    },
    getMusicUrls(e,
    t) {
      let n = [],
      o = e.songmid;
      e.types.forEach(e = > {
        n.push(N.getMusicUrl(o,
        e.type).promise)
      }),
      Promise.all(n).then(e = > {
        let n =  {
          
        };
        e.forEach(e = > {
          n[e.type] = e.url
        }),
        t(n)
      })
    },
    getPic: e = >S.getPic(e),
    getMusicDetailPageUrl: e = >`http://www.kuwo.cn/play_detail/${e.songmid}`
  },
  V = N;
  let L = [ {
    id: "kg__8888",
    name: "TOP500",
    bangid: "8888"
  },
   {
    id: "kg__6666",
    name: "飙升榜",
    bangid: "6666"
  },
   {
    id: "kg__59703",
    name: "蜂鸟流行音乐榜",
    bangid: "59703"
  },
   {
    id: "kg__52144",
    name: "抖音热歌榜",
    bangid: "52144"
  },
   {
    id: "kg__52767",
    name: "快手热歌榜",
    bangid: "52767"
  },
   {
    id: "kg__24971",
    name: "DJ热歌榜",
    bangid: "24971"
  },
   {
    id: "kg__23784",
    name: "网络红歌榜",
    bangid: "23784"
  },
   {
    id: "kg__44412",
    name: "说唱先锋榜",
    bangid: "44412"
  },
   {
    id: "kg__31308",
    name: "内地榜",
    bangid: "31308"
  },
   {
    id: "kg__33160",
    name: "电音榜",
    bangid: "33160"
  },
   {
    id: "kg__31313",
    name: "香港地区榜",
    bangid: "31313"
  },
   {
    id: "kg__51341",
    name: "民谣榜",
    bangid: "51341"
  },
   {
    id: "kg__54848",
    name: "台湾地区榜",
    bangid: "54848"
  },
   {
    id: "kg__31310",
    name: "欧美榜",
    bangid: "31310"
  },
   {
    id: "kg__33162",
    name: "ACG新歌榜",
    bangid: "33162"
  },
   {
    id: "kg__31311",
    name: "韩国榜",
    bangid: "31311"
  },
   {
    id: "kg__31312",
    name: "日本榜",
    bangid: "31312"
  },
   {
    id: "kg__49225",
    name: "80后热歌榜",
    bangid: "49225"
  },
   {
    id: "kg__49223",
    name: "90后热歌榜",
    bangid: "49223"
  },
   {
    id: "kg__49224",
    name: "00后热歌榜",
    bangid: "49224"
  },
   {
    id: "kg__33165",
    name: "粤语金曲榜",
    bangid: "33165"
  },
   {
    id: "kg__33166",
    name: "欧美金曲榜",
    bangid: "33166"
  },
   {
    id: "kg__33163",
    name: "影视金曲榜",
    bangid: "33163"
  },
   {
    id: "kg__51340",
    name: "伤感榜",
    bangid: "51340"
  },
   {
    id: "kg__35811",
    name: "会员专享榜",
    bangid: "35811"
  },
   {
    id: "kg__37361",
    name: "雷达榜",
    bangid: "37361"
  },
   {
    id: "kg__21101",
    name: "分享榜",
    bangid: "21101"
  },
   {
    id: "kg__46910",
    name: "综艺新歌榜",
    bangid: "46910"
  },
   {
    id: "kg__30972",
    name: "酷狗音乐人原创榜",
    bangid: "30972"
  },
   {
    id: "kg__60170",
    name: "闽南语榜",
    bangid: "60170"
  },
   {
    id: "kg__65234",
    name: "儿歌榜",
    bangid: "65234"
  },
   {
    id: "kg__4681",
    name: "美国BillBoard榜",
    bangid: "4681"
  },
   {
    id: "kg__25028",
    name: "Beatport电子舞曲榜",
    bangid: "25028"
  },
   {
    id: "kg__4680",
    name: "英国单曲榜",
    bangid: "4680"
  },
   {
    id: "kg__38623",
    name: "韩国Melon音乐榜",
    bangid: "38623"
  },
   {
    id: "kg__42807",
    name: "joox本地热歌榜",
    bangid: "42807"
  },
   {
    id: "kg__36107",
    name: "小语种热歌榜",
    bangid: "36107"
  },
   {
    id: "kg__4673",
    name: "日本公信榜",
    bangid: "4673"
  },
   {
    id: "kg__46868",
    name: "日本SPACE SHOWER榜",
    bangid: "46868"
  },
   {
    id: "kg__42808",
    name: "KKBOX风云榜",
    bangid: "42808"
  },
   {
    id: "kg__60171",
    name: "越南语榜",
    bangid: "60171"
  },
   {
    id: "kg__60172",
    name: "泰语榜",
    bangid: "60172"
  },
   {
    id: "kg__59895",
    name: "R&B榜",
    bangid: "59895"
  },
   {
    id: "kg__59896",
    name: "摇滚榜",
    bangid: "59896"
  },
   {
    id: "kg__59897",
    name: "爵士榜",
    bangid: "59897"
  },
   {
    id: "kg__59898",
    name: "乡村音乐榜",
    bangid: "59898"
  },
   {
    id: "kg__59900",
    name: "纯音乐榜",
    bangid: "59900"
  },
   {
    id: "kg__59899",
    name: "古典榜",
    bangid: "59899"
  },
   {
    id: "kg__22603",
    name: "5sing音乐榜",
    bangid: "22603"
  },
   {
    id: "kg__21335",
    name: "繁星音乐榜",
    bangid: "21335"
  },
   {
    id: "kg__33161",
    name: "古风新歌榜",
    bangid: "33161"
  }];
  const T =  {
    listDetailLimit: 100,
    list: [ {
      id: "kgtop500",
      name: "TOP500",
      bangid: "8888"
    },
     {
      id: "kgwlhgb",
      name: "网络榜",
      bangid: "23784"
    },
     {
      id: "kgbsb",
      name: "飙升榜",
      bangid: "6666"
    },
     {
      id: "kgfxb",
      name: "分享榜",
      bangid: "21101"
    },
     {
      id: "kgcyyb",
      name: "纯音乐榜",
      bangid: "33164"
    },
     {
      id: "kggfjqb",
      name: "古风榜",
      bangid: "33161"
    },
     {
      id: "kgyyjqb",
      name: "粤语榜",
      bangid: "33165"
    },
     {
      id: "kgomjqb",
      name: "欧美榜",
      bangid: "33166"
    },
     {
      id: "kgdyrgb",
      name: "电音榜",
      bangid: "33160"
    },
     {
      id: "kgjdrgb",
      name: "DJ热歌榜",
      bangid: "24971"
    },
     {
      id: "kghyxgb",
      name: "华语新歌榜",
      bangid: "31308"
    }],
    getUrl: (e,
    t,
    n) = >`http://mobilecdnbj.kugou.com/api/v3/rank/song?version=9108&ranktype=1&plat=0&pagesize=${n}&area_code=1&page=${e}&rankid=${t}&with_res_tag=0&show_portrait_mv=1`,
    regExps:  {
      total: /total: '(\d+)',/,
      page: /page: '(\d+)',/,
      limit: /pagesize: '(\d+)',/,
      listData: /global\.features = (\[.+\]);/
    },
    _requestBoardsObj: null,
    getBoardsData() {
      return this._requestBoardsObj&&this._requestBoardsObj.cancelHttp(),
      this._requestBoardsObj = (0,
      o.tX)("http://mobilecdnbj.kugou.com/api/v5/rank/list?version=9108&plat=0&showtype=2&parentid=0&apiver=6&area_code=1&withsong=1"),
      this._requestBoardsObj.promise
    },
    getData: e = >(0,
    o.tX)(e).promise,
    getSinger(e) {
      let t = [];
      return e.forEach(e = > {
        t.push(e.author_name)
      }),
      t.join("、")
    },
    filterData: e = >e.map(e = > {
      const t = [],
      n =  {
        
      };
      if(0! =  = e.filesize) {
        let o = (0,
        s.Uk)(e.filesize);
        t.push( {
          type: "128k",
          size: o,
          hash: e.hash
        }),
        n["128k"] =  {
          size: o,
          hash: e.hash
        }
      }if(0! =  = e["320filesize"]) {
        let o = (0,
        s.Uk)(e["320filesize"]);
        t.push( {
          type: "320k",
          size: o,
          hash: e["320hash"]
        }),
        n["320k"] =  {
          size: o,
          hash: e["320hash"]
        }
      }if(0! =  = e.sqfilesize) {
        let o = (0,
        s.Uk)(e.sqfilesize);
        t.push( {
          type: "flac",
          size: o,
          hash: e.sqhash
        }),
        n.flac =  {
          size: o,
          hash: e.sqhash
        }
      }if(0! =  = e.filesize_high) {
        let o = (0,
        s.Uk)(e.filesize_high);
        t.push( {
          type: "flac24bit",
          size: o,
          hash: e.hash_high
        }),
        n.flac24bit =  {
          size: o,
          hash: e.hash_high
        }
      }return {
        singer: c(e.authors,
        "author_name"),
        name: (0,
        s.e4)(e.songname),
        albumName: (0,
        s.e4)(e.remark),
        albumId: e.album_id,
        songmid: e.audio_id,
        albumAudioId: e.album_audio_id,
        source: "kg",
        interval: (0,
        s.Zg)(e.duration),
        img: null,
        lrc: null,
        hash: e.hash,
        otherSource: null,
        types: t,
        _types: n,
        typeUrl:  {
          
        }
      }
    }),
    filterBoardsData(e) {
      let t = [];
      for(const n of e)1 =  = n.isvol&&t.push( {
        id: "kg__"+n.rankid,
        name: n.rankname,
        bangid: String(n.rankid)
      });
      return t
    },
    async getBoards(e = 0) {
      return this.list = L,
       {
        list: L,
        source: "kg"
      }
    },
    async getList(e,
    t,
    n = 0) {
      if(++n>3)throw new Error("try max num");
      const {
        body: o
      } = await this.getData(this.getUrl(t,
      e,
      this.listDetailLimit));
      if(0! = o.errcode)return this.getList(e,
      t,
      n);
      return {
        total: o.data.total,
        list: this.filterData(o.data.info),
        limit: 100,
        page: t,
        source: "kg"
      }
    },
    getDetailPageUrl: e = >("string" =  = typeof e&&(e = e.replace("kg__",
    "")),
    `https://www.kugou.com/yy/rank/home/1-${e}.html`)
  };
  var I = n(88795),
  B = n.n(I);
  const $ = (e,
  t = "android",
  n = "") = > {
    let o = "OIlwieks28dk2k092lksi2UIkp";
    "web" =  =  = t&&(o = "NVPh5oo715z5DIWAeQlhMDsWXXQV4hwt");
    let i = e.split("&");
    i.sort();
    let s = `${o}${i.join("")}${n}${o}`;
    return r(s)
  },
  D =  {
    _requestObj_tags: null,
    _requestObj_listInfo: null,
    _requestObj_list: null,
    _requestObj_listRecommend: null,
    listDetailLimit: 1e4,
    currentTagInfo:  {
      id: void 0,
      info: void 0
    },
    sortList: [ {
      name: "推荐",
      id: "5"
    },
     {
      name: "最热",
      id: "6"
    },
     {
      name: "最新",
      id: "7"
    },
     {
      name: "热藏",
      id: "3"
    },
     {
      name: "飙升",
      id: "8"
    }],
    cache: new Map,
    regExps:  {
      listData: /global\.data = (\[.+\]);/,
      listInfo: /global = {[\s\S]+?name: "(.+)"[\s\S]+?pic: "(.+)"[\s\S]+?};/,
      listDetailLink: /^.+\/(\d+)\.html(?:\?.*|&.*$|#.*$|$)/
    },
    parseHtmlDesc(e) {
      const t = '<div class="pc_specail_text pc_singer_tab_content" id="specailIntroduceWrap">';
      let n = e.indexOf(t);
      if(n<0)return null;
      const o = e.substring(n+77);
      return n = o.indexOf("</div>"),
      n<0?null: (0,
      s.e4)(o.substring(0,
      n))
    },
    async getListDetailBySpecialId(e,
    t,
    n = 0) {
      if(n>2)throw new Error("try max num");
      const {
        body: i
      } = await(0,
      o.tX)(this.getSongListDetailUrl(e)).promise;
      let s = i.match(this.regExps.listData),
      a = i.match(this.regExps.listInfo);
      if(!s)return this.getListDetailBySpecialId(e,
      t,
      ++n);
      let l,
      r,
      c = await this.getMusicInfos(JSON.parse(s[1]));
      a&&(l = a[1],
      r = a[2]);
      let d = this.parseHtmlDesc(i);
      return {
        list: c,
        page: 1,
        limit: 1e4,
        total: c.length,
        source: "kg",
        info:  {
          name: l,
          img: r,
          desc: d
        }
      }
    },
    getInfoUrl: e = >e?`http://www2.kugou.kugou.com/yueku/v9/special/getSpecial?is_smarty=1&cdn=cdn&t=5&c=${e}`: "http://www2.kugou.kugou.com/yueku/v9/special/getSpecial?is_smarty=1&",
    getSongListUrl: (e,
    t,
    n) = >(null =  = t&&(t = ""),
    `http://www2.kugou.kugou.com/yueku/v9/special/getSpecial?is_ajax=1&cdn=cdn&t=${e}&c=${t}&p=${n}`),
    getSongListDetailUrl: e = >`http://www2.kugou.kugou.com/yueku/v9/special/single/${e}-5-9999.html`,
    filterInfoHotTag(e) {
      const t = [];
      if(1! =  = e.status)return t;
      for(const n of Object.keys(e.data)) {
        let o = e.data[n];
        t.push( {
          id: o.special_id,
          name: o.special_name,
          source: "kg"
        })
      }return t
    },
    filterTagInfo(e) {
      const t = [];
      for(const n of Object.keys(e))t.push( {
        name: n,
        list: e[n].data.map(e = >( {
          parent_id: e.parent_id,
          parent_name: e.pname,
          id: e.id,
          name: e.name,
          source: "kg"
        }))
      });
      return t
    },
    getSongList(e,
    t,
    n,
    i = 0) {
      return this._requestObj_list&&this._requestObj_list.cancelHttp(),
      i>2?Promise.reject(new Error("try max num")): (this._requestObj_list = (0,
      o.tX)(this.getSongListUrl(e,
      t,
      n)),
      this._requestObj_list.promise.then(( {
        body: o
      }) = >o&&1 =  =  = o.status?this.filterList(o.special_db): this.getSongList(e,
      t,
      n,
      ++i)))
    },
    getSongListRecommend(e = 0) {
      return this._requestObj_listRecommend&&this._requestObj_listRecommend.cancelHttp(),
      e>2?Promise.reject(new Error("try max num")): (this._requestObj_listRecommend = (0,
      o.tX)("http://everydayrec.service.kugou.com/guess_special_recommend",
       {
        method: "post",
        headers:  {
          "User-Agent": "KuGou2012-8275-web_browser_event_handler"
        },
        body:  {
          appid: 1001,
          clienttime: 1566798337219,
          clientver: 8275,
          key: "f1f93580115bb106680d2375f8032d96",
          mid: "21511157a05844bd085308bc76ef3343",
          platform: "pc",
          userid: "262643156",
          return_min: 6,
          return_max: 15
        }
      }),
      this._requestObj_listRecommend.promise.then(( {
        body: t
      }) = >1! =  = t.status?this.getSongListRecommend(++e): this.filterList(t.data.special_list)))
    },
    filterList: e = >e.map(e = >( {
      play_count: e.total_play_count||(0,
      s.pN)(e.play_count),
      id: "id_"+e.specialid,
      author: e.nickname,
      name: e.specialname,
      time: (0,
      s.i$)(e.publish_time||e.publishtime,
      "Y-M-D"),
      img: e.img||e.imgurl,
      total: e.songcount,
      grade: e.grade,
      desc: e.intro,
      source: "kg"
    })),
    async createHttp(e,
    t,
    n = 0) {
      if(n>2)throw new Error("try max num");
      let i;
      try {
        i = await(0,
        o.tX)(e,
        t).promise
      }catch(o) {
        return console.log(o),
        this.createHttp(e,
        t,
        ++n)
      }return 200! =  = i.statusCode||0! =  = (void 0! =  = i.body.error_code?i.body.error_code: void 0! =  = i.body.errcode?i.body.errcode: i.body.err_code)?this.createHttp(e,
      t,
      ++n): i.body.data?i.body.data: Array.isArray(i.body.info)?i.body: i.body.info
    },
    createTask(e) {
      let t =  {
        area_code: "1",
        show_privilege: 1,
        show_album_info: "1",
        is_publish: "",
        appid: 1005,
        clientver: 11451,
        mid: "1",
        dfid: "-",
        clienttime: Date.now(),
        key: "OIlwieks28dk2k092lksi2UIkp",
        fields: "album_info,author_name,audio_info,ori_audio_name,base,songname"
      },
      n = e,
      o = [];
      for(;
      n.length&&(o.push(Object.assign( {
        data: n.slice(0,
        100)
      },
      t)),
      !(n.length<100));
      )n = n.slice(100);
      return o.map(e = >this.createHttp("http://gateway.kugou.com/v2/album_audio/audio",
       {
        method: "POST",
        body: e,
        headers:  {
          "KG-THash": "13a3164",
          "KG-RC": "1",
          "KG-Fake": "0",
          "KG-RF": "00869891",
          "User-Agent": "Android712-AndroidPhone-11451-376-0-FeeCacheUpdate-wifi",
          "x-router": "kmr.service.kugou.com"
        }
      }).then(e = >e.map(e = >e[0])))
    },
    async getMusicInfos(e) {
      return this.filterData2(await Promise.all(this.createTask(this.deDuplication(e).map(e = >( {
        hash: e.hash
      })))).then(([...e]) = >e.flat()))
    },
    async getUserListDetailByCode(e) {
      const t = await this.createHttp("http://t.kugou.com/command/",
       {
        method: "POST",
        headers:  {
          "KG-RC": 1,
          "KG-THash": "network_super_call.cpp:3676261689:379",
          "User-Agent": ""
        },
        body:  {
          appid: 1001,
          clientver: 9020,
          mid: "21511157a05844bd085308bc76ef3343",
          clienttime: 640612895,
          key: "36164c4015e704673c588ee202b9ecb8",
          data: e
        }
      });
      let n,
      o = t.info;
      if(2 =  =  = o.type)if(!o.global_collection_id)return this.getListDetailBySpecialId(o.id);
      if(o.global_collection_id)return this.getUserListDetail2(o.global_collection_id);
      null! = o.userid&&(n = await this.createHttp("http://www2.kugou.kugou.com/apps/kucodeAndShare/app/",
       {
        method: "POST",
        headers:  {
          "KG-RC": 1,
          "KG-THash": "network_super_call.cpp:3676261689:379",
          "User-Agent": ""
        },
        body:  {
          appid: 1001,
          clientver: 9020,
          mid: "21511157a05844bd085308bc76ef3343",
          clienttime: 640612895,
          key: "36164c4015e704673c588ee202b9ecb8",
          data:  {
            id: o.id,
            type: 3,
            userid: o.userid,
            collect_type: 0,
            page: 1,
            pagesize: o.count
          }
        }
      }));
      let i = await this.getMusicInfos(n||t.list);
      return {
        list: i,
        page: 1,
        limit: o.count,
        total: i.length,
        source: "kg",
        info:  {
          name: o.name,
          img: o.img_size&&o.img_size.replace("{size}",
          240)||o.img,
          author: o.username
        }
      }
    },
    async getUserListDetail3(e,
    t) {
      const n = await this.createHttp(`http://m.kugou.com/schain/transfer?pagesize=${this.listDetailLimit}&chain=${e}&su=1&page=${t}&n=0.7928855356604456`,
       {
        headers:  {
          "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 9_1 like Mac OS X) AppleWebKit/601.1.46 (KHTML, like Gecko) Version/9.0 Mobile/13B143 Safari/601.1"
        }
      });
      if(!n.list)return n.global_collection_id?this.getListDetailByGcidChain(n.global_collection_id,
      e).catch(() = >this.getUserListDetail2(n.global_collection_id)): this.getUserListDetail4(n,
      e,
      t).catch(() = >this.getUserListDetail5(e));
      let o = await this.getMusicInfos(n.list);
      return {
        list: o,
        page: 1,
        limit: this.listDetailLimit,
        total: o.length,
        source: "kg",
        info:  {
          name: n.info.name,
          img: n.info.img,
          author: n.info.username
        }
      }
    },
    deDuplication(e) {
      let t = new Set;
      return e.filter(( {
        hash: e
      }) = >!t.has(e)&&(t.add(e),
      !0))
    },
    async decodeGcid(e) {
      const t = "dfid=-&appid=1005&mid=0&clientver=20109&clienttime=640612895&uuid=-",
      n =  {
        ret_info: 1,
        data: [ {
          id: e,
          id_type: 2
        }]
      };
      return(await this.createHttp(`https://t.kugou.com/v1/songlist/batch_decode?${t}&signature=${$(t,"android",JSON.stringify(n))}`,
       {
        method: "POST",
        headers:  {
          "User-Agent": "Mozilla/5.0 (Linux; Android 10; HUAWEI HMA-AL00) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.106 Mobile Safari/537.36",
          Referer: "https://m.kugou.com/"
        },
        body: n
      })).list[0].global_collection_id
    },
    async getUserListDetailByLink( {
      info: e
    },
    t) {
      let n = e[0],
      o = n.count,
      i = [],
      s = 0;
      for(;
      o;
      ) {
        const e = o>90?90: o;
        o- = e,
        s+ = 1,
        i.push(this.createHttp(t.replace(/pagesize=\d+/,
        "pagesize="+e).replace(/page=\d+/,
        "page="+s),
         {
          headers:  {
            "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 9_1 like Mac OS X) AppleWebKit/601.1.46 (KHTML, like Gecko) Version/9.0 Mobile/13B143 Safari/601.1",
            Referer: t
          }
        }).then(e = >e.list.info))
      }let a = await Promise.all(i).then(([...e]) = >e.flat());
      return a = await this.getMusicInfos(a),
       {
        list: a,
        page: s,
        limit: this.listDetailLimit,
        total: a.length,
        source: "kg",
        info:  {
          name: n.name,
          img: n.pic&&n.pic.replace("{size}",
          240),
          author: n.list_create_username
        }
      }
    },
    createGetListDetail2Task(e,
    t) {
      let n = [],
      o = 0;
      for(;
      t;
      ) {
        const i = t>300?300: t;
        t- = i,
        o+ = 1;
        const s = "appid=1058&global_specialid="+e+"&specialid=0&plat=0&version=8000&page="+o+"&pagesize="+i+"&srcappid=2919&clientver=20000&clienttime=1586163263991&mid=1586163263991&uuid=1586163263991&dfid=-";
        n.push(this.createHttp(`https://mobiles.kugou.com/api/v5/special/song_v2?${s}&signature=${$(s,"web")}`,
         {
          headers:  {
            mid: "1586163263991",
            Referer: "https://m3ws.kugou.com/share/index.php",
            "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1",
            dfid: "-",
            clienttime: "1586163263991"
          }
        }).then(e = >e.info))
      }return Promise.all(n).then(([...e]) = >e.flat())
    },
    async getListDetailByGcidChain(e,
    t) {
      const n = "srcappid=2919&clientver=20000&clienttime=1789566751657&mid=1789566751657&uuid=1789566751657&dfid=-&listid=15&type=0&pagesize=50&global_collection_id="+e+"&page=1&share_type=collect&appid=1058&chain="+t,
      o = await this.createHttp(`https://m3ws.kugou.com/v2/zlist/list?${n}&signature=${$(n,"web")}`,
       {
        headers:  {
          mid: "1586163263991",
          Referer: "https://m3ws.kugou.com/share/index.php",
          "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1",
          dfid: "-",
          clienttime: "1586163263991"
        }
      }),
      i = o.info[0];
      if(!i)throw new Error("get list error");
      let s = [Promise.resolve(o.list.info)],
      a = 1,
      l = Math.ceil(i.count/50);
      for(;
      a<l;
      ) {
        a+ = 1;
        const n = "srcappid=2919&clientver=20000&clienttime=1789566751657&mid=1789566751657&uuid=1789566751657&dfid=-&listid=15&type=0&pagesize="+50+"&global_collection_id="+e+"&page="+a+"&share_type=collect&appid=1058&chain="+t;
        s.push(this.createHttp(`https://m3ws.kugou.com/v2/zlist/list?${n}&signature=${$(n,"web")}`,
         {
          headers:  {
            mid: "1586163263991",
            Referer: "https://m3ws.kugou.com/share/index.php",
            "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1",
            dfid: "-",
            clienttime: "1586163263991"
          }
        }).then(e = >e.list.info))
      }const r = await Promise.all(s).then(([...e]) = >e.flat());
      return {
        list: await this.getMusicInfos(r),
        page: 1,
        limit: this.listDetailLimit,
        total: i.count,
        source: "kg",
        info:  {
          name: i.name,
          img: i.pic&&i.pic.replace("{size}",
          240),
          desc: i.intro,
          author: i.list_create_username
        }
      }
    },
    async getUserListDetail2(e) {
      let t = e;
      if(t.length>1e3)throw new Error("get list error");
      const n = "appid=1058&specialid=0&global_specialid="+t+"&format=jsonp&srcappid=2919&clientver=20000&clienttime=1586163242519&mid=1586163242519&uuid=1586163242519&dfid=-";
      let o = await this.createHttp(`https://mobiles.kugou.com/api/v5/special/info_v2?${n}&signature=${$(n,"web")}`,
       {
        headers:  {
          mid: "1586163242519",
          Referer: "https://m3ws.kugou.com/share/index.php",
          "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1",
          dfid: "-",
          clienttime: "1586163242519"
        }
      });
      const i = await this.createGetListDetail2Task(t,
      o.songcount);
      let a = await this.getMusicInfos(i);
      return {
        list: a,
        page: 1,
        limit: this.listDetailLimit,
        total: a.length,
        source: "kg",
        info:  {
          name: o.specialname,
          img: o.imgurl&&o.imgurl.replace("{size}",
          240),
          desc: o.intro,
          author: o.nickname,
          play_count: (0,
          s.pN)(o.playcount)
        }
      }
    },
    async getListInfoByChain(e) {
      if(this.cache.has(e))return this.cache.get(e);
      const {
        body: t
      } = await(0,
      o.tX)(`https://m.kugou.com/share/?chain=${e}&id=${e}`,
       {
        headers:  {
          "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1"
        }
      }).promise;
      let n = t.match(/var\sphpParam\s=\s({.+?});/);
      return n&&(n = JSON.parse(n[1])),
      this.cache.set(e,
      n),
      n
    },
    async getUserListDetailByPcChain(e) {
      let t = `${e}_pc_list`;
      if(this.cache.has(t))return this.cache.get(t);
      const {
        body: n
      } = await(0,
      o.tX)(`http://www.kugou.com/share/${e}.html`,
       {
        headers:  {
          "User-Agent": "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36"
        }
      }).promise;
      let i = n.match(/var\sdataFromSmarty\s=\s(\[.+?\])/);
      return i&&(i = JSON.parse(i[1])),
      this.cache.set(e,
      i),
      i = await this.getMusicInfos(i),
      i
    },
    async getUserListDetail4(e,
    t,
    n) {
      const[o,
      i] = await Promise.all([this.getListInfoByChain(t),
      this.getUserListDetailById(e.id,
      n,
      100)]);
      return {
        list: i||[],
        page: n,
        limit: 100,
        total: i.length??0,
        source: "kg",
        info:  {
          name: o.specialname,
          img: o.imgurl&&o.imgurl.replace("{size}",
          240),
          author: o.nickname
        }
      }
    },
    async getUserListDetail5(e) {
      const[t,
      n] = await Promise.all([this.getListInfoByChain(e),
      this.getUserListDetailByPcChain(e)]);
      return {
        list: n||[],
        page: 1,
        limit: this.listDetailLimit,
        total: n.length??0,
        source: "kg",
        info:  {
          name: t.specialname,
          img: t.imgurl&&t.imgurl.replace("{size}",
          240),
          author: t.nickname
        }
      }
    },
    async getUserListDetailById(e,
    t,
    n) {
      const o = await((e,
      t,
      n) = >new Promise((o,
      i) = > {
        B()( {
          appid: 1058,
          type: 0,
          module: "playlist",
          page: t,
          pagesize: n,
          specialid: e
        },
        null,
         {
          useH5: !0,
          isCDN: !0,
          callback(e) {
            o(e.signature)
          }
        })
      }))(e,
      t,
      n);
      let i = await this.createHttp(`https://pubsongscdn.kugou.com/v2/get_other_list_file?srcappid=2919&clientver=20000&appid=1058&type=0&module=playlist&page=${t}&pagesize=${n}&specialid=${e}&signature=${o}`,
       {
        headers:  {
          Referer: "https://m3ws.kugou.com/share/index.php",
          "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1",
          dfid: "-"
        }
      });
      return await this.getMusicInfos(i.info)
    },
    async getUserListDetail(e,
    t,
    n = 0) {
      if(n>3)return Promise.reject(new Error("link try max num"));
      if(e.includes("#")&&(e = e.replace(/#.*$/,
      "")),
      e.includes("global_collection_id")&&e.includes("chain"))return this.getListDetailByGcidChain(e.replace(/^.*?global_collection_id=(\w+)(?:&.*$|#.*$|$)/,
      "$1"),
      e.replace(/^.*?chain=(\w+)(?:&.*$|#.*$|$)/,
      "$1"));
      if(e.includes("chain="))return this.getUserListDetail3(e.replace(/^.*?chain=(\w+)(?:&.*$|#.*$|$)/,
      "$1"),
      t);
      if(e.includes("gcid_")) {
        let t = e.match(/gcid_\w+/)?.[0];
        if(t) {
          const e = await this.decodeGcid(t);
          if(e)return this.getUserListDetail2(e)
        }
      }if(e.includes(".html"))if(e.includes("zlist.html"))(e = e.replace(/^(.*)zlist\.html/,
      "https://m3ws.kugou.com/zlist/list")).includes("pagesize")?e = e.replace("pagesize=30",
      "pagesize="+this.listDetailLimit).replace("page=1",
      "page="+t): e+ = `&pagesize=${this.listDetailLimit}&page=${t}`;
      else if(!e.includes("song.html"))return this.getUserListDetail3(e.replace(/.+\/(\w+).html(?:\?.*|&.*$|#.*$|$)/,
      "$1"),
      t);
      if(e.includes("global_collection_id"))return this.getUserListDetail2(e.replace(/^.*?global_collection_id=(\w+)(?:&.*$|#.*$|$)/,
      "$1"));
      const i = (0,
      o.tX)(e,
       {
        headers:  {
          "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 9_1 like Mac OS X) AppleWebKit/601.1.46 (KHTML, like Gecko) Version/9.0 Mobile/13B143 Safari/601.1",
          Referer: e
        }
      }),
       {
        headers:  {
          location: s
        },
        statusCode: a,
        body: l
      } = await i.promise;
      if(a>400)return this.getUserListDetail(e,
      t,
      ++n);
      if(s) {
        if(s.includes("global_collection_id")&&s.includes("chain"))return this.getListDetailByGcidChain(s.replace(/^.*?global_collection_id=(\w+)(?:&.*$|#.*$|$)/,
        "$1"),
        s.replace(/^.*?chain=(\w+)(?:&.*$|#.*$|$)/,
        "$1"));
        if(s.includes("chain="))return this.getUserListDetail3(s.replace(/^.*?chain=(\w+)(?:&.*$|#.*$|$)/,
        "$1"),
        t);
        if(s.includes("gcid_")) {
          let e = s.match(/gcid_\w+/)?.[0];
          if(e) {
            const t = await this.decodeGcid(e);
            if(t)return this.getUserListDetail2(t)
          }
        }if(s.includes(".html")) {
          if(s.includes("zlist.html")) {
            let e = s.replace(/^(.*)zlist\.html/,
            "https://m3ws.kugou.com/zlist/list");
            return e.includes("pagesize")?e = e.replace("pagesize=30",
            "pagesize="+this.listDetailLimit).replace("page=1",
            "page="+t): e+ = `&pagesize=${this.listDetailLimit}&page=${t}`,
            this.getUserListDetail(e,
            t,
            ++n)
          }return this.getUserListDetail3(s.replace(/.+\/(\w+).html(?:\?.*|&.*$|#.*$|$)/,
          "$1"),
          t)
        }return s.includes("global_collection_id")?this.getUserListDetail2(s.replace(/^.*?global_collection_id=(\w+)(?:&.*$|#.*$|$)/,
        "$1")): this.getUserListDetail(s,
        t,
        ++n)
      }if("string" =  = typeof l) {
        let e = l.match(/"global_collection_id":"(\w+)"/)?.[1];
        if(!e) {
          let t = l.match(/"encode_gic":"(\w+)"/)?.[1];
          t||(t = l.match(/"encode_src_gid":"(\w+)"/)?.[1]),
          t&&(e = await this.decodeGcid(t))
        }if(!e)throw new Error("get list error");
        return this.getUserListDetail2(e)
      }return 0! =  = l.errcode?this.getUserListDetail(e,
      t,
      ++n): this.getUserListDetailByLink(l,
      e)
    },
    async getListDetail(e,
    t) {
      if((e = e.toString()).includes("special/single/"))e = e.replace(this.regExps.listDetailLink,
      "$1");
      else {
        if(/https?:/.test(e))return this.getUserListDetail(e.replace(/^.*?http/,
        "http"),
        t);
        if(/^\d+$/.test(e))return this.getUserListDetailByCode(e);
        e.startsWith("id_")&&(e = e.replace("id_",
        ""))
      }return this.getListDetailBySpecialId(e,
      t)
    },
    filterData: e = >e.map(e = > {
      const t = [],
      n =  {
        
      };
      if(0! =  = e.filesize) {
        let o = (0,
        s.Uk)(e.filesize);
        t.push( {
          type: "128k",
          size: o,
          hash: e.hash
        }),
        n["128k"] =  {
          size: o,
          hash: e.hash
        }
      }if(0! =  = e.filesize_320) {
        let o = (0,
        s.Uk)(e.filesize_320);
        t.push( {
          type: "320k",
          size: o,
          hash: e.hash_320
        }),
        n["320k"] =  {
          size: o,
          hash: e.hash_320
        }
      }if(0! =  = e.filesize_ape) {
        let o = (0,
        s.Uk)(e.filesize_ape);
        t.push( {
          type: "ape",
          size: o,
          hash: e.hash_ape
        }),
        n.ape =  {
          size: o,
          hash: e.hash_ape
        }
      }if(0! =  = e.filesize_flac) {
        let o = (0,
        s.Uk)(e.filesize_flac);
        t.push( {
          type: "flac",
          size: o,
          hash: e.hash_flac
        }),
        n.flac =  {
          size: o,
          hash: e.hash_flac
        }
      }return {
        singer: (0,
        s.e4)(e.singername),
        name: (0,
        s.e4)(e.songname),
        albumName: (0,
        s.e4)(e.album_name),
        albumId: e.album_id,
        songmid: e.audio_id,
        albumAudioId: e.album_audio_id,
        source: "kg",
        interval: (0,
        s.Zg)(e.duration/1e3),
        img: null,
        lrc: null,
        hash: e.hash,
        types: t,
        _types: n,
        typeUrl:  {
          
        }
      }
    }),
    filterData2(e) {
      let t = new Set,
      n = [];
      return e.forEach(e = > {
        if(!e)return;
        if(t.has(e.audio_info.audio_id))return;
        t.add(e.audio_info.audio_id);
        const o = [],
        i =  {
          
        };
        if("0"! =  = e.audio_info.filesize) {
          let t = (0,
          s.Uk)(parseInt(e.audio_info.filesize));
          o.push( {
            type: "128k",
            size: t,
            hash: e.audio_info.hash
          }),
          i["128k"] =  {
            size: t,
            hash: e.audio_info.hash
          }
        }if("0"! =  = e.audio_info.filesize_320) {
          let t = (0,
          s.Uk)(parseInt(e.audio_info.filesize_320));
          o.push( {
            type: "320k",
            size: t,
            hash: e.audio_info.hash_320
          }),
          i["320k"] =  {
            size: t,
            hash: e.audio_info.hash_320
          }
        }if("0"! =  = e.audio_info.filesize_flac) {
          let t = (0,
          s.Uk)(parseInt(e.audio_info.filesize_flac));
          o.push( {
            type: "flac",
            size: t,
            hash: e.audio_info.hash_flac
          }),
          i.flac =  {
            size: t,
            hash: e.audio_info.hash_flac
          }
        }if("0"! =  = e.audio_info.filesize_high) {
          let t = (0,
          s.Uk)(parseInt(e.audio_info.filesize_high));
          o.push( {
            type: "flac24bit",
            size: t,
            hash: e.audio_info.hash_high
          }),
          i.flac24bit =  {
            size: t,
            hash: e.audio_info.hash_high
          }
        }n.push( {
          singer: (0,
          s.e4)(e.author_name),
          name: (0,
          s.e4)(e.songname),
          albumName: (0,
          s.e4)(e.album_info.album_name),
          albumId: e.album_info.album_id,
          songmid: e.audio_info.audio_id,
          albumAudioId: e.audio_info.audio_group_id,
          source: "kg",
          interval: (0,
          s.Zg)(parseInt(e.audio_info.timelength)/1e3),
          img: null,
          lrc: null,
          hash: e.audio_info.hash,
          otherSource: null,
          types: o,
          _types: i,
          typeUrl:  {
            
          }
        })
      }),
      n
    },
    getListInfo(e,
    t = 0) {
      return this._requestObj_listInfo&&this._requestObj_listInfo.cancelHttp(),
      t>2?Promise.reject(new Error("try max num")): (this._requestObj_listInfo = (0,
      o.tX)(this.getInfoUrl(e)),
      this._requestObj_listInfo.promise.then(( {
        body: n
      }) = >1! =  = n.status?this.getListInfo(e,
      ++t):  {
        limit: n.data.params.pagesize,
        page: n.data.params.p,
        total: n.data.params.total,
        source: "kg"
      }))
    },
    getList(e,
    t,
    n) {
      let o = [this.getSongList(e,
      t,
      n)];
      return o.push(this.currentTagInfo.id =  =  = t?Promise.resolve(this.currentTagInfo.info): this.getListInfo(t).then(e = >(this.currentTagInfo.id = t,
      this.currentTagInfo.info = Object.assign( {
        
      },
      e),
      e))),
      t||1! =  = n||e! =  = this.sortList[0].id||o.push(this.getSongListRecommend()),
      Promise.all(o).then(([e,
      t,
      n]) = >(n&&e.unshift(...n),
       {
        list: e,
        ...t
      }))
    },
    getTags(e = 0) {
      return this._requestObj_tags&&this._requestObj_tags.cancelHttp(),
      e>2?Promise.reject(new Error("try max num")): (this._requestObj_tags = (0,
      o.tX)(this.getInfoUrl()),
      this._requestObj_tags.promise.then(( {
        body: t
      }) = >1! =  = t.status?this.getTags(++e):  {
        hotTag: this.filterInfoHotTag(t.data.hotTag),
        tags: this.filterTagInfo(t.data.tagids),
        source: "kg"
      }))
    },
    getDetailPageUrl(e) {
      if("string" =  = typeof e) {
        if(/^https?:\/\//.test(e))return e;
        e = e.replace("id_",
        "")
      }return`https://www.kugou.com/yy/special/single/${e}.html`
    },
    search: (e,
    t,
    n = 20) = >(0,
    o.tX)(`http://msearchretry.kugou.com/api/v3/search/special?keyword=${encodeURIComponent(e)}&page=${t}&pagesize=${n}&showtype=10&filter=0&version=7910&sver=2`).promise.then(( {
      body: e
    }) = > {
      if(0! = e.errcode)throw new Error("filed");
      return {
        list: e.data.info.map(e = >( {
          play_count: (0,
          s.pN)(e.playcount),
          id: "id_"+e.specialid,
          author: e.nickname,
          name: e.specialname,
          time: (0,
          s.i$)(e.publishtime,
          "Y-M-D"),
          img: e.imgurl,
          grade: e.grade,
          desc: e.intro,
          total: e.songcount,
          source: "kg"
        })),
        limit: n,
        total: e.data.total,
        source: "kg"
      }
    })
  },
  P =  {
    limit: 30,
    total: 0,
    page: 0,
    allPage: 1,
    musicSearch: (e,
    t,
    n) = >(0,
    o.tX)(`http://songsearch.kugou.com/song_search_v2?platform=AndroidFilter&iscorrection=1&keyword=${encodeURIComponent(e)}&hifiquality=0&pagesize=${n}&PrivilegeFilter=0&page=${t}`).promise.then(( {
      body: e
    }) = >e),
    filterData(e) {
      const t = [],
      n =  {
        
      };
      if(0! =  = e.FileSize) {
        let o = (0,
        s.Uk)(e.FileSize);
        t.push( {
          type: "128k",
          size: o,
          hash: e.FileHash
        }),
        n["128k"] =  {
          size: o,
          hash: e.FileHash
        }
      }if(0! =  = e.HQFileSize) {
        let o = (0,
        s.Uk)(e.HQFileSize);
        t.push( {
          type: "320k",
          size: o,
          hash: e.HQFileHash
        }),
        n["320k"] =  {
          size: o,
          hash: e.HQFileHash
        }
      }if(0! =  = e.SQFileSize) {
        let o = (0,
        s.Uk)(e.SQFileSize);
        t.push( {
          type: "flac",
          size: o,
          hash: e.SQFileHash
        }),
        n.flac =  {
          size: o,
          hash: e.SQFileHash
        }
      }if(0! =  = e.ResFileSize) {
        let o = (0,
        s.Uk)(e.ResFileSize);
        t.push( {
          type: "flac24bit",
          size: o,
          hash: e.ResFileHash
        }),
        n.flac24bit =  {
          size: o,
          hash: e.ResFileHash
        }
      }return {
        singer: (0,
        s.e4)(c(e.Singers,
        "name")),
        name: (0,
        s.e4)(`${e.OriSongName}${e.Suffix?` $ {
          e.Suffix
        }`:""}`),
        albumName: (0,
        s.e4)(e.AlbumName),
        albumId: e.AlbumID,
        songmid: e.Audioid,
        albumAudioId: e.MixSongID,
        source: "kg",
        interval: (0,
        s.Zg)(e.Duration),
        _interval: e.Duration,
        img: null,
        lrc: null,
        otherSource: null,
        hash: e.FileHash,
        types: t,
        _types: n,
        typeUrl:  {
          
        }
      }
    },
    handleResult(e) {
      let t = new Set;
      const n = [];
      return e.forEach(e = > {
        const o = e.Audioid+e.FileHash;
        if(!t.has(o)) {
          t.add(o),
          n.push(this.filterData(e));
          for(const o of e.Grp) {
            const i = e.Audioid+e.FileHash;
            t.has(i)||(t.add(i),
            n.push(this.filterData(o)))
          }
        }
      }),
      n
    },
    search(e,
    t = 1,
    n,
    o = 0) {
      return++o>3?Promise.reject(new Error("try max num")): (null =  = n&&(n = this.limit),
      this.musicSearch(e,
      t,
      n).then(i = > {
        if(!i||0! =  = i.error_code)return this.search(e,
        t,
        n,
        o);
        let s = this.handleResult(i.data.lists);
        return null =  = s?this.search(e,
        t,
        n,
        o): (this.total = i.data.total,
        this.page = t,
        this.allPage = Math.ceil(this.total/n),
        Promise.resolve( {
          list: s,
          allPage: this.allPage,
          limit: n,
          total: this.total,
          source: "kg"
        }))
      }))
    }
  },
  M =  {
    getPic: e = >(0,
    o.tX)("http://media.store.kugou.com/v1/get_res_privilege",
     {
      method: "POST",
      headers:  {
        "KG-RC": 1,
        "KG-THash": "expand_search_manager.cpp:852736169:451",
        "User-Agent": "KuGou2012-9020-ExpandSearchManager"
      },
      body:  {
        appid: 1001,
        area_code: "1",
        behavior: "play",
        clientver: "9020",
        need_hash_offset: 1,
        relate: 1,
        resource: [ {
          album_audio_id: 32 =  = e.songmid.length?e.audioId.split("_")[0]: e.songmid,
          album_id: e.albumId,
          hash: e.hash,
          id: 0,
          name: `${e.singer} - ${e.name}.mp3`,
          type: "audio"
        }],
        token: "",
        userid: 2626431536,
        vip: 1
      }
    }).promise.then(( {
      body: e
    }) = > {
      if(0! =  = e.error_code)return Promise.reject(new Error("图片获取失败"));
      let t = e.data[0].info;
      const n = t.imgsize?t.image.replace("{size}",
      t.imgsize[0]): t.image;
      return n||Promise.reject(new Error("Pic get failed"))
    })
  };
  var z = n(92617);
  const O =  {
    getIntv(e) {
      if(!e)return 0;
      let t = e.split(":"),
      n = 0,
      o = 1;
      for(;
      t.length;
      )n+ = t.pop()*o,
      o* = 60;
      return parseInt(n)
    },
    searchLyric(e,
    t,
    n,
    i = 0) {
      let s = (0,
      o.tX)(`http://lyrics.kugou.com/search?ver=1&man=yes&client=pc&keyword=${encodeURIComponent(e)}&hash=${t}&timelength=${n}&lrctxt=1`,
       {
        headers:  {
          "KG-RC": 1,
          "KG-THash": "expand_search_manager.cpp:852736169:451",
          "User-Agent": "KuGou2012-9020-ExpandSearchManager"
        }
      });
      return s.promise = s.promise.then(( {
        body: o,
        statusCode: a
      }) = > {
        if(200! =  = a) {
          if(i>5)return Promise.reject(new Error("歌词获取失败"));
          let o = this.searchLyric(e,
          t,
          n,
          ++i);
          return s.cancelHttp = o.cancelHttp.bind(o),
          o.promise
        }if(o.candidates.length) {
          let e = o.candidates[0];
          return {
            id: e.id,
            accessKey: e.accesskey,
            fmt: 1 =  = e.krctype&&1! = e.contenttype?"krc": "lrc"
          }
        }return null
      }),
      s
    },
    getLyricDownload(e,
    t,
    n,
    i = 0) {
      let s = (0,
      o.tX)(`http://lyrics.kugou.com/download?ver=1&client=pc&id=${e}&accesskey=${t}&fmt=${n}&charset=utf8`,
       {
        headers:  {
          "KG-RC": 1,
          "KG-THash": "expand_search_manager.cpp:852736169:451",
          "User-Agent": "KuGou2012-9020-ExpandSearchManager"
        }
      });
      return s.promise = s.promise.then(( {
        body: o,
        statusCode: a
      }) = > {
        if(200! =  = a) {
          if(i>5)return Promise.reject(new Error("歌词获取失败"));
          let o = this.getLyric(e,
          t,
          n,
          ++i);
          return s.cancelHttp = o.cancelHttp.bind(o),
          o.promise
        }switch(o.fmt) {
          case"krc": return(0,
          z.P)(o.content);
          case"lrc": return {
            lyric: Buffer.from(o.content,
            "base64").toString("utf-8"),
            tlyric: "",
            rlyric: "",
            lxlyric: ""
          };
          default: return Promise.reject(new Error(`未知歌词格式: ${o.fmt}`))
        }
      }),
      s
    },
    getLyric(e,
    t = 0) {
      let n = this.searchLyric(e.name,
      e.hash,
      e._interval||this.getIntv(e.interval));
      return n.promise = n.promise.then(e = > {
        if(!e)return Promise.reject(new Error("Get lyric failed"));
        let t = this.getLyricDownload(e.id,
        e.accessKey,
        e.fmt);
        return n.cancelHttp = t.cancelHttp.bind(t),
        t.promise
      }),
      n
    }
  },
  q =  {
    leaderboard: T,
    songList: D,
    musicSearch: P,
    hotSearch:  {
      _requestObj: null,
      async getList(e = 0) {
        if(this._requestObj&&this._requestObj.cancelHttp(),
        e>2)return Promise.reject(new Error("try max num"));
        const t = (0,
        o.tX)("http://gateway.kugou.com/api/v3/search/hot_tab?signature=ee44edb9d7155821412d220bcaf509dd&appid=1005&clientver=10026&plat=0",
         {
          method: "get",
          headers:  {
            dfid: "1ssiv93oVqMp27cirf2CvoF1",
            mid: "156798703528610303473757548878786007104",
            clienttime: 1584257267,
            "x-router": "msearch.kugou.com",
            "user-agent": "Android9-AndroidPhone-10020-130-0-searchrecommendprotocol-wifi",
            "kg-rc": 1
          }
        }),
         {
          body: n,
          statusCode: i
        } = await t.promise;
        if(200! = i||0! =  = n.errcode)throw new Error("获取热搜词失败");
        return {
          source: "kg",
          list: this.filterList(n.data.list)
        }
      },
      filterList(e) {
        const t = [];
        return e.forEach(e = > {
          e.keywords.map(e = >t.push((0,
          s.e4)(e.keyword)))
        }),
        t
      }
    },
    comment:  {
      _requestObj: null,
      _requestObj2: null,
      async getComment( {
        hash: e
      },
      t = 1,
      n = 20) {
        this._requestObj&&this._requestObj.cancelHttp();
        const i = `dfid=0&mid=16249512204336365674023395779019&clienttime=${Date.now()}&uuid=0&extdata=${e}&appid=1005&code=fc4be23b4e972707f36b8a828a93ba8a&schash=${e}&clientver=11409&p=${t}&clienttoken=&pagesize=${n}&ver=10&kugouid=0`,
        s = (0,
        o.tX)(`http://m.comment.service.kugou.com/r/v1/rank/newest?${i}&signature=${$(i)}`,
         {
          headers:  {
            "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36 Edg/107.0.1418.24"
          }
        }),
         {
          body: a,
          statusCode: l
        } = await s.promise;
        if(200! = l||0! =  = a.err_code)throw new Error("获取评论失败");
        const r = a.count??0;
        return {
          source: "kg",
          comments: this.filterComment(a.list||[]),
          total: r,
          page: t,
          limit: n,
          maxPage: Math.ceil(r/n)||1
        }
      },
      async getHotComment( {
        hash: e
      },
      t = 1,
      n = 20) {
        this._requestObj2&&this._requestObj2.cancelHttp();
        const i = `dfid=0&mid=16249512204336365674023395779019&clienttime=${Date.now()}&uuid=0&extdata=${e}&appid=1005&code=fc4be23b4e972707f36b8a828a93ba8a&schash=${e}&clientver=11409&p=${t}&clienttoken=&pagesize=${n}&ver=10&kugouid=0`,
        s = (0,
        o.tX)(`http://m.comment.service.kugou.com/r/v1/rank/topliked?${i}&signature=${$(i)}`,
         {
          headers:  {
            "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36 Edg/107.0.1418.24"
          }
        }),
         {
          body: a,
          statusCode: l
        } = await s.promise;
        if(200! = l||0! =  = a.err_code)throw new Error("获取热门评论失败");
        const r = a.count??0;
        return {
          source: "kg",
          comments: this.filterComment(a.list||[]),
          total: r,
          page: t,
          limit: n,
          maxPage: Math.ceil(r/n)||1
        }
      },
      async getReplyComment( {
        songmid: e,
        audioId: t
      },
      n,
      i = 1,
      s = 100) {
        this._requestObj2&&this._requestObj2.cancelHttp(),
        e = 32 =  = e.length?t.split("_")[0]: e;
        const a = (0,
        o.tX)(`http://comment.service.kugou.com/index.php?r=commentsv2/getReplyWithLike&code=fc4be23b4e972707f36b8a828a93ba8a&p=${i}&pagesize=${s}&ver=1.01&clientver=8373&kugouid=687373022&need_show_image=1&appid=1001&childrenid=${e}&tid=${n}`,
         {
          headers:  {
            "User-Agent": "Android712-AndroidPhone-8983-18-0-COMMENT-wifi"
          }
        }),
         {
          body: l,
          statusCode: r
        } = await a.promise;
        if(200! = r||0! =  = l.err_code)throw new Error("获取回复评论失败");
        return {
          source: "kg",
          comments: this.filterComment(l.list||[])
        }
      },
      replaceAt: (e,
      t) = >(t.forEach(t = > {
        e = e.replaceAll(`[at=${t.id}]`,
        `@${t.name} `)
      }),
      e),
      filterComment(e) {
        return e.map(e = > {
          let t =  {
            id: e.id,
            text: (0,
            s.e4)((e.atlist?this.replaceAt(e.content,
            e.atlist): e.content)||""),
            images: e.images?e.images.map(e = >e.url): [],
            location: e.location,
            time: e.addtime,
            timeStr: (0,
            s.Uj)(new Date(e.addtime).getTime()),
            userName: e.user_name,
            avatar: e.user_pic,
            userId: e.user_id,
            likedCount: e.like.likenum,
            replyNum: e.reply_num,
            reply: []
          };
          return e.pcontent? {
            id: e.id,
            text: (0,
            s.e4)(e.pcontent),
            time: null,
            userName: e.puser,
            avatar: null,
            userId: e.puser_id,
            likedCount: null,
            replyNum: null,
            reply: [t]
          }: t
        })
      }
    },
    getMusicUrl: (e,
    t) = >(0,
    C.O)("kg").getMusicUrl(e,
    t),
    getLyric: e = >O.getLyric(e),
    getPic: e = >M.getPic(e),
    getMusicDetailPageUrl: e = >`https://www.kugou.com/song/#hash=${e.hash}&album_id=${e.albumId}`
  };
  let R = [ {
    id: "tx__4",
    name: "流行指数榜",
    bangid: "4"
  },
   {
    id: "tx__26",
    name: "热歌榜",
    bangid: "26"
  },
   {
    id: "tx__27",
    name: "新歌榜",
    bangid: "27"
  },
   {
    id: "tx__62",
    name: "飙升榜",
    bangid: "62"
  },
   {
    id: "tx__58",
    name: "说唱榜",
    bangid: "58"
  },
   {
    id: "tx__57",
    name: "喜力电音榜",
    bangid: "57"
  },
   {
    id: "tx__28",
    name: "网络歌曲榜",
    bangid: "28"
  },
   {
    id: "tx__5",
    name: "内地榜",
    bangid: "5"
  },
   {
    id: "tx__3",
    name: "欧美榜",
    bangid: "3"
  },
   {
    id: "tx__59",
    name: "香港地区榜",
    bangid: "59"
  },
   {
    id: "tx__16",
    name: "韩国榜",
    bangid: "16"
  },
   {
    id: "tx__60",
    name: "抖快榜",
    bangid: "60"
  },
   {
    id: "tx__29",
    name: "影视金曲榜",
    bangid: "29"
  },
   {
    id: "tx__17",
    name: "日本榜",
    bangid: "17"
  },
   {
    id: "tx__52",
    name: "腾讯音乐人原创榜",
    bangid: "52"
  },
   {
    id: "tx__36",
    name: "K歌金曲榜",
    bangid: "36"
  },
   {
    id: "tx__61",
    name: "台湾地区榜",
    bangid: "61"
  },
   {
    id: "tx__63",
    name: "DJ舞曲榜",
    bangid: "63"
  },
   {
    id: "tx__64",
    name: "综艺新歌榜",
    bangid: "64"
  },
   {
    id: "tx__65",
    name: "国风热歌榜",
    bangid: "65"
  },
   {
    id: "tx__67",
    name: "听歌识曲榜",
    bangid: "67"
  },
   {
    id: "tx__72",
    name: "动漫音乐榜",
    bangid: "72"
  },
   {
    id: "tx__73",
    name: "游戏音乐榜",
    bangid: "73"
  },
   {
    id: "tx__75",
    name: "有声榜",
    bangid: "75"
  },
   {
    id: "tx__131",
    name: "校园音乐人排行榜",
    bangid: "131"
  }];
  const U =  {
    limit: 300,
    list: [ {
      id: "txlxzsb",
      name: "流行榜",
      bangid: 4
    },
     {
      id: "txrgb",
      name: "热歌榜",
      bangid: 26
    },
     {
      id: "txwlhgb",
      name: "网络榜",
      bangid: 28
    },
     {
      id: "txdyb",
      name: "抖音榜",
      bangid: 60
    },
     {
      id: "txndb",
      name: "内地榜",
      bangid: 5
    },
     {
      id: "txxgb",
      name: "香港榜",
      bangid: 59
    },
     {
      id: "txtwb",
      name: "台湾榜",
      bangid: 61
    },
     {
      id: "txoumb",
      name: "欧美榜",
      bangid: 3
    },
     {
      id: "txhgb",
      name: "韩国榜",
      bangid: 16
    },
     {
      id: "txrbb",
      name: "日本榜",
      bangid: 17
    },
     {
      id: "txtybb",
      name: "YouTube榜",
      bangid: 128
    }],
    listDetailRequest: (e,
    t,
    n) = >(0,
    o.tX)("https://u.y.qq.com/cgi-bin/musicu.fcg",
     {
      method: "post",
      headers:  {
        "User-Agent": "Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.1; WOW64; Trident/5.0)"
      },
      body:  {
        toplist:  {
          module: "musicToplist.ToplistInfoServer",
          method: "GetDetail",
          param:  {
            topid: e,
            num: n,
            period: t
          }
        },
        comm:  {
          uin: 0,
          format: "json",
          ct: 20,
          cv: 1859
        }
      }
    }).promise,
    regExps:  {
      periodList: /<i class="play_cover__btn c_tx_link js_icon_play" data-listkey=".+?" data-listname=".+?" data-tid=".+?" data-date=".+?" .+?<\/i>/g,
      period: /data-listname="(.+?)" data-tid=".*?\/(.+?)" data-date="(.+?)" .+?<\/i>/
    },
    periods:  {
      
    },
    periodUrl: "https://c.y.qq.com/node/pc/wk_v15/top.html",
    _requestBoardsObj: null,
    getBoardsData() {
      return this._requestBoardsObj&&this._requestBoardsObj.cancelHttp(),
      this._requestBoardsObj = (0,
      o.tX)("https://c.y.qq.com/v8/fcg-bin/fcg_myqq_toplist.fcg?g_tk=1928093487&inCharset=utf-8&outCharset=utf-8&notice=0&format=json&uin=0&needNewCode=1&platform=h5"),
      this._requestBoardsObj.promise
    },
    getData: e = >(0,
    o.tX)(e).promise,
    filterData: e = >e.map(e = > {
      let t = [],
      n =  {
        
      };
      if(0! =  = e.file.size_128mp3) {
        let o = (0,
        s.Uk)(e.file.size_128mp3);
        t.push( {
          type: "128k",
          size: o
        }),
        n["128k"] =  {
          size: o
        }
      }if(0! =  = e.file.size_320mp3) {
        let o = (0,
        s.Uk)(e.file.size_320mp3);
        t.push( {
          type: "320k",
          size: o
        }),
        n["320k"] =  {
          size: o
        }
      }if(0! =  = e.file.size_flac) {
        let o = (0,
        s.Uk)(e.file.size_flac);
        t.push( {
          type: "flac",
          size: o
        }),
        n.flac =  {
          size: o
        }
      }if(0! =  = e.file.size_hires) {
        let o = (0,
        s.Uk)(e.file.size_hires);
        t.push( {
          type: "flac24bit",
          size: o
        }),
        n.flac24bit =  {
          size: o
        }
      }return {
        singer: c(e.singer,
        "name"),
        name: e.title,
        albumName: e.album.name,
        albumId: e.album.mid,
        source: "tx",
        interval: (0,
        s.Zg)(e.interval),
        songId: e.id,
        albumMid: e.album.mid,
        strMediaMid: e.file.media_mid,
        songmid: e.mid,
        img: "" =  =  = e.album.name||"空" =  =  = e.album.name?e.singer?.length?`https://y.gtimg.cn/music/photo_new/T001R500x500M000${e.singer[0].mid}.jpg`: "": `https://y.gtimg.cn/music/photo_new/T002R500x500M000${e.album.mid}.jpg`,
        lrc: null,
        otherSource: null,
        types: t,
        _types: n,
        typeUrl:  {
          
        }
      }
    }),
    getPeriods(e) {
      return this.getData(this.periodUrl).then(( {
        body: t
      }) = > {
        let n = t.match(this.regExps.periodList);
        if(!n)return Promise.reject(new Error("get data failed"));
        n.forEach(e = > {
          let t = e.match(this.regExps.period);
          t&&(this.periods[t[2]] =  {
            name: t[1],
            bangid: t[2],
            period: t[3]
          })
        });
        const o = this.periods[e];
        return o&&o.period
      })
    },
    filterBoardsData(e) {
      let t = [];
      for(const n of e)201! = n.id&&(n.topTitle.startsWith("巅峰榜·")&&(n.topTitle = n.topTitle.substring(4,
      n.topTitle.length)),
      n.topTitle.endsWith("榜")||(n.topTitle+ = "榜"),
      t.push( {
        id: "tx__"+n.id,
        name: n.topTitle,
        bangid: String(n.id)
      }));
      return t
    },
    async getBoards(e = 0) {
      return this.list = R,
       {
        list: R,
        source: "tx"
      }
    },
    getList(e,
    t,
    n = 0) {
      if(++n>3)return Promise.reject(new Error("try max num"));
      e = parseInt(e);
      let o = this.periods[e];
      return(o?Promise.resolve(o.period): this.getPeriods(e)).then(o = >this.listDetailRequest(e,
      o,
      this.limit).then(o = >0! =  = o.body.code?this.getList(e,
      t,
      n):  {
        total: o.body.toplist.data.songInfoList.length,
        list: this.filterData(o.body.toplist.data.songInfoList),
        limit: this.limit,
        page: 1,
        source: "tx"
      }))
    },
    getDetailPageUrl: e = >("string" =  = typeof e&&(e = e.replace("tx__",
    "")),
    `https://y.qq.com/n/ryqq/toplist/${e}`)
  },
  j = e = > {
    let t = [];
    return e.forEach(e = > {
      t.push(e.name)
    }),
    t.join("、")
  },
  F = e = >(0,
  o.tX)("https://u.y.qq.com/cgi-bin/musicu.fcg",
   {
    method: "post",
    headers:  {
      "User-Agent": "Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.1; WOW64; Trident/5.0)"
    },
    body:  {
      comm:  {
        ct: "19",
        cv: "1859",
        uin: "0"
      },
      req:  {
        module: "music.pf_song_detail_svr",
        method: "get_song_detail_yqq",
        param:  {
          song_type: 0,
          song_mid: e
        }
      }
    }
  }).promise.then(( {
    body: e
  }) = > {
    if(0! = e.code||0! = e.req.code)return Promise.reject(new Error("获取歌曲信息失败"));
    const t = e.req.data.track_info;
    if(!t.file?.media_mid)return null;
    let n = [],
    o =  {
      
    };
    const i = t.file;
    if(0! = i.size_128mp3) {
      let e = (0,
      s.Uk)(i.size_128mp3);
      n.push( {
        type: "128k",
        size: e
      }),
      o["128k"] =  {
        size: e
      }
    }if(0! =  = i.size_320mp3) {
      let e = (0,
      s.Uk)(i.size_320mp3);
      n.push( {
        type: "320k",
        size: e
      }),
      o["320k"] =  {
        size: e
      }
    }if(0! =  = i.size_flac) {
      let e = (0,
      s.Uk)(i.size_flac);
      n.push( {
        type: "flac",
        size: e
      }),
      o.flac =  {
        size: e
      }
    }if(0! =  = i.size_hires) {
      let e = (0,
      s.Uk)(i.size_hires);
      n.push( {
        type: "flac24bit",
        size: e
      }),
      o.flac24bit =  {
        size: e
      }
    }let a = "",
    l = "";
    return t.album&&(l = t.album.name,
    a = t.album.mid),
     {
      singer: j(t.singer),
      name: t.title,
      albumName: l,
      albumId: a,
      source: "tx",
      interval: (0,
      s.Zg)(t.interval),
      songId: t.id,
      albumMid: t.album?.mid??"",
      strMediaMid: t.file.media_mid,
      songmid: t.mid,
      img: "" =  =  = a||"空" =  =  = a?t.singer?.length?`https://y.gtimg.cn/music/photo_new/T001R500x500M000${t.singer[0].mid}.jpg`: "": `https://y.gtimg.cn/music/photo_new/T002R500x500M000${a}.jpg`,
      types: n,
      _types: o,
      typeUrl:  {
        
      }
    }
  });
  (Object.getOwnPropertyDescriptor(F,
  "name")|| {
    
  }).writable||Object.defineProperty(F,
  "name",
   {
    value: "default",
    configurable: !0
  });
  const H = [[14,
  4,
  13,
  1,
  2,
  15,
  11,
  8,
  3,
  10,
  6,
  12,
  5,
  9,
  0,
  7,
  0,
  15,
  7,
  4,
  14,
  2,
  13,
  1,
  10,
  6,
  12,
  11,
  9,
  5,
  3,
  8,
  4,
  1,
  14,
  8,
  13,
  6,
  2,
  11,
  15,
  12,
  9,
  7,
  3,
  10,
  5,
  0,
  15,
  12,
  8,
  2,
  4,
  9,
  1,
  7,
  5,
  11,
  3,
  14,
  10,
  0,
  6,
  13],
  [15,
  1,
  8,
  14,
  6,
  11,
  3,
  4,
  9,
  7,
  2,
  13,
  12,
  0,
  5,
  10,
  3,
  13,
  4,
  7,
  15,
  2,
  8,
  15,
  12,
  0,
  1,
  10,
  6,
  9,
  11,
  5,
  0,
  14,
  7,
  11,
  10,
  4,
  13,
  1,
  5,
  8,
  12,
  6,
  9,
  3,
  2,
  15,
  13,
  8,
  10,
  1,
  3,
  15,
  4,
  2,
  11,
  6,
  7,
  12,
  0,
  5,
  14,
  9],
  [10,
  0,
  9,
  14,
  6,
  3,
  15,
  5,
  1,
  13,
  12,
  7,
  11,
  4,
  2,
  8,
  13,
  7,
  0,
  9,
  3,
  4,
  6,
  10,
  2,
  8,
  5,
  14,
  12,
  11,
  15,
  1,
  13,
  6,
  4,
  9,
  8,
  15,
  3,
  0,
  11,
  1,
  2,
  12,
  5,
  10,
  14,
  7,
  1,
  10,
  13,
  0,
  6,
  9,
  8,
  7,
  4,
  15,
  14,
  3,
  11,
  5,
  2,
  12],
  [7,
  13,
  14,
  3,
  0,
  6,
  9,
  10,
  1,
  2,
  8,
  5,
  11,
  12,
  4,
  15,
  13,
  8,
  11,
  5,
  6,
  15,
  0,
  3,
  4,
  7,
  2,
  12,
  1,
  10,
  14,
  9,
  10,
  6,
  9,
  0,
  12,
  11,
  7,
  13,
  15,
  1,
  3,
  14,
  5,
  2,
  8,
  4,
  3,
  15,
  0,
  6,
  10,
  10,
  13,
  8,
  9,
  4,
  5,
  11,
  12,
  7,
  2,
  14],
  [2,
  12,
  4,
  1,
  7,
  10,
  11,
  6,
  8,
  5,
  3,
  15,
  13,
  0,
  14,
  9,
  14,
  11,
  2,
  12,
  4,
  7,
  13,
  1,
  5,
  0,
  15,
  10,
  3,
  9,
  8,
  6,
  4,
  2,
  1,
  11,
  10,
  13,
  7,
  8,
  15,
  9,
  12,
  5,
  6,
  3,
  0,
  14,
  11,
  8,
  12,
  7,
  1,
  14,
  2,
  13,
  6,
  15,
  0,
  9,
  10,
  4,
  5,
  3],
  [12,
  1,
  10,
  15,
  9,
  2,
  6,
  8,
  0,
  13,
  3,
  4,
  14,
  7,
  5,
  11,
  10,
  15,
  4,
  2,
  7,
  12,
  9,
  5,
  6,
  1,
  13,
  14,
  0,
  11,
  3,
  8,
  9,
  14,
  15,
  5,
  2,
  8,
  12,
  3,
  7,
  0,
  4,
  10,
  1,
  13,
  11,
  6,
  4,
  3,
  2,
  12,
  9,
  5,
  15,
  10,
  11,
  14,
  1,
  7,
  6,
  0,
  8,
  13],
  [4,
  11,
  2,
  14,
  15,
  0,
  8,
  13,
  3,
  12,
  9,
  7,
  5,
  10,
  6,
  1,
  13,
  0,
  11,
  7,
  4,
  9,
  1,
  10,
  14,
  3,
  5,
  12,
  2,
  15,
  8,
  6,
  1,
  4,
  11,
  13,
  12,
  3,
  7,
  14,
  10,
  15,
  6,
  8,
  0,
  5,
  9,
  2,
  6,
  11,
  13,
  8,
  1,
  4,
  10,
  7,
  9,
  5,
  0,
  15,
  14,
  2,
  3,
  12],
  [13,
  2,
  8,
  4,
  6,
  15,
  11,
  1,
  10,
  9,
  3,
  14,
  5,
  0,
  12,
  7,
  1,
  15,
  13,
  8,
  10,
  3,
  7,
  4,
  12,
  5,
  6,
  11,
  0,
  14,
  9,
  2,
  7,
  11,
  4,
  1,
  9,
  12,
  14,
  2,
  0,
  6,
  10,
  13,
  15,
  3,
  5,
  8,
  2,
  1,
  14,
  7,
  4,
  10,
  8,
  13,
  15,
  12,
  9,
  0,
  3,
  5,
  6,
  11]],
  K = [1,
  1,
  2,
  2,
  2,
  2,
  2,
  2,
  1,
  2,
  2,
  2,
  2,
  2,
  2,
  1],
  G = [56,
  48,
  40,
  32,
  24,
  16,
  8,
  0,
  57,
  49,
  41,
  33,
  25,
  17,
  9,
  1,
  58,
  50,
  42,
  34,
  26,
  18,
  10,
  2,
  59,
  51,
  43,
  35],
  W = [62,
  54,
  46,
  38,
  30,
  22,
  14,
  6,
  61,
  53,
  45,
  37,
  29,
  21,
  13,
  5,
  60,
  52,
  44,
  36,
  28,
  20,
  12,
  4,
  27,
  19,
  11,
  3],
  Y = [13,
  16,
  10,
  23,
  0,
  4,
  2,
  27,
  14,
  5,
  20,
  9,
  22,
  18,
  11,
  3,
  25,
  7,
  15,
  6,
  26,
  19,
  12,
  1,
  40,
  51,
  30,
  36,
  46,
  54,
  29,
  39,
  50,
  44,
  32,
  47,
  43,
  48,
  38,
  55,
  33,
  52,
  45,
  41,
  49,
  35,
  28,
  31],
  X = (e,
  t,
  n) = >(e[4*(t/32|0)+3-(t%32/8|0)]>>>7-t%8&1)<<n>>>0,
  Z = (e,
  t,
  n) = >(e>>>31-t&1)<<n>>>0,
  Q = (e,
  t,
  n) = >(e<<t>>>0&2147483648)>>>0>>>n>>>0,
  J = e = >32&e|(31&e)>>>1|(1&e)<<4,
  ee = (e,
  t) = > {
    const n = (Q(e,
    31,
    0)|(4026531840&e)>>>0>>>1|Q(e,
    4,
    5)|Q(e,
    3,
    6)|(251658240&e)>>>0>>>3|Q(e,
    8,
    11)|Q(e,
    7,
    12)|(15728640&e)>>>0>>>5|Q(e,
    12,
    17)|Q(e,
    11,
    18)|(983040&e)>>>0>>>7|Q(e,
    16,
    23))>>>0,
    o = (Q(e,
    15,
    0)|(61440&e)<<15>>>0|Q(e,
    20,
    5)|Q(e,
    19,
    6)|(3840&e)<<13>>>0|Q(e,
    24,
    11)|Q(e,
    23,
    12)|(240&e)<<11>>>0|Q(e,
    28,
    17)|Q(e,
    27,
    18)|(15&e)<<9>>>0|Q(e,
    0,
    23))>>>0,
    i = [n>>>24&255,
    n>>>16&255,
    n>>>8&255,
    o>>>24&255,
    o>>>16&255,
    o>>>8&255];
    for(let e = 0;
    e<6;
    e++)i[e]^ = t[e];
    let s = (H[0][J(i[0]>>>2)]<<28|H[1][J((3&i[0])<<4|i[1]>>>4)]<<24|H[2][J((15&i[1])<<2|i[2]>>>6)]<<20|H[3][J(63&i[2])]<<16|H[4][J(i[3]>>>2)]<<12|H[5][J((3&i[3])<<4|i[4]>>>4)]<<8|H[6][J((15&i[4])<<2|i[5]>>>6)]<<4|H[7][J(63&i[5])])>>>0;
    return(Q(s,
    15,
    0)|Q(s,
    6,
    1)|Q(s,
    19,
    2)|Q(s,
    20,
    3)|Q(s,
    28,
    4)|Q(s,
    11,
    5)|Q(s,
    27,
    6)|Q(s,
    16,
    7)|Q(s,
    0,
    8)|Q(s,
    14,
    9)|Q(s,
    22,
    10)|Q(s,
    25,
    11)|Q(s,
    4,
    12)|Q(s,
    17,
    13)|Q(s,
    30,
    14)|Q(s,
    9,
    15)|Q(s,
    1,
    16)|Q(s,
    7,
    17)|Q(s,
    23,
    18)|Q(s,
    13,
    19)|Q(s,
    31,
    20)|Q(s,
    26,
    21)|Q(s,
    2,
    22)|Q(s,
    8,
    23)|Q(s,
    18,
    24)|Q(s,
    12,
    25)|Q(s,
    29,
    26)|Q(s,
    5,
    27)|Q(s,
    21,
    28)|Q(s,
    10,
    29)|Q(s,
    3,
    30)|Q(s,
    24,
    31))>>>0
  },
  te = (e,
  t,
  n) = > {
    let[o,
    i] = (e = >[(X(e,
    57,
    31)|X(e,
    49,
    30)|X(e,
    41,
    29)|X(e,
    33,
    28)|X(e,
    25,
    27)|X(e,
    17,
    26)|X(e,
    9,
    25)|X(e,
    1,
    24)|X(e,
    59,
    23)|X(e,
    51,
    22)|X(e,
    43,
    21)|X(e,
    35,
    20)|X(e,
    27,
    19)|X(e,
    19,
    18)|X(e,
    11,
    17)|X(e,
    3,
    16)|X(e,
    61,
    15)|X(e,
    53,
    14)|X(e,
    45,
    13)|X(e,
    37,
    12)|X(e,
    29,
    11)|X(e,
    21,
    10)|X(e,
    13,
    9)|X(e,
    5,
    8)|X(e,
    63,
    7)|X(e,
    55,
    6)|X(e,
    47,
    5)|X(e,
    39,
    4)|X(e,
    31,
    3)|X(e,
    23,
    2)|X(e,
    15,
    1)|X(e,
    7,
    0))>>>0,
    (X(e,
    56,
    31)|X(e,
    48,
    30)|X(e,
    40,
    29)|X(e,
    32,
    28)|X(e,
    24,
    27)|X(e,
    16,
    26)|X(e,
    8,
    25)|X(e,
    0,
    24)|X(e,
    58,
    23)|X(e,
    50,
    22)|X(e,
    42,
    21)|X(e,
    34,
    20)|X(e,
    26,
    19)|X(e,
    18,
    18)|X(e,
    10,
    17)|X(e,
    2,
    16)|X(e,
    60,
    15)|X(e,
    52,
    14)|X(e,
    44,
    13)|X(e,
    36,
    12)|X(e,
    28,
    11)|X(e,
    20,
    10)|X(e,
    12,
    9)|X(e,
    4,
    8)|X(e,
    62,
    7)|X(e,
    54,
    6)|X(e,
    46,
    5)|X(e,
    38,
    4)|X(e,
    30,
    3)|X(e,
    22,
    2)|X(e,
    14,
    1)|X(e,
    6,
    0))>>>0])(e);
    for(let e = 0;
    e<15;
    e++) {
      const n = i;
      i = (ee(i,
      t[e])^o)>>>0,
      o = n
    }o = (ee(i,
    t[15])^o)>>>0,
    ((e,
    t,
    n) = > {
      n[3] = 255&(Z(t,
      7,
      7)|Z(e,
      7,
      6)|Z(t,
      15,
      5)|Z(e,
      15,
      4)|Z(t,
      23,
      3)|Z(e,
      23,
      2)|Z(t,
      31,
      1)|Z(e,
      31,
      0)),
      n[2] = 255&(Z(t,
      6,
      7)|Z(e,
      6,
      6)|Z(t,
      14,
      5)|Z(e,
      14,
      4)|Z(t,
      22,
      3)|Z(e,
      22,
      2)|Z(t,
      30,
      1)|Z(e,
      30,
      0)),
      n[1] = 255&(Z(t,
      5,
      7)|Z(e,
      5,
      6)|Z(t,
      13,
      5)|Z(e,
      13,
      4)|Z(t,
      21,
      3)|Z(e,
      21,
      2)|Z(t,
      29,
      1)|Z(e,
      29,
      0)),
      n[0] = 255&(Z(t,
      4,
      7)|Z(e,
      4,
      6)|Z(t,
      12,
      5)|Z(e,
      12,
      4)|Z(t,
      20,
      3)|Z(e,
      20,
      2)|Z(t,
      28,
      1)|Z(e,
      28,
      0)),
      n[7] = 255&(Z(t,
      3,
      7)|Z(e,
      3,
      6)|Z(t,
      11,
      5)|Z(e,
      11,
      4)|Z(t,
      19,
      3)|Z(e,
      19,
      2)|Z(t,
      27,
      1)|Z(e,
      27,
      0)),
      n[6] = 255&(Z(t,
      2,
      7)|Z(e,
      2,
      6)|Z(t,
      10,
      5)|Z(e,
      10,
      4)|Z(t,
      18,
      3)|Z(e,
      18,
      2)|Z(t,
      26,
      1)|Z(e,
      26,
      0)),
      n[5] = 255&(Z(t,
      1,
      7)|Z(e,
      1,
      6)|Z(t,
      9,
      5)|Z(e,
      9,
      4)|Z(t,
      17,
      3)|Z(e,
      17,
      2)|Z(t,
      25,
      1)|Z(e,
      25,
      0)),
      n[4] = 255&(Z(t,
      0,
      7)|Z(e,
      0,
      6)|Z(t,
      8,
      5)|Z(e,
      8,
      4)|Z(t,
      16,
      3)|Z(e,
      16,
      2)|Z(t,
      24,
      1)|Z(e,
      24,
      0))
    })(o,
    i,
    n)
  },
  ne = (e,
  t) = > {
    const n = Array.from( {
      length: 16
    },
    () = >new Uint8Array(6));
    let o = 0,
    i = 0;
    for(let t = 0;
    t<28;
    t++)o = (o|X(e,
    G[t],
    31-t))>>>0,
    i = (i|X(e,
    W[t],
    31-t))>>>0;
    for(let e = 0;
    e<16;
    e++) {
      o = (4294967280&(o<<K[e]>>>0|o>>>28-K[e]))>>>0,
      i = (4294967280&(i<<K[e]>>>0|i>>>28-K[e]))>>>0;
      const s = 0 =  =  = t?15-e: e;
      for(let e = 0;
      e<24;
      e++)n[s][e/8|0]| = Z(o,
      Y[e],
      7-e%8);
      for(let e = 24;
      e<48;
      e++)n[s][e/8|0]| = Z(i,
      Y[e]-27,
      7-e%8)
    }return n
  },
  oe = (e,
  t,
  n) = > {
    const o = new Uint8Array(8);
    te(e,
    t[0],
    o),
    te(o,
    t[1],
    n),
    te(n,
    t[2],
    o),
    n.set(o)
  },
  ie = Buffer.from([33,
  64,
  35,
  41,
  40,
  42,
  36,
  37,
  49,
  50,
  51,
  90,
  88,
  67,
  33,
  64,
  33,
  64,
  35,
  41,
  40,
  78,
  72,
  76]),
  se = async e = > {
    if(!e||e.length%2! = 0)return"";
    const t = Buffer.from(e,
    "hex");
    if(0 =  =  = t.length)return"";
    const n = (o = ie,
    1 =  =  = 0?[ne(o.subarray(0,
    8),
    1),
    ne(o.subarray(8,
    16),
    0),
    ne(o.subarray(16,
    24),
    1)]: [ne(o.subarray(16,
    24),
    0),
    ne(o.subarray(8,
    16),
    1),
    ne(o.subarray(0,
    8),
    0)]);
    var o;
    const i = new Uint8Array(8);
    for(let e = 0;
    e+8< = t.length;
    e+ = 8)oe(t.subarray(e,
    e+8),
    n,
    i),
    t.set(i,
    e);
    try {
      return(await(s = t,
      new Promise((e,
      t) = > {
        (0,
        d.inflate)(s,
         {
          finishFlush: d.constants.Z_SYNC_FLUSH
        },
        (n,
        o) = > {
          n?t(n): e(o)
        })
      }))).toString("utf8")
    }catch {
      return""
    }var s
  },
  ae = new Map,
  le = new Map,
  re =  {
    rxps:  {
      info: /^{"/,
      lineTime: /^\[(\d+),\d+\]/,
      lineTime2: /^\[([\d:.]+)\]/,
      wordTime: /\(\d+,\d+\)/,
      wordTimeAll: /(\(\d+,\d+\))/g,
      timeLabelFixRxp: /(?:\.0+|0+)$/
    },
    msFormat(e) {
      if(Number.isNaN(e))return"";
      let t = (e%1e3).toString().padStart(3,
      "0");
      e/ = 1e3;
      let n = parseInt(e/60).toString().padStart(2,
      "0");
      return e% = 60,
      `[${n}:${parseInt(e).toString().padStart(2,"0")}.${t}]`
    },
    parseLyric(e) {
      if(!(e = (e = e.trim()).replace(/\r/g,
      "")))return {
        lyric: "",
        lxlyric: ""
      };
      const t = e.split("\n"),
      n = [],
      o = [];
      for(let e of t) {
        e = e.trim();
        let t = this.rxps.lineTime.exec(e);
        if(!t) {
          e.startsWith("[offset")&&(n.push(e),
          o.push(e)),
          this.rxps.lineTime2.test(e)&&o.push(e);
          continue
        }const i = parseInt(t[1]),
        s = this.msFormat(i);
        if(!s)continue;
        let a = e.replace(this.rxps.lineTime,
        "");
        o.push(`${s}${a.replace(this.rxps.wordTimeAll,"")}`);
        let l = a.match(this.rxps.wordTimeAll);
        if(!l)continue;
        l = l.map(e = > {
          const t = /\((\d+),(\d+)\)/.exec(e);
          return`<${Math.trunc(Math.max(parseInt(t[1])-i,0))},${t[2]}>`
        });
        const r = a.split(this.rxps.wordTime),
        c = l.map((e,
        t) = >`${e}${r[t]}`).join("");
        n.push(`${s}${c}`)
      }return {
        lyric: o.join("\n"),
        lxlyric: n.join("\n")
      }
    },
    parseRlyric(e) {
      if(!(e = (e = e.trim()).replace(/\r/g,
      "")))return {
        lyric: "",
        lxlyric: ""
      };
      const t = e.split("\n"),
      n = [];
      for(let e of t) {
        e = e.trim();
        let t = this.rxps.lineTime.exec(e);
        if(!t)continue;
        const o = parseInt(t[1]),
        i = this.msFormat(o);
        if(!i)continue;
        let s = e.replace(this.rxps.lineTime,
        "");
        n.push(`${i}${s.replace(this.rxps.wordTimeAll,"")}`)
      }return n.join("\n")
    },
    removeTag: e = >e.replace(/^[\S\s]*?LyricContent="/,
    "").replace(/"\/>[\S\s]*?$/,
    ""),
    getIntv(e) {
      if(!e)return 0;
      e.includes(".")||(e+ = ".0");
      let t = e.split(/:|\./);
      for(;
      t.length<3;
      )t.unshift("0");
      const[n,
      o,
      i] = t;
      return 36e5*parseInt(n)+1e3*parseInt(o)+parseInt(i)
    },
    fixRlrcTimeTag(e,
    t) {
      const n = e.split("\n");
      let o = t.split("\n"),
      i = [];
      return n.forEach(e = > {
        const t = this.rxps.lineTime2.exec(e);
        if(!t)return;
        if(!e.replace(this.rxps.lineTime2,
        "").trim())return;
        const n = this.getIntv(t[1]);
        for(;
        o.length;
        ) {
          const t = o.shift(),
          s = this.rxps.lineTime2.exec(t);
          if(!s)continue;
          const a = this.getIntv(s[1]);
          if(Math.abs(n-a)<100) {
            i.push(e.replace(this.rxps.lineTime2,
            s[0]));
            break
          }
        }
      }),
      i.join("\n")
    },
    fixTlrcTimeTag(e,
    t) {
      const n = e.split("\n");
      let o = t.split("\n"),
      i = [];
      return n.forEach(e = > {
        const t = this.rxps.lineTime2.exec(e);
        if(!t)return;
        if(!e.replace(this.rxps.lineTime2,
        "").trim())return;
        let n = t[1];
        n.includes(".")&&(n+ = "".padStart(3-n.split(".")[1].length,
        "0"));
        const s = this.getIntv(n);
        for(;
        o.length;
        ) {
          const t = o.shift(),
          n = this.rxps.lineTime2.exec(t);
          if(!n)continue;
          const a = this.getIntv(n[1]);
          if(Math.abs(s-a)<100) {
            i.push(e.replace(this.rxps.lineTime2,
            n[0]));
            break
          }
        }
      }),
      i.join("\n")
    },
    parse(e,
    t,
    n) {
      const o =  {
        lyric: "",
        tlyric: "",
        rlyric: "",
        lxlyric: ""
      };
      if(e) {
        let {
          lyric: t,
          lxlyric: n
        } = this.parseLyric(this.removeTag(e));
        o.lyric = t,
        o.lxlyric = n
      }return n&&(o.rlyric = this.fixRlrcTimeTag(this.parseRlyric(this.removeTag(n)),
      o.lyric)),
      t&&(o.tlyric = this.fixTlrcTimeTag(t,
      o.lyric)),
      o
    }
  },
  ce =  {
    successCode: 0,
    async getSongId( {
      songId: e,
      songmid: t
    }) {
      if(e)return e;
      if(ae.has(t))return ae.get(t);
      if(le.has(t))return(await le.get(t)).songId;
      const n = F(t);
      le.set(n);
      const o = await n;
      return ae.set(t,
      o.songId),
      le.delete(t),
      o.songId
    },
    async parseLyric(e,
    t,
    n) {
      const {
        lyric: o,
        tlyric: i,
        rlyric: s
      } = await(async(e,
      t,
      n) = >( {
        lyric: await se(e),
        tlyric: await se(t),
        rlyric: await se(n)
      }))(e,
      t,
      n);
      return re.parse(o,
      i,
      s)
    },
    getLyric(e,
    t = 0) {
      return t>3?Promise.reject(new Error("Get lyric failed")):  {
        cancelHttp() {
          
        },
        promise: this.getSongId(e).then(e = >(0,
        o.tX)("https://u.y.qq.com/cgi-bin/musicu.fcg",
         {
          method: "post",
          headers:  {
            referer: "https://y.qq.com",
            "user-agent": "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36"
          },
          body:  {
            comm:  {
              ct: "19",
              cv: "1859",
              uin: "0"
            },
            req:  {
              method: "GetPlayLyricInfo",
              module: "music.musichallSong.PlayLyricInfo",
              param:  {
                format: "json",
                crypt: 1,
                ct: 19,
                cv: 1873,
                interval: 0,
                lrc_t: 0,
                qrc: 1,
                qrc_t: 0,
                roma: 1,
                roma_t: 0,
                songID: e,
                trans: 1,
                trans_t: 0,
                type: -1
              }
            }
          }
        }).promise.then(( {
          body: n
        }) = > {
          if(n.code! = this.successCode||n.req.code! = this.successCode)return this.getLyric(e,
          ++t);
          const o = n.req.data;
          return this.parseLyric(o.lyric,
          o.trans,
          o.roma)
        }))
      }
    }
  },
  de =  {
    _requestObj_tags: null,
    _requestObj_hotTags: null,
    _requestObj_list: null,
    limit_list: 36,
    limit_song: 1e5,
    successCode: 0,
    sortList: [ {
      name: "推荐",
      id: -1
    },
     {
      name: "最热",
      id: 3
    },
     {
      name: "最新",
      id: 2
    }],
    regExps:  {
      hotTagHtml: /class="c_bg_link js_tag_item" data-id="\w+">.+?<\/a>/g,
      hotTag: /data-id="(\w+)">(.+?)<\/a>/,
      listDetailLink: /\/playlist\/(\d+)/,
      listDetailLink2: /id=(\d+)/
    },
    tagsUrl: "https://u.y.qq.com/cgi-bin/musicu.fcg?loginUin=0&hostUin=0&format=json&inCharset=utf-8&outCharset=utf-8&notice=0&platform=wk_v15.json&needNewCode=0&data=%7B%22tags%22%3A%7B%22method%22%3A%22get_all_categories%22%2C%22param%22%3A%7B%22qq%22%3A%22%22%7D%2C%22module%22%3A%22playlist.PlaylistAllCategoriesServer%22%7D%7D",
    hotTagUrl: "https://c.y.qq.com/node/pc/wk_v15/category_playlist.html",
    getListUrl(e,
    t) {
      return e = parseInt(e),
      `https://u.y.qq.com/cgi-bin/musicu.fcg?loginUin=0&hostUin=0&format=json&inCharset=utf-8&outCharset=utf-8&notice=0&platform=wk_v15.json&needNewCode=0&data=${encodeURIComponent(JSON.stringify({comm:{cv:1602,ct:20},playlist:{method:"get_category_content",param:{titleid:e,caller:"0",category_id:e,size:this.limit_list,page:t-1,use_page:1},module:"playlist.PlayListCategoryServer"}}))}`
    },
    getListDetailUrl: e = >`https://c.y.qq.com/qzone/fcg-bin/fcg_ucc_getcdinfo_byids_cp.fcg?type=1&json=1&utf8=1&onlysong=0&new_format=1&disstid=${e}&loginUin=0&hostUin=0&format=json&inCharset=utf8&outCharset=utf-8&notice=0&platform=yqq.json&needNewCode=0`,
    getTag(e = 0) {
      return this._requestObj_tags&&this._requestObj_tags.cancelHttp(),
      e>2?Promise.reject(new Error("try max num")): (this._requestObj_tags = (0,
      o.tX)(this.tagsUrl),
      this._requestObj_tags.promise.then(( {
        body: t
      }) = >t.code! =  = this.successCode?this.getTag(++e): this.filterTagInfo(t.tags.data.v_group)))
    },
    getHotTag(e = 0) {
      return this._requestObj_hotTags&&this._requestObj_hotTags.cancelHttp(),
      e>2?Promise.reject(new Error("try max num")): (this._requestObj_hotTags = (0,
      o.tX)(this.hotTagUrl),
      this._requestObj_hotTags.promise.then(( {
        statusCode: t,
        body: n
      }) = >200! =  = t?this.getHotTag(++e): this.filterInfoHotTag(n)))
    },
    filterInfoHotTag(e) {
      let t = e.match(this.regExps.hotTagHtml);
      const n = [];
      return t?(t.forEach(e = > {
        let t = e.match(this.regExps.hotTag);
        t&&n.push( {
          id: parseInt(t[1]),
          name: t[2],
          source: "tx"
        })
      }),
      n): n
    },
    filterTagInfo: e = >e.map(e = >( {
      name: e.group_name,
      list: e.v_item.map(t = >( {
        parent_id: e.group_id,
        parent_name: e.group_name,
        id: t.id,
        name: t.name,
        source: "tx"
      }))
    })),
    async getRecommendList(e) {
      const {
        body: t
      } = await(0,
      o.tX)(`https://u.y.qq.com/cgi-bin/musicu.fcg?loginUin=0&hostUin=0&format=json&inCharset=utf-8&outCharset=utf-8&notice=0&platform=wk_v15.json&needNewCode=0&data=${encodeURIComponent(JSON.stringify({comm:{cv:1602,ct:20},playlist:{module:"music.playlist.PlaylistSquare",method:"GetRecommendWhole",param:{IsReqFeed:!0,FeedReq:{From:(e-1)*this.limit_list,Size:this.limit_list}}}}))}`).promise;
      if(t.code! =  = this.successCode)throw new Error("tx getRecommendList failed");
      return {
        list: t.playlist.data.FeedRsp.List?.map(( {
          Playlist: e
        }) = >( {
          play_count: (0,
          s.pN)(e.basic.play_cnt),
          id: String(e.basic.tid),
          author: (0,
          s.e4)(e.basic.creator.nick),
          name: (0,
          s.e4)(e.basic.title),
          time: e.basic.modify_time?(0,
          s.i$)(1e3*e.basic.modify_time,
          "Y-M-D"): "",
          img: e.basic.cover.medium_url||e.basic.cover.default_url,
          total: e.basic.song_cnt,
          desc: (0,
          s.e4)(e.basic.desc).replace(/<br>/g,
          "\n"),
          source: "tx"
        }))||[],
        total: t.playlist.data.FeedRsp.FromLimit,
        limit: this.limit_list,
        page: e,
        source: "tx"
      }
    },
    getList(e,
    t,
    n,
    i = 0) {
      if(this._requestObj_list&&this._requestObj_list.cancelHttp(),
      i>2)return Promise.reject(new Error("try max num"));
      if(!t) {
        if(!t&&-1 =  = e)return this.getRecommendList(n);
        t = e
      }return this._requestObj_list = (0,
      o.tX)(this.getListUrl(t,
      n)),
      this._requestObj_list.promise.then(( {
        body: o
      }) = >o.code! =  = this.successCode?this.getList(e,
      t,
      n,
      ++i): this.filterList(o.playlist.data,
      n))
    },
    filterList( {
      content: e
    },
    t) {
      return {
        list: e.v_item.map(( {
          basic: e
        }) = >( {
          play_count: (0,
          s.pN)(e.play_cnt),
          id: String(e.tid),
          author: e.creator.nick,
          name: e.title,
          img: e.cover.medium_url||e.cover.default_url,
          desc: (0,
          s.e4)(e.desc).replace(/<br>/g,
          "\n"),
          source: "tx"
        })),
        total: e.total_cnt,
        page: t,
        limit: this.limit_list,
        source: "tx"
      }
    },
    async handleParseId(e,
    t = 0) {
      if(t>2)return Promise.reject(new Error("link try max num"));
      const n = (0,
      o.tX)(e),
       {
        headers:  {
          location: i
        },
        statusCode: s
      } = await n.promise;
      return s>400?this.handleParseId(e,
      ++t): i??e
    },
    async getListId(e) {
      if(/[?&:/]/.test(e)) {
        this.regExps.listDetailLink.test(e)||(e = await this.handleParseId(e));
        let t = this.regExps.listDetailLink.exec(e);
        if(!t&&(t = this.regExps.listDetailLink2.exec(e),
        !t))throw new Error("failed");
        e = t[1]
      }return e
    },
    async getListDetail2(e,
    t = 0) {
      if(t>2)return Promise.reject(new Error("try max num"));
      const n = (0,
      o.tX)("https://u.y.qq.com/cgi-bin/musicu.fcg",
       {
        method: "post",
        headers:  {
          Origin: "https://y.qq.com",
          Referer: `https://y.qq.com/n/yqq/playsquare/${e}.html`
        },
        body:  {
          comm:  {
            cv: 4747474,
            ct: 24,
            format: "json",
            inCharset: "utf-8",
            outCharset: "utf-8",
            platform: "yqq.json",
            needNewCode: 1,
            uin: 0
          },
          req_1:  {
            module: "music.srfDissInfo.aiDissInfo",
            method: "uniform_get_Dissinfo",
            param:  {
              disstid: parseInt(e),
              userinfo: 1,
              tag: 1,
              orderlist: 1,
              song_begin: 0,
              song_num: this.limit_song,
              onlysonglist: 0,
              enc_host_uin: ""
            }
          }
        }
      }),
       {
        body: i
      } = await n.promise;
      if(i.code! =  = this.successCode)return this.getListDetail2(e,
      ++t);
      if(i.req_1.code! =  = this.successCode)throw new Error("failed");
      const a = i.req_1.data,
      l = a.dirinfo;
      return {
        list: this.filterListDetail(a.songlist),
        page: 1,
        limit: this.limit_song,
        total: a.total_song_num,
        source: "tx",
        info:  {
          name: l.title,
          img: l.picurl,
          desc: (0,
          s.e4)(l.desc??"").replace(/<br>/g,
          "\n"),
          author: l.host_nick,
          play_count: (0,
          s.pN)(l.listennum)
        }
      }
    },
    async getListDetail(e,
    t = 0) {
      if(t>2)return Promise.reject(new Error("try max num"));
      e = await this.getListId(e);
      const n = (0,
      o.tX)(this.getListDetailUrl(e),
       {
        headers:  {
          Origin: "https://y.qq.com",
          Referer: `https://y.qq.com/n/yqq/playsquare/${e}.html`
        }
      }),
       {
        body: i
      } = await n.promise;
      if(i.code! =  = this.successCode)return this.getListDetail(e,
      ++t);
      if(i.subcode! =  = this.successCode||!i.cdlist)return this.getListDetail2(e);
      const a = i.cdlist[0];
      return {
        list: this.filterListDetail(a.songlist),
        page: 1,
        limit: a.songlist.length+1,
        total: a.songlist.length,
        source: "tx",
        info:  {
          name: a.dissname,
          img: a.logo,
          desc: (0,
          s.e4)(a.desc).replace(/<br>/g,
          "\n"),
          author: a.nickname,
          play_count: (0,
          s.pN)(a.visitnum)
        }
      }
    },
    filterListDetail: e = >e.map(e = > {
      let t = [],
      n =  {
        
      };
      if(0! =  = e.file.size_128mp3) {
        let o = (0,
        s.Uk)(e.file.size_128mp3);
        t.push( {
          type: "128k",
          size: o
        }),
        n["128k"] =  {
          size: o
        }
      }if(0! =  = e.file.size_320mp3) {
        let o = (0,
        s.Uk)(e.file.size_320mp3);
        t.push( {
          type: "320k",
          size: o
        }),
        n["320k"] =  {
          size: o
        }
      }if(0! =  = e.file.size_flac) {
        let o = (0,
        s.Uk)(e.file.size_flac);
        t.push( {
          type: "flac",
          size: o
        }),
        n.flac =  {
          size: o
        }
      }if(0! =  = e.file.size_hires) {
        let o = (0,
        s.Uk)(e.file.size_hires);
        t.push( {
          type: "flac24bit",
          size: o
        }),
        n.flac24bit =  {
          size: o
        }
      }return {
        singer: c(e.singer,
        "name"),
        name: e.title,
        albumName: e.album.name,
        albumId: e.album.mid,
        source: "tx",
        interval: (0,
        s.Zg)(e.interval),
        songId: e.id,
        albumMid: e.album.mid,
        strMediaMid: e.file.media_mid,
        songmid: e.mid,
        img: "" =  =  = e.album.name||"空" =  =  = e.album.name?e.singer?.length?`https://y.gtimg.cn/music/photo_new/T001R500x500M000${e.singer[0].mid}.jpg`: "": `https://y.gtimg.cn/music/photo_new/T002R500x500M000${e.album.mid}.jpg`,
        lrc: null,
        otherSource: null,
        types: t,
        _types: n,
        typeUrl:  {
          
        }
      }
    }),
    getTags() {
      return Promise.all([this.getTag(),
      this.getHotTag()]).then(([e,
      t]) = >( {
        tags: e,
        hotTag: t,
        source: "tx"
      }))
    },
    async getDetailPageUrl(e) {
      return`https://y.qq.com/n/ryqq/playlist/${e=await this.getListId(e)}`
    },
    search(e,
    t,
    n = 20,
    i = 0) {
      if(i>5)throw new Error("max retry");
      return(0,
      o.tX)(`http://c.y.qq.com/soso/fcgi-bin/client_music_search_songlist?page_no=${t-1}&num_per_page=${n}&format=json&query=${encodeURIComponent(e)}&remoteplace=txt.yqq.playlist&inCharset=utf8&outCharset=utf-8`,
       {
        headers:  {
          "User-Agent": "Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.1; WOW64; Trident/5.0)",
          Referer: "http://y.qq.com/portal/search.html"
        }
      }).promise.then(( {
        body: o
      }) = >0! = o.code?this.search(e,
      t,
      n,
      ++i):  {
        list: o.data.list.map(e = >( {
          play_count: (0,
          s.pN)(e.listennum),
          id: String(e.dissid),
          author: (0,
          s.e4)(e.creator.name),
          name: (0,
          s.e4)(e.dissname),
          time: (0,
          s.i$)(e.createtime,
          "Y-M-D"),
          img: e.imgurl,
          total: e.song_count,
          desc: (0,
          s.e4)((0,
          s.e4)(e.introduction)).replace(/<br>/g,
          "\n"),
          source: "tx"
        })),
        limit: n,
        total: o.data.sum,
        source: "tx"
      })
    }
  };
  var ue = n(77598),
  pe = n.n(ue);
  const _e = [23,
  14,
  6,
  36,
  16,
  40,
  7,
  19],
  me = [16,
  1,
  32,
  12,
  19,
  27,
  8,
  5],
  he = [89,
  39,
  179,
  150,
  218,
  82,
  58,
  252,
  177,
  52,
  186,
  123,
  120,
  64,
  242,
  133,
  143,
  161,
  121,
  179];
  function ge(e,
  t) {
    return t.map(t = >e[t]).join("")
  }async function fe(e) {
    const t = await async function(e) {
      return pe().createHash("sha1").update(e).digest("hex")
    }(e),
    n = ge(t,
    _e),
    o = ge(t,
    me),
    i = he.map((e,
    n) = >e^parseInt(t.slice(2*n,
    2*n+2),
    16));
    var s;
    return`zzc${n}${(s=i,Buffer.from(s).toString("base64").replace(/[\\/+=]/g,"")).replace(/[\\/+=]/g,"")}${o}`.toLowerCase()
  }const ye =  {
    limit: 50,
    total: 0,
    page: 0,
    allPage: 1,
    successCode: 0,
    musicSearch(e,
    t,
    n,
    i = 0) {
      if(i>5)return Promise.reject(new Error("搜索失败"));
      return(async e = > {
        const t = await fe(JSON.stringify(e));
        return(0,
        o.tX)(`https://u.y.qq.com/cgi-bin/musics.fcg?sign=${t}`,
         {
          method: "post",
          headers:  {
            "User-Agent": "QQMusic 14090508(android 12)"
          },
          body: e
        }).promise
      })( {
        comm:  {
          _channelid: "0",
          _os_version: "6.2.9200-2",
          ct: "19",
          cv: "2151",
          guid: "1F70E520B2EAA7D25E11760783C53CA9",
          patch: "118",
          psrf_access_token_expiresAt: 0,
          psrf_qqaccess_token: "",
          psrf_qqopenid: "",
          psrf_qqunionid: "",
          tmeAppID: "qqmusic",
          tmeLoginType: 0,
          uin: "0",
          wid: "7223299733393904640"
        },
        "music.search.SearchCgiService":  {
          module: "music.search.SearchCgiService",
          method: "DoSearchForQQMusicDesktop",
          param:  {
            grp: 1,
            num_per_page: n,
            page_num: t,
            query: e,
            remoteplace: "txt.newclient.top",
            search_type: 0,
            searchid: this.getSearchId()
          }
        }
      }).then(( {
        body: o
      }) = > {
        const s = o?.["music.search.SearchCgiService"]??o?.req;
        return s&&o.code =  = this.successCode&&s.code =  = this.successCode?s.data: this.musicSearch(e,
        t,
        n,
        ++i)
      })
    },
    getSearchId() {
      let e = "";
      for(let t = 0;
      t<32;
      t++)e+ = Math.floor(16*Math.random()).toString(16);
      return e.toUpperCase()+String(Math.floor(1e5*Math.random())).padStart(5,
      "0")
    },
    handleResult(e) {
      if(!e||!Array.isArray(e))return[];
      const t = [];
      return e.forEach(e = > {
        if(!e.file?.media_mid)return;
        let n = [],
        o =  {
          
        };
        const i = e.file;
        if(0! = i.size_128mp3) {
          let e = (0,
          s.Uk)(i.size_128mp3);
          n.push( {
            type: "128k",
            size: e
          }),
          o["128k"] =  {
            size: e
          }
        }if(0! =  = i.size_320mp3) {
          let e = (0,
          s.Uk)(i.size_320mp3);
          n.push( {
            type: "320k",
            size: e
          }),
          o["320k"] =  {
            size: e
          }
        }if(0! =  = i.size_flac) {
          let e = (0,
          s.Uk)(i.size_flac);
          n.push( {
            type: "flac",
            size: e
          }),
          o.flac =  {
            size: e
          }
        }if(0! =  = i.size_hires) {
          let e = (0,
          s.Uk)(i.size_hires);
          n.push( {
            type: "flac24bit",
            size: e
          }),
          o.flac24bit =  {
            size: e
          }
        }let a = "",
        l = "";
        e.album&&(l = e.album.name,
        a = e.album.mid),
        t.push( {
          singer: c(e.singer,
          "name"),
          name: e.title,
          albumName: l,
          albumId: a,
          source: "tx",
          interval: e.interval?(0,
          s.Zg)(e.interval): null,
          songId: e.id,
          albumMid: e.album?.mid??"",
          strMediaMid: e.file.media_mid,
          songmid: e.mid,
          img: "" =  =  = a||"空" =  =  = a?e.singer?.length?`https://y.gtimg.cn/music/photo_new/T001R500x500M000${e.singer[0].mid}.jpg`: "": `https://y.gtimg.cn/music/photo_new/T002R500x500M000${a}.jpg`,
          types: n,
          _types: o,
          typeUrl:  {
            
          }
        })
      }),
      t
    },
    search(e,
    t = 1,
    n) {
      return null =  = n&&(n = this.limit),
      this.musicSearch(e,
      t,
      n).then(( {
        body: e,
        meta: o
      }) = > {
        let i = this.handleResult(e.song.list);
        return this.total = o.sum,
        this.page = t,
        this.allPage = Math.ceil(this.total/n),
        Promise.resolve( {
          list: i,
          allPage: this.allPage,
          limit: n,
          total: this.total,
          source: "tx"
        })
      })
    }
  },
  ve =  {
    _requestObj: null,
    async getList(e = 0) {
      if(this._requestObj&&this._requestObj.cancelHttp(),
      e>2)return Promise.reject(new Error("try max num"));
      const t = (0,
      o.tX)("https://u.y.qq.com/cgi-bin/musicu.fcg",
       {
        method: "post",
        body:  {
          comm:  {
            ct: "19",
            cv: "1803",
            guid: "0",
            patch: "118",
            psrf_access_token_expiresAt: 0,
            psrf_qqaccess_token: "",
            psrf_qqopenid: "",
            psrf_qqunionid: "",
            tmeAppID: "qqmusic",
            tmeLoginType: 0,
            uin: "0",
            wid: "0"
          },
          hotkey:  {
            method: "GetHotkeyForQQMusicPC",
            module: "tencent_musicsoso_hotkey.HotkeyService",
            param:  {
              search_id: "",
              uin: 0
            }
          }
        },
        headers:  {
          Referer: "https://y.qq.com/portal/player.html"
        }
      }),
       {
        body: n,
        statusCode: i
      } = await t.promise;
      if(200! = i||0! =  = n.code)throw new Error("获取热搜词失败");
      return {
        source: "tx",
        list: this.filterList(n.hotkey.data.vec_hotkey)
      }
    },
    filterList: e = >e.map(e = >e.query)
  },
  we =  {
    e400846: "😘",
    e400874: "😴",
    e400825: "😃",
    e400847: "😙",
    e400835: "😍",
    e400873: "😳",
    e400836: "😎",
    e400867: "😭",
    e400832: "😊",
    e400837: "😏",
    e400875: "😫",
    e400831: "😉",
    e400855: "😡",
    e400823: "😄",
    e400862: "😨",
    e400844: "😖",
    e400841: "😓",
    e400830: "😈",
    e400828: "😆",
    e400833: "😋",
    e400822: "😀",
    e400843: "😕",
    e400829: "😇",
    e400824: "😂",
    e400834: "😌",
    e400877: "😷",
    e400132: "🍉",
    e400181: "🍺",
    e401067: "☕️",
    e400186: "🥧",
    e400343: "🐷",
    e400116: "🌹",
    e400126: "🍃",
    e400613: "💋",
    e401236: "❤️",
    e400622: "💔",
    e400637: "💣",
    e400643: "💩",
    e400773: "🔪",
    e400102: "🌛",
    e401328: "🌞",
    e400420: "👏",
    e400914: "🙌",
    e400408: "👍",
    e400414: "👎",
    e401121: "✋",
    e400396: "👋",
    e400384: "👉",
    e401115: "✊",
    e400402: "👌",
    e400905: "🙈",
    e400906: "🙉",
    e400907: "🙊",
    e400562: "👻",
    e400932: "🙏",
    e400644: "💪",
    e400611: "💉",
    e400185: "🎁",
    e400655: "💰",
    e400325: "🐥",
    e400612: "💊",
    e400198: "🎉",
    e401685: "⚡️",
    e400631: "💝",
    e400768: "🔥",
    e400432: "👑"
  },
  be = new Map,
  ke = new Map,
  Ae =  {
    _requestObj: null,
    _requestObj2: null,
    async getSongId( {
      songId: e,
      songmid: t
    }) {
      if(e)return e;
      if(be.has(t))return be.get(t);
      if(ke.has(t))return(await ke.get(t)).songId;
      const n = F(t);
      ke.set(n);
      const o = await n;
      return be.set(t,
      o.songId),
      ke.delete(t),
      o.songId
    },
    async getComment(e,
    t = 1,
    n = 20) {
      this._requestObj&&this._requestObj.cancelHttp();
      const i = await this.getSongId(e),
      s = (0,
      o.tX)("http://c.y.qq.com/base/fcgi-bin/fcg_global_comment_h5.fcg",
       {
        method: "POST",
        headers:  {
          "User-Agent": "Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.1; WOW64; Trident/5.0)"
        },
        form:  {
          uin: "0",
          format: "json",
          cid: "205360772",
          reqtype: "2",
          biztype: "1",
          topid: i,
          cmd: "8",
          needmusiccrit: "1",
          pagenum: t-1,
          pagesize: n
        }
      }),
       {
        body: a,
        statusCode: l
      } = await s.promise;
      if(200! = l||0! =  = a.code)throw new Error("获取评论失败");
      const r = a.comment;
      return {
        source: "tx",
        comments: this.filterNewComment(r.commentlist),
        total: r.commenttotal,
        page: t,
        limit: n,
        maxPage: Math.ceil(r.commenttotal/n)||1
      }
    },
    async getHotComment(e,
    t = 1,
    n = 20) {
      this._requestObj2&&this._requestObj2.cancelHttp();
      const i = await this.getSongId(e),
      s = (0,
      o.tX)("https://u.y.qq.com/cgi-bin/musicu.fcg",
       {
        method: "POST",
        body:  {
          comm:  {
            cv: 4747474,
            ct: 24,
            format: "json",
            inCharset: "utf-8",
            outCharset: "utf-8",
            notice: 0,
            platform: "yqq.json",
            needNewCode: 1,
            uin: 0
          },
          req:  {
            module: "music.globalComment.CommentRead",
            method: "GetHotCommentList",
            param:  {
              BizType: 1,
              BizId: String(i),
              LastCommentSeqNo: "",
              PageSize: n,
              PageNum: t-1,
              HotType: 1,
              WithAirborne: 0,
              PicEnable: 1
            }
          }
        },
        headers:  {
          "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36 Edg/113.0.0.0",
          referer: "https://y.qq.com/",
          origin: "https://y.qq.com"
        }
      }),
       {
        body: a,
        statusCode: l
      } = await s.promise;
      if(200! = l||0! =  = a.code||0! =  = a.req.code)throw new Error("获取热门评论失败");
      const r = a.req.data.CommentList;
      return {
        source: "tx",
        comments: this.filterHotComment(r.Comments),
        total: r.Total,
        page: t,
        limit: n,
        maxPage: Math.ceil(r.Total/n)||1
      }
    },
    filterNewComment(e) {
      return e.map(e = > {
        let t = this.formatTime(e.time),
        n = t?(0,
        s.Uj)(t): null;
        if(e.middlecommentcontent) {
          let t = e.middlecommentcontent[0];
          t.avatarurl = e.avatarurl,
          t.praisenum = e.praisenum,
          e.avatarurl = null,
          e.praisenum = null,
          e.middlecommentcontent.reverse()
        }return {
          id: `${e.rootcommentid}_${e.commentid}`,
          rootId: e.rootcommentid,
          text: e.rootcommentcontent?this.replaceEmoji(e.rootcommentcontent).replace(/\\n/g,
          "\n"): "",
          time: e.rootcommentid =  = e.commentid?t: null,
          timeStr: e.rootcommentid =  = e.commentid?n: null,
          userName: e.rootcommentnick?e.rootcommentnick.substring(1): "",
          avatar: e.avatarurl,
          userId: e.encrypt_rootcommentuin,
          likedCount: e.praisenum,
          reply: e.middlecommentcontent?e.middlecommentcontent.map(o = >( {
            id: `sub_${e.rootcommentid}_${o.subcommentid}`,
            text: this.replaceEmoji(o.subcommentcontent).replace(/\\n/g,
            "\n"),
            time: o.subcommentid =  = e.commentid?t: null,
            timeStr: o.subcommentid =  = e.commentid?n: null,
            userName: o.replynick.substring(1),
            avatar: o.avatarurl,
            userId: o.encrypt_replyuin,
            likedCount: o.praisenum
          })): []
        }
      })
    },
    filterHotComment(e) {
      return e.map(e = >( {
        id: `${e.SeqNo}_${e.CmId}`,
        rootId: e.SeqNo,
        text: e.Content?this.replaceEmoji(e.Content).replace(/\\n/g,
        "\n"): "",
        time: e.PubTime?this.formatTime(e.PubTime): null,
        timeStr: e.PubTime?(0,
        s.Uj)(this.formatTime(e.PubTime)): null,
        userName: e.Nick??"",
        images: e.Pic?[e.Pic]: [],
        avatar: e.Avatar,
        location: e.Location?e.Location: "",
        userId: e.EncryptUin,
        likedCount: e.PraiseNum,
        reply: e.SubComments?e.SubComments.map(e = >( {
          id: `sub_${e.SeqNo}_${e.CmId}`,
          text: this.replaceEmoji(e.Content).replace(/\\n/g,
          "\n"),
          time: e.PubTime?this.formatTime(e.PubTime): null,
          timeStr: e.PubTime?(0,
          s.Uj)(this.formatTime(e.PubTime)): null,
          userName: e.Nick??"",
          avatar: e.Avatar,
          images: e.Pic?[e.Pic]: [],
          userId: e.EncryptUin,
          likedCount: e.PraiseNum
        })): []
      }))
    },
    replaceEmoji(e) {
      let t = /^\[em\](e\d+)\[\/em\]$/,
      n = e.match(/\[em\]e\d+\[\/em\]/g);
      if(!n)return e;
      n = Array.from(new Set(n));
      for(let o of n) {
        let n = o.replace(t,
        "$1");
        e = e.replace(new RegExp(o.replace("[em]",
        "\\[em\\]").replace("[/em]",
        "\\[\\/em\\]"),
        "g"),
        we[n]||"")
      }return e
    },
    formatTime: e = >String(e).length<10?null: parseInt(e+"000")
  },
  xe =  {
    leaderboard: U,
    songList: de,
    musicSearch: ye,
    hotSearch: ve,
    comment: Ae,
    getMusicUrl: (e,
    t) = >(0,
    C.O)("tx").getMusicUrl(e,
    t),
    getLyric: e = >ce.getLyric(e),
    getPic: async e = >`https://y.gtimg.cn/music/photo_new/T002R500x500M000${e.albumId}.jpg`,
    getMusicDetailPageUrl: e = >`https://y.qq.com/n/yqq/song/${e.songmid}.html`
  };
  const Se = Buffer.from("0102030405060708"),
  Ce = Buffer.from("0CoJUm6Qyw8W8jud"),
  Ee = Buffer.from("rFgB&h#%2?^eDg:Q"),
  Ne = "e82ckenh8dichen8",
  Ve = (e,
  t,
  n,
  o) = > {
    const i = (0,
    a.createCipheriv)(t,
    n,
    o);
    return Buffer.concat([i.update(e),
    i.final()])
  },
  Le = e = > {
    const t = JSON.stringify(e),
    n = (0,
    a.randomBytes)(16).map(e = >"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789".charAt(e%62).charCodeAt());
    return {
      params: Ve(Buffer.from(Ve(Buffer.from(t),
      "aes-128-cbc",
      Ce,
      Se).toString("base64")),
      "aes-128-cbc",
      n,
      Se).toString("base64"),
      encSecKey: (o = n.reverse(),
      i = "-----BEGIN PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDgtQn2JZ34ZC28NWYpAUd98iZ37BUrX/aKzmFbt7clFSs6sXqHauqKWqdtLkF2KexO40H1YTX8z2lSgBBOAxLsvaklV8k4cBFK9snQXE9/DDaFt6Rr7iVZMldczhC0JNgTz+SHXT6CBHuX3e9SdB1Ua44oncaTWz7OBGLbCiK45wIDAQAB\n-----END PUBLIC KEY-----",
      o = Buffer.concat([Buffer.alloc(128-o.length),
      o]),
      (0,
      a.publicEncrypt)( {
        key: i,
        padding: a.constants.RSA_NO_PADDING
      },
      o)).toString("hex")
    };
    var o,
    i
  },
  Te = e = > {
    const t = JSON.stringify(e);
    return {
      eparams: Ve(Buffer.from(t),
      "aes-128-ecb",
      Ee,
      "").toString("hex").toUpperCase()
    }
  },
  Ie = (e,
  t) = > {
    const n = "object" =  = typeof t?JSON.stringify(t): t,
    o = `nobody${e}use${n}md5forencrypt`,
    i = `${e}-36cd479b6b5-${n}-36cd479b6b5-${(0,a.createHash)("md5").update(o).digest("hex")}`;
    return {
      params: Ve(Buffer.from(i),
      "aes-128-ecb",
      Ne,
      "").toString("hex").toUpperCase()
    }
  },
  Be =  {
    getSinger(e) {
      let t = [];
      return e?.forEach(e = > {
        t.push(e.name)
      }),
      t.join("、")
    },
    filterList( {
      songs: e,
      privileges: t
    }) {
      const n = [];
      return e.forEach((e,
      o) = > {
        const i = [],
        a =  {
          
        };
        let l,
        r = t[o];
        if(r.id! =  = e.id&&(r = t.find(t = >t.id =  =  = e.id)),
        r) {
          switch("hires" =  = r.maxBrLevel&&(l = e.hr?(0,
          s.Uk)(e.hr.size): null,
          i.push( {
            type: "flac24bit",
            size: l
          }),
          a.flac24bit =  {
            size: l
          }),
          r.maxbr) {
            case 999e3: l = e.sq?(0,
            s.Uk)(e.sq.size): null,
            i.push( {
              type: "flac",
              size: l
            }),
            a.flac =  {
              size: l
            };
            case 32e4: l = e.h?(0,
            s.Uk)(e.h.size): null,
            i.push( {
              type: "320k",
              size: l
            }),
            a["320k"] =  {
              size: l
            };
            case 192e3: case 128e3: l = e.l?(0,
            s.Uk)(e.l.size): null,
            i.push( {
              type: "128k",
              size: l
            }),
            a["128k"] =  {
              size: l
            }
          }i.reverse(),
          e.pc?n.push( {
            singer: e.pc.ar??"",
            name: e.pc.sn??"",
            albumName: e.pc.alb??"",
            albumId: e.al?.id,
            source: "wy",
            interval: (0,
            s.Zg)(e.dt/1e3),
            songmid: e.id,
            img: e.al?.picUrl??"",
            lrc: null,
            otherSource: null,
            types: i,
            _types: a,
            typeUrl:  {
              
            }
          }): n.push( {
            singer: this.getSinger(e.ar),
            name: e.name??"",
            albumName: e.al?.name,
            albumId: e.al?.id,
            source: "wy",
            interval: (0,
            s.Zg)(e.dt/1e3),
            songmid: e.id,
            img: e.al?.picUrl,
            lrc: null,
            otherSource: null,
            types: i,
            _types: a,
            typeUrl:  {
              
            }
          })
        }
      }),
      n
    },
    async getList(e = [],
    t = 0) {
      if(t>2)return Promise.reject(new Error("try max num"));
      const n = (0,
      o.tX)("https://music.163.com/weapi/v3/song/detail",
       {
        method: "post",
        headers:  {
          "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.90 Safari/537.36",
          origin: "https://music.163.com"
        },
        form: Le( {
          c: "["+e.map(e = >'{"id":'+e+"}").join(",")+"]",
          ids: "["+e.join(",")+"]"
        })
      }),
       {
        body: i,
        statusCode: s
      } = await n.promise;
      if(200! = s||200! =  = i.code)throw new Error("获取歌曲详情失败");
      return {
        source: "wy",
        list: this.filterList(i)
      }
    }
  },
  $e = [ {
    id: "wy__19723756",
    name: "飙升榜",
    bangid: "19723756"
  },
   {
    id: "wy__3779629",
    name: "新歌榜",
    bangid: "3779629"
  },
   {
    id: "wy__2884035",
    name: "原创榜",
    bangid: "2884035"
  },
   {
    id: "wy__3778678",
    name: "热歌榜",
    bangid: "3778678"
  },
   {
    id: "wy__991319590",
    name: "说唱榜",
    bangid: "991319590"
  },
   {
    id: "wy__71384707",
    name: "古典榜",
    bangid: "71384707"
  },
   {
    id: "wy__1978921795",
    name: "电音榜",
    bangid: "1978921795"
  },
   {
    id: "wy__5453912201",
    name: "黑胶VIP爱听榜",
    bangid: "5453912201"
  },
   {
    id: "wy__71385702",
    name: "ACG榜",
    bangid: "71385702"
  },
   {
    id: "wy__745956260",
    name: "韩语榜",
    bangid: "745956260"
  },
   {
    id: "wy__10520166",
    name: "国电榜",
    bangid: "10520166"
  },
   {
    id: "wy__180106",
    name: "UK排行榜周榜",
    bangid: "180106"
  },
   {
    id: "wy__60198",
    name: "美国Billboard榜",
    bangid: "60198"
  },
   {
    id: "wy__3812895",
    name: "Beatport全球电子舞曲榜",
    bangid: "3812895"
  },
   {
    id: "wy__21845217",
    name: "KTV唛榜",
    bangid: "21845217"
  },
   {
    id: "wy__60131",
    name: "日本Oricon榜",
    bangid: "60131"
  },
   {
    id: "wy__2809513713",
    name: "欧美热歌榜",
    bangid: "2809513713"
  },
   {
    id: "wy__2809577409",
    name: "欧美新歌榜",
    bangid: "2809577409"
  },
   {
    id: "wy__27135204",
    name: "法国 NRJ Vos Hits 周榜",
    bangid: "27135204"
  },
   {
    id: "wy__3001835560",
    name: "ACG动画榜",
    bangid: "3001835560"
  },
   {
    id: "wy__3001795926",
    name: "ACG游戏榜",
    bangid: "3001795926"
  },
   {
    id: "wy__3001890046",
    name: "ACG VOCALOID榜",
    bangid: "3001890046"
  },
   {
    id: "wy__3112516681",
    name: "中国新乡村音乐排行榜",
    bangid: "3112516681"
  },
   {
    id: "wy__5059644681",
    name: "日语榜",
    bangid: "5059644681"
  },
   {
    id: "wy__5059633707",
    name: "摇滚榜",
    bangid: "5059633707"
  },
   {
    id: "wy__5059642708",
    name: "国风榜",
    bangid: "5059642708"
  },
   {
    id: "wy__5338990334",
    name: "潜力爆款榜",
    bangid: "5338990334"
  },
   {
    id: "wy__5059661515",
    name: "民谣榜",
    bangid: "5059661515"
  },
   {
    id: "wy__6688069460",
    name: "听歌识曲榜",
    bangid: "6688069460"
  },
   {
    id: "wy__6723173524",
    name: "网络热歌榜",
    bangid: "6723173524"
  },
   {
    id: "wy__6732051320",
    name: "俄语榜",
    bangid: "6732051320"
  },
   {
    id: "wy__6732014811",
    name: "越南语榜",
    bangid: "6732014811"
  },
   {
    id: "wy__6886768100",
    name: "中文DJ榜",
    bangid: "6886768100"
  },
   {
    id: "wy__6939992364",
    name: "俄罗斯top hit流行音乐榜",
    bangid: "6939992364"
  },
   {
    id: "wy__7095271308",
    name: "泰语榜",
    bangid: "7095271308"
  },
   {
    id: "wy__7356827205",
    name: "BEAT排行榜",
    bangid: "7356827205"
  },
   {
    id: "wy__7325478166",
    name: "编辑推荐榜VOL.44 天才女子摇滚乐队boygenius剖白卑微心迹",
    bangid: "7325478166"
  },
   {
    id: "wy__7603212484",
    name: "LOOK直播歌曲榜",
    bangid: "7603212484"
  },
   {
    id: "wy__7775163417",
    name: "赏音榜",
    bangid: "7775163417"
  },
   {
    id: "wy__7785123708",
    name: "黑胶VIP新歌榜",
    bangid: "7785123708"
  },
   {
    id: "wy__7785066739",
    name: "黑胶VIP热歌榜",
    bangid: "7785066739"
  },
   {
    id: "wy__7785091694",
    name: "黑胶VIP爱搜榜",
    bangid: "7785091694"
  }],
  De =  {
    limit: 1e5,
    list: [ {
      id: "wybsb",
      name: "飙升榜",
      bangid: "19723756"
    },
     {
      id: "wyrgb",
      name: "热歌榜",
      bangid: "3778678"
    },
     {
      id: "wyxgb",
      name: "新歌榜",
      bangid: "3779629"
    },
     {
      id: "wyycb",
      name: "原创榜",
      bangid: "2884035"
    },
     {
      id: "wygdb",
      name: "古典榜",
      bangid: "71384707"
    },
     {
      id: "wydouyb",
      name: "抖音榜",
      bangid: "2250011882"
    },
     {
      id: "wyhyb",
      name: "韩语榜",
      bangid: "745956260"
    },
     {
      id: "wydianyb",
      name: "电音榜",
      bangid: "1978921795"
    },
     {
      id: "wydjb",
      name: "电竞榜",
      bangid: "2006508653"
    },
     {
      id: "wyktvbb",
      name: "KTV唛榜",
      bangid: "21845217"
    }],
    getUrl: e = >`https://music.163.com/discover/toplist?id=${e}`,
    regExps:  {
      list: /<textarea id="song-list-pre-data" style="display:none;">(.+?)<\/textarea>/
    },
    _requestBoardsObj: null,
    getBoardsData() {
      return this._requestBoardsObj&&this._requestBoardsObj.cancelHttp(),
      this._requestBoardsObj = (0,
      o.tX)("https://music.163.com/weapi/toplist",
       {
        method: "post",
        form: Le( {
          
        })
      }),
      this._requestBoardsObj.promise
    },
    getData: e = >(0,
    o.tX)("https://music.163.com/weapi/v3/playlist/detail",
     {
      method: "post",
      form: Le( {
        id: e,
        n: 1e5,
        p: 1
      })
    }).promise,
    filterBoardsData(e) {
      let t = [];
      for(const n of e)t.push( {
        id: "wy__"+n.id,
        name: n.name,
        bangid: String(n.id)
      });
      return t
    },
    async getBoards(e = 0) {
      return this.list = $e,
       {
        list: $e,
        source: "wy"
      }
    },
    async getList(e,
    t,
    n = 0) {
      if(++n>6)return Promise.reject(new Error("try max num"));
      let o,
      i;
      try {
        o = await this.getData(e)
      }catch(o) {
        if("try max num" =  = o.message)throw o;
        return this.getList(e,
        t,
        n)
      }if(200! =  = o.statusCode||200! =  = o.body.code)return this.getList(e,
      t,
      n);
      try {
        i = await Be.getList(o.body.playlist.trackIds.map(e = >e.id))
      }catch(o) {
        if(console.log(o),
        "try max num" =  = o.message)throw o;
        return this.getList(e,
        t,
        n)
      }return {
        total: i.list.length,
        list: i.list,
        limit: this.limit,
        page: t,
        source: "wy"
      }
    },
    getDetailPageUrl: e = >("string" =  = typeof e&&(e = e.replace("wy__",
    "")),
    `https://music.163.com/#/discover/toplist?id=${e}`)
  },
  Pe =  {
    rxps:  {
      info: /^{"/,
      lineTime: /^\[(\d+),\d+\]/,
      wordTime: /\(\d+,\d+,\d+\)/,
      wordTimeAll: /(\(\d+,\d+,\d+\))/g,
      timeMs2: /\[\d+:\d+\.\d{2}]/,
      timeMs3: /\[\d+:\d+\.\d{3}]/
    },
    msFormat(e,
    t = !0) {
      if(Number.isNaN(e))return"";
      let n = (e%1e3).toString().padStart(t?3: 2,
      "0");
      !t&&n.length>2&&(n = n.slice(0,
      2)),
      e/ = 1e3;
      let o = parseInt(e/60).toString().padStart(2,
      "0");
      return e% = 60,
      `[${o}:${parseInt(e).toString().padStart(2,"0")}.${n}]`
    },
    parseLyric(e) {
      const t = [],
      n = [];
      for(let o of e) {
        o = o.trim();
        let e = this.rxps.lineTime.exec(o);
        if(!e) {
          o.startsWith("[offset")&&(t.push(o),
          n.push(o));
          continue
        }const i = parseInt(e[1]),
        s = this.msFormat(i);
        if(!s)continue;
        let a = o.replace(this.rxps.lineTime,
        "");
        n.push(`${s}${a.replace(this.rxps.wordTimeAll,"")}`);
        let l = a.match(this.rxps.wordTimeAll);
        if(!l)continue;
        l = l.map(e = > {
          const t = /\((\d+),(\d+),\d+\)/.exec(e);
          return`<${Math.trunc(Math.max(parseInt(t[1])-i,0))},${t[2]}>`
        });
        const r = a.split(this.rxps.wordTime);
        r.shift();
        const c = l.map((e,
        t) = >`${e}${r[t]}`).join("");
        t.push(`${s}${c}`)
      }return {
        lyric: n.join("\n"),
        lxlyric: t.join("\n")
      }
    },
    parseHeaderInfo(e) {
      if(!(e = (e = e.trim()).replace(/\r/g,
      "")))return null;
      const t = this.rxps.timeMs3.test(e)||!this.rxps.timeMs2.test(e);
      return e.split("\n").map(e = > {
        if(!this.rxps.info.test(e))return e;
        try {
          const n = JSON.parse(e),
          o = this.msFormat(n.t,
          t);
          return o?`${o}${n.c.map(e=>e.tx).join("")}`: ""
        }catch {
          return""
        }
      })
    },
    getIntv(e) {
      if(!e)return 0;
      e.includes(".")||(e+ = ".0");
      let t = e.split(/:|\./);
      for(;
      t.length<3;
      )t.unshift("0");
      const[n,
      o,
      i] = t;
      return 36e5*parseInt(n)+1e3*parseInt(o)+parseInt(i)
    },
    fixTimeTag(e,
    t) {
      let n = e.split("\n");
      const o = t.split("\n"),
      i = /^\[([\d:.]+)\]/;
      let s = [],
      a = [];
      return o.forEach(e = > {
        const t = i.exec(e);
        if(!t)return;
        if(!e.replace(i,
        "").trim())return;
        const o = this.getIntv(t[1]);
        for(;
        n.length;
        ) {
          const t = n.shift(),
          l = i.exec(t);
          if(!l)continue;
          const r = this.getIntv(l[1]);
          if(Math.abs(o-r)<100) {
            const t = e.replace(i,
            l[0]).trim();
            if(!t)continue;
            a.push(t);
            break
          }s.push(t)
        }n = [...s,
        ...n],
        s = []
      }),
      a.join("\n")
    },
    parse(e,
    t,
    n,
    o,
    i,
    s) {
      const a =  {
        lyric: "",
        tlyric: "",
        rlyric: "",
        lxlyric: ""
      };
      if(e) {
        let o = this.parseHeaderInfo(e);
        if(o) {
          const e = this.parseLyric(o);
          if(t) {
            const n = this.parseHeaderInfo(t);
            n&&(a.tlyric = this.fixTimeTag(e.lyric,
            n.join("\n")))
          }if(n) {
            const t = this.parseHeaderInfo(n);
            t&&(a.rlyric = this.fixTimeTag(e.lyric,
            t.join("\n")))
          }const i = /^\[[\d:.]+\]/,
          s = o.filter(e = >i.test(e)).join("\n");
          return a.lyric = `${s}\n${e.lyric}`,
          a.lxlyric = e.lxlyric,
          a
        }
      }if(o) {
        const e = this.parseHeaderInfo(o);
        e&&(a.lyric = e.join("\n"))
      }if(i) {
        const e = this.parseHeaderInfo(i);
        e&&(a.tlyric = e.join("\n"))
      }if(s) {
        const e = this.parseHeaderInfo(s);
        e&&(a.rlyric = e.join("\n"))
      }return a
    }
  },
  Me = e = > {
    const t = (n = "/api/song/lyric/v1",
    i =  {
      id: e,
      cp: !1,
      tv: 0,
      lv: 0,
      rv: 0,
      kv: 0,
      yv: 0,
      ytv: 0,
      yrv: 0
    },
    (0,
    o.tX)("https://interface3.music.163.com/eapi/song/lyric/v1",
     {
      method: "post",
      headers:  {
        "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.90 Safari/537.36",
        origin: "https://music.163.com"
      },
      form: Ie(n,
      i)
    }));
    var n,
    i;
    return t.promise = t.promise.then(( {
      body: e
    }) = > {
      if(200! =  = e.code||!e?.lrc?.lyric)return Promise.reject(new Error("Get lyric failed"));
      const t = ((e,
      t,
      n) = > {
        if(e) {
          let o = e.replace(/\[(\d{2}:\d{2}):(\d{2})]/g,
          "[$1.$2]"),
          i = t?.replace(/\[(\d{2}:\d{2}):(\d{2})]/g,
          "[$1.$2]")??t;
          o =  = e&&i =  = t||(e = o,
          t = i,
          n&&(n = n.replace(/\[(\d{2}:\d{2}):(\d{2,3})]/g,
          "[$1.$2]").replace(/\[(\d{2}:\d{2}\.\d{2})0]/g,
          "[$1]")))
        }return {
          lrc: e,
          tlrc: t,
          romalrc: n
        }
      })(e.lrc.lyric,
      e.tlyric?.lyric,
      e.romalrc?.lyric),
      n = Pe.parse(e.yrc?.lyric,
      e.ytlrc?.lyric,
      e.yromalrc?.lyric,
      t.lrc,
      t.tlrc,
      t.romalrc);
      return n.lyric?n: Promise.reject(new Error("Get lyric failed"))
    }),
    t
  };
  (Object.getOwnPropertyDescriptor(Me,
  "name")|| {
    
  }).writable||Object.defineProperty(Me,
  "name",
   {
    value: "default",
    configurable: !0
  });
  const ze = e = > {
    const t = (0,
    o.tX)("https://music.163.com/weapi/v3/song/detail",
     {
      method: "post",
      headers:  {
        "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.90 Safari/537.36",
        Referer: "https://music.163.com/song?id="+e,
        origin: "https://music.163.com"
      },
      form: Le( {
        c: `[{"id":${e}}]`,
        ids: `[${e}]`
      })
    });
    return t.promise = t.promise.then(( {
      body: e
    }) = >200 =  =  = e.code&&e.songs.length?e.songs[0]: Promise.reject(new Error("获取歌曲信息失败"))),
    t
  };
  (Object.getOwnPropertyDescriptor(ze,
  "name")|| {
    
  }).writable||Object.defineProperty(ze,
  "name",
   {
    value: "default",
    configurable: !0
  });
  const Oe = (e,
  t) = >(0,
  o.tX)("http://interface.music.163.com/eapi/batch",
   {
    method: "post",
    headers:  {
      "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.90 Safari/537.36",
      origin: "https://music.163.com"
    },
    form: Ie(e,
    t)
  }),
  qe =  {
    limit: 30,
    total: 0,
    page: 0,
    allPage: 1,
    musicSearch: (e,
    t,
    n) = >Oe("/api/search/song/list/page",
     {
      keyword: e,
      needCorrect: "1",
      channel: "typing",
      offset: n*(t-1),
      scene: "normal",
      total: 1 =  = t,
      limit: n
    }).promise.then(( {
      body: e
    }) = >e),
    getSinger(e) {
      let t = [];
      return e.forEach(e = > {
        t.push(e.name)
      }),
      t.join("、")
    },
    handleResult(e) {
      return e?e.map(e = > {
        const t = [],
        n =  {
          
        };
        let o;
        switch("hires" =  = (e = e.baseInfo.simpleSongData).privilege.maxBrLevel&&(o = e.hr?(0,
        s.Uk)(e.hr.size): null,
        t.push( {
          type: "flac24bit",
          size: o
        }),
        n.flac24bit =  {
          size: o
        }),
        e.privilege.maxbr) {
          case 999e3: o = e.sq?(0,
          s.Uk)(e.sq.size): null,
          t.push( {
            type: "flac",
            size: o
          }),
          n.flac =  {
            size: o
          };
          case 32e4: o = e.h?(0,
          s.Uk)(e.h.size): null,
          t.push( {
            type: "320k",
            size: o
          }),
          n["320k"] =  {
            size: o
          };
          case 192e3: case 128e3: o = e.l?(0,
          s.Uk)(e.l.size): null,
          t.push( {
            type: "128k",
            size: o
          }),
          n["128k"] =  {
            size: o
          }
        }return t.reverse(),
         {
          singer: this.getSinger(e.ar),
          name: e.name,
          albumName: e.al.name,
          albumId: e.al.id,
          source: "wy",
          interval: (0,
          s.Zg)(e.dt/1e3),
          songmid: e.id,
          img: e.al.picUrl,
          lrc: null,
          types: t,
          _types: n,
          typeUrl:  {
            
          }
        }
      }): []
    },
    search(e,
    t = 1,
    n,
    o = 0) {
      return++o>3?Promise.reject(new Error("try max num")): (null =  = n&&(n = this.limit),
      this.musicSearch(e,
      t,
      n).then(i = > {
        if(!i||200! =  = i.code)return this.search(e,
        t,
        n,
        o);
        let s = this.handleResult(i.data.resources||[]);
        return null =  = s?this.search(e,
        t,
        n,
        o): (this.total = i.data.totalCount||0,
        this.page = t,
        this.allPage = Math.ceil(this.total/this.limit),
         {
          list: s,
          allPage: this.allPage,
          limit: this.limit,
          total: this.total,
          source: "wy"
        })
      }))
    }
  },
  Re =  {
    _requestObj_tags: null,
    _requestObj_hotTags: null,
    _requestObj_list: null,
    limit_list: 30,
    limit_song: 1e5,
    successCode: 200,
    cookie: "MUSIC_U=",
    sortList: [ {
      name: "最热",
      id: "hot"
    }],
    regExps:  {
      listDetailLink: /^.+(?:\?|&)id=(\d+)(?:&.*$|#.*$|$)/,
      listDetailLink2: /^.+\/playlist\/(\d+)\/\d+\/.+$/
    },
    async handleParseId(e,
    t = 0) {
      if(t>2)throw new Error("link try max num");
      const n = (0,
      o.tX)(e),
       {
        headers:  {
          location: i
        },
        statusCode: s
      } = await n.promise;
      if(s>400)return this.handleParseId(e,
      ++t);
      const a = i??e;
      return this.regExps.listDetailLink.test(a)?a.replace(this.regExps.listDetailLink,
      "$1"): a.replace(this.regExps.listDetailLink2,
      "$1")
    },
    async getListId(e) {
      let t;
      if(/###/.test(e)) {
        const[n,
        o] = e.split("###");
        e = n,
        t = `MUSIC_U=${o}`
      }return/[?&: /]/.test(e)&&(e = this.regExps.listDetailLink.test(e)?e.replace(this.regExps.listDetailLink,
      "$1"): this.regExps.listDetailLink2.test(e)?e.replace(this.regExps.listDetailLink2,
      "$1"): await this.handleParseId(e)),
       {
        id: e,
        cookie: t
      }
    },
    async getListDetail(e,
    t,
    n = 0) {
      if(n>2)return Promise.reject(new Error("try max num"));
      const {
        id: i,
        cookie: a
      } = await this.getListId(e);
      a&&(this.cookie = a);
      const l = (0,
      o.tX)("https://music.163.com/api/linux/forward",
       {
        method: "post",
        headers:  {
          "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.90 Safari/537.36",
          Cookie: this.cookie
        },
        form: Te( {
          method: "POST",
          url: "https://music.163.com/api/v3/playlist/detail",
          params:  {
            id: i,
            n: this.limit_song,
            s: 8
          }
        })
      }),
       {
        statusCode: r,
        body: c
      } = await l.promise;
      if(200! =  = r||c.code! =  = this.successCode)return this.getListDetail(i,
      t,
      ++n);
      let d,
      u = 1e3,
      p = (t-1)*u;
      if(c.playlist.trackIds.length =  = c.privileges.length)d = this.filterListDetail(c);
      else try {
        d = (await Be.getList(c.playlist.trackIds.slice(p,
        u*t).map(e = >e.id))).list
      }catch(e) {
        if(console.log(e),
        "try max num" =  = e.message)throw e;
        return this.getListDetail(i,
        t,
        ++n)
      }return {
        list: d,
        page: t,
        limit: u,
        total: c.playlist.trackIds.length,
        source: "wy",
        info:  {
          play_count: (0,
          s.pN)(c.playlist.playCount),
          name: c.playlist.name,
          img: c.playlist.coverImgUrl,
          desc: c.playlist.description,
          author: c.playlist.creator.nickname
        }
      }
    },
    filterListDetail( {
      playlist:  {
        tracks: e
      },
      privileges: t
    }) {
      const n = [];
      return e.forEach((e,
      o) = > {
        const i = [],
        a =  {
          
        };
        let l,
        r = t[o];
        if(r.id! =  = e.id&&(r = t.find(t = >t.id =  =  = e.id)),
        r) {
          switch("hires" =  = r.maxBrLevel&&(l = e.hr?(0,
          s.Uk)(e.hr.size): null,
          i.push( {
            type: "flac24bit",
            size: l
          }),
          a.flac24bit =  {
            size: l
          }),
          r.maxbr) {
            case 999e3: l = null,
            i.push( {
              type: "flac",
              size: l
            }),
            a.flac =  {
              size: l
            };
            case 32e4: l = e.h?(0,
            s.Uk)(e.h.size): null,
            i.push( {
              type: "320k",
              size: l
            }),
            a["320k"] =  {
              size: l
            };
            case 192e3: case 128e3: l = e.l?(0,
            s.Uk)(e.l.size): null,
            i.push( {
              type: "128k",
              size: l
            }),
            a["128k"] =  {
              size: l
            }
          }i.reverse(),
          e.pc?n.push( {
            singer: e.pc.ar??"",
            name: e.pc.sn??"",
            albumName: e.pc.alb??"",
            albumId: e.al?.id,
            source: "wy",
            interval: (0,
            s.Zg)(e.dt/1e3),
            songmid: e.id,
            img: e.al?.picUrl??"",
            lrc: null,
            otherSource: null,
            types: i,
            _types: a,
            typeUrl:  {
              
            }
          }): n.push( {
            singer: c(e.ar,
            "name"),
            name: e.name??"",
            albumName: e.al?.name,
            albumId: e.al?.id,
            source: "wy",
            interval: (0,
            s.Zg)(e.dt/1e3),
            songmid: e.id,
            img: e.al?.picUrl,
            lrc: null,
            otherSource: null,
            types: i,
            _types: a,
            typeUrl:  {
              
            }
          })
        }
      }),
      n
    },
    getList(e,
    t,
    n,
    i = 0) {
      return i>2?Promise.reject(new Error("try max num")): (this._requestObj_list&&this._requestObj_list.cancelHttp(),
      this._requestObj_list = (0,
      o.tX)("https://music.163.com/weapi/playlist/list",
       {
        method: "post",
        form: Le( {
          cat: t||"全部",
          order: e,
          limit: this.limit_list,
          offset: this.limit_list*(n-1),
          total: !0
        })
      }),
      this._requestObj_list.promise.then(( {
        body: o
      }) = >o.code! =  = this.successCode?this.getList(e,
      t,
      n,
      ++i):  {
        list: this.filterList(o.playlists),
        total: parseInt(o.total),
        page: n,
        limit: this.limit_list,
        source: "wy"
      }))
    },
    filterList: e = >e.map(e = >( {
      play_count: (0,
      s.pN)(e.playCount),
      id: String(e.id),
      author: e.creator.nickname,
      name: e.name,
      time: e.createTime?(0,
      s.i$)(e.createTime,
      "Y-M-D"): "",
      img: e.coverImgUrl,
      grade: e.grade,
      total: e.trackCount,
      desc: e.description,
      source: "wy"
    })),
    getTag(e = 0) {
      return this._requestObj_tags&&this._requestObj_tags.cancelHttp(),
      e>2?Promise.reject(new Error("try max num")): (this._requestObj_tags = (0,
      o.tX)("https://music.163.com/weapi/playlist/catalogue",
       {
        method: "post",
        form: Le( {
          
        })
      }),
      this._requestObj_tags.promise.then(( {
        body: t
      }) = >t.code! =  = this.successCode?this.getTag(++e): this.filterTagInfo(t)))
    },
    filterTagInfo( {
      sub: e,
      categories: t
    }) {
      const n =  {
        
      };
      for(const o of e)n[o.category]||(n[o.category] = []),
      n[o.category].push( {
        parent_id: t[o.category],
        parent_name: t[o.category],
        id: o.name,
        name: o.name,
        source: "wy"
      });
      const o = [];
      for(const e of Object.keys(t))o.push( {
        name: t[e],
        list: n[e],
        source: "wy"
      });
      return o
    },
    getHotTag(e = 0) {
      return this._requestObj_hotTags&&this._requestObj_hotTags.cancelHttp(),
      e>2?Promise.reject(new Error("try max num")): (this._requestObj_hotTags = (0,
      o.tX)("https://music.163.com/weapi/playlist/hottags",
       {
        method: "post",
        form: Le( {
          
        })
      }),
      this._requestObj_hotTags.promise.then(( {
        body: t
      }) = >t.code! =  = this.successCode?this.getTag(++e): this.filterHotTagInfo(t.tags)))
    },
    filterHotTagInfo: e = >e.map(e = >( {
      id: e.playlistTag.name,
      name: e.playlistTag.name,
      source: "wy"
    })),
    getTags() {
      return Promise.all([this.getTag(),
      this.getHotTag()]).then(([e,
      t]) = >( {
        tags: e,
        hotTag: t,
        source: "wy"
      }))
    },
    async getDetailPageUrl(e) {
      const {
        id: t
      } = await this.getListId(e);
      return`https://music.163.com/#/playlist?id=${t}`
    },
    search(e,
    t,
    n = 20) {
      return Oe("/api/cloudsearch/pc",
       {
        s: e,
        type: 1e3,
        limit: n,
        total: 1 =  = t,
        offset: n*(t-1)
      }).promise.then(( {
        body: e
      }) = > {
        if(e.code! = this.successCode)throw new Error("filed");
        return {
          list: this.filterList(e.result.playlists),
          limit: n,
          total: e.result.playlistCount,
          source: "wy"
        }
      })
    }
  },
  Ue =  {
    _requestObj: null,
    async getList(e = 0) {
      if(this._requestObj&&this._requestObj.cancelHttp(),
      e>2)return Promise.reject(new Error("try max num"));
      const t = Oe("/api/search/chart/detail",
       {
        id: "HOT_SEARCH_SONG#@#"
      }),
       {
        body: n,
        statusCode: o
      } = await t.promise;
      if(200! = o||200! =  = n.code)throw new Error("获取热搜词失败");
      return {
        source: "wy",
        list: this.filterList(n.data.itemList)
      }
    },
    filterList: e = >e.map(e = >e.searchWord)
  },
  je = [["大笑",
  "😃"],
  ["可爱",
  "😊"],
  ["憨笑",
  "☺️"],
  ["色",
  "😍"],
  ["亲亲",
  "😙"],
  ["惊恐",
  "😱"],
  ["流泪",
  "😭"],
  ["亲",
  "😚"],
  ["呆",
  "😳"],
  ["哀伤",
  "😔"],
  ["呲牙",
  "😁"],
  ["吐舌",
  "😝"],
  ["撇嘴",
  "😒"],
  ["怒",
  "😡"],
  ["奸笑",
  "😏"],
  ["汗",
  "😓"],
  ["痛苦",
  "😖"],
  ["惶恐",
  "😰"],
  ["生病",
  "😨"],
  ["口罩",
  "😷"],
  ["大哭",
  "😂"],
  ["晕",
  "😵"],
  ["发怒",
  "👿"],
  ["开心",
  "😄"],
  ["鬼脸",
  "😜"],
  ["皱眉",
  "😞"],
  ["流感",
  "😢"],
  ["爱心",
  "❤️"],
  ["心碎",
  "💔"],
  ["钟情",
  "💘"],
  ["星星",
  "⭐️"],
  ["生气",
  "💢"],
  ["便便",
  "💩"],
  ["强",
  "👍"],
  ["弱",
  "👎"],
  ["拜",
  "🙏"],
  ["牵手",
  "👫"],
  ["跳舞",
  "👯‍♀️"],
  ["禁止",
  "🙅‍♀️"],
  ["这边",
  "💁‍♀️"],
  ["爱意",
  "💏"],
  ["示爱",
  "👩‍❤️‍👨"],
  ["嘴唇",
  "👄"],
  ["狗",
  "🐶"],
  ["猫",
  "🐱"],
  ["猪",
  "🐷"],
  ["兔子",
  "🐰"],
  ["小鸡",
  "🐤"],
  ["公鸡",
  "🐔"],
  ["幽灵",
  "👻"],
  ["圣诞",
  "🎅"],
  ["外星",
  "👽"],
  ["钻石",
  "💎"],
  ["礼物",
  "🎁"],
  ["男孩",
  "👦"],
  ["女孩",
  "👧"],
  ["蛋糕",
  "🎂"],
  ["18",
  "🔞"],
  ["圈",
  "⭕"],
  ["叉",
  "❌"]],
  Fe = e = > {
    for(const t of je)e = e.replaceAll(`[${t[0]}]`,
    t[1]);
    return e
  };
  let He =  {
    cache:  {
      
    },
    getCursor(e,
    t,
    n) {
      let o,
      i,
      s,
      a = this.cache[e];
      return a||(a = this.cache[e] =  {
        
      }),
      1 =  = t?(a.page = 1,
      i = a.cursor = a.prevCursor = Date.now(),
      o = 1,
      s = 0): a.page&&(i = a.cursor,
      t>a.page?(o = 1,
      s = (t-a.page-1)*n): t<a.page?(o = 0,
      s = (a.page-t-1)*n): (i = a.cursor = a.prevCursor,
      s = a.offset,
      o = a.orderType)),
       {
        orderType: o,
        cursor: i,
        offset: s
      }
    },
    setCursor(e,
    t,
    n,
    o,
    i) {
      let s = this.cache[e];
      s||(s = this.cache[e] =  {
        
      }),
      s.prevCursor = s.cursor,
      s.cursor = t,
      s.orderType = n,
      s.offset = o,
      s.page = i
    }
  };
  const Ke =  {
    leaderboard: De,
    musicSearch: qe,
    songList: Re,
    hotSearch: Ue,
    comment:  {
      _requestObj: null,
      _requestObj2: null,
      async getComment( {
        songmid: e
      },
      t = 1,
      n = 20) {
        this._requestObj&&this._requestObj.cancelHttp();
        const i = "R_SO_4_"+e,
        s = He.getCursor(e,
        t,
        n),
        a = (0,
        o.tX)("https://music.163.com/weapi/comment/resource/comments/get",
         {
          method: "post",
          headers:  {
            "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.90 Safari/537.36",
            origin: "https://music.163.com",
            Refere: "http://music.163.com/"
          },
          form: Le( {
            cursor: s.cursor,
            offset: s.offset,
            orderType: s.orderType,
            pageNo: t,
            pageSize: n,
            rid: i,
            threadId: i
          })
        }),
         {
          body: l,
          statusCode: r
        } = await a.promise;
        if(200! = r||200! =  = l.code)throw new Error("获取评论失败");
        return He.setCursor(e,
        l.data.cursor,
        s.orderType,
        s.offset,
        t),
         {
          source: "wy",
          comments: this.filterComment(l.data.comments),
          total: l.data.totalCount,
          page: t,
          limit: n,
          maxPage: Math.ceil(l.data.totalCount/n)||1
        }
      },
      async getHotComment( {
        songmid: e
      },
      t = 1,
      n = 100) {
        this._requestObj2&&this._requestObj2.cancelHttp();
        const i = "R_SO_4_"+e;
        t- = 1;
        const s = (0,
        o.tX)(`https://music.163.com/weapi/v1/resource/hotcomments/${i}`,
         {
          method: "post",
          headers:  {
            "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.90 Safari/537.36",
            origin: "https://music.163.com",
            Refere: "http://music.163.com/"
          },
          form: Le( {
            rid: i,
            limit: n,
            offset: n*t,
            beforeTime: Date.now().toString()
          })
        }),
         {
          body: a,
          statusCode: l
        } = await s.promise;
        if(200! = l||200! =  = a.code)throw new Error("获取热门评论失败");
        const r = a.total??0;
        return {
          source: "wy",
          comments: this.filterComment(a.hotComments),
          total: r,
          page: t,
          limit: n,
          maxPage: Math.ceil(r/n)||1
        }
      },
      filterComment: e = >e.map(e = > {
        let t =  {
          id: e.commentId,
          text: e.content?Fe(e.content): "",
          time: e.time?e.time: "",
          timeStr: e.time?(0,
          s.Uj)(e.time): "",
          location: e.ipLocation?.location,
          userName: e.user.nickname,
          avatar: e.user.avatarUrl,
          userId: e.user.userId,
          likedCount: e.likedCount,
          reply: []
        },
        n = e.beReplied&&e.beReplied[0];
        return n? {
          id: e.commentId,
          rootId: n.beRepliedCommentId,
          text: n.content?Fe(n.content): "",
          time: e.time,
          timeStr: null,
          location: n.ipLocation?.location,
          userName: n.user.nickname,
          avatar: n.user.avatarUrl,
          userId: n.user.userId,
          likedCount: null,
          reply: [t]
        }: t
      })
    },
    getMusicUrl: (e,
    t) = >(0,
    C.O)("wy").getMusicUrl(e,
    t),
    getLyric: e = >Me(e.songmid),
    getPic: e = >ze(e.songmid).promise.then(e = >e.al.picUrl),
    getMusicDetailPageUrl: e = >`https://music.163.com/#/song?id=${e.songmid}`
  },
  Ge = async(e,
  t,
  n = 0) = > {
    if(n>2)throw new Error("try max num");
    let i;
    try {
      i = await(0,
      o.tX)(e,
      t).promise
    }catch(o) {
      return console.log(o),
      Ge(e,
      t,
      ++n)
    }return 200! =  = i.statusCode||"000000"! =  = (void 0! =  = i.body.code?i.body.code: void 0! =  = i.body.returnCode?i.body.returnCode: i.body.code)?Ge(e,
    t,
    ++n): i.body.data?i.body.data: i.body
  },
  We = e = > {
    let t = new Set;
    const n = [];
    return e.forEach(e = > {
      if(!e.songId||t.has(e.songId))return;
      t.add(e.songId);
      const o = [],
      i =  {
        
      };
      e.newRateFormats?.forEach(e = > {
        let t;
        switch(e.formatType) {
          case"PQ": t = (0,
          s.Uk)(e.size??e.androidSize),
          o.push( {
            type: "128k",
            size: t
          }),
          i["128k"] =  {
            size: t
          };
          break;
          case"HQ": t = (0,
          s.Uk)(e.size??e.androidSize),
          o.push( {
            type: "320k",
            size: t
          }),
          i["320k"] =  {
            size: t
          };
          break;
          case"SQ": t = (0,
          s.Uk)(e.size??e.androidSize),
          o.push( {
            type: "flac",
            size: t
          }),
          i.flac =  {
            size: t
          };
          break;
          case"ZQ": t = (0,
          s.Uk)(e.size??e.androidSize),
          o.push( {
            type: "flac24bit",
            size: t
          }),
          i.flac24bit =  {
            size: t
          }
        }
      });
      const a = /(\d\d:\d\d)$/.test(e.length);
      n.push( {
        singer: c(e.artists,
        "name"),
        name: e.songName,
        albumName: e.album,
        albumId: e.albumId,
        songmid: e.songId,
        copyrightId: e.copyrightId,
        source: "mg",
        interval: a?RegExp.$1: null,
        img: e.albumImgs?.length?e.albumImgs[0].img: null,
        lrc: null,
        lrcUrl: e.lrcUrl,
        mrcUrl: e.mrcUrl,
        trcUrl: e.trcUrl,
        otherSource: null,
        types: o,
        _types: i,
        typeUrl:  {
          
        }
      })
    }),
    n
  },
  Ye = e = > {
    let t = new Set;
    const n = [];
    return e.forEach(e = > {
      if(!e.songId||t.has(e.songId))return;
      t.add(e.songId);
      const o = [],
      i =  {
        
      };
      e.audioFormats?.forEach(e = > {
        let t;
        switch(e.formatType) {
          case"PQ": t = (0,
          s.Uk)(e.size??e.androidSize),
          o.push( {
            type: "128k",
            size: t
          }),
          i["128k"] =  {
            size: t
          };
          break;
          case"HQ": t = (0,
          s.Uk)(e.size??e.androidSize),
          o.push( {
            type: "320k",
            size: t
          }),
          i["320k"] =  {
            size: t
          };
          break;
          case"SQ": t = (0,
          s.Uk)(e.size??e.androidSize),
          o.push( {
            type: "flac",
            size: t
          }),
          i.flac =  {
            size: t
          };
          break;
          case"ZQ": t = (0,
          s.Uk)(e.size??e.androidSize),
          o.push( {
            type: "flac24bit",
            size: t
          }),
          i.flac24bit =  {
            size: t
          }
        }
      }),
      n.push( {
        singer: c(e.singerList,
        "name"),
        name: e.songName,
        albumName: e.album,
        albumId: e.albumId,
        songmid: e.songId,
        copyrightId: e.copyrightId,
        source: "mg",
        interval: (0,
        s.Zg)(e.duration),
        img: e.img3||e.img2||e.img1||null,
        lrc: null,
        lrcUrl: e.lrcUrl,
        mrcUrl: e.mrcUrl,
        trcUrl: e.trcUrl,
        otherSource: null,
        types: o,
        _types: i,
        typeUrl:  {
          
        }
      })
    }),
    n
  },
  Xe = async e = >Ze([e]).then(e = >e[0]),
  Ze = async e = >We(await Promise.all((e = > {
    let t = e,
    n = [];
    for(;
    t.length&&(n.push(t.slice(0,
    100)),
    !(t.length<100));
    )t = t.slice(100);
    return Promise.all(n.map(e = >Ge("https://c.musicapp.migu.cn/MIGUM2.0/v1.0/content/resourceinfo.do?resourceType=2",
     {
      method: "POST",
      form:  {
        resourceId: e.join("|")
      }
    }).then(e = >e.resource)))
  })(e)).then(e = >e.flat())),
  Qe = [ {
    id: "mg__27553319",
    name: "新歌榜",
    bangid: "27553319",
    source: "mg"
  },
   {
    id: "mg__27186466",
    name: "热歌榜",
    bangid: "27186466",
    source: "mg"
  },
   {
    id: "mg__27553408",
    name: "原创榜",
    bangid: "27553408",
    source: "mg"
  },
   {
    id: "mg__75959118",
    name: "音乐风向榜",
    bangid: "75959118",
    source: "mg"
  },
   {
    id: "mg__76557036",
    name: "彩铃分贝榜",
    bangid: "76557036",
    source: "mg"
  },
   {
    id: "mg__76557745",
    name: "会员臻爱榜",
    bangid: "76557745",
    source: "mg"
  },
   {
    id: "mg__23189800",
    name: "港台榜",
    bangid: "23189800",
    source: "mg"
  },
   {
    id: "mg__23189399",
    name: "内地榜",
    bangid: "23189399",
    source: "mg"
  },
   {
    id: "mg__19190036",
    name: "欧美榜",
    bangid: "19190036",
    source: "mg"
  },
   {
    id: "mg__83176390",
    name: "国风金曲榜",
    bangid: "83176390",
    source: "mg"
  }],
  Je =  {
    limit: 200,
    list: [ {
      id: "mgyyb",
      name: "音乐榜",
      bangid: "27553319"
    },
     {
      id: "mgysb",
      name: "影视榜",
      bangid: "23603721"
    },
     {
      id: "mghybnd",
      name: "华语内地榜",
      bangid: "23603926"
    },
     {
      id: "mghyjqbgt",
      name: "华语港台榜",
      bangid: "23603954"
    },
     {
      id: "mgomb",
      name: "欧美榜",
      bangid: "23603974"
    },
     {
      id: "mgrhb",
      name: "日韩榜",
      bangid: "23603982"
    },
     {
      id: "mgwlb",
      name: "网络榜",
      bangid: "23604058"
    },
     {
      id: "mgclb",
      name: "彩铃榜",
      bangid: "23604023"
    },
     {
      id: "mgktvb",
      name: "KTV榜",
      bangid: "23604040"
    },
     {
      id: "mgrcb",
      name: "原创榜",
      bangid: "23604032"
    }],
    getUrl: (e,
    t) = >`https://app.c.nf.migu.cn/MIGUM2.0/v1.0/content/querycontentbyId.do?columnId=${e}&needAll=0`,
    successCode: "000000",
    requestBoardsObj: null,
    getBoardsData() {
      return this.requestBoardsObj&&this._requestBoardsObj.cancelHttp(),
      this.requestBoardsObj = (0,
      o.tX)("https://app.c.nf.migu.cn/pc/bmw/rank/rank-index/v1.0",
       {
        headers:  {
          Referer: "https://app.c.nf.migu.cn/",
          "User-Agent": "Mozilla/5.0 (Linux; Android 5.1.1; Nexus 6 Build/LYZ28E) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Mobile Safari/537.36",
          channel: "0146921"
        }
      }),
      this.requestBoardsObj.promise
    },
    getData: e = >(0,
    o.tX)(e).promise,
    async getBoards(e = 0) {
      return this.list = Qe,
       {
        list: Qe,
        source: "mg"
      }
    },
    getList(e,
    t,
    n = 0) {
      return++n>3?Promise.reject(new Error("try max num")): this.getData(this.getUrl(e,
      t)).then(( {
        statusCode: o,
        body: i
      }) = > {
        if(200! =  = o||i.code! =  = this.successCode)return this.getList(e,
        t,
        n);
        const s = We(i.columnInfo.contents.map(e = >e.objectInfo));
        return {
          total: s.length,
          list: s,
          limit: this.limit,
          page: t,
          source: "mg"
        }
      })
    },
    getDetailPageUrl(e) {
      "string" =  = typeof e&&(e = e.replace("mg__",
      ""));
      for(const t of Qe)if(t.bangid =  = e)return`https://music.migu.cn/v3/music/top/${t.webId}`;
      return null
    }
  },
  et = (e,
  t) = > {
    const n = "963B7AA0D21511ED807EE5846EC87D20";
    return {
      sign: r(`${t}6cdc72a439cef99a3418d2a78aa28c73yyapp2d16148780a1dcc7408e06336b98cfd50${n}${e}`),
      deviceId: n
    }
  },
  tt =  {
    limit: 20,
    total: 0,
    page: 0,
    allPage: 1,
    musicSearch(e,
    t,
    n) {
      const i = Date.now().toString(),
      s = et(i,
      e);
      return(0,
      o.tX)(`https://jadeite.migu.cn/music_search/v3/search/searchAll?isCorrect=0&isCopyright=1&searchSwitch=%7B%22song%22%3A1%2C%22album%22%3A0%2C%22singer%22%3A0%2C%22tagSong%22%3A1%2C%22mvSong%22%3A0%2C%22bestShow%22%3A1%2C%22songlist%22%3A0%2C%22lyricSong%22%3A0%7D&pageSize=${n}&text=${encodeURIComponent(e)}&pageNo=${t}&sort=0&sid=USS`,
       {
        headers:  {
          uiVersion: "A_music_3.6.1",
          deviceId: s.deviceId,
          timestamp: i,
          sign: s.sign,
          channel: "0146921",
          "User-Agent": "Mozilla/5.0 (Linux; U; Android 11.0.0; zh-cn; MI 11 Build/OPR1.170623.032) AppleWebKit/534.30 (KHTML, like Gecko) Version/4.0 Mobile Safari/534.30"
        }
      }).promise.then(( {
        body: e
      }) = >e)
    },
    filterData(e) {
      const t = [],
      n = new Set;
      return e.forEach(e = > {
        e.forEach(e = > {
          if(!e.songId||!e.copyrightId||n.has(e.copyrightId))return;
          n.add(e.copyrightId);
          const o = [],
          i =  {
            
          };
          e.audioFormats&&e.audioFormats.forEach(e = > {
            let t;
            switch(e.formatType) {
              case"PQ": t = (0,
              s.Uk)(e.asize??e.isize),
              o.push( {
                type: "128k",
                size: t
              }),
              i["128k"] =  {
                size: t
              };
              break;
              case"HQ": t = (0,
              s.Uk)(e.asize??e.isize),
              o.push( {
                type: "320k",
                size: t
              }),
              i["320k"] =  {
                size: t
              };
              break;
              case"SQ": t = (0,
              s.Uk)(e.asize??e.isize),
              o.push( {
                type: "flac",
                size: t
              }),
              i.flac =  {
                size: t
              };
              break;
              case"ZQ24": t = (0,
              s.Uk)(e.asize??e.isize),
              o.push( {
                type: "flac24bit",
                size: t
              }),
              i.flac24bit =  {
                size: t
              }
            }
          });
          let a = e.img3||e.img2||e.img1||null;
          a&&!/https?:/.test(e.img3)&&(a = "http://d.musicapp.migu.cn"+a),
          t.push( {
            singer: c(e.singerList),
            name: e.name,
            albumName: e.album,
            albumId: e.albumId,
            songmid: e.songId,
            copyrightId: e.copyrightId,
            source: "mg",
            interval: (0,
            s.Zg)(e.duration),
            img: a,
            lrc: null,
            lrcUrl: e.lrcUrl,
            mrcUrl: e.mrcurl,
            trcUrl: e.trcUrl,
            types: o,
            _types: i,
            typeUrl:  {
              
            }
          })
        })
      }),
      t
    },
    search(e,
    t = 1,
    n,
    o = 0) {
      return++o>3?Promise.reject(new Error("try max num")): (null =  = n&&(n = this.limit),
      this.musicSearch(e,
      t,
      n).then(i = > {
        if(!i||"000000"! =  = i.code)return Promise.reject(new Error(i?i.info: "搜索失败"));
        const s = i.songResultData|| {
          resultList: [],
          totalCount: 0
        };
        let a = this.filterData(s.resultList);
        return null =  = a?this.search(e,
        t,
        n,
        o): (this.total = parseInt(s.totalCount),
        this.page = t,
        this.allPage = Math.ceil(this.total/n),
         {
          list: a,
          allPage: this.allPage,
          limit: n,
          total: this.total,
          source: "mg"
        })
      }))
    }
  },
  nt =  {
    _requestObj_tags: null,
    _requestObj_list: null,
    limit_list: 30,
    limit_song: 50,
    successCode: "000000",
    cachedDetailInfo:  {
      
    },
    cachedUrl:  {
      
    },
    sortList: [ {
      name: "推荐",
      id: "15127315"
    }],
    regExps:  {
      list: /<li><div class="thumb">.+?<\/li>/g,
      listInfo: /.+data-original="(.+?)".*data-id="(\d+)".*<div class="song-list-name"><a\s.*?>(.+?)<\/a>.+<i class="iconfont cf-bofangliang"><\/i>(.+?)<\/div>/,
      listDetailLink: /^.+\/playlist\/(\d+)(?:\?.*|&.*$|#.*$|$)/
    },
    tagsUrl: "https://app.c.nf.migu.cn/pc/v1.0/template/musiclistplaza-taglist/release",
    getSongListUrl: (e,
    t,
    n) = >t?`https://app.c.nf.migu.cn/pc/v1.0/template/musiclistplaza-listbytag/release?pageNumber=${n}&templateVersion=2&tagId=${t}`: `https://app.c.nf.migu.cn/pc/bmw/page-data/playlist-square-recommend/v1.0?templateVersion=2&pageNo=${n}`,
    getSongListDetailUrl(e,
    t) {
      return`https://app.c.nf.migu.cn/MIGUM3.0/resource/playlist/song/v2.0?pageNo=${t}&pageSize=${this.limit_song}&playlistId=${e}`
    },
    defaultHeaders:  {
      "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
      Referer: "https://m.music.migu.cn/"
    },
    getListDetailList(e,
    t,
    n = 0) {
      if(n>2)return Promise.reject(new Error("try max num"));
      return(0,
      o.tX)(this.getSongListDetailUrl(e,
      t),
       {
        headers: this.defaultHeaders
      }).promise.then(( {
        body: o
      }) = >o.code! =  = this.successCode?this.getListDetailList(e,
      t,
      ++n):  {
        list: Ye(o.data.songList),
        page: t,
        limit: this.limit_song,
        total: o.data.totalCount,
        source: "mg"
      })
    },
    getListDetailInfo(e,
    t = 0) {
      if(t>2)return Promise.reject(new Error("try max num"));
      if(this.cachedDetailInfo[e])return Promise.resolve(this.cachedDetailInfo[e]);
      return(0,
      o.tX)(`https://c.musicapp.migu.cn/MIGUM3.0/resource/playlist/v2.0?playlistId=${e}`,
       {
        headers: this.defaultHeaders
      }).promise.then(( {
        body: n
      }) = > {
        if(n.code! =  = this.successCode)return this.getListDetail(e,
        ++t);
        return this.cachedDetailInfo[e] =  {
          name: n.data.title,
          img: n.data.imgItem.img,
          desc: n.data.summary,
          author: n.data.ownerName,
          play_count: (0,
          s.pN)(n.data.opNumItem.playNum)
        }
      })
    },
    async getDetailUrl(e,
    t,
    n = 0) {
      if(n>3)return Promise.reject(new Error("link try max num"));
      const i = (0,
      o.tX)(e,
       {
        headers:  {
          "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 9_1 like Mac OS X) AppleWebKit/601.1.46 (KHTML, like Gecko) Version/9.0 Mobile/13B143 Safari/601.1",
          Referer: e
        }
      }),
       {
        headers:  {
          location: s
        },
        statusCode: a
      } = await i.promise;
      return a>400?this.getDetailUrl(e,
      t,
      ++n): s?(this.cachedUrl[e] = s,
      this.getListDetail(s,
      t)): Promise.reject(new Error("link get failed"))
    },
    getListDetail(e,
    t,
    n = 0) {
      if(/\/playlist[/?]/.test(e)) {
        if(e = /(?:playlistId|id)=(\d+)/.exec(e)?.[1],
        !e)throw new Error("list detail id parse failed")
      }else if(this.regExps.listDetailLink.test(e))e = e.replace(this.regExps.listDetailLink,
      "$1");
      else if(/[?&:/]/.test(e)) {
        const n = this.cachedUrl[e];
        return n?this.getListDetail(n,
        t): this.getDetailUrl(e,
        t)
      }return Promise.all([this.getListDetailList(e,
      t,
      n),
      this.getListDetailInfo(e,
      n)]).then(([e,
      t]) = >(e.info = t,
      e))
    },
    getList(e,
    t,
    n,
    i = 0) {
      return this._requestObj_list&&this._requestObj_list.cancelHttp(),
      i>2?Promise.reject(new Error("try max num")): (this._requestObj_list = (0,
      o.tX)(this.getSongListUrl(e,
      t,
      n),
       {
        headers: this.defaultHeaders
      }),
      this._requestObj_list.promise.then(( {
        body: o
      }) = > {
        if("000000"! =  = o.code)return this.getList(e,
        t,
        n,
        ++i);
        return {
          list: o.data.contents?this.filterList2(o.data.contents): this.filterList(o.data.contentItemList[1].itemList),
          total: 99999,
          page: n,
          limit: this.limit_list,
          source: "mg"
        }
      }))
    },
    filterList2(e,
    t = [],
    n = new Set) {
      for(const o of e)o.contents?this.filterList2(o.contents,
      t,
      n): "2021"! = o.resType||n.has(o.resId)||(n.add(o.resId),
      t.push( {
        id: String(o.resId),
        author: "",
        name: o.txt,
        img: o.img,
        desc: o.txt2,
        source: "mg"
      }));
      return t
    },
    filterList: e = >e.map(e = >( {
      play_count: e.barList[0]?.title,
      id: String(e.logEvent.contentId),
      author: "",
      name: e.title,
      img: e.imageUrl,
      desc: "",
      source: "mg"
    })),
    getTag(e = 0) {
      return this._requestObj_tags&&this._requestObj_tags.cancelHttp(),
      e>2?Promise.reject(new Error("try max num")): (this._requestObj_tags = (0,
      o.tX)(this.tagsUrl,
       {
        headers: this.defaultHeaders
      }),
      this._requestObj_tags.promise.then(( {
        body: t
      }) = >t.code! =  = this.successCode?this.getTag(++e): this.filterTagInfo(t.data)))
    },
    filterTagInfo: e = >( {
      hotTag: e[0].content.map(( {
        texts: [e,
        t]
      }) = >( {
        id: t,
        name: e,
        source: "mg"
      })),
      tags: e.slice(1).map(( {
        header: e,
        content: t
      }) = >( {
        name: e.title,
        list: t.map(( {
          texts: [e,
          t]
        }) = >( {
          id: t,
          name: e,
          source: "mg"
        }))
      })),
      source: "mg"
    }),
    getTags() {
      return this.getTag()
    },
    getDetailPageUrl(e) {
      return/playlist\/index\.html\?/.test(e)?e=e.replace(/.*(?: \?|&)id = (\d+)(?: &.*|$)/,
      "$1"): this.regExps.listDetailLink.test(e)&&(e = e.replace(this.regExps.listDetailLink,
      "$1")),
      `https://music.migu.cn/v3/music/playlist/${e}`
    },
    filterSongListResult(e) {
      const t = [];
      return e.forEach(e = > {
        if(!e.id)return;
        const n = parseInt(e.playNum);
        t.push( {
          play_count: isNaN(n)?0: (0,
          s.pN)(n),
          id: e.id,
          author: e.userName,
          name: e.name,
          img: e.musicListPicUrl,
          total: e.musicNum,
          source: "mg"
        })
      }),
      t
    },
    search(e,
    t,
    n = 20) {
      const o = Date.now().toString(),
      i = et(o,
      e);
      return Ge(`https://jadeite.migu.cn/music_search/v3/search/searchAll?isCorrect=1&isCopyright=1&searchSwitch=%7B%22song%22%3A0%2C%22album%22%3A0%2C%22singer%22%3A0%2C%22tagSong%22%3A0%2C%22mvSong%22%3A0%2C%22bestShow%22%3A0%2C%22songlist%22%3A1%2C%22lyricSong%22%3A0%7D&pageSize=${n}&text=${encodeURIComponent(e)}&pageNo=${t}&sort=0&sid=USS`,
       {
        headers:  {
          uiVersion: "A_music_3.6.1",
          deviceId: i.deviceId,
          timestamp: o,
          sign: i.sign,
          channel: "0146921",
          "User-Agent": "Mozilla/5.0 (Linux; U; Android 11.0.0; zh-cn; MI 11 Build/OPR1.170623.032) AppleWebKit/534.30 (KHTML, like Gecko) Version/4.0 Mobile Safari/534.30"
        }
      }).