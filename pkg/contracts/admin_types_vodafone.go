package contracts

import "time"

// VodafoneAuthState описує поточний локальний стан авторизації у Vodafone API.
type VodafoneAuthState struct {
	Phone          string
	Authorized     bool
	TokenExpiresAt time.Time
}

// VodafoneConnectivityStatus містить статус підключення SIM у мережі.
type VodafoneConnectivityStatus struct {
	OperationStatus   string
	SIMStatus         string
	BaseStationStatus string
	LBSStatusKey      string
	ConnectionTime    time.Time
	ConnectionTimeRaw string
}

// VodafoneLastEvent містить інформацію про останню подію по номеру.
type VodafoneLastEvent struct {
	CallType     string
	EventTime    time.Time
	EventTimeRaw string
}

// VodafoneSIMBlockingStatus містить поточний стан блокування номера.
type VodafoneSIMBlockingStatus struct {
	Status                 string
	BlockingDate           time.Time
	BlockingDateRaw        string
	BlockingRequestDate    time.Time
	BlockingRequestDateRaw string
	UpdateDate             time.Time
	UpdateDateRaw          string
}

// VodafoneSIMStatus об'єднує ознаки доступності SIM, статус у мережі та останню подію.
type VodafoneSIMStatus struct {
	MSISDN            string
	Available         bool
	SubscriberName    string
	SubscriberComment string
	Connectivity      VodafoneConnectivityStatus
	Blocking          VodafoneSIMBlockingStatus
	LastEvent         VodafoneLastEvent
}

// VodafoneSIMRebootResult містить результат постановки заявки на перезавантаження SIM.
type VodafoneSIMRebootResult struct {
	OrderID string
	State   string
}

// VodafoneSIMBarringResult містить результат заявки на блокування або розблокування номера.
type VodafoneSIMBarringResult struct {
	OrderID   string
	State     string
	Operation string
}
