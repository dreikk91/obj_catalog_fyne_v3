package viewmodels

// ObjectWizardSIMUsageStateViewModel залишено для зворотної сумісності.
type ObjectWizardSIMUsageStateViewModel = SIMUsageStateViewModel

func NewObjectWizardSIMUsageStateViewModel() *ObjectWizardSIMUsageStateViewModel {
	return NewSIMUsageStateViewModel()
}
