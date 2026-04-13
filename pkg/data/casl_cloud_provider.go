package data

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"obj_catalog_fyne_v3/pkg/models"

	"github.com/rs/zerolog/log"
)

const (
	caslCommandPath    = "/command"
	caslLoginPath      = "/login"
	caslDefaultBaseURL = "http://127.0.0.1:50003"

	caslHTTPTimeout       = 12 * time.Second
	caslObjectsCacheTTL   = 20 * time.Second
	caslUsersCacheTTL     = 5 * time.Minute
	caslObjectEventsTTL   = 10 * time.Second
	caslObjectEventsSpan  = 7 * 24 * time.Hour
	caslJournalEventsSpan = 72 * time.Hour
	caslStatsSpan         = 30 * 24 * time.Hour
	caslObjectsStatTTL    = 20 * time.Second
	caslDictionaryTTL     = 15 * time.Minute
	caslTranslatorTTL     = 15 * time.Minute
	caslProbeEventsSpan   = 2 * time.Minute
	caslRealtimeBackoff   = 10 * time.Second

	caslMaxCachedEvents = 2000
	caslReadLimit       = 100000
	caslDebugBodyLimit  = 8192

	caslObjectStatusText = "НОРМА"
)

// CASLCloudProvider реалізує DataProvider для CASL Cloud API.
// Підтримує:
//   - login + автоматичне оновлення token;
//   - список об'єктів (read_grd_object);
//   - базові деталі об'єкта (кімнати, відповідальні, події, стан обладнання, статистика).
type CASLCloudProvider struct {
	baseURL string
	pultID  int64
	email   string
	pass    string

	httpClient *http.Client

	authMu sync.Mutex
	mu     sync.RWMutex

	token  string
	wsURL  string
	userID string

	cachedObjects      []caslGrdObject
	cachedObjectsAt    time.Time
	objectByInternalID map[int]caslGrdObject
	deviceByDeviceID   map[string]caslDevice
	deviceByObjectID   map[string]caslDevice
	deviceByNumber     map[int64]caslDevice
	cachedDevicesAt    time.Time

	cachedUsers   map[string]caslUser
	cachedUsersAt time.Time

	cachedObjectEvents   map[int][]models.Event
	cachedObjectEventsAt map[int]time.Time

	cachedGroupStats   map[string]map[int]int
	cachedGroupStatsAt time.Time

	cachedEvents    []models.Event
	eventsStartAtMs int64
	eventsCursorMs  int64
	eventsRevision  int64

	cachedDictionary        map[string]any
	cachedDictionaryAt      time.Time
	cachedAlarmEvents       map[string]bool
	cachedAlarmEventsAt     time.Time
	dictionaryLang          string
	cachedTranslators       map[string]map[string]string
	cachedTranslatorAlarms  map[string]map[string]bool
	cachedTransAt           map[string]time.Time
	translatorDisabledUntil time.Time

	realtimeMu         sync.Mutex
	realtimeCancel     context.CancelFunc
	realtimeRunning    bool
	realtimeSubscribed bool

	realtimeAlarmByObjID map[string]models.Alarm
}

func NewCASLCloudProvider(baseURL string, token string, pultID int64, credentials ...string) *CASLCloudProvider {
	nowMS := time.Now().UnixMilli()
	email := ""
	pass := ""
	if len(credentials) > 0 {
		email = strings.TrimSpace(credentials[0])
	}
	if len(credentials) > 1 {
		pass = strings.TrimSpace(credentials[1])
	}

	return &CASLCloudProvider{
		baseURL: normalizeCASLBaseURL(baseURL),
		pultID:  pultID,
		email:   email,
		pass:    pass,
		token:   strings.TrimSpace(token),
		httpClient: &http.Client{
			Timeout: caslHTTPTimeout,
		},
		objectByInternalID:     make(map[int]caslGrdObject),
		deviceByDeviceID:       make(map[string]caslDevice),
		deviceByObjectID:       make(map[string]caslDevice),
		deviceByNumber:         make(map[int64]caslDevice),
		cachedUsers:            make(map[string]caslUser),
		cachedObjectEvents:     make(map[int][]models.Event),
		cachedObjectEventsAt:   make(map[int]time.Time),
		cachedGroupStats:       make(map[string]map[int]int),
		cachedAlarmEvents:      make(map[string]bool),
		dictionaryLang:         "uk",
		cachedTranslators:      make(map[string]map[string]string),
		cachedTranslatorAlarms: make(map[string]map[string]bool),
		cachedTransAt:          make(map[string]time.Time),
		eventsStartAtMs:        nowMS,
		eventsCursorMs:         nowMS,
		realtimeAlarmByObjID:   make(map[string]models.Alarm),
	}
}

