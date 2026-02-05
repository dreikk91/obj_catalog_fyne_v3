// pkg/data/db_provider.go
package data

import (
	"context"
	"obj_catalog_fyne_v3/pkg/database"
	"obj_catalog_fyne_v3/pkg/models"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// DBDataProvider реалізує інтерфейс DataProvider для реальної БД
type DBDataProvider struct {
	db *sqlx.DB

	// Поля для інкрементального завантаження подій
	lastEventID  int64
	cachedEvents []models.Event
	eventMutex   sync.RWMutex

	// Базовий DSN для підключення до інших БД на тому ж сервері
	baseDSN string
}

func NewDBDataProvider(db *sqlx.DB, baseDSN string) *DBDataProvider {
	provider := &DBDataProvider{db: db, baseDSN: baseDSN}
	log.Debug().Msg("DBDataProvider ініціалізовано з підключенням до БД")
	return provider
}

// GetObjects отримує список об'єктів з БД (швидкий запит)
func (p *DBDataProvider) GetObjects() []models.Object {
	if p.db == nil {
		log.Warn().Msg("Спроба отримати об'єкти без активного з'єднання БД")
		return nil
	}

	log.Debug().Msg("Завантаження списку об'єктів з БД...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := database.GetObjectsList(ctx, p.db)
	if err != nil {
		log.Error().Err(err).Msg("Помилка завантаження списку об'єктів")
		return nil
	}

	var objects []models.Object
	for _, row := range rows {
		objects = append(objects, mapObjectRowToModel(row))
	}
	log.Debug().Int("objectsCount", len(objects)).Msg("Список об'єктів завантажено")
	return objects
}

// GetObjectByID отримує базову інформацію про об'єкт
func (p *DBDataProvider) GetObjectByID(idStr string) *models.Object {
	if p.db == nil {
		log.Warn().Str("id", idStr).Msg("Спроба отримати об'єкт без активного з'єднання БД")
		return nil
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Warn().Err(err).Str("id", idStr).Msg("Невірний формат ID об'єкта")
		return nil
	}

	log.Debug().Int64("objectID", id).Msg("Завантаження деталей об'єкта...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	row, err := database.GetObjectDetail(ctx, p.db, id)
	if err != nil {
		log.Warn().Err(err).Int64("objectID", id).Msg("Об'єкт не знайдено або помилка запиту")
		return nil
	}

	obj := &models.Object{
		ID:          int(row.Objn),
		Name:        ptrToString(row.ObjFullName1),
		Address:     ptrToString(row.Address1),
		ContractNum: ptrToString(row.Contract1),
		Phone:       ptrToString(row.Phones1),
		DeviceType:  ptrToString(row.ObjType1),
		PanelMark:   ptrToString(row.PanelMark1),
		SIM1:        ptrToString(row.GsmPhone),
		SIM2:        ptrToString(row.GsmPhone2),
		Status:      mapStateToStatus(row.AlarmState1, row.IsConnState1),
		StatusText:  mapStateToStatusText(row.AlarmState1, row.TechAlarmState1, row.IsConnState1),

		ObjChan:   ptrToInt(row.ObjChan),
		Phones1:   ptrToString(row.Phones1),
		Notes1:    ptrToString(row.Notes1),
		Location1: ptrToString(row.Location1),

		AkbState:      ptrToInt64(row.AkbState),
		PowerFault:    ptrToInt64(row.PowerFault),
		TestControl:   ptrToInt64(row.TestControl1),
		TestTime:      ptrToInt64(row.TestTime1),
		AutoTestHours: int(ptrToInt64(row.TestTime1)) / 60,
	}

	// Оновлюємо PowerSource на основі PowerFault
	if obj.PowerFault > 0 {
		obj.PowerSource = models.PowerBattery
	} else {
		obj.PowerSource = models.PowerMains
	}

	log.Debug().Int("objectID", obj.ID).Str("name", obj.Name).Str("status", obj.StatusText).Msg("Деталі об'єкта завантажено")

	return obj
}

// GetZones отримує зони об'єкта
func (p *DBDataProvider) GetZones(idStr string) []models.Zone {
	id, _ := strconv.ParseInt(idStr, 10, 64)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	dbZones, err := database.GetObjectZones(ctx, p.db, id)
	if err != nil {
		return nil
	}

	var zones []models.Zone
	for _, dz := range dbZones {
		zones = append(zones, models.Zone{
			Number:     int(ptrToInt64(dz.Zonen)),
			Name:       ptrToString(dz.ZoneDescr1),
			SensorType: mapZoneType(dz.ZoneType1),
			Status:     mapZoneStatus(dz.AlarmState1),
		})
	}
	return zones
}

// GetEmployees отримує персонал об'єкта
func (p *DBDataProvider) GetEmployees(idStr string) []models.Contact {
	id, _ := strconv.ParseInt(idStr, 10, 64)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Нам потрібен ObjUin для GetObjectEmployees. Отримуємо його через деталі.
	row, err := database.GetObjectDetail(ctx, p.db, id)
	if err != nil {
		return nil
	}

	dbPers, err := database.GetObjectEmployees(ctx, p.db, row.ObjUin)
	if err != nil {
		return nil
	}

	var contacts []models.Contact
	for _, dp := range dbPers {
		contacts = append(contacts, models.Contact{
			Name:     ptrToString(dp.Surname1) + " " + ptrToString(dp.Name1),
			Position: ptrToString(dp.Status1),
			Phone:    ptrToString(dp.Phones1),
			Priority: int(ptrToInt16(dp.Order1)),
		})
	}
	return contacts
}

// GetEvents отримує глобальні події (інкрементально)
func (p *DBDataProvider) GetEvents() []models.Event {
	if p.db == nil {
		log.Warn().Msg("Спроба отримати eventos без активного з'єднання БД")
		return nil
	}

	p.eventMutex.Lock()
	defer p.eventMutex.Unlock()

	log.Debug().Msg("Завантаження глобальних подій...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Якщо це перший запуск, отримуємо останній ID з бази
	if p.lastEventID == 0 {
		log.Debug().Msg("Перший запуск GetEvents - отримання останнього ID...")
		lastID, err := database.GetLastEventID(ctx, p.db)
		if err == nil {
			p.lastEventID = lastID
			log.Debug().Int64("lastEventID", lastID).Msg("Останній ID подій встановлено")
		} else {
			log.Warn().Err(err).Msg("Помилка отримання останнього ID подій")
		}
	}

	// 2. Отримуємо тільки нові події
	rows, err := database.GetGlobalEvents(ctx, p.db, p.lastEventID)
	if err != nil {
		log.Error().Err(err).Msg("Помилка завантаження подій")
		return p.cachedEvents
	}

	// 3. Додаємо нові події в кеш
	if len(rows) > 0 {
		log.Debug().Int("newEventsCount", len(rows)).Msg("Знайдено нові события")
		var newEvents []models.Event
		for _, row := range rows {
			newEvents = append(newEvents, mapEventRowToModel(row, 0))
			if row.ID > p.lastEventID {
				p.lastEventID = row.ID
			}
		}

		// Перевертаємо нові події, щоб остання була першою в списку (традиційний вигляд журналу)
		for i, j := 0, len(newEvents)-1; i < j; i, j = i+1, j-1 {
			newEvents[i], newEvents[j] = newEvents[j], newEvents[i]
		}

		// Об'єднуємо: спочатку нові, потім старі
		p.cachedEvents = append(newEvents, p.cachedEvents...)

		// Обмежуємо розміри журналу
		if len(p.cachedEvents) > 2000 {
			log.Debug().Int("cachedEventsBefore", len(p.cachedEvents)).Msg("Кеш подій перевищує 2000, обрізаємо...")
			p.cachedEvents = p.cachedEvents[:2000]
			log.Debug().Int("cachedEventsAfter", len(p.cachedEvents)).Msg("Кеш обрізаний")
		}
	}

	log.Debug().Int("totalCachedEvents", len(p.cachedEvents)).Msg("Події завантажено")
	return p.cachedEvents
}

// GetObjectEvents отримує події конкретного об'єкта
func (p *DBDataProvider) GetObjectEvents(objectID string) []models.Event {
	if p.db == nil {
		log.Warn().Str("objectID", objectID).Msg("Спроба отримати события об'єкта без активного з'єднання БД")
		return nil
	}
	id, err := strconv.ParseInt(objectID, 10, 64)
	if err != nil {
		log.Warn().Err(err).Str("objectID", objectID).Msg("Невірний формат ID об'єкта при запиті подій")
		return nil
	}

	log.Debug().Int64("objectID", id).Msg("Завантаження подій об'єкта...")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Для GetObjectEvents нам потрібен ObjUin. Отримуємо його через деталі.
	row, err := database.GetObjectDetail(ctx, p.db, id)
	if err != nil {
		log.Error().Err(err).Int64("objn", id).Msg("Помилка отримання деталей об'єкта для подій")
		return nil
	}

	rows, err := database.GetObjectEvents(ctx, p.db, row.ObjUin)
	if err != nil {
		log.Error().Err(err).Int64("objn", id).Int64("objuin", row.ObjUin).Msg("Помилка отримання подій об'єкта")
		return nil
	}

	var events []models.Event
	for _, row := range rows {
		events = append(events, mapEventRowToModel(row, int(id)))
	}
	return events
}

// GetAlarms отримує список активних тривог (оптимізовано)
func (p *DBDataProvider) GetAlarms() []models.Alarm {
	if p.db == nil {
		log.Warn().Msg("Спроба отримати тривоги без активного з'єднання БД")
		return nil
	}

	log.Debug().Msg("Завантаження активних тривог з БД...")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := database.GetAlarmsList(ctx, p.db)
	if err != nil {
		log.Error().Err(err).Msg("Помилка завантаження тривог")
		return nil
	}

	var alarms []models.Alarm
	for _, row := range rows {
		// Визначаємо тип тривоги за SC1
		alarmType := models.AlarmFire
		if row.Sc1 != nil && *row.Sc1 == 1 {
			alarmType = models.AlarmFire
		} else {
			alarmType = models.AlarmFault
		}

		details := ptrToString(row.Ukr1)
		info1 := ptrToString(row.Info1)
		if info1 != "" {
			if details != "" {
				details += " (" + info1 + ")"
			} else {
				details = info1
			}
		}

		alarm := models.Alarm{
			ID:         int(ptrToInt64(row.ObjN)),
			ObjectID:   int(ptrToInt64(row.ObjN)),
			ObjectName: ptrToString(row.ObjShortName1),
			Address:    ptrToString(row.Address1),
			Details:    details,
			Time:       ptrToTime(row.EvTime1),
			Type:       alarmType,
		}
		alarms = append(alarms, alarm)
	}

	log.Debug().Int("alarmsCount", len(alarms)).Msg("Тривоги завантажено")
	if len(alarms) > 0 {
		
		// log.Info().Int("count", len(alarms)).Msg("Активні тривоги знайдено!")
		// for _, a := range alarms {
		// 	log.Warn().Int("id", a.ID).Str("object", a.ObjectName).Str("address", a.Address).Msg("Тривога")
		// }
	}
	return alarms
}

func (p *DBDataProvider) ProcessAlarm(id string, user string, note string) {
	log.Info().Str("alarmID", id).Str("user", user).Str("note", note).Msg("Обробка тривоги")
	// Not implemented
}

func (p *DBDataProvider) GetExternalData(objectID string) (signal string, lastTestMsg string, lastTest time.Time, lastMsg time.Time) {
	messages := p.GetTestMessages(objectID)
	if len(messages) > 0 {
		last := messages[0]
		// Парсимо рівень сигналу з Info
		signal = "—"
		if strings.Contains(last.Info, "dBm") {
			if idx := strings.LastIndex(last.Info, "["); idx != -1 {
				if endIdx := strings.LastIndex(last.Info, "]"); endIdx > idx {
					signal = last.Info[idx : endIdx+1]
				}
			}
		} else if strings.Contains(strings.ToUpper(last.Info), "GPRS") {
			signal = "GPRS"
		} else if last.Info == "" || strings.Contains(strings.ToUpper(last.Info), "AVD") {
			signal = "AVD"
		} else {
			signal = last.Info
		}
		lastTestMsg = last.Details
	} else {
		signal = "—"
		lastTestMsg = "—"
	}

	// Отримуємо часи з TBL_TESTCONTROL (AVD_MAIN або GPRS_TC)
	id, _ := strconv.ParseInt(objectID, 10, 64)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pathInfo, err := database.GetObjectDbPath(ctx, p.db, id)
	if err == nil && pathInfo.Sbpdb != nil {
		fileName := "GPRS_TC.FDB"
		if pathInfo.SbType != nil && *pathInfo.SbType == 4 {
			fileName = "AVD_MAIN.FDB"
		}

		fullPath := strings.ReplaceAll(*pathInfo.Sbpdb+fileName, "\\", "/")
		extDSN := p.buildExtDSN(fullPath)
		if extDSN == "" {
			log.Warn().Int64("objn", id).Str("filename", fileName).Msg("Не вдалося побудувати DSN для TBL_TESTCONTROL")
			return signal, lastTestMsg, lastTest, lastMsg
		}

		tcDB, err := sqlx.Connect("firebirdsql", extDSN)
		if err != nil {
			log.Error().Err(err).Str("dsn", extDSN).Msg("Помилка підключення до бази TBL_TESTCONTROL")
			return signal, lastTestMsg, lastTest, lastMsg
		}
		defer tcDB.Close()

		tcRow, err := database.GetTestControl(ctx, tcDB, id)
		if err != nil {
			log.Error().Err(err).Int64("objn", id).Msg("Помилка отримання даних з TBL_TESTCONTROL")
			return signal, lastTestMsg, lastTest, lastMsg
		}

		lastTest = ptrToTime(tcRow.LastTestTime1)
		lastMsg = ptrToTime(tcRow.LastMessTime1)
	}

	return signal, lastTestMsg, lastTest, lastMsg
}

// Допоміжна функція для побудови DSN
func (p *DBDataProvider) buildExtDSN(fullPath string) string {
	atIdx := strings.Index(p.baseDSN, "@")
	if atIdx == -1 {
		return ""
	}
	slashIdx := strings.Index(p.baseDSN[atIdx:], "/")
	if slashIdx == -1 {
		return ""
	}
	slashIdx += atIdx
	prefix := p.baseDSN[:slashIdx+1]

	params := ""
	if qIdx := strings.Index(p.baseDSN, "?"); qIdx != -1 {
		params = p.baseDSN[qIdx:]
	}

	return prefix + fullPath + params
}

func (p *DBDataProvider) GetTestMessages(objectID string) []models.TestMessage {
	id, _ := strconv.ParseInt(objectID, 10, 64)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 1. Отримуємо шлях до бази
	dbPathInfo, err := database.GetObjectDbPath(ctx, p.db, id)
	if err != nil || dbPathInfo.Sbpdb == nil {
		return nil
	}

	// 2. Будуємо шлях до файлу бази
	fileName := "GPRS_TRP.FDB"
	if dbPathInfo.SbType != nil && *dbPathInfo.SbType == 4 {
		fileName = "AVD_TRP.FDB"
	}

	fullPath := strings.ReplaceAll(*dbPathInfo.Sbpdb+fileName, "\\", "/")

	// 3. Створюємо DSN для зовнішньоі бази (копіюємо параметри з основного)
	extDSN := p.buildExtDSN(fullPath)
	if extDSN == "" {
		log.Warn().Int64("objn", id).Msg("Не вдалося побудувати DSN для зовнішньої бази")
		return nil
	}

	// 4. Підключаємось
	extDB, err := sqlx.Connect("firebirdsql", extDSN)
	if err != nil {
		log.Error().Err(err).Str("dsn", extDSN).Msg("Помилка підключення до зовнішньої бази TRP")
		return nil
	}
	defer extDB.Close()

	// 5. Отримуємо повідомлення
	rows, err := database.GetTestMessages(ctx, extDB, id)
	if err != nil {
		log.Error().Err(err).Int64("objn", id).Msg("Помилка отримання тестових повідомлень")
		return nil
	}

	var results []models.TestMessage
	for _, row := range rows {
		results = append(results, models.TestMessage{
			Time:    ptrToTime(row.MsgDTime1),
			Info:    ptrToString(row.MsgInfo),
			Details: ptrToString(row.MsgText),
		})
	}
	return results
}

// Допоміжні функції для мапінгу
func mapObjectRowToModel(row database.ObjectInfoRow) models.Object {
	return models.Object{
		ID:          int(row.Objn),
		Name:        ptrToString(row.ObjShortName1),
		Address:     ptrToString(row.Address1),
		ContractNum: ptrToString(row.Contract1),
		Phone:       ptrToString(row.GsmPhone),
		Status:      mapStateToStatus(row.AlarmState1, row.IsConnState1),
		StatusText:  mapStateToStatusText(row.AlarmState1, row.TechAlarmState1, row.IsConnState1),
		GSMLevel:    0,

		AlarmState:     ptrToInt64(row.AlarmState1),
		GuardState:     ptrToInt64(row.GuardState1),
		TechAlarmState: ptrToInt64(row.TechAlarmState1),
		IsConnState:    ptrToInt64(row.IsConnState1),
	}
}

func mapEventRowToModel(row database.EventRow, objID int) models.Event {
	id := objID
	if row.ObjN != nil {
		id = int(*row.ObjN)
	}

	e := models.Event{
		ID:         int(row.ID),
		ObjectID:   id,
		ObjectName: ptrToString(row.ObjShortName1),
		Details:    ptrToString(row.Ukr1),
		SC1:        0,
	}

	if row.Sc1 != nil {
		e.SC1 = *row.Sc1
	}

	// Додаємо інформацію з INFO1 до деталей, якщо вона є
	info := ptrToString(row.Info1)
	if info != "" {
		if e.Details != "" {
			e.Details += " (" + info + ")"
		} else {
			e.Details = info
		}
	}

	if row.EvTime1 != nil {
		e.Time = *row.EvTime1
	} else {
		e.Time = time.Now()
	}

	if row.Zonen != nil {
		e.ZoneNumber = int(*row.Zonen)
	}
	if row.Sc1 != nil {
		sc1 := *row.Sc1
		switch sc1 {
		case 1:
			e.Type = models.EventFire
		case 2:
			e.Type = models.EventFault
		case 3:
			e.Type = models.EventFault
		case 5, 9, 13, 17:
			e.Type = models.EventRestore
		case 10:
			e.Type = models.EventArm
		case 11:
			e.Type = models.EventDisarm
		case 12:
			e.Type = models.EventOffline
		case 14, 18:
			e.Type = models.EventDisarm // Часткова постановка/зняття
		default:
			e.Type = models.SystemEvent
		}
	} else {
		e.Type = models.SystemEvent
	}
	return e
}

func ptrToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func ptrToInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func ptrToInt16(p *int16) int16 {
	if p == nil {
		return 0
	}
	return *p
}

func ptrToInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func ptrToTime(p *time.Time) time.Time {
	if p == nil {
		return time.Now()
	}
	return *p
}

func mapStateToStatus(alarm *int64, conn *int64) models.ObjectStatus {
	if conn != nil && *conn == 0 {
		return models.StatusOffline
	}
	if alarm != nil && *alarm > 0 {
		return models.StatusFire
	}
	return models.StatusNormal
}

func mapStateToStatusText(alarm *int64, tech *int64, conn *int64) string {
	if conn != nil && *conn == 0 {
		return "НЕМАЄ ЗВ'ЯЗКУ"
	}
	if alarm != nil && *alarm > 0 {
		return "ПОЖЕЖА"
	}
	if tech != nil && *tech > 0 {
		return "НЕСПРАВНІСТЬ"
	}
	return "НОРМА"
}

func mapZoneType(zt *int64) string {
	if zt == nil {
		return "Невідомо"
	}
	switch *zt {
	case 1:
		return "Димовий"
	case 2:
		return "Тепловий"
	default:
		return "Ручний/Інше"
	}
}

func mapZoneStatus(as *int64) models.ZoneStatus {
	if as == nil || *as == 0 {
		return models.ZoneNormal
	}
	return models.ZoneFire
}
