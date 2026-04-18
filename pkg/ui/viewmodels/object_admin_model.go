package viewmodels

import "obj_catalog_fyne_v3/pkg/contracts"

type ObjectPersonal struct {
	ID          int64
	SourceObjN  int64
	Number      int64
	Surname     string
	Name        string
	SecName     string
	Address     string
	Phones      string
	Position    string
	Notes       string
	IsTRKTester bool
	Access1     int64
	IsRang      bool
	ViberID     string
	TelegramID  string
	CreatedAt   string
}

type ObjectZone struct {
	ID            int64
	ZoneNumber    int64
	ZoneType      int64
	Description   string
	EntryDelaySec int64
}

type ObjectCoordinates struct {
	Latitude  string
	Longitude string
}

type SIMPhoneUsage struct {
	ObjN int64
	Name string
	Slot string
}

func ObjectPersonalFromContracts(item contracts.AdminObjectPersonal) ObjectPersonal {
	return ObjectPersonal{
		ID:          item.ID,
		SourceObjN:  item.SourceObjN,
		Number:      item.Number,
		Surname:     item.Surname,
		Name:        item.Name,
		SecName:     item.SecName,
		Address:     item.Address,
		Phones:      item.Phones,
		Position:    item.Position,
		Notes:       item.Notes,
		IsTRKTester: item.IsTRKTester,
		Access1:     item.Access1,
		IsRang:      item.IsRang,
		ViberID:     item.ViberID,
		TelegramID:  item.TelegramID,
		CreatedAt:   item.CreatedAt,
	}
}

func ObjectPersonalsFromContracts(items []contracts.AdminObjectPersonal) []ObjectPersonal {
	result := make([]ObjectPersonal, 0, len(items))
	for _, item := range items {
		result = append(result, ObjectPersonalFromContracts(item))
	}
	return result
}

func (item ObjectPersonal) ToContracts() contracts.AdminObjectPersonal {
	return contracts.AdminObjectPersonal{
		ID:          item.ID,
		SourceObjN:  item.SourceObjN,
		Number:      item.Number,
		Surname:     item.Surname,
		Name:        item.Name,
		SecName:     item.SecName,
		Address:     item.Address,
		Phones:      item.Phones,
		Position:    item.Position,
		Notes:       item.Notes,
		IsTRKTester: item.IsTRKTester,
		Access1:     item.Access1,
		IsRang:      item.IsRang,
		ViberID:     item.ViberID,
		TelegramID:  item.TelegramID,
		CreatedAt:   item.CreatedAt,
	}
}

func ObjectZoneFromContracts(item contracts.AdminObjectZone) ObjectZone {
	return ObjectZone{
		ID:            item.ID,
		ZoneNumber:    item.ZoneNumber,
		ZoneType:      item.ZoneType,
		Description:   item.Description,
		EntryDelaySec: item.EntryDelaySec,
	}
}

func ObjectZonesFromContracts(items []contracts.AdminObjectZone) []ObjectZone {
	result := make([]ObjectZone, 0, len(items))
	for _, item := range items {
		result = append(result, ObjectZoneFromContracts(item))
	}
	return result
}

func (item ObjectZone) ToContracts() contracts.AdminObjectZone {
	return contracts.AdminObjectZone{
		ID:            item.ID,
		ZoneNumber:    item.ZoneNumber,
		ZoneType:      item.ZoneType,
		Description:   item.Description,
		EntryDelaySec: item.EntryDelaySec,
	}
}

func ObjectCoordinatesFromContracts(item contracts.AdminObjectCoordinates) ObjectCoordinates {
	return ObjectCoordinates{
		Latitude:  item.Latitude,
		Longitude: item.Longitude,
	}
}

func (item ObjectCoordinates) ToContracts() contracts.AdminObjectCoordinates {
	return contracts.AdminObjectCoordinates{
		Latitude:  item.Latitude,
		Longitude: item.Longitude,
	}
}

func SIMPhoneUsagesFromContracts(items []contracts.AdminSIMPhoneUsage) []SIMPhoneUsage {
	result := make([]SIMPhoneUsage, 0, len(items))
	for _, item := range items {
		result = append(result, SIMPhoneUsage{
			ObjN: item.ObjN,
			Name: item.Name,
			Slot: item.Slot,
		})
	}
	return result
}
