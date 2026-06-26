package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/xuri/excelize/v2"
	"obj_catalog_fyne_v3/pkg/gdrive"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

// GoogleDriveUploadError indicates that the Excel row was appended successfully but the Google Drive upload failed.
type GoogleDriveUploadError struct {
	Err error
}

func (e *GoogleDriveUploadError) Error() string {
	return fmt.Sprintf("помилка завантаження на Google Drive: %v", e.Err)
}

// ContactInfo stores basic contact information
type ContactInfo struct {
	Name  string
	Phone string
}

// ObjectReportRow represents a row in the accepted/deleted objects sheet
type ObjectReportRow struct {
	ObjN         int64
	DisplayNum   string
	LaunchDate   string
	Contract     string
	FullName     string
	LegalAddr    string
	ShortName    string
	PhysicalAddr string
	PKK          string
	SCS          string
	SIM1         string
	SIM2         string
	Payment      string
	Email        string
	ManagerName  string
	ManagerPhone string
	Notes        string
}

// parseLaunchDate tries to parse common date formats to help sort objects by creation date
func parseLaunchDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	// Try parsing just the date if it's mixed with contract text, e.g. "1001ос 17.11.21" -> find date part
	fields := strings.Fields(s)
	for _, f := range fields {
		f = strings.TrimSpace(f)
		// Try dd.mm.yy
		t, err := time.Parse("02.01.06", f)
		if err == nil {
			return t
		}
		// Try dd.mm.yyyy
		t, err = time.Parse("02.01.2006", f)
		if err == nil {
			return t
		}
		// Try yyyy-mm-dd
		t, err = time.Parse("2006-01-02", f)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

// GenerateAcceptedObjectsReport queries all accepted objects from the DB, sorts them by date, and writes them to the Excel sheet 'Прийом'
func GenerateAcceptedObjectsReport(db *sqlx.DB, filePath string) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Query all objects (excluding type 1)
	queryObjects := `
		SELECT 
			oi.OBJUIN,
			oi.OBJN,
			COALESCE(oi.GSMHIDENINT, 0) AS GSMHIDENINT,
			COALESCE(oi.RESERVTEXT, '') AS RESERVTEXT,
			COALESCE(oi.CONTRACT1, '') AS CONTRACT1,
			COALESCE(oi.OBJFULLNAME1, '') AS OBJFULLNAME1,
			COALESCE(oi.ADDRESS1, '') AS ADDRESS1,
			COALESCE(oi.OBJSHORTNAME1, '') AS OBJSHORTNAME1,
			COALESCE(ot.OBJTYPE1, '') AS OBJTYPE1,
			COALESCE(p.PANELMARK1, '') AS PANELMARK1,
			COALESCE(oi.GSMPHONE, '') AS GSMPHONE,
			COALESCE(oi.GSMPHONE2, '') AS GSMPHONE2,
			COALESCE(oi.NOTES1, '') AS NOTES1
		FROM OBJECTS_INFO oi
		LEFT JOIN OBJTYPES ot ON ot.ID = oi.OBJTYPEID
		LEFT JOIN PPK p ON oi.PPKID = p.ID + 100
		WHERE oi.OBJTYPEID <> 1
		ORDER BY oi.OBJN
	`

	type dbRow struct {
		ObjUIN        int64  `db:"OBJUIN"`
		ObjN          int64  `db:"OBJN"`
		GSMHiddenN    int64  `db:"GSMHIDENINT"`
		ReservText    string `db:"RESERVTEXT"`
		Contract1     string `db:"CONTRACT1"`
		ObjFullName1  string `db:"OBJFULLNAME1"`
		Address1      string `db:"ADDRESS1"`
		ObjShortName1 string `db:"OBJSHORTNAME1"`
		ObjType1      string `db:"OBJTYPE1"`
		PanelMark1    string `db:"PANELMARK1"`
		GsmPhone      string `db:"GSMPHONE"`
		GsmPhone2     string `db:"GSMPHONE2"`
		Notes1        string `db:"NOTES1"`
	}

	var dbObjects []dbRow
	if err := db.SelectContext(ctx, &dbObjects, db.Rebind(queryObjects)); err != nil {
		return fmt.Errorf("failed to query objects: %w", err)
	}

	// 2. Query contacts to map the first manager/contact for each object
	queryContacts := `
		SELECT 
			OBJUIN,
			COALESCE(SURNAME1, '') AS SURNAME1,
			COALESCE(NAME1, '') AS NAME1,
			COALESCE(SECNAME1, '') AS SECNAME1,
			COALESCE(PHONES1, '') AS PHONES1,
			ORDER1
		FROM PERSONAL
		ORDER BY OBJUIN, COALESCE(ORDER1, 32767), ID
	`

	type contactRow struct {
		ObjUIN   int64  `db:"OBJUIN"`
		Surname1 string `db:"SURNAME1"`
		Name1    string `db:"NAME1"`
		SecName1 string `db:"SECNAME1"`
		Phones1  string `db:"PHONES1"`
		Order1   *int16 `db:"ORDER1"`
	}

	var dbContacts []contactRow
	if err := db.SelectContext(ctx, &dbContacts, db.Rebind(queryContacts)); err != nil {
		return fmt.Errorf("failed to query contacts: %w", err)
	}

	contactsMap := make(map[int64]ContactInfo)
	for _, c := range dbContacts {
		if _, exists := contactsMap[c.ObjUIN]; !exists {
			fullName := strings.TrimSpace(c.Surname1 + " " + c.Name1 + " " + c.SecName1)
			contactsMap[c.ObjUIN] = ContactInfo{
				Name:  fullName,
				Phone: strings.TrimSpace(c.Phones1),
			}
		}
	}

	// 3. Map to report rows
	rows := make([]ObjectReportRow, 0, len(dbObjects))
	for _, obj := range dbObjects {
		contact := contactsMap[obj.ObjUIN]
		displayNumber := fmt.Sprintf("%d", obj.ObjN)
		if obj.GSMHiddenN > 0 {
			displayNumber = fmt.Sprintf("%d", obj.GSMHiddenN)
		}
		rows = append(rows, ObjectReportRow{
			ObjN:         obj.ObjN,
			DisplayNum:   displayNumber,
			LaunchDate:   strings.TrimSpace(obj.ReservText),
			Contract:     strings.TrimSpace(obj.Contract1),
			FullName:     strings.TrimSpace(obj.ObjFullName1),
			LegalAddr:    strings.TrimSpace(obj.Address1),
			ShortName:    strings.TrimSpace(obj.ObjShortName1),
			PhysicalAddr: strings.TrimSpace(obj.Address1),
			PKK:          strings.TrimSpace(obj.ObjType1),
			SCS:          strings.TrimSpace(obj.PanelMark1),
			SIM1:         formatSIMTo9Digits(obj.GsmPhone),
			SIM2:         formatSIMTo9Digits(obj.GsmPhone2),
			ManagerName:  contact.Name,
			ManagerPhone: contact.Phone,
			Notes:        strings.TrimSpace(obj.Notes1),
		})
	}

	// 4. Sort rows by launch date (chronological)
	sort.SliceStable(rows, func(i, j int) bool {
		t1 := parseLaunchDate(rows[i].LaunchDate)
		t2 := parseLaunchDate(rows[j].LaunchDate)
		if t1.Equal(t2) {
			return rows[i].ObjN < rows[j].ObjN
		}
		return t1.Before(t2)
	})

	// 5. Open or create Excel file
	var f *excelize.File
	var err error
	if _, errStat := os.Stat(filePath); errStat == nil {
		f, err = excelize.OpenFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to open existing file: %w", err)
		}
	} else {
		f = excelize.NewFile()
	}
	defer f.Close()

	// Ensure sheet 'Прийом' exists and clear it
	sheetName := "Прийом"
	f.DeleteSheet(sheetName) // Delete old sheet if it exists
	f.NewSheet(sheetName)

	// Set headers
	headers := []string{
		"собсс", "Дата підключен. до ПЦС", "Дата угоди",
		"Юридична назва, згідно угоди", "Юридична адреса, згідно угоди",
		"Фізична назва об’єкту по вивисці", "Фізична адреса об’єкту",
		"ПКП", "СЦС", "Основний канал зв’язку / телефон підключення ",
		"Резервний канал зв’язку / телефон підключення ", "Місячна оплата",
		"Електронна пошта об’єкту", "Керівник об’єкту", "Контакт керівника", "Примітки",
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#D9E1F2"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    []excelize.Border{{Type: "left", Color: "D9D9D9", Style: 1}, {Type: "right", Color: "D9D9D9", Style: 1}, {Type: "top", Color: "D9D9D9", Style: 1}, {Type: "bottom", Color: "D9D9D9", Style: 1}},
	})

	cellStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{WrapText: true, Vertical: "center"},
		Border:    []excelize.Border{{Type: "left", Color: "E0E0E0", Style: 1}, {Type: "right", Color: "E0E0E0", Style: 1}, {Type: "top", Color: "E0E0E0", Style: 1}, {Type: "bottom", Color: "E0E0E0", Style: 1}},
	})

	for colIdx, h := range headers {
		colName, _ := excelize.ColumnNumberToName(colIdx + 1)
		cell := fmt.Sprintf("%s1", colName)
		_ = f.SetCellValue(sheetName, cell, h)
		_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}
	_ = f.SetRowHeight(sheetName, 1, 28)

	// Write rows
	for rowIdx, row := range rows {
		rNum := rowIdx + 2
		vals := []interface{}{
			row.DisplayNum,
			row.LaunchDate,
			row.Contract,
			row.FullName,
			row.LegalAddr,
			row.ShortName,
			row.PhysicalAddr,
			row.PKK,
			row.SCS,
			row.SIM1,
			row.SIM2,
			row.Payment,
			row.Email,
			row.ManagerName,
			row.ManagerPhone,
			row.Notes,
		}

		for colIdx, val := range vals {
			colName, _ := excelize.ColumnNumberToName(colIdx + 1)
			cell := fmt.Sprintf("%s%d", colName, rNum)
			_ = f.SetCellValue(sheetName, cell, val)
			_ = f.SetCellStyle(sheetName, cell, cell, cellStyle)
		}
		_ = f.SetRowHeight(sheetName, rNum, 22)
	}

	// Set default column widths
	colWidths := map[string]float64{
		"A": 12, "B": 18, "C": 18, "D": 35, "E": 35, "F": 30, "G": 30,
		"H": 15, "I": 15, "J": 20, "K": 20, "L": 12, "M": 20, "N": 25, "O": 20, "P": 30,
	}
	for col, w := range colWidths {
		_ = f.SetColWidth(sheetName, col, col, w)
	}

	// If it was a new file, active sheet might be the default "Sheet1"
	if f.GetSheetName(f.GetActiveSheetIndex()) == "Sheet1" {
		f.DeleteSheet("Sheet1")
	}

	// Create parent directory if it does not exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := f.SaveAs(filePath); err != nil {
		return fmt.Errorf("failed to save accepted objects report: %w", err)
	}

	return nil
}

