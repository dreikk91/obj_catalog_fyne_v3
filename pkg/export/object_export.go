package export

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
	"golang.org/x/image/font/gofont/goregular"
)

var invalidFilenameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

type ZoneExportRow struct {
	Number string
	Name   string
	Type   string
	Status string
}

type ResponsibleExportRow struct {
	Name  string
	Phone string
	Note  string
}

// ObjectExportData contains all information required for object export.
type ObjectExportData struct {
	Number         int
	Name           string
	Address        string
	ContractNumber string
	LaunchDate     string
	SimCard        string
	DeviceType     string
	TestPeriod     string
	LastEvent      string
	LastTest       string
	Channel        string
	ObjectPhone    string
	Location       string
	AdditionalInfo string
	Zones          []ZoneExportRow
	Responsibles   []ResponsibleExportRow
}

// ExportObjectToPDF exports object data to a PDF file.
func ExportObjectToPDF(data ObjectExportData, outputDir string) (string, error) {
	filePath, err := buildFilePath(data.Number, "pdf", outputDir)
	if err != nil {
		return "", err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(fmt.Sprintf("Object #%d", data.Number), false)
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 12)
	pdf.AddPage()

	if err := setUnicodeFont(pdf); err != nil {
		return "", err
	}

	pdf.SetFont("goregular", "", 14)
	pdf.CellFormat(0, 9, fmt.Sprintf("Інформація про об'єкт №%d", data.Number), "", 1, "", false, 0, "")
	pdf.Ln(1)

	fields := []struct {
		label string
		value string
	}{
		{label: "Номер", value: fmt.Sprintf("%d", data.Number)},
		{label: "Назва", value: normalizeValue(data.Name)},
		{label: "Адреса", value: normalizeValue(data.Address)},
		{label: "Номер договору", value: normalizeValue(data.ContractNumber)},
		{label: "Дата запуску", value: normalizeValue(data.LaunchDate)},
		{label: "Номер сім карти", value: normalizeValue(data.SimCard)},
		{label: "Тип приладу", value: normalizeValue(data.DeviceType)},
		{label: "Період тесту", value: normalizeValue(data.TestPeriod)},
		{label: "Остання подія", value: normalizeValue(data.LastEvent)},
		{label: "Останній тест", value: normalizeValue(data.LastTest)},
		{label: "Канал", value: normalizeValue(data.Channel)},
		{label: "Телефон об'єкту", value: normalizeValue(data.ObjectPhone)},
		{label: "Розташування", value: normalizeValue(data.Location)},
		{label: "Додаткова інформація", value: normalizeValue(data.AdditionalInfo)},
	}

	writePDFSectionHeader(pdf, "ЗАГАЛЬНА ІНФОРМАЦІЯ")
	for _, field := range fields {
		writePDFKeyValue(pdf, field.label, field.value)
	}
	pdf.Ln(1)

	writePDFSectionHeader(pdf, "СПИСОК ЗОН")
	writeZoneTableToPDF(pdf, ensureZones(data.Zones))
	pdf.Ln(1)

	writePDFSectionHeader(pdf, "СПИСОК ВІДПОВІДАЛЬНИХ")
	writeResponsibleTableToPDF(pdf, ensureResponsibles(data.Responsibles))

	if err := pdf.OutputFileAndClose(filePath); err != nil {
		return "", fmt.Errorf("failed to write PDF: %w", err)
	}

	return filePath, nil
}

