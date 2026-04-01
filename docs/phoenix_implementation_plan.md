# Технічне завдання: Впровадження підтримки БД Phoenix (MSSQL)

## 1. Загальний огляд
Проект використовує архітектуру на основі інтерфейсу `contracts.DataProvider`. Для підтримки БД Phoenix (MSSQL) необхідно реалізувати новий провайдер даних, який буде взаємодіяти з MSSQL сервером замість Firebird.

## 2. Архітектурні зміни

### 2.1. Новий пакет `pkg/database/phoenix`
Створення окремого пакету для SQL-запитів Phoenix (MSSQL):
- `pkg/database/phoenix/queries.go`: SQL запити до таблиць Phoenix.
- `pkg/database/phoenix/models.go`: Структури даних (rows), що повертаються MSSQL.

### 2.2. Провайдер `pkg/data/phoenix_provider.go`
Реалізація структури `PhoenixDataProvider`, яка імплементує інтерфейс `contracts.DataProvider`.
- Повинна підтримувати інкрементальне завантаження подій (аналогічно `DBDataProvider`).
- Повинна обробляти специфічні для Phoenix формати ID об'єктів (String `panel_id` vs Int `ObjectID`).

## 3. Мапінг даних

### 3.1. Об'єкти (`models.Object`)
| Поле моделі | Таблиця/Колонка Phoenix | Коментар |
| :--- | :--- | :--- |
| `ID` | `RealPanel.ObjectID` | Використовувати числовий ID для внутрішньої логіки |
| `DisplayNumber` | `RealPanel.panel_id` | Рядковий номер (напр. "L00028") для відображення |
| `Name` | `Company.CompanyName` | Назва компанії/об'єкта |
| `Address` | `Company.Address` | Адреса об'єкта |
| `Status` | `Groups.isOpen` | 1 - Відкрито, 0 - Закрито (потребує уточнення) |
| `DeviceType` | `MphoneRadioType.Message` | Тип приладу Лунь |
| `SIM1` | `Sim.SimNumber` | Номер першої SIM-карти |

### 3.2. Події (`models.Event`)
| Поле моделі | Таблиця/Колонка Phoenix | Коментар |
| :--- | :--- | :--- |
| `ID` | `vwArchives.Event_id` | Унікальний ID події |
| `Time` | `vwArchives.TimeEvent` | Дата та час події |
| `Type` | `TypeCode.idTCode` | Мапінг за `idTCode` на `models.EventType` |
| `Details` | `vwArchives.CodeMessage` | Формований опис події |

### 3.3. Тривоги (`models.Alarm`)
| Поле моделі | Таблиця/Колонка Phoenix | Коментар |
| :--- | :--- | :--- |
| `ID` | `CurrentAlarms.ID` | ID тривоги |
| `ObjectID` | `CurrentAlarms.Panel_id` | Потрібен мапінг на числовий ID |
| `Type` | `CurrentAlarms.Code` | Визначення типу (Пожежа/Охорона) за кодом |

## 4. Технічні вимоги

### 4.1. Драйвер та підключення
- Додати драйвер: `github.com/microsoft/go-mssqldb`.
- Формат DSN: `sqlserver://user:password@host:port?database=dbname`.

### 4.2. Конфігурація
- Оновити `pkg/config/db_config.go`, щоб дозволити вибір типу драйвера (`firebirdsql` або `sqlserver`).

## 5. Питання, що потребують уточнення (Missing Info)

1. **Логіка обробки тривог:** Які дії (UPDATE/DELETE/SP) потрібно виконати в Phoenix для підтвердження або скидання тривоги?
2. **Повний довідник кодів:** Необхідна таблиця відповідності `idTCode` та `Code` до типів подій `models.EventType`.
3. **Обробка груп:** Кожен об'єкт може мати кілька груп. Чи потрібно їх розділяти в загальному списку?
4. **Статуси:** Уточнення станів `Groups.isOpen` та як вони корелюють з "Пожежа", "Норма", "Технічна несправність".
5. **Real-time:** Чи використовувати polling для `vwArchives` чи існують альтернативні методи (Trigger + Pipe/Notification)?
