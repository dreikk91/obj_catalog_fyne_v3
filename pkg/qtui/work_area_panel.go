//go:build qt

package qtui

import (
	"fmt"
	"slices"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	objexport "obj_catalog_fyne_v3/pkg/export"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

type WorkAreaPanel struct {
	*qt.QWidget
	tabs                     *qt.QTabWidget
	headerName               *qt.QLabel
	headerAddress            *qt.QLabel
	cardFields               map[string]*qt.QLineEdit
	cardNotes                *qt.QTextEdit
	deviceVM                 *viewmodels.WorkAreaDeviceViewModel
	zonesModel               *qt.QStandardItemModel
	contactsModel            *qt.QStandardItemModel
	eventsModel              *qt.QStandardItemModel
	zonesTree                *qt.QTreeView
	contactsTable            *qt.QTableView
	eventsTable              *qt.QTableView
	autoSized                bool
	OnEditObjectRequested    func()
	OnSIMManagementRequested func()

	// Export fields
	currentObject     *models.Object
	zones             []models.Zone
	contacts          []models.Contact
	events            []models.Event
	dataProvider      contracts.DataProvider
	viewModel         *viewmodels.WorkAreaViewModel
	exportVM          *viewmodels.WorkAreaExportViewModel
	exportPDFBtn      *qt.QPushButton
	exportXLSXBtn     *qt.QPushButton
	copyExcelBtn      *qt.QPushButton
	addToDeletedBtn   *qt.QPushButton
	OnRunOnMainThread func(f func())
}

func NewWorkAreaPanel() *WorkAreaPanel {
	panel := &WorkAreaPanel{
		QWidget:   qt.NewQWidget2(),
		deviceVM:  viewmodels.NewWorkAreaDeviceViewModel(),
		viewModel: viewmodels.NewWorkAreaViewModel(),
		exportVM:  viewmodels.NewWorkAreaExportViewModel(),
	}
	layout := qt.NewQVBoxLayout(panel.QWidget)
	panel.headerName = qt.NewQLabel3("Оберіть об'єкт зі списку")
	panel.headerName.SetStyleSheet("font-weight: 600; font-size: 12pt;")
	panel.headerAddress = qt.NewQLabel3("")
	panel.headerAddress.SetWordWrap(true)
	actionsLayout := qt.NewQHBoxLayout2()
	editButton := qt.NewQPushButton3("Редагувати")
	editButton.OnClicked(func() {
		if panel.OnEditObjectRequested != nil {
			panel.OnEditObjectRequested()
		}
	})
	simButton := qt.NewQPushButton3("SIM")
	simButton.OnClicked(func() {
		if panel.OnSIMManagementRequested != nil {
			panel.OnSIMManagementRequested()
		}
	})
	actionsLayout.AddWidget(editButton.QWidget)
	actionsLayout.AddWidget(simButton.QWidget)
	actionsLayout.AddStretch()
	panel.tabs = qt.NewQTabWidget2()
	panel.tabs.AddTab(panel.buildObjectCardTab(), "Картка")
	panel.zonesModel = qt.NewQStandardItemModel2(0, 6)
	panel.zonesTree = newTree(panel.zonesModel, zoneTreeHeaders())
	panel.tabs.AddTab(panel.zonesTree.QWidget, "Зони")
	panel.contactsModel = qt.NewQStandardItemModel2(0, 4)
	panel.contactsTable = newTable(panel.contactsModel, []string{"Ім'я", "Посада", "Телефон", "Група"})
	panel.tabs.AddTab(panel.contactsTable.QWidget, "Контакти")
	panel.eventsModel = qt.NewQStandardItemModel2(0, 3)
	panel.eventsTable = newTable(panel.eventsModel, []string{"Час", "Подія", "Опис"})
	panel.tabs.AddTab(panel.eventsTable.QWidget, "Журнал")
	panel.tabs.AddTab(panel.buildExportTab(), "Експорт")
	layout.AddWidget(panel.headerName.QWidget)
	layout.AddWidget(panel.headerAddress.QWidget)
	layout.AddLayout(actionsLayout.QLayout)
	layout.AddWidget(panel.tabs.QWidget)
	panel.SetLayout(layout.QLayout)
	return panel
}

func (panel *WorkAreaPanel) SetObject(object models.Object, zones []models.Zone, contacts []models.Contact, events []models.Event) {
	if panel == nil {
		return
	}
	panel.currentObject = &object
	panel.zones = zones
	panel.contacts = contacts
	panel.events = events

	panel.headerName.SetText(strings.TrimSpace(object.Name) + " (№" + viewmodels.ObjectDisplayNumber(object) + ")")
	panel.headerAddress.SetText(workAreaHeaderAddress(object))

	presentation := panel.deviceVM.BuildObjectPresentation(object)
	panel.setObjectCard(object, presentation)

	if panel.dataProvider != nil {
		go func(id int) {
			externalData := panel.viewModel.LoadExternalData(panel.dataProvider, id)
			updateUI := func() {
				if panel.currentObject == nil || panel.currentObject.ID != id {
					return
				}
				panel.setCardValue("Тест-сигнал", externalData.TestMessage)
				panel.setCardValue("Рівень сигналу", externalData.Signal)

				lastTestStr := "—"
				if !externalData.LastTest.IsZero() {
					lastTestStr = externalData.LastTest.Format("02.01.2006 15:04:05")
				}
				panel.setCardValue("Останній тест", lastTestStr)

				lastMsgStr := "—"
				if !externalData.LastMessage.IsZero() {
					lastMsgStr = externalData.LastMessage.Format("02.01.2006 15:04:05")
				}
				panel.setCardValue("Ост. повідомлення", lastMsgStr)
			}

			if panel.OnRunOnMainThread != nil {
				panel.OnRunOnMainThread(updateUI)
			} else {
				updateUI()
			}
		}(object.ID)
	}

	setZoneRows(panel.zonesModel, zones)
	setContactRows(panel.contactsModel, contacts)
	setEventRows(panel.eventsModel, events)

	if panel.exportPDFBtn != nil {
		panel.exportPDFBtn.SetEnabled(true)
	}
	if panel.exportXLSXBtn != nil {
		panel.exportXLSXBtn.SetEnabled(true)
	}
	if panel.copyExcelBtn != nil {
		panel.copyExcelBtn.SetEnabled(true)
	}
	if panel.addToDeletedBtn != nil {
		panel.addToDeletedBtn.SetEnabled(true)
	}

	if !panel.autoSized {
		resizeTreeToContents(panel.zonesTree)
		resizeTableToContents(panel.contactsTable)
		resizeTableToContents(panel.eventsTable)
		panel.autoSized = true
	}
}

func (panel *WorkAreaPanel) SetLoading(object models.Object) {
	if panel == nil {
		return
	}
	panel.currentObject = &object
	panel.zones = nil
	panel.contacts = nil
	panel.events = nil

	if panel.exportPDFBtn != nil {
		panel.exportPDFBtn.SetEnabled(false)
	}
	if panel.exportXLSXBtn != nil {
		panel.exportXLSXBtn.SetEnabled(false)
	}
	if panel.copyExcelBtn != nil {
		panel.copyExcelBtn.SetEnabled(false)
	}
	if panel.addToDeletedBtn != nil {
		panel.addToDeletedBtn.SetEnabled(false)
	}

	panel.headerName.SetText(strings.TrimSpace(object.Name) + " (№" + viewmodels.ObjectDisplayNumber(object) + ")")
	panel.headerAddress.SetText(workAreaHeaderAddress(object))
	panel.setCardValue("Назва", "Завантаження картки об'єкта...")
}

func (panel *WorkAreaPanel) buildObjectCardTab() *qt.QWidget {
	panel.cardFields = make(map[string]*qt.QLineEdit)
	panel.cardNotes = qt.NewQTextEdit2()
	panel.cardNotes.SetReadOnly(true)
	panel.cardNotes.SetMinimumHeight(72)
	panel.cardNotes.SetMaximumHeight(120)

	content := qt.NewQWidget2()
	grid := qt.NewQGridLayout(content)
	grid.SetHorizontalSpacing(12)
	grid.SetVerticalSpacing(6)
	grid.SetColumnStretch(1, 1)
	grid.SetColumnStretch(3, 1)
	grid.SetColumnStretch(5, 1)

	row := 0
	row = panel.addCardSection(grid, row, "Основне")
	row = panel.addCardFields(grid, row, []string{"Номер", "Договір", "Телефон"})
	row = panel.addCardFields(grid, row, []string{"Назва", "Район", "Адреса"})
	row = panel.addCardFields(grid, row, []string{"Координати", "Геокодування", "Словник об'єкта"})

	row = panel.addCardSection(grid, row, "Обладнання і зв'язок")
	row = panel.addCardFields(grid, row, []string{"Прилад", "Шифр приладу", "Контроль тестів"})
	row = panel.addCardFields(grid, row, []string{"Групи", "Взяття/Зняття", "SIM-карта"})
	row = panel.addCardFields(grid, row, []string{"SIM 1", "SIM 2", "Живлення"})

	row = panel.addCardSection(grid, row, "Поточний оперативний стан")
	row = panel.addCardFields(grid, row, []string{"Охорона", "Зв'язок", "Ост. повідомлення"})
	row = panel.addCardFields(grid, row, []string{"АКБ", "Канал", "Тест-сигнал"})
	row = panel.addCardFields(grid, row, []string{"Рівень сигналу", "Останній тест", "Напрямок"})

	row = panel.addCardSection(grid, row, "Додатково")
	grid.AddWidget3(qt.NewQLabel3("Примітки").QWidget, row, 0, 1, 1)
	grid.AddWidget3(panel.cardNotes.QWidget, row, 1, 1, 5)
	row++

	scroll := qt.NewQScrollArea2()
	scroll.SetWidgetResizable(true)
	scroll.SetWidget(content)
	return scroll.QWidget
}

func (panel *WorkAreaPanel) addCardSection(grid *qt.QGridLayout, row int, title string) int {
	label := qt.NewQLabel3(title)
	label.SetStyleSheet("font-weight: bold; color: #1a73e8; padding-top: 10px;")
	grid.AddWidget3(label.QWidget, row, 0, 1, 6)
	return row + 1
}

func (panel *WorkAreaPanel) addCardWideField(grid *qt.QGridLayout, row int, labelText string) int {
	grid.AddWidget3(qt.NewQLabel3(labelText).QWidget, row, 0, 1, 1)
	field := qt.NewQLineEdit2()
	field.SetReadOnly(true)
	grid.AddWidget3(field.QWidget, row, 1, 1, 5)
	panel.cardFields[labelText] = field
	return row + 1
}

func (panel *WorkAreaPanel) addCardFields(grid *qt.QGridLayout, row int, labels []string) int {
	col := 0
	for _, label := range labels {
		grid.AddWidget3(qt.NewQLabel3(label).QWidget, row, col, 1, 1)
		field := qt.NewQLineEdit2()
		field.SetReadOnly(true)
		grid.AddWidget3(field.QWidget, row, col+1, 1, 1)
		panel.cardFields[label] = field
		col += 2
	}
	return row + 1
}

func (panel *WorkAreaPanel) setObjectCard(object models.Object, presentation viewmodels.WorkAreaDevicePresentation) {
	panel.setCardValue("Номер", viewmodels.ObjectDisplayNumber(object))
	panel.setCardValue("Договір", object.ContractNum)
	panel.setCardValue("Телефон", presentation.PhoneCopyText)

	panel.setCardValue("Назва", strings.TrimSpace(object.Name))
	panel.setCardValue("Район", "")
	panel.setCardValue("Адреса", strings.TrimSpace(object.Address))

	panel.setCardValue("Координати", object.Location1)
	panel.setCardValue("Геокодування", "")
	panel.setCardValue("Словник об'єкта", "")

	panel.setCardValue("Прилад", trimPresentationPrefix(presentation.DeviceTypeText))
	panel.setCardValue("Шифр приладу", trimPresentationPrefix(presentation.PanelMarkText))
	panel.setCardValue("Контроль тестів", trimPresentationPrefix(presentation.TestControlText))

	panel.setCardValue("Групи", trimPresentationPrefix(presentation.GroupsText))
	panel.setCardValue("Взяття/Зняття", trimPresentationPrefix(presentation.GuardText))
	panel.setCardValue("SIM-карта", trimPresentationPrefix(presentation.SIMText))

	panel.setCardValue("SIM 1", trimPresentationPrefix(presentation.SIM1Text))
	panel.setCardValue("SIM 2", trimPresentationPrefix(presentation.SIM2Text))
	panel.setCardValue("Живлення", trimPresentationPrefix(presentation.PowerText))

	panel.setCardValue("Охорона", objectCardGuardText(object, presentation))
	panel.setCardValue("Зв'язок", objectCardConnectionText(object, presentation))

	lastMsgStr := "—"
	if !object.LastMessageTime.IsZero() {
		lastMsgStr = object.LastMessageTime.Format("02.01.2006 15:04:05")
	}
	panel.setCardValue("Ост. повідомлення", lastMsgStr)

	panel.setCardValue("АКБ", trimPresentationPrefix(presentation.AkbText))
	panel.setCardValue("Канал", trimPresentationPrefix(presentation.ChannelText))
	panel.setCardValue("Тест-сигнал", "Завантаження...")
	panel.setCardValue("Рівень сигналу", object.SignalStrength)

	lastTestStr := "—"
	if !object.LastTestTime.IsZero() {
		lastTestStr = object.LastTestTime.Format("02.01.2006 15:04:05")
	}
	panel.setCardValue("Останній тест", lastTestStr)
	panel.setCardValue("Напрямок", "")

	panel.cardNotes.SetPlainText(emptyDash(object.Notes1))
}

func (panel *WorkAreaPanel) setCardValue(label string, value string) {
	if field, ok := panel.cardFields[label]; ok {
		field.SetText(emptyDash(value))
	}
}

func trimPresentationPrefix(value string) string {
	if idx := strings.Index(value, ": "); idx >= 0 {
		return strings.TrimSpace(value[idx+2:])
	}
	return value
}

func objectCardConnectionText(object models.Object, presentation viewmodels.WorkAreaDevicePresentation) string {
	val := trimPresentationPrefix(presentation.ConnectionText)
	if val == "невідомо" && object.Status == models.StatusNormal {
		return "в нормі"
	}
	return val
}

func objectCardGuardText(object models.Object, presentation viewmodels.WorkAreaDevicePresentation) string {
	val := trimPresentationPrefix(presentation.GuardText)
	if val == "невідомо" && object.Status == models.StatusNormal {
		return "в нормі"
	}
	return val
}

func (panel *WorkAreaPanel) SelectTab(index int) {
	if panel == nil || panel.tabs == nil {
		return
	}
	if index >= 0 && index < panel.tabs.Count() {
		panel.tabs.SetCurrentIndex(index)
	}
}

func (panel *WorkAreaPanel) buildExportTab() *qt.QWidget {
	widget := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(widget)

	title := qt.NewQLabel3("Експорт даних об'єкта")
	title.SetStyleSheet("font-weight: bold; font-size: 11pt; margin-bottom: 8px;")
	layout.AddWidget(title.QWidget)

	info := qt.NewQLabel3("Виберіть потрібний формат для експорту поточного стану об'єкта, його зон, контактів та журналу подій.")
	info.SetWordWrap(true)
	layout.AddWidget(info.QWidget)

	layout.AddSpacing(12)

	panel.exportPDFBtn = qt.NewQPushButton3("Експортувати в PDF")
	panel.exportPDFBtn.SetStyleSheet("padding: 6px; font-weight: 500;")
	panel.exportPDFBtn.OnClicked(func() {
		panel.exportObject("pdf")
	})

	panel.exportXLSXBtn = qt.NewQPushButton3("Експортувати в XLSX (Excel)")
	panel.exportXLSXBtn.SetStyleSheet("padding: 6px; font-weight: 500;")
	panel.exportXLSXBtn.OnClicked(func() {
		panel.exportObject("xlsx")
	})

	panel.copyExcelBtn = qt.NewQPushButton3("Копіювати рядок для Excel (TSV)")
	panel.copyExcelBtn.SetStyleSheet("padding: 6px; font-weight: 500;")
	panel.copyExcelBtn.OnClicked(func() {
		panel.copyRowForExcel()
	})

	panel.addToDeletedBtn = qt.NewQPushButton3("Додати в звіт видалених об'єктів")
	panel.addToDeletedBtn.SetStyleSheet("padding: 6px; font-weight: 500;")
	panel.addToDeletedBtn.OnClicked(func() {
		panel.addObjectToDeleted()
	})

	panel.exportPDFBtn.SetEnabled(false)
	panel.exportXLSXBtn.SetEnabled(false)
	panel.copyExcelBtn.SetEnabled(false)
	panel.addToDeletedBtn.SetEnabled(false)

	layout.AddWidget(panel.exportPDFBtn.QWidget)
	layout.AddWidget(panel.exportXLSXBtn.QWidget)
	layout.AddWidget(panel.copyExcelBtn.QWidget)
	layout.AddWidget(panel.addToDeletedBtn.QWidget)

	layout.AddStretch()
	return widget
}

func (panel *WorkAreaPanel) uiPreferences() config.Preferences {
	return config.NewQtPreferences("MOST", "ObjCatalogQt")
}

func (panel *WorkAreaPanel) exportObject(format string) {
	if panel == nil || panel.currentObject == nil {
		return
	}

	obj := *panel.currentObject
	zones := slices.Clone(panel.zones)
	contacts := slices.Clone(panel.contacts)
	events := slices.Clone(panel.events)

	panel.exportPDFBtn.SetEnabled(false)
	panel.exportXLSXBtn.SetEnabled(false)

	go func() {
		externalData := panel.viewModel.LoadExternalData(panel.dataProvider, obj.ID)
		exportData := panel.exportVM.BuildObjectExportData(obj, zones, contacts, events, externalData)

		uiCfg := config.LoadUIConfig(panel.uiPreferences())
		exportDir := uiCfg.ExportDir

		var (
			filePath string
			err      error
		)

		switch strings.ToLower(format) {
		case "pdf":
			filePath, err = objexport.ExportObjectToPDF(exportData, exportDir)
		case "xlsx":
			filePath, err = objexport.ExportObjectToXLSX(exportData, exportDir)
		default:
			err = fmt.Errorf("unsupported export format: %s", format)
		}

		runCallback := func() {
			panel.exportPDFBtn.SetEnabled(true)
			panel.exportXLSXBtn.SetEnabled(true)

			if err != nil {
				qt.QMessageBox_Critical(panel.QWidget, "Помилка експорту", err.Error())
				return
			}
			qt.QMessageBox_Information(panel.QWidget, "Експорт виконано", "Файл створено:\n"+filePath)
		}

		if panel.OnRunOnMainThread != nil {
			panel.OnRunOnMainThread(runCallback)
		} else {
			runCallback()
		}
	}()
}

func (panel *WorkAreaPanel) copyRowForExcel() {
	if panel == nil || panel.currentObject == nil {
		return
	}

	row := panel.exportVM.BuildExcelRowTSV(*panel.currentObject, panel.contacts)

	clipboard := qt.QGuiApplication_Clipboard()
	if clipboard != nil {
		clipboard.SetText(row)
		qt.QMessageBox_Information(panel.QWidget, "Буфер обміну", "Рядок для Excel скопійовано в буфер обміну")
	} else {
		qt.QMessageBox_Critical(panel.QWidget, "Помилка", "Не вдалося отримати доступ до буфера обміну")
	}
}

func (panel *WorkAreaPanel) addObjectToDeleted() {
	if panel == nil || panel.currentObject == nil {
		return
	}

	excelProvider, ok := panel.dataProvider.(contracts.ExcelReportingProvider)
	if !ok {
		qt.QMessageBox_Information(panel.QWidget, "Помилка", "Поточний провайдер не підтримує експорт в Excel.")
		return
	}

	displayName := viewmodels.ObjectDisplayNumber(*panel.currentObject)
	filePath := `D:\goproject\obj_catalog_fyne_v3\Звіт прийнятих-знятих об’єктів (1).xlsx`

	reply := qt.QMessageBox_Question(panel.QWidget, "Підтвердження", fmt.Sprintf("Додати об'єкт №%s в звіт видалених?", displayName))
	if reply != qt.QMessageBox__Yes {
		return
	}

	panel.addToDeletedBtn.SetEnabled(false)

	go func() {
		obj := *panel.currentObject
		zones := slices.Clone(panel.zones)
		contacts := slices.Clone(panel.contacts)
		events := slices.Clone(panel.events)

		externalData := panel.viewModel.LoadExternalData(panel.dataProvider, obj.ID)
		exportData := panel.exportVM.BuildObjectExportData(obj, zones, contacts, events, externalData)

		uiCfg := config.LoadUIConfig(panel.uiPreferences())
		exportDir := uiCfg.ExportDir

		tempPDFPath, pdfErr := objexport.ExportObjectToPDF(exportData, exportDir)
		if pdfErr != nil {
			runCallback := func() {
				panel.addToDeletedBtn.SetEnabled(true)
				qt.QMessageBox_Critical(panel.QWidget, "Помилка генерації PDF", pdfErr.Error())
			}
			if panel.OnRunOnMainThread != nil {
				panel.OnRunOnMainThread(runCallback)
			} else {
				runCallback()
			}
			return
		}

		err := excelProvider.AppendObjectToDeletedReport(&obj, contacts, tempPDFPath, filePath)

		runCallback := func() {
			panel.addToDeletedBtn.SetEnabled(true)
			if err != nil {
				if gdriveErr, ok := err.(*objexport.GoogleDriveUploadError); ok {
					qt.QMessageBox_Warning(panel.QWidget, "Увага", fmt.Sprintf("Об'єкт додано в Excel, але не завантажено на Google Drive: %v", gdriveErr.Err))
				} else {
					qt.QMessageBox_Critical(panel.QWidget, "Помилка", err.Error())
				}
				return
			}
			qt.QMessageBox_Information(panel.QWidget, "Готово", "Об'єкт додано до знятих/видалених Excel та Google Drive")
		}

		if panel.OnRunOnMainThread != nil {
			panel.OnRunOnMainThread(runCallback)
		} else {
			runCallback()
		}
	}()
}

func (panel *WorkAreaPanel) SetDataProvider(provider contracts.DataProvider) {
	if panel == nil {
		return
	}
	panel.dataProvider = provider
}
