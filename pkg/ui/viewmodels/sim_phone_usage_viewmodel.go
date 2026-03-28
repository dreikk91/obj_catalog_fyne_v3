package viewmodels

import (
	"fmt"
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
)

// SIMPhoneUsageLookup описує мінімальний контракт для перевірки зайнятості SIM-номера.
type SIMPhoneUsageLookup interface {
	FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]contracts.AdminSIMPhoneUsage, error)
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
		return "Не вдалося перевірити номер у базі"
	}
	return vm.FormatUsageList(usages)
}

func (vm *SIMPhoneUsageViewModel) FormatUsageList(usages []contracts.AdminSIMPhoneUsage) string {
	if len(usages) == 0 {
		return ""
	}

	parts := make([]string, 0, len(usages))
	for _, u := range usages {
		name := strings.TrimSpace(u.Name)
		if name != "" {
			parts = append(parts, fmt.Sprintf("#%d (%s, %s)", u.ObjN, name, u.Slot))
			continue
		}
		parts = append(parts, fmt.Sprintf("#%d (%s)", u.ObjN, u.Slot))
	}
	return "Номер вже використовується: " + strings.Join(parts, "; ")
}
