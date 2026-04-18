package viewmodels

import (
	"fmt"
	"strings"
)

// ObjectAdditionalTabViewModel керує state-логікою вкладки "Додатково".
type ObjectAdditionalTabViewModel struct {
	lastGeoAddress       string
	lastGeoDistrictHints []string
}

func NewObjectAdditionalTabViewModel() *ObjectAdditionalTabViewModel {
	return &ObjectAdditionalTabViewModel{}
}

func (vm *ObjectAdditionalTabViewModel) AddressFromObjectTab(getAddress func() string) (string, bool) {
	if getAddress == nil {
		return "", false
	}
	address := strings.TrimSpace(getAddress())
	if address == "" {
		return "", false
	}
	return address, true
}

func (vm *ObjectAdditionalTabViewModel) RequireLookupAddress(raw string) (string, error) {
	address := strings.TrimSpace(raw)
	if address == "" {
		return "", fmt.Errorf("вкажіть адресу")
	}
	return address, nil
}

func (vm *ObjectAdditionalTabViewModel) RememberGeocode(address string, districtHints []string) {
	vm.lastGeoAddress = strings.TrimSpace(address)
	vm.lastGeoDistrictHints = append([]string(nil), districtHints...)
}

func (vm *ObjectAdditionalTabViewModel) CachedDistrictHintsForAddress(address string) ([]string, bool) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, false
	}
	if !strings.EqualFold(strings.TrimSpace(vm.lastGeoAddress), address) || len(vm.lastGeoDistrictHints) == 0 {
		return nil, false
	}
	return append([]string(nil), vm.lastGeoDistrictHints...), true
}

func (vm *ObjectAdditionalTabViewModel) BuildCoordinates(latitudeRaw string, longitudeRaw string) ObjectCoordinates {
	return ObjectCoordinates{
		Latitude:  strings.TrimSpace(latitudeRaw),
		Longitude: strings.TrimSpace(longitudeRaw),
	}
}