func (p *CASLCloudProvider) Shutdown() {
	if p == nil {
		return
	}

	p.realtimeMu.Lock()
	cancel := p.realtimeCancel
	p.realtimeCancel = nil
	p.realtimeRunning = false
	p.realtimeSubscribed = false
	p.realtimeMu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// GetLatestEventID повертає компактний курсор змін для scheduler.

// Оновлюємо всі активні тривоги об'єкта даними про оператора/ГМР.

func (p *CASLCloudProvider) readBasketCount(ctx context.Context) (int, error) {
	if !p.hasAuthData() {
		return 0, nil
	}

	payload := map[string]any{"type": "read_count_in_basket"}

	var resp caslBasketResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return 0, err
	}
	return resp.Count, nil
}

func (p *CASLCloudProvider) readPultsPublic(ctx context.Context) ([]caslPult, error) {
	payload := map[string]any{"type": "read_pult", "skip": 0, "limit": caslReadLimit}

	var resp caslReadPultResponse
	if err := p.postCommand(ctx, payload, &resp, false); err != nil {
		return nil, err
	}
	if err := validateCASLPults(resp.Data); err != nil {
		return nil, err
	}

	return append([]caslPult(nil), resp.Data...), nil
}

func (p *CASLCloudProvider) postCommand(ctx context.Context, payload map[string]any, out any, requireAuth bool) error {
	return p.postCommandWithRetry(ctx, payload, out, requireAuth, true)
}

func (p *CASLCloudProvider) postCommandWithRetry(ctx context.Context, payload map[string]any, out any, requireAuth bool, allowRelogin bool) error {
	requestPayload := copyStringAnyMap(payload)
	if requireAuth {
		token, err := p.ensureToken(ctx)
		if err != nil {
			return err
		}
		if strings.TrimSpace(asString(requestPayload["token"])) == "" {
			requestPayload["token"] = token
		}
	}

	body, status, err := p.doJSONRequest(ctx, caslCommandPath, requestPayload)
	if err != nil {
		return err
	}

	if !statusIsOK(status.Status) {
		if requireAuth && allowRelogin && isCASLAuthError(status.Error) && p.canRelogin() {
			if reloginErr := p.refreshToken(ctx, true); reloginErr != nil {
				return fmt.Errorf("casl relogin failed: %w", reloginErr)
			}
			return p.postCommandWithRetry(ctx, payload, out, requireAuth, false)
		}
		return fmt.Errorf("casl command %q status=%q error=%q", asString(payload["type"]), status.Status, status.Error)
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("casl decode response: %w", err)
	}
	return nil
}

func (p *CASLCloudProvider) doJSONRequest(ctx context.Context, path string, payload any) ([]byte, caslStatusOnlyResponse, error) {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, caslStatusOnlyResponse{}, fmt.Errorf("casl marshal payload: %w", err)
	}
	startedAt := time.Now()

	log.Debug().
		Str("method", http.MethodPost).
		Str("path", path).
		Msg("CASL HTTP request")
	logCASLHTTPBody(http.MethodPost, path, "request", requestBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+path, bytes.NewReader(requestBody))
	if err != nil {
		return nil, caslStatusOnlyResponse{}, fmt.Errorf("casl create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, caslStatusOnlyResponse{}, fmt.Errorf("casl request failed: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, caslStatusOnlyResponse{}, fmt.Errorf("casl read response: %w", readErr)
	}

	log.Debug().
		Str("method", http.MethodPost).
		Str("path", path).
		Int("statusCode", resp.StatusCode).
		Dur("duration", time.Since(startedAt)).
		Msg("CASL HTTP response")
	logCASLHTTPBody(http.MethodPost, path, "response", body)

	var status caslStatusOnlyResponse
	_ = json.Unmarshal(body, &status)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if strings.TrimSpace(status.Error) != "" {
			return nil, status, fmt.Errorf("casl http %d: %s", resp.StatusCode, status.Error)
		}
		return nil, status, fmt.Errorf("casl unexpected http status: %d", resp.StatusCode)
	}

	return body, status, nil
}

