package viewmodels

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/simoperator"
	"obj_catalog_fyne_v3/pkg/utils"
)

const (
	DefaultCASLSIMReportLimit = 1001
	simLookupConcurrencyLimit = 6
	SIMInventorySourceBridge  = "БД/МІСТ"
	SIMInventorySourcePhoenix = "Phoenix"
	SIMInventorySourceCASL    = "CASL Cloud"
)

// SIMInventoryReportProvider описує мінімальний API для зведеного звіту по SIM-картах.
type SIMInventoryReportProvider interface {
	GetObjects() []models.Object
	GetObjectByID(id string) *models.Object
	GetStatisticReport(ctx context.Context, name string, limit int) ([]map[string]any, error)
	GetVodafoneSIMStatus(msisdn string) (contracts.VodafoneSIMStatus, error)
	GetKyivstarSIMStatus(msisdn string) (contracts.KyivstarSIMStatus, error)
	SupportsCASLReports() bool
	ListVodafoneSIMInventory() (map[string]contracts.VodafoneSIMInventoryEntry, error)
	ListKyivstarSIMInventory(numbers []string) (map[string]contracts.KyivstarSIMInventoryEntry, error)
}

type simInventoryBaseRow struct {
	Source       string
	ObjectNumber string
	ObjectName   string
	SIM1         string
	SIM2         string
}

type simInventoryLookupInfo struct {
	Operator string
	Found    bool
	FoundSet bool
	Active   string
	Status   string
	Name     string
	Comment  string
	Error    string
}

type SIMInventoryReportRow struct {
	Source       string
	ObjectNumber string
	ObjectName   string
	SIM1         string
	SIM1Operator string
	SIM1Found    string
	SIM1Active   string
	SIM1Status   string
	SIM1Name     string
	SIM1Comment  string
	SIM2         string
	SIM2Operator string
	SIM2Found    string
	SIM2Active   string
	SIM2Status   string
	SIM2Name     string
	SIM2Comment  string
}

type SIMInventoryReportResult struct {
	Rows                    []SIMInventoryReportRow
	ObjectsCount            int
	SIMCount                int
	LookupErrors            int
	UnknownSIMs             int
	CASLRowsCount           int
	VodafoneInventoryCount  int
	KyivstarInventoryCount  int
	VodafoneInventoryLoaded bool
	KyivstarInventoryLoaded bool
}

type SIMInventoryProgressFunc func(stage string)

type SIMInventoryViewModel struct{}

func NewSIMInventoryViewModel() *SIMInventoryViewModel {
	return &SIMInventoryViewModel{}
}

