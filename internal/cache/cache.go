package cache

import "encoding/base64"

const Addr = "127.0.0.1:11211"

func Key(q string) string {
	return "s:" + base64.RawURLEncoding.EncodeToString([]byte(q))
}
