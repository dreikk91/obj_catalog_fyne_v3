package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// Значення нижче можуть бути перевизначені на етапі збірки через -ldflags:
// -X obj_catalog_fyne_v3/pkg/version.Version=v1.2.3
// -X obj_catalog_fyne_v3/pkg/version.Commit=<sha>
// -X obj_catalog_fyne_v3/pkg/version.BuildTime=<RFC3339>
var (
	Version   = "dev"
	Commit    = ""
	BuildTime = ""
)

// Info містить версію/ревізію поточного білду.
type Info struct {
	Version   string
	Commit    string
	BuildTime string
	Dirty     bool
	GoVersion string
}

func shortSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// Current повертає фактичні версійні дані білду.
// Порядок пріоритетів:
// 1) Значення з ldflags (Version/Commit/BuildTime)
// 2) Автоматичні VCS-налаштування з debug.ReadBuildInfo()
func Current() Info {
	info := Info{
		Version:   strings.TrimSpace(Version),
		Commit:    strings.TrimSpace(Commit),
		BuildTime: strings.TrimSpace(BuildTime),
		GoVersion: runtime.Version(),
	}

	if bi, ok := debug.ReadBuildInfo(); ok {
		if strings.TrimSpace(bi.GoVersion) != "" {
			info.GoVersion = strings.TrimSpace(bi.GoVersion)
		}
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if info.Commit == "" {
					info.Commit = strings.TrimSpace(s.Value)
				}
			case "vcs.time":
				if info.BuildTime == "" {
					info.BuildTime = strings.TrimSpace(s.Value)
				}
			case "vcs.modified":
				if strings.EqualFold(strings.TrimSpace(s.Value), "true") {
					info.Dirty = true
				}
			}
		}
	}

	if info.Version == "" {
		info.Version = "dev"
	}
	if strings.EqualFold(info.Version, "dev") && info.Commit != "" {
		info.Version = "dev-" + shortSHA(info.Commit)
	}
	if info.Dirty && !strings.Contains(strings.ToLower(info.Version), "dirty") {
		info.Version += "+dirty"
	}

	return info
}

// Label повертає коротке представлення для заголовків UI.
func (i Info) Label() string {
	v := strings.TrimSpace(i.Version)
	if v == "" {
		v = "dev"
	}
	return v
}

// String повертає компактний рядок версії для логів.
func (i Info) String() string {
	parts := []string{i.Label()}
	if sha := shortSHA(i.Commit); sha != "" && !strings.Contains(strings.ToLower(i.Label()), sha) {
		parts = append(parts, sha)
	}
	return strings.Join(parts, " ")
}

// FullText повертає детальну інформацію для діалогу "Про версію".
func (i Info) FullText() string {
	commit := strings.TrimSpace(i.Commit)
	if commit == "" {
		commit = "невідомо"
	}
	buildTime := strings.TrimSpace(i.BuildTime)
	if buildTime == "" {
		buildTime = "невідомо"
	}

	dirty := "ні"
	if i.Dirty {
		dirty = "так"
	}

	return fmt.Sprintf(
		"Версія: %s\nCommit: %s\nDirty: %s\nЧас збірки: %s\nGo: %s",
		i.Label(),
		commit,
		dirty,
		buildTime,
		i.GoVersion,
	)
}