func (p *CASLCloudProvider) ensureToken(ctx context.Context) (string, error) {
	p.mu.RLock()
	token := strings.TrimSpace(p.token)
	p.mu.RUnlock()
	if token != "" {
		return token, nil
	}

	if !p.canRelogin() {
		return "", errors.New("casl: token is empty and credentials are not configured")
	}

	if err := p.refreshToken(ctx, false); err != nil {
		return "", err
	}

	p.mu.RLock()
	token = strings.TrimSpace(p.token)
	p.mu.RUnlock()
	if token == "" {
		return "", errors.New("casl: login succeeded without token")
	}
	return token, nil
}

func (p *CASLCloudProvider) refreshToken(ctx context.Context, force bool) error {
	if !p.canRelogin() {
		return errors.New("casl: credentials are not configured")
	}

	p.authMu.Lock()
	defer p.authMu.Unlock()

	if force {
		p.mu.Lock()
		p.token = ""
		p.mu.Unlock()
	} else {
		p.mu.RLock()
		token := strings.TrimSpace(p.token)
		p.mu.RUnlock()
		if token != "" {
			return nil
		}
	}

	loginPultID := p.resolveLoginPultID(ctx)
	payload := map[string]any{"email": p.email, "pwd": p.pass, "pult_id": loginPultID, "captcha": ""}

	body, status, err := p.doJSONRequest(ctx, caslLoginPath, payload)
	if err != nil {
		return err
	}

	var resp caslLoginResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("casl decode login response: %w", err)
	}

	if !statusIsOK(resp.Status) && !statusIsOK(status.Status) {
		authErr := strings.TrimSpace(resp.Error)
		if authErr == "" {
			authErr = strings.TrimSpace(status.Error)
		}
		return fmt.Errorf("casl login status=%q error=%q", resp.Status, authErr)
	}
	if strings.TrimSpace(resp.Token) == "" {
		return errors.New("casl login did not return token")
	}

	p.mu.Lock()
	p.token = strings.TrimSpace(resp.Token)
	p.wsURL = strings.TrimSpace(resp.WSURL)
	p.userID = strings.TrimSpace(resp.UserID)
	if p.pultID <= 0 {
		if parsed := parseCASLID(loginPultID); parsed > 0 {
			p.pultID = int64(parsed)
		}
	}
	p.mu.Unlock()

	p.restartRealtimeStream()

	return nil
}

func (p *CASLCloudProvider) resolveLoginPultID(ctx context.Context) string {
	p.mu.RLock()
	if p.pultID > 0 {
		value := strconv.FormatInt(p.pultID, 10)
		p.mu.RUnlock()
		return value
	}
	p.mu.RUnlock()

	pults, err := p.readPultsPublic(ctx)
	if err == nil {
		for _, item := range pults {
			candidate := strings.TrimSpace(item.PultID)
			if candidate != "" {
				return candidate
			}
		}
	}

	return "1"
}

func (p *CASLCloudProvider) canRelogin() bool {
	return strings.TrimSpace(p.email) != "" && strings.TrimSpace(p.pass) != ""
}

func (p *CASLCloudProvider) hasAuthData() bool {
	p.mu.RLock()
	token := strings.TrimSpace(p.token)
	p.mu.RUnlock()
	return token != "" || p.canRelogin()
}

// Після завантаження словника одразу попередньо завантажуємо транслятори
// для всіх типів пристроїв, що вказані у user_device_types.
// Виконується у окремій горутині, щоб не блокувати поточний запит.

// preloadTranslatorsFromDict читає список user_device_types зі словника
// та послідовно завантажує транслятор (get_msg_translator_by_device_type) для кожного типу.
// Викликається один раз після кешування словника.

// extractCASLUserDeviceTypes витягує всі типи приладів зі словника.
// Читає два поля:
//   - user_device_types: ["AX_PRO","Ajax Pro","MAKS_PRO","SATEL",...]
//   - devices: [{"type":"TYPE_DEVICE_Ajax",...}, {"type":"TYPE_DEVICE_Dunay_4_3",...}, ...]
//
// Результат — об'єднаний дедупльований список.

// Поле 1: user_device_types (рядковий масив)

// Поле 2: devices (масив об'єктів з полем "type")

// На частині інсталяцій CASL endpoint повертає WRONG_FORMAT, якщо передати device_type.
// Пробуємо одноразово загальний виклик без device_type і витягаємо мапу по типу локально.