func (vm *SIMInventoryViewModel) BuildReport(ctx context.Context, provider SIMInventoryReportProvider, caslLimit int, progress ...SIMInventoryProgressFunc) (SIMInventoryReportResult, error) {
	reportProgress := func(stage string) {
		if len(progress) == 0 || progress[0] == nil {
			return
		}
		progress[0](stage)
	}

	reportProgress("Етап 1/5: збираю об'єкти з БД/МІСТ та Phoenix...")
	baseRows, caslRowsCount, err := loadSIMInventoryBaseRows(ctx, provider, caslLimit, reportProgress)
	if err != nil {
		return SIMInventoryReportResult{}, err
	}

	reportProgress("Етап 2/5: завантажую масовий список Vodafone...")
	vodafoneInventory, err := provider.ListVodafoneSIMInventory()
	vodafoneInventoryLoaded := err == nil
	if err != nil {
		vodafoneInventory = nil
		reportProgress("Етап 2/5: масовий список Vodafone недоступний, продовжую без нього")
	} else {
		reportProgress(fmt.Sprintf("Етап 2/5: Vodafone отримано %d номерів", len(vodafoneInventory)))
	}

	reportProgress("Етап 3/5: завантажую масовий список Kyivstar...")
	kyivstarNumbers := collectSIMInventoryNumbers(baseRows, simoperator.Kyivstar)
	kyivstarInventory, err := provider.ListKyivstarSIMInventory(kyivstarNumbers)
	kyivstarInventoryLoaded := err == nil
	if err != nil {
		kyivstarInventory = nil
		reportProgress("Етап 3/5: масовий список Kyivstar недоступний, продовжую без нього")
	} else {
		reportProgress(fmt.Sprintf("Етап 3/5: Kyivstar отримано %d номерів", len(kyivstarInventory)))
	}

	reportProgress("Етап 4/5: звіряю SIM-карти з операторами...")
	lookups, lookupErrors, unknownSIMs := resolveSIMInventoryLookups(
		provider,
		baseRows,
		vodafoneInventory,
		vodafoneInventoryLoaded,
		kyivstarInventory,
		kyivstarInventoryLoaded,
		reportProgress,
	)

	reportProgress("Етап 5/5: формую таблицю звіту...")
	rows := make([]SIMInventoryReportRow, 0, len(baseRows))
	simCount := 0
	for _, base := range baseRows {
		row := SIMInventoryReportRow{
			Source:       base.Source,
			ObjectNumber: base.ObjectNumber,
			ObjectName:   base.ObjectName,
			SIM1:         strings.TrimSpace(base.SIM1),
			SIM2:         strings.TrimSpace(base.SIM2),
		}
		if row.SIM1 != "" {
			simCount++
			applySIMInventoryLookup(&row.SIM1Operator, &row.SIM1Found, &row.SIM1Active, &row.SIM1Status, &row.SIM1Name, &row.SIM1Comment, lookups[NormalizeSIMLookupKey(row.SIM1)])
		}
		if row.SIM2 != "" {
			simCount++
			applySIMInventoryLookup(&row.SIM2Operator, &row.SIM2Found, &row.SIM2Active, &row.SIM2Status, &row.SIM2Name, &row.SIM2Comment, lookups[NormalizeSIMLookupKey(row.SIM2)])
		}
		rows = append(rows, row)
	}

	return SIMInventoryReportResult{
		Rows:                    rows,
		ObjectsCount:            len(rows),
		SIMCount:                simCount,
		LookupErrors:            lookupErrors,
		UnknownSIMs:             unknownSIMs,
		CASLRowsCount:           caslRowsCount,
		VodafoneInventoryCount:  len(vodafoneInventory),
		KyivstarInventoryCount:  len(kyivstarInventory),
		VodafoneInventoryLoaded: vodafoneInventoryLoaded,
		KyivstarInventoryLoaded: kyivstarInventoryLoaded,
	}, nil
}

func (vm *SIMInventoryViewModel) BuildTSV(rows []SIMInventoryReportRow) string {
	lines := make([]string, 0, len(rows)+1)
	header := simInventoryHeader()
	for i := range header {
		header[i] = cleanTSV(header[i])
	}
	lines = append(lines, strings.Join(header, "\t"))
	for _, row := range rows {
		values := simInventoryRowValues(row)
		for i := range values {
			values[i] = cleanTSV(values[i])
		}
		lines = append(lines, strings.Join(values, "\t"))
	}
	return strings.Join(lines, "\n")
}

func (vm *SIMInventoryViewModel) BuildCSV(rows []SIMInventoryReportRow) string {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	writer.Comma = ';'
	_ = writer.Write(simInventoryHeader())
	for _, row := range rows {
		_ = writer.Write(simInventoryRowValues(row))
	}
	writer.Flush()
	return buffer.String()
}

func (vm *SIMInventoryViewModel) FormatSummary(result SIMInventoryReportResult) string {
	return fmt.Sprintf(
		"Об'єктів: %d | SIM: %d | Vodafone: %s | Kyivstar: %s | CASL rows: %d | невідомий оператор: %d | помилок операторних запитів: %d",
		result.ObjectsCount,
		result.SIMCount,
		formatSIMInventoryOperatorCount(result.VodafoneInventoryLoaded, result.VodafoneInventoryCount),
		formatSIMInventoryOperatorCount(result.KyivstarInventoryLoaded, result.KyivstarInventoryCount),
		result.CASLRowsCount,
		result.UnknownSIMs,
		result.LookupErrors,
	)
}

