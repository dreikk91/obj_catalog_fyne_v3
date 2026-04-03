package contracts

import "time"

// KyivstarAuthState описує локальний стан авторизації Kyivstar IoT API.
type KyivstarAuthState struct {
	ClientID       string
	UserEmail      string
	Configured     bool
	Authorized     bool
	TokenExpiresAt time.Time
}

// KyivstarSIMServiceStatus описує стан окремого сервісу номера.
type KyivstarSIMServiceStatus struct {
	ServiceID        string
	Name             string
	Status           string
	AvailableActions []string
}

// KyivstarSIMStatus містить агрегований стан SIM Kyivstar та базову довідкову інформацію.
type KyivstarSIMStatus struct {
	MSISDN           string
	Available        bool
	NumberStatus     string
	AvailableActions []string
	DeviceName       string
	DeviceID         string
	ICCID            string
	IMEI             string
	TariffPlan       string
	Account          string
	DataUsage        string
	SMSUsage         string
	VoiceUsage       string
	IsTestPeriod     bool
	IsOnline         bool
	Services         []KyivstarSIMServiceStatus
}

// KyivstarSIMOperationResult містить результат зміни стану номера або сервісів.
type KyivstarSIMOperationResult struct {
	MSISDN    string
	Operation string
}

// KyivstarSIMResetResult містить результат постановки запиту на reset SIM.
type KyivstarSIMResetResult struct {
	MSISDN string
	Email  string
}
