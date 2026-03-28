package viewmodels

import "fmt"

const (
	defaultObjectChannelCode   int64 = 1
	defaultObjectTypeID        int64 = 0
	defaultObjectRegionID      int64 = 1
	defaultObjectSubServerBind       = ""
	defaultTestIntervalMinRaw        = "9"
)

// ObjectCardFormReferenceLookup описує мінімальні довідникові lookup-операції для формування ObjectCardInput.
type ObjectCardFormReferenceLookup interface {
	ObjectTypeID(label string) int64
	RegionID(label string) int64
	PPKID(label string) int64
	SubServerBind(label string) string
}

// ObjectCardFormSnapshot описує поточний стан полів форми (без Fyne-залежностей).
type ObjectCardFormSnapshot struct {
	ObjNRaw       string
	ShortName     string
	FullName      string
	Address       string
	Phones        string
	Contract      string
	StartDate     string
	Location      string
	Notes         string
	GSMPhone1     string
	GSMPhone2     string
	GSMHiddenNRaw string

	ChannelLabel       string
	TestControlEnabled bool
	TestIntervalMinRaw string
	ObjectTypeLabel    string
	RegionLabel        string
	PPKLabel           string
	SubServerALabel    string
	SubServerBLabel    string
}

// ObjectCardFormDefaults містить дефолтні значення форми.
type ObjectCardFormDefaults struct {
	ChannelCode        int64
	ObjectTypeID       int64
	RegionID           int64
	SubServerBind      string
	TestControlEnabled bool
	TestIntervalMinRaw string
}

// ObjectCardFormViewModel інкапсулює спільну логіку підготовки input для create/edit форм.
type ObjectCardFormViewModel struct{}

func NewObjectCardFormViewModel() *ObjectCardFormViewModel {
	return &ObjectCardFormViewModel{}
}

func (vm *ObjectCardFormViewModel) Defaults() ObjectCardFormDefaults {
	return ObjectCardFormDefaults{
		ChannelCode:        defaultObjectChannelCode,
		ObjectTypeID:       defaultObjectTypeID,
		RegionID:           defaultObjectRegionID,
		SubServerBind:      defaultObjectSubServerBind,
		TestControlEnabled: true,
		TestIntervalMinRaw: defaultTestIntervalMinRaw,
	}
}

func (vm *ObjectCardFormViewModel) BuildInput(
	snapshot ObjectCardFormSnapshot,
	references ObjectCardFormReferenceLookup,
	channelLabelToCode map[string]int64,
) (ObjectCardInput, error) {
	channelCode, ok := channelLabelToCode[snapshot.ChannelLabel]
	if !ok || channelCode <= 0 {
		return ObjectCardInput{}, fmt.Errorf("виберіть канал зв'язку")
	}

	return ObjectCardInput{
		ObjNRaw:            snapshot.ObjNRaw,
		ShortName:          snapshot.ShortName,
		FullName:           snapshot.FullName,
		Address:            snapshot.Address,
		Phones:             snapshot.Phones,
		Contract:           snapshot.Contract,
		StartDate:          snapshot.StartDate,
		Location:           snapshot.Location,
		Notes:              snapshot.Notes,
		GSMPhone1:          snapshot.GSMPhone1,
		GSMPhone2:          snapshot.GSMPhone2,
		GSMHiddenNRaw:      snapshot.GSMHiddenNRaw,
		ChannelCode:        channelCode,
		TestControlEnabled: snapshot.TestControlEnabled,
		TestIntervalMinRaw: snapshot.TestIntervalMinRaw,
		ObjTypeID:          references.ObjectTypeID(snapshot.ObjectTypeLabel),
		ObjRegID:           references.RegionID(snapshot.RegionLabel),
		PPKID:              references.PPKID(snapshot.PPKLabel),
		SubServerA:         references.SubServerBind(snapshot.SubServerALabel),
		SubServerB:         references.SubServerBind(snapshot.SubServerBLabel),
	}, nil
}