// ExportObjectToXLSX exports object data to an XLSX file.
func ExportObjectToXLSX(data ObjectExportData, outputDir string) (string, error) {
	filePath, err := buildFilePath(data.Number, "xlsx", outputDir)
	if err != nil {
		return "", err
	}

	f := excelize.NewFile()
	defer f.Close()

	sheet := f.GetSheetName(f.GetActiveSheetIndex())

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#D9E1F2"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	labelStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"},
		Border:    []excelize.Border{{Type: "left", Color: "D9D9D9", Style: 1}, {Type: "right", Color: "D9D9D9", Style: 1}, {Type: "top", Color: "D9D9D9", Style: 1}, {Type: "bottom", Color: "D9D9D9", Style: 1}},
	})
	cellStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"},
		Border:    []excelize.Border{{Type: "left", Color: "D9D9D9", Style: 1}, {Type: "right", Color: "D9D9D9", Style: 1}, {Type: "top", Color: "D9D9D9", Style: 1}, {Type: "bottom", Color: "D9D9D9", Style: 1}},
	})
	tableHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#F2F2F2"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    []excelize.Border{{Type: "left", Color: "D9D9D9", Style: 1}, {Type: "right", Color: "D9D9D9", Style: 1}, {Type: "top", Color: "D9D9D9", Style: 1}, {Type: "bottom", Color: "D9D9D9", Style: 1}},
	})

	baseRows := []struct {
		key   string
		value string
	}{
		{key: "Номер", value: fmt.Sprintf("%d", data.Number)},
		{key: "Назва", value: normalizeValue(data.Name)},
		{key: "Адреса", value: normalizeValue(data.Address)},
		{key: "Номер договору", value: normalizeValue(data.ContractNumber)},
		{key: "Дата запуску", value: normalizeValue(data.LaunchDate)},
		{key: "Номер сім карти", value: normalizeValue(data.SimCard)},
		{key: "Тип приладу", value: normalizeValue(data.DeviceType)},
		{key: "Період тесту", value: normalizeValue(data.TestPeriod)},
		{key: "Остання подія", value: normalizeValue(data.LastEvent)},
		{key: "Останній тест", value: normalizeValue(data.LastTest)},
		{key: "Канал", value: normalizeValue(data.Channel)},
		{key: "Телефон об'єкту", value: normalizeValue(data.ObjectPhone)},
		{key: "Розташування", value: normalizeValue(data.Location)},
		{key: "Додаткова інформація", value: normalizeValue(data.AdditionalInfo)},
	}

	row := 1
	row = addSectionHeaderRange(f, sheet, row, "ЗАГАЛЬНА ІНФОРМАЦІЯ", "A", "D", headerStyle)
	for _, item := range baseRows {
		setKV(f, sheet, row, item.key, item.value, labelStyle, cellStyle)
		row++
	}

	row++
	row = addSectionHeaderRange(f, sheet, row, "СПИСОК ЗОН", "A", "D", headerStyle)
	setTableHeaders(f, sheet, row, []string{"№ зони", "Назва", "Тип", "Стан"}, tableHeaderStyle)
	row++
	for _, z := range ensureZones(data.Zones) {
		setTableRow(f, sheet, row, []string{normalizeValue(z.Number), normalizeValue(z.Name), normalizeValue(z.Type), normalizeValue(z.Status)}, cellStyle)
		row++
	}

	row++
	row = addSectionHeaderRange(f, sheet, row, "СПИСОК ВІДПОВІДАЛЬНИХ", "A", "C", headerStyle)
	setTableHeaders(f, sheet, row, []string{"Ім'я", "Телефон", "Примітка"}, tableHeaderStyle)
	row++
	for _, p := range ensureResponsibles(data.Responsibles) {
		setTableRow(f, sheet, row, []string{normalizeValue(p.Name), normalizeValue(p.Phone), normalizeValue(p.Note)}, cellStyle)
		row++
	}

	_ = f.SetColWidth(sheet, "A", "A", 14)
	_ = f.SetColWidth(sheet, "B", "B", 42)
	_ = f.SetColWidth(sheet, "C", "C", 32)
	_ = f.SetColWidth(sheet, "D", "D", 24)
	for i := 1; i <= row; i++ {
		_ = f.SetRowHeight(sheet, i, 24)
	}

	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("failed to write XLSX: %w", err)
	}

	return filePath, nil
}

func setUnicodeFont(pdf *gofpdf.Fpdf) error {
	fontFile, err := os.CreateTemp("", "goregular-*.ttf")
	if err != nil {
		return fmt.Errorf("failed to create temp font file: %w", err)
	}
	defer os.Remove(fontFile.Name())

	if _, err := fontFile.Write(goregular.TTF); err != nil {
		_ = fontFile.Close()
		return fmt.Errorf("failed to write temp font file: %w", err)
	}
	if err := fontFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp font file: %w", err)
	}

	pdf.AddUTF8Font("goregular", "", fontFile.Name())
	if pdf.Error() != nil {
		return fmt.Errorf("failed to register UTF-8 font: %w", pdf.Error())
	}
	return nil
}

func writePDFSectionHeader(pdf *gofpdf.Fpdf, title string) {
	pdf.SetFillColor(217, 225, 242)
	pdf.SetDrawColor(185, 185, 185)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("goregular", "", 11)
	pdf.CellFormat(0, 7, title, "1", 1, "L", true, 0, "")
}

func writePDFKeyValue(pdf *gofpdf.Fpdf, key string, value string) {
	pdf.SetFillColor(245, 245, 245)
	pdf.SetFont("goregular", "", 10)
	pdf.CellFormat(0, 6, normalizeValue(key), "1", 1, "L", true, 0, "")
	pdf.SetFillColor(255, 255, 255)
	pdf.MultiCell(0, 6, normalizeValue(value), "1", "L", true)
	pdf.Ln(0.8)
}

