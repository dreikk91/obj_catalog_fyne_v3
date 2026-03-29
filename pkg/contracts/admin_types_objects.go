package contracts

// PPKConstructorItem - запис довідника "Конструктор ППК".
type PPKConstructorItem struct {
	ID        int64
	Name      string
	Channel   int64
	ZoneCount int64
}

// AdminObjectCard - дані картки об'єкта (вкладка "Об'єкт").
type AdminObjectCard struct {
	ObjUIN int64
	ObjN   int64
	GrpN   int64

	ShortName string
	FullName  string
	ObjTypeID int64
	ObjRegID  int64

	Address   string
	Phones    string
	Contract  string
	StartDate string
	Location  string
	Notes     string

	ChannelCode int64
	PPKID       int64 // ID з довідника PPK (без +100)
	GSMPhone1   string
	GSMPhone2   string
	GSMHiddenN  int64
	SubServerA  string
	SubServerB  string

	TestControlEnabled bool
	TestIntervalMin    int64
}

// AdminSubServer - запис підсервера (SBS).
type AdminSubServer struct {
	ID    int64
	Info  string
	Bind  string
	Host  string
	Type  int64
	Host2 string
}

// AdminSubServerObject - об'єкт з прив'язкою до підсервера (SBSA/SBSB).
type AdminSubServerObject struct {
	ObjN       int64
	Name       string
	Address    string
	SubServerA string
	SubServerB string
}

// AdminObjectPersonal - контакт/відповідальна особа по об'єкту (вкладка "В/О").
type AdminObjectPersonal struct {
	ID          int64
	SourceObjN  int64
	Number      int64
	Surname     string
	Name        string
	SecName     string
	Address     string
	Phones      string
	Position    string
	Notes       string
	IsTRKTester bool
	Access1     int64
	IsRang      bool
	ViberID     string
	TelegramID  string
	CreatedAt   string
}

// AdminObjectZone - запис шлейфу/зони об'єкта (вкладка "Зони").
type AdminObjectZone struct {
	ID            int64
	ZoneNumber    int64
	ZoneType      int64
	Description   string
	EntryDelaySec int64
}

// AdminObjectCoordinates - координати об'єкта (вкладка "Додатково").
type AdminObjectCoordinates struct {
	Latitude  string
	Longitude string
}

// AdminSIMPhoneUsage - використання SIM-номера в картках об'єктів.
type AdminSIMPhoneUsage struct {
	ObjN int64
	Name string
	Slot string
}
