package viewmodels

// ObjectCardDefaultsReferenceLookup описує мінімальні lookup-операції довідників для дефолтного стану форми.
type ObjectCardDefaultsReferenceLookup interface {
	ObjectTypeLabelByID(id int64) string
	RegionLabelByID(id int64) string
	SubServerLabelByBind(bind string) string
}

// ObjectCardDefaultsPresentation містить підготовлений дефолтний стан форми картки об'єкта.
type ObjectCardDefaultsPresentation struct {
	ObjNText            string
	ShortName           string
	FullName            string
	Address             string
	Phones              string
	Contract            string
	StartDateText       string
	Location            string
	Notes               string
	ChannelLabel        string
	ChannelCode         int64
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

// ObjectCardDefaultsViewModel формує дефолтний стан форми без прив'язки до Fyne.
type ObjectCardDefaultsViewModel struct{}

func NewObjectCardDefaultsViewModel() *ObjectCardDefaultsViewModel {
	return &ObjectCardDefaultsViewModel{}
}

func (vm *ObjectCardDefaultsViewModel) BuildPresentation(
	defaults ObjectCardFormDefaults,
	references ObjectCardDefaultsReferenceLookup,
	channelCodeToLabel map[int64]string,
	startDateText string,
) ObjectCardDefaultsPresentation {
	channelLabel, channelCode := ResolveObjectChannel(defaults.ChannelCode, channelCodeToLabel)

	presentation := ObjectCardDefaultsPresentation{
		ObjNText:            "",
		ShortName:           "",
		FullName:            "",
		Address:             "",
		Phones:              "",
		Contract:            "",
		StartDateText:       startDateText,
		Location:            "",
		Notes:               "",
		ChannelLabel:        channelLabel,
		ChannelCode:         channelCode,
		GSMPhone1:           "",
		GSMPhone2:           "",
		GSMHiddenNText:      "",
		TestControlEnabled:  defaults.TestControlEnabled,
		TestIntervalMinText: defaults.TestIntervalMinRaw,
	}

	if references != nil {
		presentation.ObjectTypeLabel = references.ObjectTypeLabelByID(defaults.ObjectTypeID)
		presentation.RegionLabel = references.RegionLabelByID(defaults.RegionID)
		presentation.SubServerALabel = references.SubServerLabelByBind(defaults.SubServerBind)
		presentation.SubServerBLabel = references.SubServerLabelByBind(defaults.SubServerBind)
	}

	return presentation
}
