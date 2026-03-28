package viewmodels

import "strings"

// ObjectWizardCoordinatesActionResult описує результат дії над координатами для UI.
type ObjectWizardCoordinatesActionResult struct {
	Latitude  string
	Longitude string
	Status    string
}

// ObjectWizardCoordinatesFlowViewModel інкапсулює сценарії кроку координат.
type ObjectWizardCoordinatesFlowViewModel struct{}

func NewObjectWizardCoordinatesFlowViewModel() *ObjectWizardCoordinatesFlowViewModel {
	return &ObjectWizardCoordinatesFlowViewModel{}
}

func (vm *ObjectWizardCoordinatesFlowViewModel) PreparePickerInput(latitude string, longitude string) (string, string) {
	return strings.TrimSpace(latitude), strings.TrimSpace(longitude)
}

func (vm *ObjectWizardCoordinatesFlowViewModel) ApplyPicked(latitude string, longitude string) ObjectWizardCoordinatesActionResult {
	return ObjectWizardCoordinatesActionResult{
		Latitude:  strings.TrimSpace(latitude),
		Longitude: strings.TrimSpace(longitude),
		Status:    "Координати вибрано на карті",
	}
}

func (vm *ObjectWizardCoordinatesFlowViewModel) Clear() ObjectWizardCoordinatesActionResult {
	return ObjectWizardCoordinatesActionResult{
		Latitude:  "",
		Longitude: "",
		Status:    "Координати очищено",
	}
}