var caslContactIDFallbackTemplates = map[string]string{
	// Базові Contact-ID коди
	"E100":  "Медична тривога",
	"R100":  "Медична тривога — відновлення",
	"E110":  "Пожежна тривога",
	"R110":  "Відновлення після пожежної тривоги",
	"E111":  "Пожежна тривога (дим)",
	"R111":  "Відновлення після пожежної тривоги",
	"E114":  "Тепловий сенсор — тривога",
	"R114":  "Тепловий сенсор — норма",
	"E115":  "Пожежну тривогу активовано вручну",
	"R115":  "Пожежну тривогу деактивовано вручну",
	"E120":  "Тривожна кнопка",
	"R120":  "Відновлення після тривожної кнопки",
	"E129":  "Верифікація тривоги (паніка)",
	"E130":  "Тривога проникнення",
	"R130":  "Відновлення після тривоги проникнення",
	"E131":  "Тривога: периметральна зона",
	"R131":  "Норма: периметральна зона",
	"E132":  "Тривога: внутрішня зона",
	"R132":  "Норма: внутрішня зона",
	"E133":  "Тривога: 24-годинна зона",
	"R133":  "Норма: 24-годинна зона",
	"E134":  "Тривога IO",
	"R134":  "Норма IO",
	"E137":  "Маскування лінії",
	"R137":  "Норма маскування лінії",
	"E139":  "Верифікація тривоги (проникнення)",
	"E140":  "Загальна тривога",
	"R140":  "Загальна тривога — норма",
	"E141":  "Відключення Ring",
	"R141":  "Підключення Ring",
	"E144":  "Тампер датчика",
	"R144":  "Норма тамперу датчика",
	"E145":  "Тампер Hub",
	"R145":  "Норма тамперу Hub",
	"E150":  "Вібрація",
	"E151":  "Газова тривога",
	"R151":  "Газова тривога — відновлення",
	"E154":  "Протікання води",
	"R154":  "Протікання води — відновлення",
	"E158":  "Висока температура",
	"R158":  "Температура в нормі",
	"E159":  "Низька температура",
	"R159":  "Температура в нормі (після низької)",
	"E162":  "Чадний газ (CO)",
	"R162":  "Чадний газ (CO) — норма",
	"E165":  "Тампер Hub",
	"R165":  "Норма тамперу Hub",
	"E300":  "Низький заряд АКБ",
	"R300":  "АКБ в нормі",
	"E301":  "Втрата живлення 220В",
	"R301":  "Відновлення живлення 220В",
	"E302":  "Низький заряд АКБ",
	"R302":  "Відновлення АКБ",
	"E305":  "Перезавантаження",
	"R305":  "Перезавантаження",
	"E306":  "Повне перезавантаження",
	"R306":  "Повне перезавантаження",
	"E307":  "Несправний детектор",
	"E308":  "Зупинка системи",
	"E309":  "Акумулятор приладу несправний",
	"R309":  "Акумулятор приладу в нормі",
	"E311":  "Низький заряд АКБ пристрою",
	"R311":  "АКБ пристрою в нормі",
	"E314":  "Блок живлення несправний",
	"R314":  "Блок живлення в нормі",
	"E315":  "Замок відчинено",
	"R315":  "Замок зачинено",
	"E330":  "Несправність пристрою в кімнаті",
	"R330":  "Несправність пристрою усунена",
	"E337":  "Відсутнє зовнішнє живлення",
	"R337":  "Зовнішнє живлення відновлено",
	"E344":  "Глушіння радіоканалу",
	"R344":  "Глушіння радіоканалу — норма",
	"E350":  "Немає зв'язку з ППК",
	"R350":  "Зв'язок з ППК відновлено",
	"E351":  "Втрата з'єднання через Ethernet",
	"R351":  "Відновлення з'єднання через Ethernet",
	"E352":  "Втрата з'єднання через GSM",
	"R352":  "Відновлення з'єднання через GSM",
	"E353":  "Пристрій у кімнаті не відповідає",
	"E363":  "Втрата з'єднання через Wi-Fi",
	"R363":  "Відновлення з'єднання через Wi-Fi",
	"R373":  "Пожежний шлейф — норма",
	"E374":  "Пристрій не закрито під час спроби постановки",
	"E377":  "Несправність акселерометра",
	"R377":  "Акселерометр — норма",
	"E378":  "Несправність зони",
	"R378":  "Зона — норма",
	"E380":  "Несправність під час перевірки цілісності системи",
	"E381":  "Немає зв'язку з датчиком",
	"R381":  "Зв'язок з датчиком відновлено",
	"E383":  "Акселерометр — тривога",
	"R383":  "Тампер ON (новий)",
	"E384":  "Несправність батареї датчика",
	"R384":  "Батарея датчика — норма",
	"E389":  "Апаратна несправність",
	"R389":  "Глибоке перезавантаження",
	"E391":  "Втрата з'єднання з фото",
	"R391":  "З'єднання з фото відновлено",
	"E393":  "Запорошення датчика",
	"R393":  "Датчик очищено",
	"E390":  "Не прийшло опитування за вказаний час",
	"R390":  "Відновлення опитування",
	"E400":  "Зняття групи (з Дуная)",
	"R400":  "Постановка групи (з Дуная)",
	"E401":  "Зняття групи № {number}",
	"R401":  "Постановка групи № {number}",
	"E402":  "Зняття секції",
	"R402":  "Взяття групи № {number}",
	"E403":  "Сценарій вимкнено",
	"R403":  "Сценарій увімкнено",
	"E406":  "Система відновлена після тривоги користувачем",
	"E409":  "Зняття групи (брелок/клавіатура)",
	"R409":  "Постановка групи (брелок/клавіатура)",
	"E423":  "Ідентифікація користувача {number}",
	"E441":  "Зняття під тиском",
	"R441":  "Постановка під тиском",
	"E442":  "Нічний режим — вимкнено",
	"R442":  "Нічний режим — увімкнено",
	"E451":  "Зняття до часу",
	"R451":  "Постановка до часу",
	"E452":  "Зняття після часу",
	"R452":  "Постановка після часу",
	"E453":  "Невчасне зняття з охорони",
	"E454":  "Невчасна постановка під охорону",
	"E455":  "Не вдалося автопостановку",
	"E456":  "Часткове зняття",
	"R456":  "Залишаємось вдома (STAYIN_HOME)",
	"E459":  "Тривога через 2 хв після постановки",
	"E461":  "Брутфорс",
	"R461":  "Брутфорс скасовано",
	"E531":  "Тампер розпайки підключено",
	"E532":  "Тампер розпайки відключено",
	"E550":  "Фото по запиту — увімкнено",
	"R550":  "Фото по запиту — вимкнено",
	"E570":  "Пристрій тимчасово деактивовано",
	"R570":  "Пристрій активовано знову",
	"E572":  "Сповіщення кришки — вимкнено",
	"R572":  "Сповіщення кришки — увімкнено",
	"E573":  "Автодеактивація тривог або закінчення строку",
	"R573":  "Автодеактивація тривог — відновлено",
	"E577":  "Клавіатура заблокована",
	"R577":  "Клавіатура розблокована",
	"E601":  "Камера задимлення — норма",
	"E602":  "Опитування (PING)",
	"E627":  "Старт процесу оновлення чи застосування нових налаштувань",
	"R627":  "Старт процесу оновлення чи застосування нових налаштувань",
	"E628":  "Завершення процесу оновлення чи застосування нових налаштувань",
	"R628":  "Завершення процесу оновлення чи застосування нових налаштувань",
	"E730":  "Фотоверифікація",
	"E731":  "Отримано фото за розкладом",
	"R731":  "Завершено отримання фото за розкладом",
	"E750":  "Фото по сценарію тривоги — увімкнено",
	"R750":  "Фото по сценарію тривоги — вимкнено",
	"E835":  "Hub у режимі збереження батареї",
	"R835":  "Hub вийшов з режиму збереження батареї",
	"61184": "Відповідь на опитування - норма шлейфа № {number}",
}

