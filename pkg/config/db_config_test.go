package config

import "testing"

func TestNormalizeBackendMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "firebird default", input: "", want: BackendModeFirebird},
		{name: "firebird explicit", input: BackendModeFirebird, want: BackendModeFirebird},
		{name: "phoenix explicit", input: BackendModePhoenix, want: BackendModePhoenix},
		{name: "phoenix mixed case", input: "  PHOENIX  ", want: BackendModePhoenix},
		{name: "casl explicit", input: BackendModeCASLCloud, want: BackendModeCASLCloud},
		{name: "casl mixed case", input: "  CASL_CLOUD  ", want: BackendModeCASLCloud},
		{name: "unknown fallback", input: "unknown", want: BackendModeFirebird},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeBackendMode(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeBackendMode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDBConfig_NormalizedMode(t *testing.T) {
	t.Parallel()

	cfg := DBConfig{Mode: BackendModeCASLCloud}
	if got := cfg.NormalizedMode(); got != BackendModeCASLCloud {
		t.Fatalf("NormalizedMode() = %q, want %q", got, BackendModeCASLCloud)
	}

	cfg.Mode = BackendModePhoenix
	if got := cfg.NormalizedMode(); got != BackendModePhoenix {
		t.Fatalf("NormalizedMode() = %q, want %q", got, BackendModePhoenix)
	}

	cfg.Mode = "invalid"
	if got := cfg.NormalizedMode(); got != BackendModeFirebird {
		t.Fatalf("NormalizedMode() fallback = %q, want %q", got, BackendModeFirebird)
	}
}

func TestDBConfig_PhoenixDSN(t *testing.T) {
	t.Parallel()

	cfg := DBConfig{
		PhoenixUser:     "sa",
		PhoenixPassword: "",
		PhoenixHost:     "localhost",
		PhoenixInstance: "PHOENIX4",
		PhoenixDatabase: "Pult4DB",
		PhoenixParams:   "encrypt=disable&trustservercertificate=true",
	}

	got := cfg.PhoenixDSN()
	want := "sqlserver://sa@localhost?database=Pult4DB&dial+timeout=5&encrypt=disable&instance=PHOENIX4&keepalive=5&trustservercertificate=true"
	if got != want {
		t.Fatalf("PhoenixDSN() = %q, want %q", got, want)
	}
}

func TestDBConfig_PhoenixDSNPreservesConfiguredTransportTimeouts(t *testing.T) {
	t.Parallel()

	cfg := DBConfig{
		PhoenixHost:   "phoenix.local",
		PhoenixParams: "dial timeout=9&keepalive=12",
	}

	got := cfg.PhoenixDSN()
	want := "sqlserver://phoenix.local?dial+timeout=9&keepalive=12"
	if got != want {
		t.Fatalf("PhoenixDSN() = %q, want %q", got, want)
	}
}

func TestLoadDBConfigTrimsAndFallsBackForRequiredFields(t *testing.T) {
	t.Parallel()

	prefs := memoryPreferences{
		strings: map[string]string{
			PrefUser:            " SYSDBA ",
			PrefHost:            " ",
			PrefPort:            "",
			PrefPath:            "\t",
			PrefParams:          " charset=WIN1251 ",
			PrefPhoenixHost:     "",
			PrefPhoenixInstance: " ",
			PrefPhoenixDatabase: "\n",
		},
	}

	cfg := LoadDBConfig(prefs)
	if cfg.User != "SYSDBA" {
		t.Fatalf("User = %q, want trimmed SYSDBA", cfg.User)
	}
	if cfg.Host != "localhost" {
		t.Fatalf("Host = %q, want localhost fallback", cfg.Host)
	}
	if cfg.Port != "3050" {
		t.Fatalf("Port = %q, want 3050 fallback", cfg.Port)
	}
	if cfg.Path != "C:/MOST.PM/BASE/MOST5.FDB" {
		t.Fatalf("Path = %q, want default path fallback", cfg.Path)
	}
	if cfg.Params != "charset=WIN1251" {
		t.Fatalf("Params = %q, want trimmed params", cfg.Params)
	}
	if cfg.PhoenixHost != "localhost" || cfg.PhoenixInstance != "PHOENIX4" || cfg.PhoenixDatabase != "Pult4DB" {
		t.Fatalf("Phoenix fallbacks not applied: %+v", cfg)
	}
}

func TestSaveAndLoadDBConfigPreservesPhoenixOperatorSettings(t *testing.T) {
	prefs := memoryPreferences{
		strings: map[string]string{},
		bools:   map[string]bool{},
		ints:    map[string]int{},
		floats:  map[string]float64{},
	}
	want := DBConfig{
		PhoenixControlHost:      "10.32.1.200",
		PhoenixOperatorID:       42,
		PhoenixOperatorName:     "Operator",
		PhoenixOperatorPassword: "secret",
		PhoenixClientRole:       PhoenixClientRoleAdministrator,
	}

	SaveDBConfig(prefs, want)
	got := LoadDBConfig(prefs)

	if got.PhoenixControlHost != want.PhoenixControlHost ||
		got.PhoenixOperatorID != want.PhoenixOperatorID ||
		got.PhoenixOperatorName != want.PhoenixOperatorName ||
		got.PhoenixOperatorPassword != want.PhoenixOperatorPassword ||
		got.PhoenixClientRole != want.PhoenixClientRole {
		t.Fatalf("Phoenix operator settings = %+v, want %+v", got, want)
	}
}

func TestPhoenixLoginConfigured(t *testing.T) {
	complete := DBConfig{
		PhoenixControlHost:      "10.32.1.200",
		PhoenixOperatorID:       3,
		PhoenixOperatorPassword: "secret",
	}
	if !PhoenixLoginConfigured(complete) {
		t.Fatal("complete Phoenix login was not recognized")
	}
	complete.PhoenixOperatorPassword = ""
	if PhoenixLoginConfigured(complete) {
		t.Fatal("Phoenix login without password must be incomplete")
	}
}

func TestNormalizePhoenixClientRole(t *testing.T) {
	if got := NormalizePhoenixClientRole("admin"); got != PhoenixClientRoleAdministrator {
		t.Fatalf("admin role = %q", got)
	}
	if got := NormalizePhoenixClientRole(""); got != PhoenixClientRoleDuty {
		t.Fatalf("default role = %q", got)
	}
}

func TestDBConfigFirebirdDSNUsesTrimmedHostPortAndPath(t *testing.T) {
	t.Parallel()

	cfg := DBConfig{
		User:     " SYSDBA ",
		Password: "masterkey",
		Host:     " 10.32.1.101 ",
		Port:     " 3050 ",
		Path:     " C:/MOST.PM/BASE/MOST5.FDB ",
		Params:   " charset=WIN1251&auth_plugin_name=Srp ",
	}

	got := cfg.FirebirdDSN()
	want := "SYSDBA:masterkey@10.32.1.101:3050/C:/MOST.PM/BASE/MOST5.FDB?charset=WIN1251&auth_plugin_name=Srp"
	if got != want {
		t.Fatalf("FirebirdDSN() = %q, want %q", got, want)
	}
}

func TestDBConfigFirebirdDSNUsesIPv4LoopbackForLocalhost(t *testing.T) {
	t.Parallel()

	cfg := DBConfig{
		User:     "SYSDBA",
		Password: "masterkey",
		Host:     "localhost",
		Port:     "3050",
		Path:     "C:/MOST.PM/BASE/MOST5.FDB",
	}

	got := cfg.FirebirdDSN()
	want := "SYSDBA:masterkey@127.0.0.1:3050/C:/MOST.PM/BASE/MOST5.FDB"
	if got != want {
		t.Fatalf("FirebirdDSN() = %q, want %q", got, want)
	}
}

type memoryPreferences struct {
	strings map[string]string
	bools   map[string]bool
	ints    map[string]int
	floats  map[string]float64
}

func (p memoryPreferences) BoolWithFallback(key string, fallback bool) bool {
	if value, ok := p.bools[key]; ok {
		return value
	}
	return fallback
}

func (p memoryPreferences) FloatWithFallback(key string, fallback float64) float64 {
	if value, ok := p.floats[key]; ok {
		return value
	}
	return fallback
}

func (p memoryPreferences) IntWithFallback(key string, fallback int) int {
	if value, ok := p.ints[key]; ok {
		return value
	}
	return fallback
}

func (p memoryPreferences) String(key string) string {
	return p.StringWithFallback(key, "")
}

func (p memoryPreferences) StringWithFallback(key string, fallback string) string {
	if value, ok := p.strings[key]; ok {
		return value
	}
	return fallback
}

func (p memoryPreferences) SetBool(key string, value bool) {
	if p.bools != nil {
		p.bools[key] = value
	}
}

func (p memoryPreferences) SetFloat(key string, value float64) {
	if p.floats != nil {
		p.floats[key] = value
	}
}

func (p memoryPreferences) SetInt(key string, value int) {
	if p.ints != nil {
		p.ints[key] = value
	}
}

func (p memoryPreferences) SetString(key string, value string) {
	if p.strings != nil {
		p.strings[key] = value
	}
}
