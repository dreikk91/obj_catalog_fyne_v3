package viewmodels

import "fmt"

// ObjectWizardStepsViewModel інкапсулює стан навігації по кроках майстра.
type ObjectWizardStepsViewModel struct {
	stepTitles  []string
	currentStep int
}

func NewObjectWizardStepsViewModel(stepTitles []string) *ObjectWizardStepsViewModel {
	titles := append([]string(nil), stepTitles...)
	return &ObjectWizardStepsViewModel{
		stepTitles:  titles,
		currentStep: 0,
	}
}

func (vm *ObjectWizardStepsViewModel) CurrentStep() int {
	return vm.currentStep
}

func (vm *ObjectWizardStepsViewModel) TotalSteps() int {
	return len(vm.stepTitles)
}

func (vm *ObjectWizardStepsViewModel) IsLastStep() bool {
	total := vm.TotalSteps()
	return total > 0 && vm.currentStep >= total-1
}

func (vm *ObjectWizardStepsViewModel) CanGoBack() bool {
	return vm.currentStep > 0
}

func (vm *ObjectWizardStepsViewModel) CanGoNext() bool {
	total := vm.TotalSteps()
	return total > 0 && vm.currentStep < total-1
}

func (vm *ObjectWizardStepsViewModel) CanCreate() bool {
	return vm.IsLastStep()
}

func (vm *ObjectWizardStepsViewModel) GoBack() bool {
	if !vm.CanGoBack() {
		return false
	}
	vm.currentStep--
	return true
}

func (vm *ObjectWizardStepsViewModel) GoNext() bool {
	if !vm.CanGoNext() {
		return false
	}
	vm.currentStep++
	return true
}

func (vm *ObjectWizardStepsViewModel) StatusText() string {
	total := vm.TotalSteps()
	if total <= 0 {
		return "Крок 0/0"
	}
	stepName := ""
	if vm.currentStep >= 0 && vm.currentStep < len(vm.stepTitles) {
		stepName = vm.stepTitles[vm.currentStep]
	}
	return fmt.Sprintf("Крок %d/%d: %s", vm.currentStep+1, total, stepName)
}
