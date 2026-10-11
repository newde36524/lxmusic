package lx

import "testing"

// TestKugouEntries covers the gateway's unstable data field. A hash the server
// knows is [[{...}]]; one it does not know is [{}], a one-level array holding a
// single empty object. That empty reply is the server's way of saying "nothing
// here", so it must come back as zero entries with a nil error - the gateway
// returns error_code 0 with it - rather than as a Go unmarshal error leaking
// into the output when a user pastes a truncated id.
func TestKugouEntries(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
		err  bool
	}{
		// The live shape for 周杰伦 - 晴天, 2026-10-10.
		{"nested one row", `[[{"songname":"晴天","author_name":"周杰伦","audio_info":{"hash":"B3A52A7A","filesize":"4317292"},"album_info":{"album_id":"966846"}}]]`, 1, false},
		{"nested several rows", `[[{"songname":"a"},{"songname":"b"}]]`, 2, false},
		{"unknown hash", `[{}]`, 0, false},
		{"empty array", `[]`, 0, false},
		{"empty nested", `[[]]`, 0, false},
		{"null", `null`, 0, false},
		{"empty raw", ``, 0, false},
		{"whitespace only", `  `, 0, false},
		// A padding row before the real one.
		{"leading empty row", `[{},{"songname":"晴"}]`, 1, false},
		{"flat one row", `[{"songname":"晴"}]`, 1, false},
		{"malformed", `not json`, 0, true},
		{"number", `123`, 0, true},
		{"object", `{"data":1}`, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := kugouEntries([]byte(tt.raw))
			if (err != nil) != tt.err {
				t.Fatalf("kugouEntries(%q) error = %v, want error %v", tt.raw, err, tt.err)
			}
			if tt.err {
				return
			}
			if len(got) != tt.want {
				t.Fatalf("kugouEntries(%q) = %d entries, want %d", tt.raw, len(got), tt.want)
			}
			if tt.want > 0 {
				if got[0].SongName == "" {
					t.Fatalf("kugouEntries(%q) kept an entry with no song name", tt.raw)
				}
			}
		})
	}
}

// TestKugouEntriesFields checks that the nested and flat shapes decode into the
// same fields, so the two shapes are really interchangeable.
func TestKugouEntriesFields(t *testing.T) {
	row := `{"songname":"晴天","author_name":"周杰伦","audio_info":{"hash":"B3A52A7A958BF0AED0EBFBA2E9A818B7","filesize":"4317292","hash_128":"B3A52A7A958BF0AED0EBFBA2E9A818B7","filesize_128":"4317292","hash_320":"1B56126A8A03924F1DD066259C095CBC","filesize_320":"10792943","hash_flac":"0A69169202DE95AAF24A9944CCF0730D","filesize_flac":"31633244","hash_high":"D667BC5EE93F201126697CB3BA745009","filesize_high":"55397039"},"album_info":{"album_id":"966846","album_name":"叶惠美"}}`
	for _, raw := range []string{"[[" + row + "]]", "[" + row + "]"} {
		got, err := kugouEntries([]byte(raw))
		if err != nil {
			t.Fatalf("kugouEntries(%q) error: %v", raw, err)
		}
		if len(got) != 1 {
			t.Fatalf("kugouEntries(%q) = %d entries, want 1", raw, len(got))
		}
		e := got[0]
		if e.SongName != "晴天" {
			t.Errorf("songname = %q", e.SongName)
		}
		if e.AuthorName != "周杰伦" {
			t.Errorf("author_name = %q", e.AuthorName)
		}
		if e.AudioInfo.Hash128 != "B3A52A7A958BF0AED0EBFBA2E9A818B7" {
			t.Errorf("hash_128 = %q", e.AudioInfo.Hash128)
		}
		if e.AudioInfo.FilesizeHigh != "55397039" {
			t.Errorf("filesize_high = %q", e.AudioInfo.FilesizeHigh)
		}
		if e.AlbumInfo.AlbumID != "966846" {
			t.Errorf("album_id = %q", e.AlbumInfo.AlbumID)
		}
		if e.AlbumInfo.AlbumName != "叶惠美" {
			t.Errorf("album_name = %q", e.AlbumInfo.AlbumName)
		}
	}
}
