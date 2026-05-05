# Технічне завдання: Впровадження підтримки бази даних Phoenix (MSSQL)

## 1. Загальні положення
Цей документ описує вимоги та кроки для додавання підтримки бази даних Phoenix (на базі MSSQL) до проекту ARM. Phoenix має замінити або працювати паралельно з існуючою системою Firebird, забезпечуючи моніторинг об'єктів, подій та тривог.

## 2. Зміни в архітектурі даних
### 2.1. Перехід на рядкові ідентифікатори об'єктів
У Phoenix основним ідентифікатором об'єкта є `panel_id` (VARCHAR), наприклад `L00028`.
*   **Вимога**: Змінити типи полів `ID` та `ObjectID` у структурах `models.Object`, `models.Event`, `models.Alarm`, `models.Zone` та відповідних інтерфейсах з `int` на `string`.
*   **Обґрунтування**: Рядкові ID дозволяють уникнути колізій між різними системами та відповідають структурі Phoenix.

### 2.2. Новий провайдер даних: `PhoenixDataProvider`
Створити реалізацію інтерфейсу `contracts.DataProvider` для MSSQL.
*   **Розташування**: `pkg/data/phoenix_provider.go`.
*   **Драйвер**: `github.com/microsoft/go-mssqldb`.
*   **Функціональність**:
    *   `GetObjects()`: Отримання списку об'єктів (враховуючи групи як окремі записи).
    *   `GetEvents()`: Інкрементальне завантаження подій через `vwArchives`.
    *   `GetAlarms()`: Отримання активних тривог.
    *   `GetExternalData()`: Отримання стану каналів зв'язку через `vwMPhoneDeviceChannels`.

## 3. Реалізація SQL запитів

### 3.1. Список об'єктів та груп
Запит має об'єднувати `vwRealPanel` та `Groups`. Кожна група відображається як окремий об'єкт.
```sql
SELECT
    RealPanel.panel_id,
    groups.group_,
    groups.message AS group_name,
    Company.CompanyName,
    Company.Address,
    Groups.isOpen AS status,
    Groups.TimeEvent AS status_time
FROM vwRealPanel AS RealPanel
LEFT JOIN Groups ON RealPanel.Realpanel_id = groups.panel_id
LEFT JOIN company ON groups.companyID = company.ID
WHERE COALESCE(Pults.IsOutPult, 0) = 0
```

### 3.2. Журнал подій
Використовувати наданий запит до `vwArchives` з фільтрацією за датою та інкрементальним `Event_id`.
```sql
SELECT TOP 100 * FROM vwArchives
WHERE Event_id > @lastID
ORDER BY Event_id DESC
```

### 3.3. Канали зв'язку (External Data)
Вся інформація доступна в основній базі через `vwMPhoneDeviceChannels`. Більше не потрібно підключатися до зовнішніх `.FDB` файлів.

## 4. Мапінг типів подій (idTCode)
Необхідно впровадити мапінг `idTCode` до `models.EventType`:
*   `14, 129, 131` -> `EventFire`
*   `1, 22, 28, 128` -> `EventBurglary` (або `EventPanic`)
*   `10, 143, 146, 148, 150` -> `EventArm`
*   `11, 144, 145, 147, 149` -> `EventDisarm`
*   `15, 126` -> `EventFault`
*   `6` -> `EventPowerFail`, `7` -> `EventPowerOK`
*   `18` -> `EventOffline`, `19` -> `EventOnline`
*   `5, 92` -> `EventTest`

## 5. Відсутня інформація (Потребує уточнення)
На основі аналізу проекту, для повного впровадження не вистачає наступних даних:

1.  **Активні тривоги (Active Alarms)**: У Phoenix документації згадується `CurrentAlarms`. Потрібно підтвердити, чи це основна таблиця для моніторингу тривог, які ще не були закриті оператором, та яка її структура.
2.  **Обробка тривог (ProcessAlarm)**: Як саме відбувається "квітування" або закриття тривоги в Phoenix? Чи це запис в базу (напр. в `CurrentAlarms` або `ArchiveAlarms`), чи виклик спеціальної збереженої процедури?
3.  **Параметри підключення**: Типові налаштування MSSQL (Instance name, чи використовується Windows Authentication або SQL Login).
4.  **Зони та Відповідальні**: Запит для отримання списку зон та контактних осіб для конкретного `panel_id` та `group_` (у документації є приклади, але потрібно уточнити відповідність полів нашим моделям).

## 6. Етапи впровадження
1.  **Моделі**: Рефакторинг `models` для підтримки `string ID`.
2.  **Драйвер**: Додавання `go-mssqldb` у проект.
3.  **Провайдер**: Створення `PhoenixDataProvider`.
4.  **Інтеграція**: Оновлення `CombinedDataProvider` для роботи з Phoenix як з одним із джерел.
5.  **Тестування**: Верифікація отримання даних на тестовому стенді MSSQL.