// AppendObjectToDeletedXLSX queries the details of the object and appends it to the 'Зняття' sheet in the Excel report file
// AppendObjectToDeletedXLSX appends the object details to the 'Зняття' sheet in the Excel report file
func AppendObjectToDeletedXLSX(obj *models.Object, contacts []models.Contact, pdfFilePath string, filePath string) error {
	if obj == nil {
		return fmt.Errorf("object is nil")
	}

	var managerName, managerPhone string
	if len(contacts) > 0 {
		managerName = contacts[0].Name
		managerPhone = contacts[0].Phone
	}

	displayNumber := obj.DisplayNumber
	if obj.GSMHiddenN > 0 {
		displayNumber = fmt.Sprintf("%d", obj.GSMHiddenN)
	} else if displayNumber == "" {
		displayNumber = fmt.Sprintf("%d", obj.ID)
	}

	// 1. Determine Google Drive folder name
	folderName := "Об'єкти МОСТП"
	if ids.IsCASLObjectID(obj.ID) {
		folderName = "Об'єкти casl"
	} else if ids.IsPhoenixObjectID(obj.ID) {
		folderName = "Об'єкти фенікс"
	}
	log.Info().Int("object_id", obj.ID).Str("folder_name", folderName).Msg("Determined Google Drive folder for object deletion")

	// 2. Google Drive upload
	var driveLink string
	var uploadErr error
	credentialsPath := "credentials.json"
	tokenPath := "token.json"

	if pdfFilePath == "" {
		uploadErr = fmt.Errorf("pdf file path is empty")
		log.Error().Msg("Cannot upload to Google Drive: PDF file path is empty")
	} else if _, err := os.Stat(pdfFilePath); os.IsNotExist(err) {
		uploadErr = fmt.Errorf("pdf file does not exist locally: %s", pdfFilePath)
		log.Error().Str("pdf_path", pdfFilePath).Msg("Cannot upload to Google Drive: PDF file does not exist locally")
	} else if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		uploadErr = fmt.Errorf("credentials.json not found in root")
		log.Error().Msg("Cannot upload to Google Drive: credentials.json not found in root")
	} else {
		log.Info().Str("credentials_path", credentialsPath).Msg("Initializing Google Drive OAuth2 client...")
		srv, err := gdrive.NewService(credentialsPath, tokenPath)
		if err != nil {
			uploadErr = fmt.Errorf("failed to initialize google drive: %w", err)
			log.Error().Err(err).Msg("Google Drive initialization failed")
		} else {
			fileName := filepath.Base(pdfFilePath)
			if fileName == "." || fileName == "/" || fileName == "\\" || fileName == "" {
				fileName = fmt.Sprintf("Object_%d.pdf", obj.ID)
			}
			log.Info().Str("fileName", fileName).Str("pdf_path", pdfFilePath).Str("folder", folderName).Msg("Uploading PDF to Google Drive...")
			driveLink, uploadErr = srv.UploadAndShareFile(context.Background(), pdfFilePath, fileName, folderName)
			if uploadErr != nil {
				log.Error().Err(uploadErr).Msg("Failed to upload PDF to Google Drive")
			} else {
				log.Info().Str("link", driveLink).Msg("Successfully uploaded PDF to Google Drive and shared publicly")
			}
		}
	}

	notesVal := obj.Notes1
	if uploadErr != nil {
		if notesVal != "" {
			notesVal += " | "
		}
		notesVal += fmt.Sprintf("[Локальний файл: %s] (Не завантажено на Google Drive: %v)", filepath.Base(pdfFilePath), uploadErr)
	} else if driveLink != "" {
		notesVal = driveLink
	}

	// 3. Open Excel file
	log.Info().Str("excel_path", filePath).Msg("Opening Excel deleted report...")
	var f *excelize.File
	var err error
	if _, errStat := os.Stat(filePath); errStat == nil {
		f, err = excelize.OpenFile(filePath)
		if err != nil {
			log.Error().Err(err).Str("excel_path", filePath).Msg("Failed to open existing Excel file")
			return fmt.Errorf("failed to open existing file: %w", err)
		}
	} else {
		f = excelize.NewFile()
		log.Info().Str("excel_path", filePath).Msg("Created new Excel file since it did not exist")
	}
	defer f.Close()

	sheetName := "Зняття"

	// Create sheet if it does not exist
	sheetIndex, err := f.GetSheetIndex(sheetName)
	if err != nil || sheetIndex < 0 {
		f.NewSheet(sheetName)
		log.Info().Str("sheet", sheetName).Msg("Created new sheet 'Зняття' in Excel file")
		// Write headers if new sheet
		headers := []string{
			"Пультовий номер", "Дата відключен. від ПЦС", "Дата припинен. угоди",
			"Юридична назва, згідно угоди", "Юридична адреса, згідно угоди",
			"Фізична назва об’єкту по вивисці", "Фізична адреса об’єкту",
			"ПКП", "СЦС", "Основний канал зв’язку / телефон підключення ",
			"Резервний канал зв’язку / телефон підключення ", "Місячна оплата",
			"Електронна пошта об’єкту", "Керівник об’єкту", "Контакт керівника", "Примітки",
		}
		headerStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true},
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"#F2F2F2"}, Pattern: 1},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
			Border:    []excelize.Border{{Type: "left", Color: "D9D9D9", Style: 1}, {Type: "right", Color: "D9D9D9", Style: 1}, {Type: "top", Color: "D9D9D9", Style: 1}, {Type: "bottom", Color: "D9D9D9", Style: 1}},
		})
		for colIdx, h := range headers {
			colName, _ := excelize.ColumnNumberToName(colIdx + 1)
			cell := fmt.Sprintf("%s1", colName)
			_ = f.SetCellValue(sheetName, cell, h)
			_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}
		_ = f.SetRowHeight(sheetName, 1, 28)
	}

	// 4. Find next row
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Error().Err(err).Str("sheet", sheetName).Msg("Failed to read rows from sheet")
		return fmt.Errorf("failed to get rows of sheet %s: %w", sheetName, err)
	}
	nextRow := len(rows) + 1
	log.Info().Int("next_row", nextRow).Msg("Inserting data row into Excel")

	// 5. Append data row
	todayStr := time.Now().Format("02.01.06") // Format dd.mm.yy
	vals := []interface{}{
		displayNumber,
		todayStr,
		obj.ContractNum,
		obj.Name,
		obj.Address,
		obj.Name,
		obj.Address,
		obj.DeviceType,
		obj.PanelMark,
		formatSIMTo9Digits(obj.SIM1),
		formatSIMTo9Digits(obj.SIM2),
		"", // Місячна оплата
		"", // Електронна пошта
		managerName,
		managerPhone,
		notesVal,
	}

	cellStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{WrapText: true, Vertical: "center"},
		Border:    []excelize.Border{{Type: "left", Color: "E0E0E0", Style: 1}, {Type: "right", Color: "E0E0E0", Style: 1}, {Type: "top", Color: "E0E0E0", Style: 1}, {Type: "bottom", Color: "E0E0E0", Style: 1}},
	})

	for colIdx, val := range vals {
		colName, _ := excelize.ColumnNumberToName(colIdx + 1)
		cell := fmt.Sprintf("%s%d", colName, nextRow)
		_ = f.SetCellValue(sheetName, cell, val)
		_ = f.SetCellStyle(sheetName, cell, cell, cellStyle)
	}
	_ = f.SetRowHeight(sheetName, nextRow, 22)

	// Set column widths if new
	if len(rows) <= 1 {
		colWidths := map[string]float64{
			"A": 12, "B": 18, "C": 18, "D": 35, "E": 35, "F": 30, "G": 30,
			"H": 15, "I": 15, "J": 20, "K": 20, "L": 12, "M": 20, "N": 25, "O": 20, "P": 30,
		}
		for col, w := range colWidths {
			_ = f.SetColWidth(sheetName, col, col, w)
		}
	}

	// Clean default sheet
	if f.GetSheetName(f.GetActiveSheetIndex()) == "Sheet1" {
		f.DeleteSheet("Sheet1")
	}

	// Save file
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Error().Err(err).Str("dir", dir).Msg("Failed to create Excel output directory")
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := f.SaveAs(filePath); err != nil {
		log.Error().Err(err).Str("excel_path", filePath).Msg("Failed to save Excel file")
		return fmt.Errorf("failed to save deleted objects report: %w", err)
	}
	log.Info().Str("excel_path", filePath).Msg("Successfully saved Excel deleted report")

	if uploadErr != nil {
		return &GoogleDriveUploadError{Err: uploadErr}
	}

	return nil
}

// formatSIMTo9Digits normalizes a Ukrainian mobile phone number to operator code + 7 digits (e.g. 501234567)
func formatSIMTo9Digits(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Filter only digits
	var sb strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	digits := sb.String()
	if len(digits) >= 9 {
		return digits[len(digits)-9:]
	}
	return raw
}
