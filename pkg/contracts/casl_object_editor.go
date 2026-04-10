package contracts

import "context"

// CASLPhoneNumber описує номер телефону користувача в CASL.
type CASLPhoneNumber struct {
	Active bool
	Number string
}

// CASLUserProfile описує користувача CASL.
type CASLUserProfile struct {
	UserID       string
	Email        string
	LastName     string
	FirstName    string
	MiddleName   string
	Role         string
	Tag          string
	PhoneNumbers []CASLPhoneNumber
}

// CASLPultRef описує пульт CASL.
type CASLPultRef struct {
	PultID   string
	Name     string
	Nickname string
}

// CASLRoomUserLink описує прив'язку користувача до приміщення.
type CASLRoomUserLink struct {
	UserID   string
	Priority int
	HozNum   string
}

// CASLRoomLineLink описує прив'язку зони до приміщення.
type CASLRoomLineLink struct {
	LineNumber    int
	AdapterType   string
	GroupNumber   int
	AdapterNumber int
}

// CASLRoomDetails описує приміщення об'єкта.
type CASLRoomDetails struct {
	RoomID      string
	Name        string
	Description string
	Images      []string
	RTSP        string
	Users       []CASLRoomUserLink
	Lines       []CASLRoomLineLink
}

// CASLDeviceLineDetails описує шлейф/зону обладнання.
type CASLDeviceLineDetails struct {
	LineID        *int64
	LineNumber    int
	GroupNumber   int
	AdapterType   string
	AdapterNumber int
	Description   string
	LineType      string
	IsBlocked     bool
	RoomID        string
}

// CASLDeviceDetails описує обладнання CASL.
type CASLDeviceDetails struct {
	DeviceID          string
	ObjID             string
	Number            int64
	Name              string
	Type              string
	Timeout           int64
	SIM1              string
	SIM2              string
	TechnicianID      string
	Units             string
	Requisites        string
	ChangeDate        int64
	ReglamentDate     int64
	MoreAlarmTime     []any
	IgnoringAlarmTime []any
	LicenceKey        string
	PasswRemote       string
	LastPingDate      int64
	Lines             []CASLDeviceLineDetails
}

// CASLGuardObjectDetails описує повний знімок охоронного об'єкта.
type CASLGuardObjectDetails struct {
	ObjID          string
	Name           string
	Address        string
	Lat            string
	Long           string
	Description    string
	PultID         string
	ReactingPultID string
	Contract       string
	ManagerID      string
	Note           string
	StartDate      int64
	ObjectType     string
	IDRequest      string
	GeoZoneID      int64
	BusinessCoeff  *float64
	Rooms          []CASLRoomDetails
	Device         CASLDeviceDetails
	ObjectStatus   string
	DeviceBlocked  bool
	BlockMessage   string
	TimeUnblock    int64
	Images         []string
}

// CASLObjectEditorSnapshot містить дані, потрібні для редагування CASL-об'єкта.
type CASLObjectEditorSnapshot struct {
	Object     CASLGuardObjectDetails
	Users      []CASLUserProfile
	Pults      []CASLPultRef
	Dictionary map[string]any
}

// CASLGuardObjectUpdate описує оновлення основних полів об'єкта.
type CASLGuardObjectUpdate struct {
	ObjID          string
	Name           string
	Address        string
	Long           string
	Lat            string
	Description    string
	Contract       string
	ManagerID      string
	Note           string
	StartDate      int64
	Status         string
	ObjectType     string
	IDRequest      string
	ReactingPultID string
	GeoZoneID      int64
	BusinessCoeff  *float64
	Images         []string
}

// CASLGuardObjectCreate описує створення нового охоронного об'єкта.
type CASLGuardObjectCreate struct {
	Name           string
	Address        string
	Long           string
	Lat            string
	Description    string
	Contract       string
	ManagerID      string
	Note           string
	StartDate      int64
	Status         string
	ObjectType     string
	IDRequest      string
	ReactingPultID string
	GeoZoneID      int64
	BusinessCoeff  *float64
}

// CASLRoomUpdate описує оновлення приміщення.
type CASLRoomUpdate struct {
	ObjID       string
	RoomID      string
	Name        string
	Description string
	Images      []string
	RTSP        string
}

// CASLRoomCreate описує створення приміщення.
type CASLRoomCreate struct {
	ObjID       string
	Name        string
	Description string
	Images      []string
	RTSP        string
}

// CASLDeviceUpdate описує оновлення обладнання.
type CASLDeviceUpdate struct {
	DeviceID          string
	Number            int64
	Name              string
	DeviceType        string
	Timeout           int64
	SIM1              string
	SIM2              string
	TechnicianID      string
	Units             string
	Requisites        string
	ChangeDate        int64
	ReglamentDate     int64
	MoreAlarmTime     []any
	IgnoringAlarmTime []any
	LicenceKey        string
	PasswRemote       string
}

