package contracts

type AMISettings struct {
	Enabled   bool   `json:"enabled"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Username  string `json:"username"`
	Secret    string `json:"secret"`
	Extension string `json:"extension"`
	Context   string `json:"context"`
}

type AMISettingsProvider interface {
	GetAMISettings() AMISettings
	SaveAMISettings(AMISettings) error
}
