package viewmodels

import (
	"strconv"

	"obj_catalog_fyne_v3/pkg/contracts"
)

const defaultTestIntervalMinText = "9"

// ObjectCardLoadReferenceLookup описує мінімальні lookup-операції довідників для формування стану edit-форми.
type ObjectCardLoadReferenceLookup interface {
	ObjectTypeLabelByID(id int64) string
	RegionLabelByID(id int64) string
	SubServerLabelByBind(bind string) string
}

// ObjectCardLoadPresentation містить підготовлений стан полів для заповнення edit-форми.
type ObjectCardLoadPresentation struct {
	ObjNText            string
	ShortName           string
	FullName            string
	Address             string
	Phones              string
	Contract            string
	StartDate           string
	Location            string
	Notes               string
	ChannelLabel        string
	ChannelCode         int64
	PPKID               int64
	GSMPhone1           string
	GSMPhone2           string
	GSMHiddenNText      string
	TestControlEnabled  bool
	TestIntervalMinText string
	ObjectTypeLabel     string
	RegionLabel         string
	SubServerALabel     string
	SubServerBLabel     string
}

// ObjectCardLoadViewModel формує стан edit-форми з моделі об'єкта без прив'язки до Fyne.
type ObjectCardLoadViewModel struct{}

func NewObjectCardLoadViewModel() *ObjectCardLoadViewModel {
	return &ObjectCardLoadViewModel{}
}

func (vm *ObjectCardLoadViewModel) BuildPresentation(
	card contracts.AdminObjectCard,
	references ObjectCardLoadReferenceLookup,
	channelCodeToLabel map[int64]string,
) ObjectCardLoadPresentation {
	channelLabel, channelCode := ResolveObjectChannel(card.ChannelCode, channelCodeToLabel)

	regionID := card.ObjRegID
	if regionID <= 0 {
		regionID = 1
	}

	testInterval := defaultTestIntervalMinText
	if card.TestIntervalMin > 0 {
		testInterval = strconv.FormatInt(card.TestIntervalMin, 10)
	}

	hiddenN := ""
	if card.GSMHiddenN > 0 {
		hiddenN = strconv.FormatInt(card.GSMHiddenN, 10)
	}

	presentation := ObjectCardLoadPresentation{
		ObjNText:            strconv.FormatInt(card.ObjN, 10),
		ShortName:           card.ShortName,
		FullName:            card.FullName,
		Address:             card.Address,
		Phones:              card.Phones,
		Contract:            card.Contract,
		StartDate:           card.StartDate,
		Location:            card.Location,
		Notes:               card.Notes,
		ChannelLabel:        channelLabel,
		ChannelCode:         channelCode,
		PPKID:               card.PPKID,
		GSMPhone1:           card.GSMPhone1,
		GSMPhone2:           card.GSMPhone2,
		GSMHiddenNText:      hiddenN,
		TestControlEnabled:  card.TestControlEnabled,
		TestIntervalMinText: testInterval,
	}

	if references != nil {
		presentation.ObjectTypeLabel = references.ObjectTypeLabelByID(card.ObjTypeID)
		presentation.RegionLabel = references.RegionLabelByID(regionID)
		presentation.SubServerALabel = references.SubServerLabelByBind(card.SubServerA)
		presentation.SubServerBLabel = references.SubServerLabelByBind(card.SubServerB)
	}

	return presentation
}