var caslMessageKeyFallbackTemplates = map[string]string{
	"GROUP_ON":        "Постановка групи {number}",
	"OO_GROUP_ON":     "Постановка групи {number}",
	"GROUP_OFF":       "Зняття групи № {number}",
	"OO_GROUP_OFF":    "Зняття групи № {number}",
	"LINE_BRK":        "Обрив шлейфа № {number}",
	"OO_LINE_BRK":     "Обрив шлейфа № {number}",
	"LINE_NORM":       "Норма шлейфа № {number}",
	"OO_LINE_NORM":    "Норма шлейфа № {number}",
	"LINE_KZ":         "Коротке замикання шлейфа № {number}",
	"OO_LINE_KZ":      "Коротке замикання шлейфа № {number}",
	"LINE_BAD":        "Несправність шлейфа № {number}",
	"OO_LINE_BAD":     "Несправність шлейфа № {number}",
	"ATTACK":          "Напад № {number}",
	"OO_ATTACK":       "Прихований напад № {number}",
	"ZONE_ALM":        "Тривога в зоні № {number}",
	"ZONE_NORM":       "Норма в зоні № {number}",
	"ALM_INNER_ZONE":  "Тривога внутрішньої зони № {number}",
	"NORM_INNER_ZONE": "Норма внутрішньої зони № {number}",
	"NORM_IO":         "Норма IO № {number}",
	"NO_220":          "Втрата живлення 220В",
	"OK_220":          "Відновлення живлення 220В",
	"PPK_NO_CONN":     "Немає зв'язку з ППК",
	"PPK_CONN_OK":     "Зв'язок з ППК відновлено",
	"ACC_BAD":         "Низький заряд АКБ",
	"ACC_OK":          "АКБ в нормі",
	"DOOR_OP":         "Відкриття корпусу/дверей",
	"DOOR_CL":         "Закриття корпусу/дверей",
	"CHECK_CONN":      "Перевірка зв'язку",
	"ENABLED":         "Прилад увімкнено",
	"DISABLED":        "Прилад вимкнено",
	"FULL_REBOOT":     "Повне перезавантаження ППК",
	"ID_HOZ":          "Ідентифікація користувача {number}",
	"PRIMUS":          "Ідентифікація користувача {number}",
	"UPD_START":       "Старт процесу оновлення чи застосування нових налаштувань",
	"UPD_END":         "Завершення процесу оновлення чи застосування нових налаштувань",
}

