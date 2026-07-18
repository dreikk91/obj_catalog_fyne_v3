package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/utils"
)

var contactsCSVHeader = []string{
	"groupname",
	"grouptype",
	"displayname",
	"fname",
	"lname",
	"title",
	"company",
	"address",
	"phone_1_type",
	"phone_1_number",
	"phone_1_extension",
	"phone_1_flags",
	"phone_1_speeddial",
	"phone_2_type",
	"phone_2_number",
	"phone_2_extension",
	"phone_2_flags",
	"phone_2_speeddial",
}

// ContactExportObject contains one object's contacts prepared for CSV export.
type ContactExportObject struct {
	Source       contracts.FrontendSource
	ObjectNumber string
	Object       models.Object
	Contacts     []models.Contact
}

// WriteContactsCSV writes contacts in the CSV format used by the phone directory.
func WriteContactsCSV(filePath string, objects []ContactExportObject) (int, error) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return 0, fmt.Errorf("шлях до CSV-файлу порожній")
	}

	file, err := os.Create(filePath)
	if err != nil {
		return 0, fmt.Errorf("створити CSV-файл: %w", err)
	}

	writer := csv.NewWriter(file)
	writer.UseCRLF = true
	if err := writer.Write(contactsCSVHeader); err != nil {
		_ = file.Close()
		return 0, fmt.Errorf("записати заголовок CSV: %w", err)
	}

	sortedObjects := append([]ContactExportObject(nil), objects...)
	sort.SliceStable(sortedObjects, func(i, j int) bool {
		leftOrder := contactSourceOrder(sortedObjects[i].Source)
		rightOrder := contactSourceOrder(sortedObjects[j].Source)
		if leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		return strings.ToLower(strings.TrimSpace(sortedObjects[i].ObjectNumber)) <
			strings.ToLower(strings.TrimSpace(sortedObjects[j].ObjectNumber))
	})

	written := 0
	for _, item := range sortedObjects {
		contacts := append([]models.Contact(nil), item.Contacts...)
		sort.SliceStable(contacts, func(i, j int) bool {
			left := contacts[i].Priority
			right := contacts[j].Priority
			if left <= 0 && right <= 0 {
				return false
			}
			if left <= 0 {
				return false
			}
			if right <= 0 {
				return true
			}
			return left < right
		})

		for index, contact := range contacts {
			for _, record := range contactCSVRecords(item, contact, index+1) {
				if err := writer.Write(record); err != nil {
					_ = file.Close()
					return written, fmt.Errorf("записати контакт у CSV: %w", err)
				}
				written++
			}
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		_ = file.Close()
		return written, fmt.Errorf("завершити запис CSV: %w", err)
	}
	if err := file.Close(); err != nil {
		return written, fmt.Errorf("закрити CSV-файл: %w", err)
	}
	return written, nil
}

func contactCSVRecords(item ContactExportObject, contact models.Contact, ordinal int) [][]string {
	phones := splitContactPhones(contact.Phone)
	if len(phones) <= 2 {
		return [][]string{contactCSVRecord(item, contact, ordinal)}
	}

	records := make([][]string, 0, (len(phones)+1)/2)
	for index := 0; index < len(phones); index += 2 {
		end := min(index+2, len(phones))
		phoneContact := contact
		phoneContact.Phone = strings.Join(phones[index:end], " / ")
		recordOrdinal := 0
		if index == 0 {
			recordOrdinal = ordinal
		}
		records = append(records, contactCSVRecord(item, phoneContact, recordOrdinal))
	}
	return records
}

func contactCSVRecord(item ContactExportObject, contact models.Contact, ordinal int) []string {
	phones := splitContactPhones(contact.Phone)
	displayName, firstName := contactCSVNames(contact.Name, item.ObjectNumber)
	phone1 := ""
	phone2 := ""
	if len(phones) > 0 {
		phone1 = phones[0]
	}
	if len(phones) > 1 {
		phone2 = phones[1]
	}

	speedDial := ""
	if phone1 != "" {
		speedDial = contactSpeedDial(item.Source, item.ObjectNumber, ordinal)
	}

	return []string{
		contactSourceGroup(item.Source),
		"external",
		displayName,
		firstName,
		"",
		strings.TrimSpace(contact.Position),
		strings.TrimSpace(item.Object.Name),
		strings.TrimSpace(item.Object.Address),
		contactPhoneType(phone1),
		phone1,
		"",
		"",
		speedDial,
		contactPhoneType(phone2),
		phone2,
		"",
		"",
		"",
	}
}

