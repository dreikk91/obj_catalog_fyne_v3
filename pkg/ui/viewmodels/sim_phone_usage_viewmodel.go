package viewmodels

import (
	"fmt"
	"strings"
)

// SIMPhoneUsageLookup описує мінімальний контракт для перевірки зайнятості SIM-номера.
type SIMPhoneUsageLookup interface {
	FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]SIMPhoneUsage, error)
}

// SIMPhoneUsageViewModel містить presentation-логіку повідомлень про використання SIM.
type SIMPhoneUsageViewModel struct{}

func NewSIMPhoneUsageViewModel() *SIMPhoneUsageViewModel {
	return &SIMPhoneUsageViewModel{}
}

func (vm *SIMPhoneUsageViewModel) ResolveUsageText(
	lookup SIMPhoneUsageLookup,
	rawPhone string,
	excludeObjN *int64,
) string {
	phone := strings.TrimSpace(rawPhone)
	if phone == "" {
		return ""
	}
	if lookup == nil {
		return "Не вдалося перевірити номер у базі"
	}

	usages, err := lookup.FindObjectsBySIMPhone(phone, excludeObjN)
	if err != nil {
		if len(usages) > 0 {
			return vm.FormatUsageList(usages) + ". Не вдалося перевірити всі джерела"
		}
		return "Не вдалося перевірити номер у всіх джерелах"
	}
	return vm.FormatUsageList(usages)
}

func (vm *SIMPhoneUsageViewModel) FormatUsageList(usages []SIMPhoneUsage) string {
	if len(usages) == 0 {
		return ""
	}

	parts := make([]string, 0, len(usages))
	for _, u := range usages {
		objectNumber := strings.TrimSpace(u.DisplayNumber)
		if objectNumber == "" {
			objectNumber = fmt.Sprintf("%d", u.ObjN)
		}
		details := make([]string, 0, 3)
		name := strings.TrimSpace(u.Name)
		if name != "" {
			details = append(details, name)
		}
		if slot := strings.TrimSpace(u.Slot); slot != "" {
			details = append(details, slot)
		}
		if source := strings.TrimSpace(u.Source); source != "" {
			details = append(details, source)
		}
		if len(details) == 0 {
			parts = append(parts, "#"+objectNumber)
			continue
		}
		parts = append(parts, fmt.Sprintf("#%s (%s)", objectNumber, strings.Join(details, ", ")))
	}
	return "Номер вже використовується: " + strings.Join(parts, "; ")
}
