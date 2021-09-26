package redigo

import "time"

func formatExpirationArgs(ttl time.Duration) []interface{} {
	if ttl == 0 {
		return []interface{}{}
	}

	var opt string
	var t int64

	if isPX(ttl) {
		opt = "PX"
		t = int64(ttl / time.Millisecond)
	} else {
		opt = "EX"
		t = int64(ttl / time.Second)
	}

	// Assume 1 if less than 0 after conversion.
	if t < 1 {
		t = 1
	}

	return []interface{}{opt, t}
}

func isPX(ttl time.Duration) bool {
	return ttl < time.Second || ttl%time.Second != 0
}
