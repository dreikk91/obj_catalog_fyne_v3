package viewmodels

// ObjectChannelChange описує результат обробки зміни каналу зв'язку у формі.
type ObjectChannelChange struct {
	ChannelCode    int64
	PreferredPPKID int64
}

// ObjectChannelFlowViewModel інкапсулює загальну поведінку зміни каналу в картці/майстрі об'єкта.
type ObjectChannelFlowViewModel struct{}

func NewObjectChannelFlowViewModel() *ObjectChannelFlowViewModel {
	return &ObjectChannelFlowViewModel{}
}

func (vm *ObjectChannelFlowViewModel) ResolveChange(
	selectedChannelLabel string,
	selectedPPKLabel string,
	channelLabelToCode map[string]int64,
	ppkIDLookup func(label string) int64,
) ObjectChannelChange {
	channelCode := ObjectChannelCodeAutoDial
	if code, ok := channelLabelToCode[selectedChannelLabel]; ok && code > 0 {
		channelCode = code
	}

	preferredPPKID := int64(0)
	if ppkIDLookup != nil {
		preferredPPKID = ppkIDLookup(selectedPPKLabel)
	}

	return ObjectChannelChange{
		ChannelCode:    channelCode,
		PreferredPPKID: preferredPPKID,
	}
}
