package viewmodels

// ObjectCardSIMUsageStateViewModel залишено для зворотної сумісності.
type ObjectCardSIMUsageStateViewModel = SIMUsageStateViewModel

func NewObjectCardSIMUsageStateViewModel() *ObjectCardSIMUsageStateViewModel {
	return NewSIMUsageStateViewModel()
}
