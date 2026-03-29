package ui

import "fyne.io/fyne/v2/data/binding"

// SetUntypedList заповнює binding.UntypedList значеннями типізованого зрізу.
func SetUntypedList[T any](list binding.UntypedList, values []T) error {
	if list == nil {
		return nil
	}
	items := make([]any, len(values))
	for i := range values {
		items[i] = values[i]
	}
	return list.Set(items)
}