func (vm *SIMInventoryViewModel) FormatReadyStatus(result SIMInventoryReportResult) string {
	return fmt.Sprintf("Готово: %d рядків, звіт готовий до експорту", len(result.Rows))
}

func NormalizeSIMLookupKey(raw string) string {
	return NormalizeSIMInventoryNumber(raw)
}

func NormalizeSIMInventoryNumber(raw string) string {
	digits := utils.DigitsOnly(raw)
	switch {
	case len(digits) == 12 && strings.HasPrefix(digits, "380"):
		return digits
	case len(digits) == 11 && strings.HasPrefix(digits, "80"):
		return "3" + digits
	case len(digits) == 10 && strings.HasPrefix(digits, "0"):
		return "38" + digits
	case len(digits) == 9:
		return "380" + digits
	default:
		return digits
	}
}

func loadSIMInventoryBaseRows(ctx context.Context, provider SIMInventoryReportProvider, caslLimit int, progress SIMInventoryProgressFunc) ([]simInventoryBaseRow, int, error) {
	baseRows := make([]simInventoryBaseRow, 0, 128)

	for _, obj := range provider.GetObjects() {
		if ids.IsCASLObjectID(obj.ID) {
			continue
		}

		item := simInventoryBaseRow{
			Source:       simInventorySourceForObjectID(obj.ID),
			ObjectNumber: strings.TrimSpace(ObjectDisplayNumber(obj)),
			ObjectName:   strings.TrimSpace(obj.Name),
			SIM1:         NormalizeSIMInventoryNumber(obj.SIM1),
			SIM2:         NormalizeSIMInventoryNumber(obj.SIM2),
		}
		if item.Source == SIMInventorySourcePhoenix {
			enriched := provider.GetObjectByID(strconv.Itoa(obj.ID))
			if enriched != nil {
				if value := strings.TrimSpace(ObjectDisplayNumber(*enriched)); value != "" {
					item.ObjectNumber = value
				}
				if value := strings.TrimSpace(enriched.Name); value != "" {
					item.ObjectName = value
				}
				if value := NormalizeSIMInventoryNumber(enriched.SIM1); value != "" {
					item.SIM1 = value
				}
				if value := NormalizeSIMInventoryNumber(enriched.SIM2); value != "" {
					item.SIM2 = value
				}
			}
		}

		if item.ObjectNumber == "" {
			item.ObjectNumber = strconv.Itoa(obj.ID)
		}
		if item.ObjectName == "" {
			item.ObjectName = "—"
		}
		if item.SIM1 == "" && item.SIM2 == "" {
			continue
		}
		baseRows = append(baseRows, item)
	}

	caslRowsCount := 0
	if provider.SupportsCASLReports() {
		if progress != nil {
			progress("Етап 1/5: завантажую CASL stats_devices_v2...")
		}
		caslRows, err := provider.GetStatisticReport(ctx, "stats_devices_v2", caslLimit)
		if err != nil {
			return nil, 0, err
		}
		caslRowsCount = len(caslRows)
		for _, raw := range caslRows {
			item := simInventoryBaseRow{
				Source:       SIMInventorySourceCASL,
				ObjectNumber: strings.TrimSpace(utils.AsString(raw["number"])),
				ObjectName:   strings.TrimSpace(utils.AsString(raw["name"])),
				SIM1:         NormalizeSIMInventoryNumber(utils.AsString(raw["sim1"])),
				SIM2:         NormalizeSIMInventoryNumber(utils.AsString(raw["sim2"])),
			}
			if item.ObjectNumber == "" && item.ObjectName == "" {
				continue
			}
			if item.ObjectName == "" {
				item.ObjectName = "—"
			}
			if item.SIM1 == "" && item.SIM2 == "" {
				continue
			}
			baseRows = append(baseRows, item)
		}
	}

	sort.SliceStable(baseRows, func(i int, j int) bool {
		left := baseRows[i]
		right := baseRows[j]
		if sourceCmp := compareSIMInventorySource(left.Source, right.Source); sourceCmp != 0 {
			return sourceCmp < 0
		}
		if numberCmp := compareSIMInventoryNumbers(left.ObjectNumber, right.ObjectNumber); numberCmp != 0 {
			return numberCmp < 0
		}
		return left.ObjectName < right.ObjectName
	})

	return baseRows, caslRowsCount, nil
}