func contactCSVNames(name string, objectNumber string) (string, string) {
	name = strings.TrimSpace(name)
	objectNumber = strings.TrimSpace(objectNumber)

	hasName := strings.ContainsFunc(name, func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	})
	if !hasName {
		return objectNumber, objectNumber
	}
	if objectNumber == "" {
		return name, name
	}
	return objectNumber + " " + name, name
}

func contactSourceGroup(source contracts.FrontendSource) string {
	switch source {
	case contracts.FrontendSourceBridge:
		return "МІСТ"
	case contracts.FrontendSourcePhoenix:
		return "Phoenix"
	case contracts.FrontendSourceCASL:
		return "CASL"
	default:
		return "Інше"
	}
}

func contactSourceOrder(source contracts.FrontendSource) int {
	switch source {
	case contracts.FrontendSourceBridge:
		return 0
	case contracts.FrontendSourcePhoenix:
		return 1
	case contracts.FrontendSourceCASL:
		return 2
	default:
		return 3
	}
}

func contactSourceSpeedDialPrefix(source contracts.FrontendSource) string {
	switch source {
	case contracts.FrontendSourceBridge:
		return "2"
	case contracts.FrontendSourcePhoenix:
		return "3"
	case contracts.FrontendSourceCASL:
		return "4"
	default:
		return ""
	}
}

func contactSpeedDial(source contracts.FrontendSource, objectNumber string, ordinal int) string {
	prefix := contactSourceSpeedDialPrefix(source)
	number := utils.DigitsOnly(objectNumber)
	if prefix == "" || number == "" || ordinal <= 0 {
		return ""
	}
	return fmt.Sprintf("*%s%s%d", prefix, number, ordinal)
}

func contactPhoneType(phone string) string {
	if strings.TrimSpace(phone) == "" {
		return ""
	}
	return "work"
}

func splitContactPhones(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', ';', '/', '\r', '\n':
			return true
		default:
			return false
		}
	})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		for _, phone := range splitJoinedUkrainianPhones(part) {
			phone = normalizeContactPhone(phone)
			if phone == "" {
				continue
			}
			result = append(result, phone)
		}
	}
	return result
}

func splitJoinedUkrainianPhones(raw string) []string {
	digits := utils.DigitsOnly(raw)
	if digits == "" {
		return nil
	}

	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "+") && !strings.HasPrefix(digits, "380") {
		return []string{raw}
	}
	if strings.HasPrefix(digits, "00") && !strings.HasPrefix(digits, "00380") {
		return []string{raw}
	}

	if phones, ok := splitUkrainianPhoneDigits(digits); ok {
		return phones
	}
	return []string{raw}
}

func splitUkrainianPhoneDigits(digits string) ([]string, bool) {
	if digits == "" {
		return nil, true
	}

	lengths := []int{14, 12, 10, 9}
	for _, length := range lengths {
		if len(digits) < length || !isUkrainianPhoneDigits(digits[:length]) {
			continue
		}
		tail, ok := splitUkrainianPhoneDigits(digits[length:])
		if ok {
			return append([]string{digits[:length]}, tail...), true
		}
	}
	return nil, false
}

func isUkrainianPhoneDigits(digits string) bool {
	switch len(digits) {
	case 14:
		return strings.HasPrefix(digits, "00380")
	case 12:
		return strings.HasPrefix(digits, "380")
	case 10:
		return strings.HasPrefix(digits, "0")
	case 9:
		return !strings.HasPrefix(digits, "0")
	default:
		return false
	}
}

func normalizeContactPhone(raw string) string {
	digits := utils.DigitsOnly(raw)
	if digits == "" {
		return ""
	}

	switch {
	case strings.HasPrefix(digits, "00") && len(digits) > 2:
		return "+" + digits[2:]
	case len(digits) == 10 && strings.HasPrefix(digits, "0"):
		return "+38" + digits
	case len(digits) == 9:
		return "+380" + digits
	case len(digits) == 12 && strings.HasPrefix(digits, "380"):
		return "+" + digits
	case strings.HasPrefix(strings.TrimSpace(raw), "+"):
		return "+" + digits
	default:
		return digits
	}
}