const (
	caslProtocolModelOther caslProtocolModel = iota
	caslProtocolModelRcom
	caslProtocolModelSIA
	caslProtocolModelVBD4
	caslProtocolModelDozor
	caslProtocolModelD128
)

// Основні кандидати: точний ключ + upper + lower.

// Для user_device_types приладів коди подій приходять у форматі "E627"/"R627".
// Транслятор будується з {"code":627,"typeEvent":"E"} → ключі "E627" і "627".
// Тому додаємо cross-варіанти:
//   "E627" → також пробуємо "R627" і "627" (числовий)
//   "R627" → також пробуємо "E627" і "627"
//   "627"  → також пробуємо "E627" і "R627"

// Strip prefix → numeric fallback

// Alternate prefix

// Чисто числовий ключ → пробуємо E/R варіанти

// Наразі використовуємо rcom/surgard як базовий декодер для всіх моделей.
// Для SIA/VBD/Dozor/інших моделей він теж дає коректні результати для більшості
// подій у CASL Cloud, а специфічні моделі можна додати окремими декодерами.

// Пріоритет 1: явні мапи по code.
// Транслятор (get_msg_translator_by_device_type) повертає message-key (наприклад,
// "PROG_MODE_ENTER"), а не кириличний текст. Тому після отримання ключа зі транслятора
// необхідно ще раз заглянути у словник (read_dictionary) щоб отримати локалізований опис.

// translator повернув message-key — шукаємо людський текст у словнику.

// Для ключових message key використовуємо канонічні українські тексти.

// Пріоритет 2: байтовий декодер протоколу (code -> msg key -> template).

// Аналогічно: якщо транслятор повернув message-key, дворазовий lookup через словник.

// Пріоритет 3: fallback по contact_id.