func resolveSIMInventoryLookups(
	provider SIMInventoryReportProvider,
	rows []simInventoryBaseRow,
	vodafoneInventory map[string]contracts.VodafoneSIMInventoryEntry,
	vodafoneInventoryLoaded bool,
	kyivstarInventory map[string]contracts.KyivstarSIMInventoryEntry,
	kyivstarInventoryLoaded bool,
	progress SIMInventoryProgressFunc,
) (map[string]simInventoryLookupInfo, int, int) {
	type lookupTask struct {
		key      string
		number   string
		operator simoperator.Operator
	}

	results := make(map[string]simInventoryLookupInfo)
	tasks := make([]lookupTask, 0, len(rows)*2)
	seen := make(map[string]struct{}, len(rows)*2)
	unknownCount := 0
	directResolved := 0

	addTask := func(number string) {
		number = strings.TrimSpace(number)
		if number == "" {
			return
		}
		key := NormalizeSIMLookupKey(number)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}

		operator := simoperator.Detect(number)
		if operator == simoperator.Unknown {
			results[key] = simInventoryLookupInfo{}
			unknownCount++
			return
		}
		if operator == simoperator.Lifecell {
			results[key] = simInventoryLookupInfo{
				Operator: simoperator.Label(operator),
				Status:   "API недоступне, не перевіряється",
			}
			directResolved++
			return
		}
		switch operator {
		case simoperator.Vodafone:
			if item, ok := vodafoneInventory[key]; ok {
				results[key] = simInventoryLookupInfo{
					Operator: simoperator.Label(operator),
					Found:    true,
					FoundSet: true,
					Active:   formatVodafoneBlockingActive(item.BlockingStatus),
					Status:   strings.TrimSpace(item.BlockingStatus),
					Name:     strings.TrimSpace(item.SubscriberName),
					Comment:  strings.TrimSpace(item.SubscriberComment),
				}
				directResolved++
				return
			}
			if vodafoneInventoryLoaded {
				results[key] = simInventoryLookupInfo{
					Operator: simoperator.Label(operator),
					Found:    false,
					FoundSet: true,
					Active:   "ні",
				}
				directResolved++
				return
			}
		case simoperator.Kyivstar:
			if item, ok := kyivstarInventory[key]; ok {
				results[key] = simInventoryLookupInfo{
					Operator: simoperator.Label(operator),
					Found:    true,
					FoundSet: true,
					Active:   formatKyivstarInventoryActive(item.Status),
					Status:   formatKyivstarInventoryStatus(item.Status, item.IsOnline),
					Name:     strings.TrimSpace(item.DeviceName),
					Comment:  strings.TrimSpace(item.DeviceID),
				}
				directResolved++
				return
			}
			if kyivstarInventoryLoaded {
				results[key] = simInventoryLookupInfo{
					Operator: simoperator.Label(operator),
					Found:    false,
					FoundSet: true,
					Active:   "ні",
				}
				directResolved++
				return
			}
		}
		tasks = append(tasks, lookupTask{
			key:      key,
			number:   number,
			operator: operator,
		})
	}

	for _, row := range rows {
		addTask(row.SIM1)
		addTask(row.SIM2)
	}

	if progress != nil {
		switch {
		case len(tasks) == 0:
			progress(fmt.Sprintf("Етап 4/5: усі %d SIM звірені локально, додаткові запити не потрібні", directResolved))
		default:
			progress(fmt.Sprintf("Етап 4/5: локально звірено %d SIM, точкові запити потрібні для %d", directResolved, len(tasks)))
		}
	}

	if len(tasks) == 0 {
		return results, 0, unknownCount
	}

	var (
		mu           sync.Mutex
		wg           sync.WaitGroup
		semaphore    = make(chan struct{}, simLookupConcurrencyLimit)
		lookupErrors int
		completed    int
	)

	for _, task := range tasks {
		task := task
		wg.Add(1)
		semaphore <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-semaphore }()

			info := loadSIMInventoryLookup(provider, task.number, task.operator)
			mu.Lock()
			results[task.key] = info
			if info.Error != "" {
				lookupErrors++
			}
			completed++
			if progress != nil && (completed == len(tasks) || completed%10 == 0) {
				progress(fmt.Sprintf("Етап 4/5: звіряю SIM-карти з операторами... %d/%d", completed, len(tasks)))
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	return results, lookupErrors, unknownCount
}

