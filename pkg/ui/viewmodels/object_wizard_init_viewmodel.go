package viewmodels

// ObjectWizardInitIssue описує проблему під час ініціалізації майстра об'єкта.
type ObjectWizardInitIssue struct {
	StatusMessage   string
	Err             error
	ShowErrorDialog bool
}

// ObjectWizardInitResult містить результат init-flow майстра.
type ObjectWizardInitResult struct {
	Issues []ObjectWizardInitIssue
}

// ObjectWizardInitInput описує залежності ініціалізації майстра.
type ObjectWizardInitInput struct {
	LoadReferenceData func() error
	FillDefaults      func()
}

// ObjectWizardInitViewModel інкапсулює init-сценарій майстра створення об'єкта.
type ObjectWizardInitViewModel struct{}

func NewObjectWizardInitViewModel() *ObjectWizardInitViewModel {
	return &ObjectWizardInitViewModel{}
}

func (vm *ObjectWizardInitViewModel) Initialize(input ObjectWizardInitInput) ObjectWizardInitResult {
	result := ObjectWizardInitResult{
		Issues: make([]ObjectWizardInitIssue, 0, 1),
	}

	if input.LoadReferenceData != nil {
		if err := input.LoadReferenceData(); err != nil {
			result.Issues = append(result.Issues, ObjectWizardInitIssue{
				StatusMessage:   "Не вдалося завантажити довідники",
				Err:             err,
				ShowErrorDialog: true,
			})
		}
	}
	if input.FillDefaults != nil {
		input.FillDefaults()
	}

	return result
}
