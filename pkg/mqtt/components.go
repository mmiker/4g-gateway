package mqtt

type component string

// Component names for debug output
const (
	NET component = "[net]     "
	PNG component = "[pinger]  "
	CLI component = "[client]  "
	DEC component = "[decode]  "
	MES component = "[message] "
	STR component = "[store]   "
	MID component = "[msgids]  "
	TST component = "[test]    "
	STA component = "[state]   "
	ERR component = "[error]   "
	ROU component = "[router]  "
)