func loadSIMInventoryLookup(provider SIMInventoryReportProvider, number string, operator simoperator.Operator) simInventoryLookupInfo {
	switch operator {
	case simoperator.Vodafone:
		status, err := provider.GetVodafoneSIMStatus(number)
		if err != nil {
			return simInventoryLookupInfo{
				Operator: simoperator.Label(operator),
				Error:    strings.TrimSpace(err.Error()),
			}
		}
		return simInventoryLookupInfo{
			Operator: simoperator.Label(operator),
			Found:    status.Available,
			FoundSet: true,
			Active:   formatVodafoneSIMActive(status),
			Status:   formatVodafoneSIMStatus(status),
			Name:     strings.TrimSpace(status.SubscriberName),
			Comment:  strings.TrimSpace(status.SubscriberComment),
		}
	case simoperator.Kyivstar:
		status, err := provider.GetKyivstarSIMStatus(number)
		if err != nil {
			return simInventoryLookupInfo{
				Operator: simoperator.Label(operator),
				Error:    strings.TrimSpace(err.Error()),
			}
		}
		return simInventoryLookupInfo{
			Operator: simoperator.Label(operator),
			Found:    status.Available,
			FoundSet: true,
			Active:   formatKyivstarSIMActive(status),
			Status:   formatKyivstarSIMStatus(status),
			Name:     strings.TrimSpace(status.DeviceName),
			Comment:  strings.TrimSpace(status.DeviceID),
		}
	default:
		return simInventoryLookupInfo{}
	}
}

func applySIMInventoryLookup(
	operator *string,
	found *string,
	active *string,
	status *string,
	name *string,
	comment *string,
	info simInventoryLookupInfo,
) {
	if operator != nil {
		*operator = strings.TrimSpace(info.Operator)
	}
	if found != nil {
		if info.FoundSet {
			*found = yesNo(info.Found)
		} else {
			*found = ""
		}
	}
	if active != nil {
		*active = strings.TrimSpace(info.Active)
	}
	if status != nil {
		value := strings.TrimSpace(info.Status)
		if value == "" && strings.TrimSpace(info.Error) != "" {
			value = "помилка: " + strings.TrimSpace(info.Error)
		}
		*status = value
	}
	if name != nil {
		*name = strings.TrimSpace(info.Name)
	}
	if comment != nil {
		*comment = strings.TrimSpace(info.Comment)
	}
}

func simInventoryHeader() []string {
	return []string{
		"Джерело",
		"№ об'єкта",
		"Назва об'єкта",
		"SIM 1",
		"Оператор SIM 1",
		"Є в базі SIM 1",
		"Активна SIM 1",
		"Статус SIM 1",
		"Назва / пристрій SIM 1",
		"Коментар / ID пристрою SIM 1",
		"SIM 2",
		"Оператор SIM 2",
		"Є в базі SIM 2",
		"Активна SIM 2",
		"Статус SIM 2",
		"Назва / пристрій SIM 2",
		"Коментар / ID пристрою SIM 2",
	}
}

func simInventoryRowValues(row SIMInventoryReportRow) []string {
	return []string{
		row.Source,
		row.ObjectNumber,
		row.ObjectName,
		row.SIM1,
		row.SIM1Operator,
		row.SIM1Found,
		row.SIM1Active,
		row.SIM1Status,
		row.SIM1Name,
		row.SIM1Comment,
		row.SIM2,
		row.SIM2Operator,
		row.SIM2Found,
		row.SIM2Active,
		row.SIM2Status,
		row.SIM2Name,
		row.SIM2Comment,
	}
}

