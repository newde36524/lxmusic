package tui

// Terminal escape decoding, shared by every unix platform. It is pure byte
// logic over the bytes term_unix returns, so it carries no build tag and is
// therefore covered by the test suite on any host, including Windows.
// Windows does not use it: the console reports virtual key codes directly.

var keyBuf []byte

// scan pulls the next logical key out of a chunk of raw input, returning the
// bytes left over. ok=false means the chunk is truncated mid-sequence and
// more input must be read before deciding. It is only false for a genuinely
// cut-off escape: an empty or all-discarded buffer is a verdict, not a stall.
func scan(buf []byte) (Key, []byte, bool) {
	for {
		if len(buf) == 0 {
			return KeyNone, buf, true
		}
		c := buf[0]
		if c == 0x1b {
			if len(buf) < 2 {
				return KeyNone, buf, false // incomplete, wait for the rest
			}
			switch buf[1] {
			case '[':
				k, rest, ok := scanCSI(buf)
				return k, rest, ok
			case 'O': // application-mode arrows, as tmux emits
				if len(buf) < 3 {
					return KeyNone, buf, false
				}
				switch buf[2] {
				case 'A':
					return KeyUp, buf[3:], true
				case 'B':
					return KeyDown, buf[3:], true
				}
				return KeyNone, buf[3:], true
			}
			return KeyQuit, buf[1:], true // a bare ESC
		}
		switch c {
		case '\r', '\n':
			return KeyEnter, buf[1:], true
		case 3, 4: // Ctrl+C / Ctrl+D
			return KeyQuit, buf[1:], true
		case 'q', 'Q':
			return KeyQuit, buf[1:], true
		case 'k', 'K':
			return KeyUp, buf[1:], true
		case 'j', 'J':
			return KeyDown, buf[1:], true
		case 'g', 'G':
			return KeyPageUp, buf[1:], true
		case 'y', 'Y':
			return KeyYes, buf[1:], true
		case 'n', 'N':
			return KeyNo, buf[1:], true
		default:
			buf = buf[1:]
		}
	}
}

// scanCSI parses one "\x1b[paramsF" sequence. It returns ok=false while the
// sequence is still arriving, so a truncated read is completed rather than
// mistaken for an ordinary 'x' byte.
func scanCSI(buf []byte) (Key, []byte, bool) {
	i := 2 // skip ESC and '['
	for i < len(buf) && buf[i] >= '0' && buf[i] <= '9' {
		i++
	}
	for i < len(buf) && buf[i] == ';' {
		i++
	}
	for i < len(buf) && buf[i] >= '0' && buf[i] <= '9' {
		i++
	}
	if i >= len(buf) {
		return KeyNone, buf, false
	}
	f := buf[i]
	rest := buf[i+1:]
	switch {
	case f == 'A':
		return KeyUp, rest, true
	case f == 'B':
		return KeyDown, rest, true
	case f == '~':
		switch string(buf[2:i]) {
		case "1", "5", "7": // home, page-up
			return KeyPageUp, rest, true
		case "4", "6", "8": // end, page-down
			return KeyPageDown, rest, true
		}
	}
	return KeyNone, rest, true
}