var caslDeviceTypeDisplayNames = map[string]string{
	"TYPE_DEVICE_CASL":                    "CASL",
	"TYPE_DEVICE_Dunay_8L":                "Дунай-8L",
	"TYPE_DEVICE_Dunay_16L":               "Дунай-16L",
	"TYPE_DEVICE_Dunay_4L":                "Дунай-4L",
	"TYPE_DEVICE_Lun":                     "Лунь",
	"TYPE_DEVICE_Ajax":                    "Ajax",
	"TYPE_DEVICE_Ajax_SIA":                "Ajax(SIA)",
	"TYPE_DEVICE_Bron_SIA":                "Bron(SIA)",
	"TYPE_DEVICE_CASL_PLUS":               "CASL+",
	"TYPE_DEVICE_Dozor_4":                 "Дозор-4",
	"TYPE_DEVICE_Dozor_8":                 "Дозор-8",
	"TYPE_DEVICE_Dozor_8MG":               "Дозор-8MG",
	"TYPE_DEVICE_Dunay_8_32":              "Дунай-8/32",
	"TYPE_DEVICE_Dunay_16_32":             "Дунай-16/32",
	"TYPE_DEVICE_Dunay_4_3":               "Дунай-4.3",
	"TYPE_DEVICE_Dunay_4_3S":              "Дунай-4.3.1S",
	"TYPE_DEVICE_Dunay_8(16)32_Dunay_G1R": "128 + G1R",
	"TYPE_DEVICE_Dunay_STK":               "Дунай-СТК",
	"TYPE_DEVICE_Dunay_4.2":               "4.2 + G1R",
	"TYPE_DEVICE_VBDb_2":                  "ВБД6-2 + G1R",
	"TYPE_DEVICE_VBD4":                    "ВБД4 + G1R",
	"TYPE_DEVICE_Dunay_PSPN":              "ПСПН (R.COM)",
	"TYPE_DEVICE_Dunay_PSPN_ECOM":         "ПСПН (ECOM)",
	"TYPE_DEVICE_VBD4_ECOM":               "ВБД4",
	"TYPE_DEVICE_VBD_16":                  "ВБД6-16",
}

func normalizeCASLBaseURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = caslDefaultBaseURL
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	return strings.TrimRight(value, "/")
}

type caslPhoneNumber struct {
	Active bool   `json:"active"`
	Number string `json:"number"`
}

type caslObjectEvent struct {
	PPKNum     caslInt64 `json:"ppk_num"`
	DeviceID   caslText  `json:"device_id"`
	ObjID      caslText  `json:"obj_id"`
	ObjName    caslText  `json:"obj_name"`
	ObjAddr    caslText  `json:"obj_address"`
	Action     caslText  `json:"action"`
	DictName   caslText  `json:"dict_name"`
	AlarmType  caslText  `json:"alarm_type"`
	MgrID      caslText  `json:"mgr_id"`
	UserID     caslText  `json:"user_id"`
	UserFIO    caslText  `json:"user_fio"`
	Time       caslInt64 `json:"time"`
	Code       caslText  `json:"code"`
	EventCode  caslText  `json:"event_code"`
	Type       string    `json:"type"`
	TypeEvent  caslText  `json:"type_event"`
	Module     caslText  `json:"module"`
	Number     caslInt64 `json:"number"`
	ContactID  caslText  `json:"contact_id"`
	HozUserID  caslText  `json:"hoz_user_id"`
	Cause      caslText  `json:"cause"`
	Note       caslText  `json:"note"`
	BlockMsg   caslText  `json:"block_message"`
	TimeUnlock caslInt64 `json:"time_unblock"`
	PPKAction  caslText  `json:"ppk_action_type"`
	UserAction caslText  `json:"user_action_type"`
	MgrAction  caslText  `json:"mgr_action_type"`
}

type caslLoginResponse struct {
	Status string `json:"status"`
	UserID string `json:"user_id"`
	FIO    string `json:"fio"`
	Token  string `json:"token"`
	WSURL  string `json:"ws_url"`
	Error  string `json:"error"`
}

type caslReadPultResponse struct {
	Status string     `json:"status"`
	Data   []caslPult `json:"data"`
	Error  string     `json:"error"`
}

type caslReadGrdObjectResponse struct {
	Status string          `json:"status"`
	Data   []caslGrdObject `json:"data"`
	Error  string          `json:"error"`
}

type caslReadUserResponse struct {
	Status string     `json:"status"`
	Data   []caslUser `json:"data"`
	Error  string     `json:"error"`
}

type caslReadDeviceResponse struct {
	Status string       `json:"status"`
	Data   []caslDevice `json:"data"`
	Error  string       `json:"error"`
}

type caslReadEventsByIDResponse struct {
	Status string            `json:"status"`
	Data   []caslObjectEvent `json:"data"`
	Events []caslObjectEvent `json:"events"`
	Error  string            `json:"error"`
}

type caslReadDeviceStateResponse struct {
	Status string          `json:"status"`
	State  caslDeviceState `json:"state"`
	Error  string          `json:"error"`
}

type caslGetStatisticResponse struct {
	Status string              `json:"status"`
	Data   caslStatsAlarmsData `json:"data"`
	Error  string              `json:"error"`
}

type caslBasketResponse struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
	Error  string `json:"error"`
}