func formatSIMInventoryOperatorCount(loaded bool, count int) string {
	if !loaded {
		return "помилка"
	}
	return strconv.Itoa(count)
}

func simInventorySourceForObjectID(id int) string {
	switch {
	case ids.IsPhoenixObjectID(id):
		return SIMInventorySourcePhoenix
	case ids.IsCASLObjectID(id):
		return SIMInventorySourceCASL
	default:
		return SIMInventorySourceBridge
	}
}

func compareSIMInventorySource(left string, right string) int {
	leftRank := simInventorySourceRank(left)
	rightRank := simInventorySourceRank(right)
	switch {
	case leftRank < rightRank:
		return -1
	case leftRank > rightRank:
		return 1
	default:
		return strings.Compare(left, right)
	}
}

func simInventorySourceRank(value string) int {
	switch strings.TrimSpace(value) {
	case SIMInventorySourceBridge:
		return 0
	case SIMInventorySourcePhoenix:
		return 1
	case SIMInventorySourceCASL:
		return 2
	default:
		return 99
	}
}

func compareSIMInventoryNumbers(left string, right string) int {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	leftNum, leftErr := strconv.ParseInt(left, 10, 64)
	rightNum, rightErr := strconv.ParseInt(right, 10, 64)
	switch {
	case leftErr == nil && rightErr == nil:
		switch {
		case leftNum < rightNum:
			return -1
		case leftNum > rightNum:
			return 1
		default:
			return 0
		}
	default:
		return strings.Compare(left, right)
	}
}

func formatVodafoneSIMActive(status contracts.VodafoneSIMStatus) string {
	if !status.Available {
		return "ні"
	}
	if value := strings.TrimSpace(status.Blocking.Status); value != "" {
		return formatVodafoneBlockingActive(value)
	}
	switch strings.ToLower(strings.TrimSpace(status.Connectivity.SIMStatus)) {
	case "active":
		return "так"
	case "":
		return ""
	default:
		return "ні"
	}
}

func formatVodafoneSIMStatus(status contracts.VodafoneSIMStatus) string {
	if value := strings.TrimSpace(status.Blocking.Status); value != "" {
		return value
	}
	return strings.TrimSpace(status.Connectivity.SIMStatus)
}

func formatVodafoneBlockingActive(status string) string {
	switch strings.TrimSpace(status) {
	case "":
		return ""
	case "NotBlocked":
		return "так"
	default:
		return "ні"
	}
}

func formatKyivstarSIMActive(status contracts.KyivstarSIMStatus) string {
	if !status.Available {
		return "ні"
	}
	switch strings.ToUpper(strings.TrimSpace(status.NumberStatus)) {
	case "ACTIVE":
		return "так"
	case "":
		return ""
	default:
		return "ні"
	}
}

func formatKyivstarSIMStatus(status contracts.KyivstarSIMStatus) string {
	parts := make([]string, 0, 2)
	if value := strings.TrimSpace(status.NumberStatus); value != "" {
		parts = append(parts, value)
	}
	if status.IsOnline {
		parts = append(parts, "online")
	} else if status.Available {
		parts = append(parts, "offline")
	}
	return strings.Join(parts, ", ")
}

func formatKyivstarInventoryActive(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "":
		return ""
	case "ACTIVE":
		return "так"
	default:
		return "ні"
	}
}

func formatKyivstarInventoryStatus(status string, isOnline bool) string {
	parts := make([]string, 0, 2)
	if value := strings.TrimSpace(status); value != "" {
		parts = append(parts, value)
	}
	if isOnline {
		parts = append(parts, "online")
	} else if strings.TrimSpace(status) != "" {
		parts = append(parts, "offline")
	}
	return strings.Join(parts, ", ")
}

func collectSIMInventoryNumbers(rows []simInventoryBaseRow, operator simoperator.Operator) []string {
	result := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows)*2)
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || simoperator.Detect(value) != operator {
			return
		}
		key := NormalizeSIMLookupKey(value)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	for _, row := range rows {
		add(row.SIM1)
		add(row.SIM2)
	}
	return result
}

func yesNo(value bool) string {
	if value {
		return "так"
	}
	return "ні"
}