// CASLDeviceCreate описує створення нового обладнання.
type CASLDeviceCreate struct {
	Number            int64
	Name              string
	DeviceType        string
	Timeout           int64
	SIM1              string
	SIM2              string
	TechnicianID      string
	Units             string
	Requisites        string
	ChangeDate        int64
	ReglamentDate     int64
	MoreAlarmTime     []any
	IgnoringAlarmTime []any
	LicenceKey        string
	PasswRemote       string
}

// CASLDeviceLineMutation описує створення або оновлення шлейфу.
type CASLDeviceLineMutation struct {
	DeviceID      string
	LineID        *int64
	LineNumber    int
	GroupNumber   int
	AdapterType   string
	AdapterNumber int
	Description   string
	LineType      string
	IsBlocked     bool
}

// CASLLineToRoomBinding описує прив'язку шлейфу до приміщення.
type CASLLineToRoomBinding struct {
	ObjID      string
	DeviceID   string
	LineNumber int
	RoomID     string
}

// CASLAddUserToRoomRequest описує додавання користувача до приміщення.
type CASLAddUserToRoomRequest struct {
	ObjID    string
	RoomID   string
	UserID   string
	Priority int
	HozNum   string
}

// CASLRemoveUserFromRoomRequest описує видалення користувача з приміщення.
type CASLRemoveUserFromRoomRequest struct {
	ObjID  string
	RoomID string
	UserID string
}

// CASLRoomUserPriority описує порядок користувачів у приміщенні.
type CASLRoomUserPriority struct {
	UserID   string
	RoomID   string
	Priority int
	HozNum   string
}

// CASLUserCreateRequest описує створення нового користувача CASL.
type CASLUserCreateRequest struct {
	Email        string
	Password     string
	LastName     string
	FirstName    string
	MiddleName   string
	OneboxID     string
	Role         string
	Tag          string
	PhoneNumbers []CASLPhoneNumber
	DeviceIDs    []string
}

// CASLImageCreateRequest описує додавання картинки до об'єкта або приміщення.
type CASLImageCreateRequest struct {
	ObjID     string
	RoomID    string
	ImageType string
	ImageData string
}

// CASLImageDeleteRequest описує видалення картинки з об'єкта або приміщення.
type CASLImageDeleteRequest struct {
	ObjID   string
	RoomID  string
	ImageID string
}

// CASLDeviceBlockRequest описує блокування приладу об'єкта.
type CASLDeviceBlockRequest struct {
	DeviceID     string
	DeviceNumber int64
	TimeUnblock  int64
	Message      string
}

// CASLObjectBasketItem описує один об'єкт у корзині CASL.
type CASLObjectBasketItem struct {
	BasketID   int64
	ObjID      string
	Name       string
	Address    string
	TypeData   string
	DeletedRaw string
}

// CASLObjectEditorProvider описує редагування CASL-об'єктів через CASL Cloud.
type CASLObjectEditorProvider interface {
	GetCASLObjectEditorSnapshot(ctx context.Context, objectID int64) (CASLObjectEditorSnapshot, error)
	CreateCASLObject(ctx context.Context, create CASLGuardObjectCreate) (string, error)
	UpdateCASLObject(ctx context.Context, update CASLGuardObjectUpdate) error
	DeleteCASLObject(ctx context.Context, objectID int64) error
	UpdateCASLRoom(ctx context.Context, update CASLRoomUpdate) error
	CreateCASLRoom(ctx context.Context, create CASLRoomCreate) error
	ReadCASLDeviceNumbers(ctx context.Context) ([]int64, error)
	IsCASLDeviceNumberInUse(ctx context.Context, deviceNumber int64) (bool, error)
	CreateCASLDevice(ctx context.Context, create CASLDeviceCreate) (string, error)
	UpdateCASLDevice(ctx context.Context, update CASLDeviceUpdate) error
	BlockCASLDevice(ctx context.Context, request CASLDeviceBlockRequest) error
	UnblockCASLDevice(ctx context.Context, deviceID string) error
	UpdateCASLDeviceLine(ctx context.Context, update CASLDeviceLineMutation) error
	CreateCASLDeviceLine(ctx context.Context, create CASLDeviceLineMutation) error
	AddCASLLineToRoom(ctx context.Context, binding CASLLineToRoomBinding) error
	AddCASLUserToRoom(ctx context.Context, request CASLAddUserToRoomRequest) error
	RemoveCASLUserFromRoom(ctx context.Context, request CASLRemoveUserFromRoomRequest) error
	UpdateCASLRoomUserPriorities(ctx context.Context, objectID int64, items []CASLRoomUserPriority) error
	CreateCASLUser(ctx context.Context, request CASLUserCreateRequest) (CASLUserProfile, error)
	ReadCASLObjectBasket(ctx context.Context) ([]CASLObjectBasketItem, error)
	CreateCASLImage(ctx context.Context, request CASLImageCreateRequest) error
	DeleteCASLImage(ctx context.Context, request CASLImageDeleteRequest) error
	FetchCASLImagePreview(ctx context.Context, imageID string) ([]byte, error)
}
