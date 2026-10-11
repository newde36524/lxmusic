package lx

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/url"
	"strings"
)

/*
The constants below are lifted verbatim from dist/renderer.js of
lx-music-desktop 2.12.6 (module that hosts the "wy"/"tx" providers).
*/

const (
	// NetEase eapi
	neteaseEAPIKey    = "e82ckenh8dichen8"
	neteaseEAPIMarker = "36cd479b6b5"

	// NetEase weapi
	neteaseWeAPIKey = "0CoJUm6Qyw8W8jud"
)

var neteaseWeAPIIV = []byte("0102030405060708")

const neteaseRSAPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDgtQn2JZ34ZC28NWYpAUd98iZ37BUrX/aKzmFbt7clFSs6sXqHauqKWqdtLkF2KexO40H1YTX8z2lSgBBOAxLsvaklV8k4cBFK9snQXE9/DDaFt6Rr7iVZMldczhC0JNgTz+SHXT6CBHuX3e9SdB1Ua44oncaTWz7OBGLbCiK45wIDAQAB
-----END PUBLIC KEY-----`

// Kugou sign keys: android client vs web client.
const (
	KugouKeyAndroid = "OIlwieks28dk2k092lksi2UIkp"
	KugouKeyWeb     = "NVPh5oo715z5DIWAeQlhMDsWXXQV4hwt"
)

/*
Tencent musics.fcg signature tables.
*/
var (
	txIndexA = []int{23, 14, 6, 36, 16, 40, 7, 19}
	txIndexB = []int{16, 1, 32, 12, 19, 27, 8, 5}
	txXOR    = []int{89, 39, 179, 150, 218, 82, 58, 252, 177, 52, 186, 123,
		120, 64, 242, 133, 143, 161, 121, 179}
)

/* ---------------------------------------------------------------- helpers */

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func sha1Hex(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// aesECBPKCS7 encrypts with AES-ECB (Node's crypto has no ECB either, so both
// sides roll their own per-block loop).
func aesECBPKCS7(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize()
	padded := pkcs7Pad(plaintext, bs)
	out := make([]byte, len(padded))
	for i := 0; i < len(padded); i += bs {
		block.Encrypt(out[i:i+bs], padded[i:i+bs])
	}
	return out, nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	n := blockSize - len(data)%blockSize
	if n == 0 {
		n = blockSize
	}
	return append(data, bytesOf(byte(n), n)...)
}

func bytesOf(v byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = v
	}
	return b
}

func aesCBCEncrypt(plaintext, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padded := pkcs7Pad(plaintext, block.BlockSize())
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, padded)
	return out, nil
}

func mustParseNeteaseRSA() *rsa.PublicKey {
	block, _ := pem.Decode([]byte(neteaseRSAPublicKeyPEM))
	if block == nil {
		panic("bad pem")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		panic("not rsa")
	}
	return rsaPub
}

// rsaEncryptRaw is RSA with no padding (Node's RSA_NO_PADDING): c = m^e mod n.
// The standard library only offers PKCS1v15, so the modpow is done by hand.
func rsaEncryptRaw(pub *rsa.PublicKey, msg []byte) ([]byte, error) {
	if len(msg) == 0 || len(msg) > pub.N.BitLen()/8 {
		return nil, fmt.Errorf("rsa: message length %d out of range for a %d-bit key", len(msg), pub.N.BitLen())
	}
	m := new(big.Int).SetBytes(msg)
	if m.Cmp(pub.N) >= 0 {
		return nil, fmt.Errorf("rsa: message larger than modulus")
	}
	e := new(big.Int).SetInt64(int64(pub.E))
	ct := new(big.Int).Exp(m, e, pub.N)
	out := make([]byte, pub.N.BitLen()/8)
	ct.FillBytes(out)
	return out, nil
}

/* -------------------------------------------------------------- NetEase API */

// NeteaseEAPIForm builds the `params` form field for interface.music.163.com/eapi/batch.
//
// Original:
//
//	o = `nobody${path}use${json}md5forencrypt`
//	i = `${path}-36cd479b6b5-${json}-36cd479b6b5-${md5(o)}`
//	params = AES-128-ECB(i, "e82ckenh8dichen8").hex.toUpperCase()
func NeteaseEAPIForm(path string, data any) (url.Values, error) {
	var jsonStr string
	if s, ok := data.(string); ok {
		jsonStr = s
	} else {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		jsonStr = string(b)
	}
	o := "nobody" + path + "use" + jsonStr + "md5forencrypt"
	plain := fmt.Sprintf("%s-%s-%s-%s-%s", path, neteaseEAPIMarker, jsonStr, neteaseEAPIMarker, md5Hex(o))
	enc, err := aesECBPKCS7([]byte(plain), []byte(neteaseEAPIKey))
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("params", strings.ToUpper(hex.EncodeToString(enc)))
	return form, nil
}

// NeteaseWeAPIForm builds the `params` + `encSecKey` form fields used by
// music.163.com/weapi/*.
//
// Original:
//
//	n = randomBytes(16).map(b => "abc...9".charAt(b%62).charCodeAt())
//	params    = base64(AES-CBC(base64(AES-CBC(json, Ce, IV)), n, IV))
//	encSecKey = hex(RSA_nopad(leftZeroPad(reverse(n), 128)))
func NeteaseWeAPIForm(data any) (url.Values, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	n := make([]byte, 16)
	if _, err := rand.Read(n); err != nil {
		return nil, err
	}
	for i, v := range n {
		n[i] = alphabet[v%62]
	}

	// first layer: AES-CBC(json, key=Ce, iv=Se) -> base64
	l1, err := aesCBCEncrypt(b, []byte(neteaseWeAPIKey), neteaseWeAPIIV)
	if err != nil {
		return nil, err
	}
	mid := base64.StdEncoding.EncodeToString(l1)

	// second layer: AES-CBC(base64, key=n, iv=Se) -> base64
	l2, err := aesCBCEncrypt([]byte(mid), n, neteaseWeAPIIV)
	if err != nil {
		return nil, err
	}
	params := base64.StdEncoding.EncodeToString(l2)

	// encSecKey: reverse n, left-pad to 128 bytes, RSA raw encrypt
	reversed := make([]byte, 16)
	for i, v := range n {
		reversed[15-i] = v
	}
	padded := make([]byte, 128)
	copy(padded[128-16:], reversed)
	pub := mustParseNeteaseRSA()
	ct, err := rsaEncryptRaw(pub, padded)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("params", params)
	form.Set("encSecKey", hex.EncodeToString(ct))
	return form, nil
}

/* ------------------------------------------------------------- Tencent API */

// TencentSign computes the `sign` query parameter of u.y.qq.com/cgi-bin/musics.fcg.
//
// Original (renderer.js):
//
//	t = sha1(payload).hex                       // 40 hex chars
//	n = t[_e.map]    o = t[me.map]               // _e contains index 40 -> undefined, dropped by join
//	i = he.map((v,k) => v ^ parseInt(t.slice(2k,2k+2),16))
//	sign = (`zzc${n}${base64(i) without /+=}${o}`).toLowerCase()
func TencentSign(payload []byte) string {
	t := sha1Hex(string(payload))
	var sb strings.Builder
	sb.WriteString("zzc")
	for _, idx := range txIndexA {
		if idx >= 0 && idx < len(t) { // index 40 is intentionally out of range
			sb.WriteByte(t[idx])
		}
	}
	i := make([]byte, len(txXOR))
	for k, v := range txXOR {
		i[k] = byte(v ^ hexPair(t[2*k:2*k+2]))
	}
	sb.WriteString(strings.NewReplacer("/", "", "+", "", "=", "").Replace(base64.StdEncoding.EncodeToString(i)))
	for _, idx := range txIndexB {
		if idx >= 0 && idx < len(t) {
			sb.WriteByte(t[idx])
		}
	}
	return strings.ToLower(sb.String())
}

// hexPair parses the two hex chars at one position of the sha1 digest as a
// number. Node uses parseInt(t.slice(2k,2k+2),16), i.e. a hex numeral, not the
// two ASCII bytes interpreted as a 16-bit integer.
func hexPair(s string) int {
	n := func(c byte) int {
		switch {
		case c >= '0' && c <= '9':
			return int(c - '0')
		case c >= 'a' && c <= 'f':
			return int(c-'a') + 10
		default:
			return int(c-'A') + 10
		}
	}
	return n(s[0])*16 + n(s[1])
}

/* ------------------------------------------------------------------ Kugou */

// KugouToken builds the kugou `key`/`token` md5 used by song_search and
// media.store endpoints.
func KugouToken(hash, key string) string { return md5Hex(hash + key) }

// KugouSignedParam replicates the desktop app's `$(params, platform, extra)`
// helper: md5 of `key + sorted(params) + extra + key`.
func KugouSignedParam(params, platform, extra string) string {
	key := KugouKeyAndroid
	if platform == "web" {
		key = KugouKeyWeb
	}
	parts := strings.Split(params, "&")
	// JS Array.sort() is a lexicographic sort of the whole string
	for i := 1; i < len(parts); i++ {
		for j := i; j > 0 && parts[j] < parts[j-1]; j-- {
			parts[j-1], parts[j] = parts[j], parts[j-1]
		}
	}
	return md5Hex(key + strings.Join(parts, "") + extra + key)
}

/* --------------------------------------------------------------- utilities */
