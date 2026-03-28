package viewmodels

// ObjectCardInitIssue описує проблему, виявлену під час ініціалізації діалогу картки.
type ObjectCardInitIssue struct {
	StatusMessage   string
	Err             error
	ShowErrorDialog bool
}

// ObjectCardInitResult містить результати ініціалізаційного сценарію.
type ObjectCardInitResult struct {
	Issues []ObjectCardInitIssue
}

// ObjectCardInitInput описує залежності для ініціалізації діалогу картки об'єкта.
type ObjectCardInitInput struct {
	EditObjN          *int64
	LoadReferenceData func() error
	PrepareEditMode   func()
	LoadCard          func(objn int64) error
	FillDefaults      func()
}

// ObjectCardInitViewModel інкапсулює init-flow create/edit діалогу об'єкта.
type ObjectCardInitViewModel struct{}

func NewObjectCardInitViewModel() *ObjectCardInitViewModel {
	return &ObjectCardInitViewModel{}
}

func (vm *ObjectCardInitViewModel) Initialize(input ObjectCardInitInput) ObjectCardInitResult {
	result := ObjectCardInitResult{
		Issues: make([]ObjectCardInitIssue, 0, 2),
	}

	if input.LoadReferenceData != nil {
		if err := input.LoadReferenceData(); err != nil {
			result.Issues = append(result.Issues, ObjectCardInitIssue{
				StatusMessage:   "Не вдалося завантажити довідники",
				Err:             err,
				ShowErrorDialog: true,
			})
		}
	}

	isEdit := input.EditObjN != nil && *input.EditObjN > 0
	if isEdit {
		if input.PrepareEditMode != nil {
			input.PrepareEditMode()
		}
		if input.LoadCard != nil {
			if err := input.LoadCard(*input.EditObjN); err != nil {
				result.Issues = append(result.Issues, ObjectCardInitIssue{
					StatusMessage:   "Не вдалося завантажити об'єкт для редагування",
					Err:             err,
					ShowErrorDialog: true,
				})
			}
		}
		return result
	}

	if input.FillDefaults != nil {
		input.FillDefaults()
	}
	return result
}