func writeZoneTableToPDF(pdf *gofpdf.Fpdf, rows []ZoneExportRow) {
	pdf.SetFont("goregular", "", 9)
	pdf.SetFillColor(242, 242, 242)
	pdf.CellFormat(24, 6, "№ зони", "1", 0, "C", true, 0, "")
	pdf.CellFormat(70, 6, "Назва", "1", 0, "C", true, 0, "")
	pdf.CellFormat(45, 6, "Тип", "1", 0, "C", true, 0, "")
	pdf.CellFormat(0, 6, "Стан", "1", 1, "C", true, 0, "")

	pdf.SetFillColor(255, 255, 255)
	for _, z := range rows {
		pdf.CellFormat(24, 6, normalizeValue(z.Number), "1", 0, "L", false, 0, "")
		pdf.CellFormat(70, 6, normalizeValue(z.Name), "1", 0, "L", false, 0, "")
		pdf.CellFormat(45, 6, normalizeValue(z.Type), "1", 0, "L", false, 0, "")
		pdf.CellFormat(0, 6, normalizeValue(z.Status), "1", 1, "L", false, 0, "")
	}
}

func writeResponsibleTableToPDF(pdf *gofpdf.Fpdf, rows []ResponsibleExportRow) {
	pdf.SetFont("goregular", "", 9)
	pdf.SetFillColor(242, 242, 242)
	pdf.CellFormat(70, 6, "Ім'я", "1", 0, "C", true, 0, "")
	pdf.CellFormat(50, 6, "Телефон", "1", 0, "C", true, 0, "")
	pdf.CellFormat(0, 6, "Примітка", "1", 1, "C", true, 0, "")

	pdf.SetFillColor(255, 255, 255)
	for _, p := range rows {
		pdf.CellFormat(70, 6, normalizeValue(p.Name), "1", 0, "L", false, 0, "")
		pdf.CellFormat(50, 6, normalizeValue(p.Phone), "1", 0, "L", false, 0, "")
		pdf.CellFormat(0, 6, normalizeValue(p.Note), "1", 1, "L", false, 0, "")
	}
}

func normalizeValue(v string) string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "Немає"
	}
	return trimmed
}

func ensureZones(items []ZoneExportRow) []ZoneExportRow {
	if len(items) == 0 {
		return []ZoneExportRow{{Number: "Немає", Name: "Немає", Type: "Немає", Status: "Немає"}}
	}
	return items
}

func ensureResponsibles(items []ResponsibleExportRow) []ResponsibleExportRow {
	if len(items) == 0 {
		return []ResponsibleExportRow{{Name: "Немає", Phone: "Немає", Note: "Немає"}}
	}
	return items
}

func buildFilePath(objectID int, ext string, outputDir string) (string, error) {
	baseDir := strings.TrimSpace(outputDir)
	if baseDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
		baseDir = wd
	}

	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}

	absDir, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve export directory: %w", err)
	}

	ts := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("object_%d_export_%s.%s", objectID, ts, ext)
	fileName = sanitizeFileName(fileName)

	return filepath.Join(absDir, fileName), nil
}

func sanitizeFileName(name string) string {
	clean := invalidFilenameChars.ReplaceAllString(name, "_")
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return "export_file"
	}
	return clean
}

func addSectionHeaderRange(f *excelize.File, sheet string, row int, title string, fromCol string, toCol string, styleID int) int {
	_ = f.SetCellValue(sheet, fmt.Sprintf("%s%d", fromCol, row), title)
	_ = f.MergeCell(sheet, fmt.Sprintf("%s%d", fromCol, row), fmt.Sprintf("%s%d", toCol, row))
	if styleID > 0 {
		_ = f.SetCellStyle(sheet, fmt.Sprintf("%s%d", fromCol, row), fmt.Sprintf("%s%d", toCol, row), styleID)
	}
	return row + 1
}

func setKV(f *excelize.File, sheet string, row int, key string, value string, keyStyleID int, valueStyleID int) {
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), key)
	_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), normalizeValue(value))
	_ = f.MergeCell(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("D%d", row))
	if keyStyleID > 0 {
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), keyStyleID)
	}
	if valueStyleID > 0 {
		_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("D%d", row), valueStyleID)
	}
}

func setTableHeaders(f *excelize.File, sheet string, row int, headers []string, styleID int) {
	for i, h := range headers {
		col := string(rune('A' + i))
		_ = f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), h)
		if styleID > 0 {
			_ = f.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", col, row), styleID)
		}
	}
}

func setTableRow(f *excelize.File, sheet string, row int, values []string, styleID int) {
	for i, v := range values {
		col := string(rune('A' + i))
		_ = f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), v)
		if styleID > 0 {
			_ = f.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", col, row), styleID)
		}
	}
}
