package viewmodels

// ObjectWizardFlowViewModel керує переходами майстра та викликом валідації кроків.
type ObjectWizardFlowViewModel struct {
	stepsVM *ObjectWizardStepsViewModel
}

func NewObjectWizardFlowViewModel(stepTitles []string) *ObjectWizardFlowViewModel {
	return &ObjectWizardFlowViewModel{
		stepsVM: NewObjectWizardStepsViewModel(stepTitles),
	}
}

func (vm *ObjectWizardFlowViewModel) CurrentStep() int {
	return vm.stepsVM.CurrentStep()
}

func (vm *ObjectWizardFlowViewModel) TotalSteps() int {
	return vm.stepsVM.TotalSteps()
}

func (vm *ObjectWizardFlowViewModel) StatusText() string {
	return vm.stepsVM.StatusText()
}

func (vm *ObjectWizardFlowViewModel) IsLastStep() bool {
	return vm.stepsVM.IsLastStep()
}

func (vm *ObjectWizardFlowViewModel) CanGoBack() bool {
	return vm.stepsVM.CanGoBack()
}

func (vm *ObjectWizardFlowViewModel) CanGoNext() bool {
	return vm.stepsVM.CanGoNext()
}

func (vm *ObjectWizardFlowViewModel) CanCreate() bool {
	return vm.stepsVM.CanCreate()
}

func (vm *ObjectWizardFlowViewModel) GoBack() bool {
	return vm.stepsVM.GoBack()
}

func (vm *ObjectWizardFlowViewModel) GoNext(validateStep func(step int) error) (bool, error) {
	if !vm.stepsVM.CanGoNext() {
		return false, nil
	}
	if validateStep != nil {
		if err := validateStep(vm.stepsVM.CurrentStep()); err != nil {
			return false, err
		}
	}
	return vm.stepsVM.GoNext(), nil
}

func (vm *ObjectWizardFlowViewModel) ValidateCreate(validateStep func(step int) error) error {
	if validateStep == nil {
		return nil
	}
	total := vm.stepsVM.TotalSteps()
	if total <= 0 {
		return nil
	}
	return validateStep(total - 1)
}
