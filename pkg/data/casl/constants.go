package casl

import (
	"time"
)

// CASLConstants hold system-wide CASL configuration and limits.
const (
	CommandPath    = "/command"
	LoginPath      = "/login"
	DefaultBaseURL = "http://127.0.0.1:50003"

	HTTPTimeout       = 12 * time.Second
	ObjectsCacheTTL   = 20 * time.Second
	UsersCacheTTL     = 5 * time.Minute
	ObjectEventsTTL   = 10 * time.Second
	ObjectEventsSpan  = 7 * 24 * time.Hour
	JournalEventsSpan = 72 * time.Hour
	StatsSpan         = 30 * 24 * time.Hour
	DictionaryTTL     = 15 * time.Minute
	TranslatorTTL     = 15 * time.Minute
	ProbeEventsSpan   = 2 * time.Minute
	RealtimeBackoff   = 10 * time.Second

	MaxCachedEvents = 2000
	ReadLimit       = 100000
	DebugBodyLimit  = 8192

	ObjectStatusText = "НОРМА"

	ObjectIDNamespaceStart = 1_500_000_000
	ObjectIDNamespaceEnd   = 1_999_999_999
	ObjectIDNamespaceSize  = ObjectIDNamespaceEnd - ObjectIDNamespaceStart + 1
)

const (
	CaptchaShowPath  = "/captchaShow"
	TimeServerPath   = "/get_time_server"
	SubscribePath    = "/subscribe"
	DefaultPageLimit = 1000
)
