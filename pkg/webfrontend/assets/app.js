const config = window.APP_CONFIG || { apiBasePath: "/api/frontend/v1" };

const state = {
  objects: [],
  filteredObjects: [],
  selectedObjectID: null,
  selectedObject: null,
  events: [],
  alarms: [],
  activeMainTab: "object",
  activeObjectTab: "info",
};

const elements = {
  refreshAllButton: document.getElementById("refreshAllButton"),
  refreshAllButtonSidebar: document.getElementById("refreshAllButtonSidebar"),
  refreshObjectsButton: document.getElementById("refreshObjectsButton"),
  refreshObjectButton: document.getElementById("refreshObjectButton"),
  refreshEventsButton: document.getElementById("refreshEventsButton"),
  refreshAlarmsButton: document.getElementById("refreshAlarmsButton"),
  objectSearchInput: document.getElementById("objectSearchInput"),
  objectListBody: document.getElementById("objectListBody"),
  objectsMeta: document.getElementById("objectsMeta"),
  eventsMeta: document.getElementById("eventsMeta"),
  alarmsMeta: document.getElementById("alarmsMeta"),
  currentObjectNumber: document.getElementById("currentObjectNumber"),
  currentObjectName: document.getElementById("currentObjectName"),
  currentObjectAddress: document.getElementById("currentObjectAddress"),
  currentObjectSource: document.getElementById("currentObjectSource"),
  currentObjectStatusBadge: document.getElementById("currentObjectStatusBadge"),
  currentObjectStatusText: document.getElementById("currentObjectStatusText"),
  objectEmptyState: document.getElementById("objectEmptyState"),
  objectWorkspace: document.getElementById("objectWorkspace"),
  objectInfoGrid: document.getElementById("objectInfoGrid"),
  zonesTableContainer: document.getElementById("zonesTableContainer"),
  contactsTableContainer: document.getElementById("contactsTableContainer"),
  objectEventsTableContainer: document.getElementById("objectEventsTableContainer"),
  eventsTableContainer: document.getElementById("eventsTableContainer"),
  alarmsTableContainer: document.getElementById("alarmsTableContainer"),
  zonesCount: document.getElementById("zonesCount"),
  contactsCount: document.getElementById("contactsCount"),
  objectEventsCount: document.getElementById("objectEventsCount"),
  statusSelectedObject: document.getElementById("statusSelectedObject"),
  statusEventsCount: document.getElementById("statusEventsCount"),
  statusAlarmsCount: document.getElementById("statusAlarmsCount"),
  clock: document.getElementById("clock"),
  mainTabs: Array.from(document.querySelectorAll(".main-tab")),
  mainPanels: {
    object: document.getElementById("main-panel-object"),
    events: document.getElementById("main-panel-events"),
    alarms: document.getElementById("main-panel-alarms"),
  },
  objectTabButtons: Array.from(document.querySelectorAll(".object-tab-btn")),
  objectPanels: {
    info: document.getElementById("object-panel-info"),
    zones: document.getElementById("object-panel-zones"),
    contacts: document.getElementById("object-panel-contacts"),
    "object-events": document.getElementById("object-panel-object-events"),
  },
};

function init() {
  bindEvents();
  updateClock();
  window.setInterval(updateClock, 1000);
  loadInitialData();
  window.setInterval(refreshJournals, 15000);
}

function bindEvents() {
  const refreshAll = () => loadInitialData();
  elements.refreshAllButton.addEventListener("click", refreshAll);
  elements.refreshAllButtonSidebar.addEventListener("click", refreshAll);
  elements.refreshObjectsButton.addEventListener("click", () => loadObjects());
  elements.refreshObjectButton.addEventListener("click", () => {
    if (state.selectedObjectID) {
      loadObjectDetails(state.selectedObjectID);
    }
  });
  elements.refreshEventsButton.addEventListener("click", () => loadGeneralEvents());
  elements.refreshAlarmsButton.addEventListener("click", () => loadAlarms());
  elements.objectSearchInput.addEventListener("input", () => {
    applyObjectFilter(elements.objectSearchInput.value);
  });
  elements.mainTabs.forEach((button) => {
    button.addEventListener("click", () => activateMainTab(button.dataset.mainTab));
  });
  elements.objectTabButtons.forEach((button) => {
    button.addEventListener("click", () => activateObjectTab(button.dataset.objectTab));
  });
}

function updateClock() {
  const now = new Date();
  elements.clock.textContent = new Intl.DateTimeFormat("uk-UA", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(now);
}

async function loadInitialData() {
  await Promise.all([loadObjects(), loadGeneralEvents(), loadAlarms()]);
}

async function refreshJournals() {
  const tasks = [loadGeneralEvents(), loadAlarms()];
  if (state.selectedObjectID) {
    tasks.push(loadObjectDetails(state.selectedObjectID, true));
  }
  await Promise.all(tasks);
}

async function loadObjects() {
  setMeta(elements.objectsMeta, "Завантаження списку об'єктів...");
  try {
    const payload = await fetchJSON(`${config.apiBasePath}/objects`);
    state.objects = Array.isArray(payload.items) ? payload.items : [];
    applyObjectFilter(elements.objectSearchInput.value);
    setMeta(elements.objectsMeta, `Об'єктів: ${state.objects.length}`);
    if (state.selectedObjectID && !state.objects.some((item) => item.ID === state.selectedObjectID)) {
      clearSelectedObject();
    }
  } catch (error) {
    renderObjectListError(error.message);
    setMeta(elements.objectsMeta, error.message);
  }
}

async function loadObjectDetails(objectID, silent = false) {
  if (!objectID) {
    return;
  }
  try {
    const details = await fetchJSON(`${config.apiBasePath}/objects/${objectID}`);
    state.selectedObjectID = objectID;
    state.selectedObject = details;
    renderObjectDetails();
    renderObjectList();
    if (!silent) {
      activateMainTab("object");
    }
  } catch (error) {
    if (!silent) {
      clearSelectedObject();
      elements.currentObjectName.textContent = "Помилка завантаження";
      elements.currentObjectAddress.textContent = error.message;
      elements.currentObjectStatusText.textContent = "Помилка";
    }
  }
}

async function loadGeneralEvents() {
  setMeta(elements.eventsMeta, "Завантаження подій...");
  try {
    const payload = await fetchJSON(`${config.apiBasePath}/events`);
    state.events = Array.isArray(payload.items) ? payload.items : [];
    renderEventsTable(elements.eventsTableContainer, state.events, false);
    setMeta(elements.eventsMeta, `Подій: ${state.events.length}`);
    elements.statusEventsCount.textContent = String(state.events.length);
  } catch (error) {
    renderErrorState(elements.eventsTableContainer, error.message);
    setMeta(elements.eventsMeta, error.message);
  }
}

async function loadAlarms() {
  setMeta(elements.alarmsMeta, "Завантаження тривог...");
  try {
    const payload = await fetchJSON(`${config.apiBasePath}/alarms`);
    state.alarms = Array.isArray(payload.items) ? payload.items : [];
    renderAlarmsTable(elements.alarmsTableContainer, state.alarms);
    setMeta(elements.alarmsMeta, `Тривог: ${state.alarms.length}`);
    elements.statusAlarmsCount.textContent = String(state.alarms.length);
  } catch (error) {
    renderErrorState(elements.alarmsTableContainer, error.message);
    setMeta(elements.alarmsMeta, error.message);
  }
}

function applyObjectFilter(rawQuery) {
  const query = String(rawQuery || "").trim().toLowerCase();
  if (!query) {
    state.filteredObjects = [...state.objects];
  } else {
    state.filteredObjects = state.objects.filter((item) => {
      const haystack = [
        item.DisplayNumber,
        item.Name,
        item.Address,
        item.Phone,
        item.ContractNumber,
        item.DeviceType,
      ].join(" ").toLowerCase();
      return haystack.includes(query);
    });
  }
  renderObjectList();
}

function renderObjectList() {
  const tbody = elements.objectListBody;
  tbody.innerHTML = "";

  if (!state.filteredObjects.length) {
    tbody.appendChild(buildEmptyTableRow(4, "Об'єкти не знайдено"));
    return;
  }

  state.filteredObjects.forEach((item) => {
    const row = document.createElement("tr");
    if (item.ID === state.selectedObjectID) {
      row.classList.add("selected");
    }
    row.innerHTML = `
      <td><span class="status-dot ${dotClassForSummary(item)}"></span></td>
      <td class="object-number-cell bright">${escapeHTML(item.DisplayNumber || String(item.ID))}</td>
      <td>
        <div class="bright">${escapeHTML(item.Name || "Без назви")}</div>
        <div class="dim">${escapeHTML(item.Address || "Адреса не вказана")}</div>
      </td>
      <td><span class="status-pill ${severityClassFromSummary(item)}">${escapeHTML(item.StatusText || "—")}</span></td>
    `;
    row.addEventListener("click", () => loadObjectDetails(item.ID));
    tbody.appendChild(row);
  });
}

function renderObjectListError(message) {
  elements.objectListBody.innerHTML = "";
  elements.objectListBody.appendChild(buildEmptyTableRow(4, message || "Сталася помилка"));
}

function renderObjectDetails() {
  const details = state.selectedObject;
  if (!details || !details.Summary) {
    clearSelectedObject();
    return;
  }

  const summary = details.Summary;
  elements.objectEmptyState.classList.add("hidden");
  elements.objectWorkspace.classList.remove("hidden");
  elements.currentObjectNumber.textContent = stringifyValue(summary.DisplayNumber || summary.ID);
  elements.currentObjectName.textContent = stringifyValue(summary.Name);
  elements.currentObjectAddress.textContent = stringifyValue(summary.Address);
  elements.currentObjectSource.textContent = stringifyValue(summary.Source);
  elements.currentObjectStatusText.textContent = stringifyValue(summary.StatusText || "—");
  elements.currentObjectStatusBadge.className = `obj-status ${severityClassFromSummary(summary)}`;
  elements.statusSelectedObject.textContent = `${stringifyValue(summary.DisplayNumber || summary.ID)} · ${stringifyValue(summary.Name)}`;

  renderObjectInfo(details);
  renderZonesTable(details.Zones || []);
  renderContactsTable(details.Contacts || []);
  renderEventsTable(elements.objectEventsTableContainer, details.Events || [], true);

  elements.zonesCount.textContent = String((details.Zones || []).length);
  elements.contactsCount.textContent = String((details.Contacts || []).length);
  elements.objectEventsCount.textContent = String((details.Events || []).length);
}

function renderObjectInfo(details) {
  const summary = details.Summary || {};
  const guardStatusText = guardStatusCaption(summary.GuardStatus);
  const connectionStatusText = connectionStatusCaption(summary.ConnectionStatus);
  const monitoringStatusText = monitoringStatusCaption(summary.MonitoringStatus);
  const groups = [
    {
      title: "Загальні відомості",
      items: [
        ["Номер об'єкта", summary.DisplayNumber || summary.ID],
        ["Статус", summary.StatusText || "—"],
        ["Охорона", guardStatusText],
        ["Зв'язок", connectionStatusText],
        ["Моніторинг", monitoringStatusText],
        ["Тип пристрою", summary.DeviceType || "—"],
        ["Пульт / ППК", summary.PanelMark || "—"],
        ["Канал", details.ChannelCode || "—"],
        ["SubServer A", details.SubServerA || "—"],
        ["SubServer B", details.SubServerB || "—"],
      ],
    },
    {
      title: "Контактні та технічні дані",
      items: [
        ["Телефон", details.Phones || summary.Phone || "—"],
        ["SIM 1", summary.SIM1 || "—"],
        ["SIM 2", summary.SIM2 || "—"],
        ["Рівень GSM", details.GSMLevel || "—"],
        ["Живлення", details.PowerSource || "—"],
        ["AKB state", details.AKBState || "—"],
        ["Power fault", details.PowerFault || "—"],
        ["Тест-контроль", details.TestControl ? "увімкнено" : "вимкнено"],
        ["Інтервал тесту", details.TestIntervalMin ? `${details.TestIntervalMin} хв` : "—"],
        ["AutoTestHours", details.AutoTestHours || "—"],
      ],
    },
    {
      title: "Документи і місце",
      items: [
        ["Адреса", summary.Address || "—"],
        ["Договір", summary.ContractNumber || "—"],
        ["Опис місця", details.Location || "—"],
        ["Примітки", details.Notes || "—"],
        ["Дата запуску", details.LaunchDate || "—"],
        ["Останній тест", formatDate(summary.LastTestTime)],
        ["Останнє повідомлення", formatDate(summary.LastMessageTime)],
        ["Зовнішній сигнал", details.ExternalSignal || "—"],
        ["Останній зовнішній тест", formatDate(details.ExternalLastTest)],
        ["Останнє зовн. повідомлення", formatDate(details.ExternalLastMessage)],
      ],
    },
    {
      title: "Поточна оперативна довідка",
      items: [
        ["Джерело", summary.Source || "—"],
        ["Native ID", summary.NativeID || "—"],
        ["Status code", summary.StatusCode || "—"],
        ["Signal strength", summary.SignalStrength || "—"],
        ["External test message", details.ExternalTestMessage || "—"],
        ["Призначено групу", summary.HasAssignment ? "так" : "ні"],
      ],
    },
  ];

  const fragment = document.createDocumentFragment();
  groups.forEach((group) => {
    const section = document.createElement("section");
    section.className = "info-section";
    const rows = group.items.map(([label, value]) => {
      return `
        <div class="info-row">
          <label>${escapeHTML(label)}</label>
          <span class="val">${escapeHTML(stringifyValue(value))}</span>
        </div>
      `;
    }).join("");
    section.innerHTML = `
      <div class="section-title">${escapeHTML(group.title)}</div>
      <div class="info-rows">${rows}</div>
    `;
    fragment.appendChild(section);
  });
  elements.objectInfoGrid.replaceChildren(fragment);
}

function renderZonesTable(items) {
  renderDataTable(
    elements.zonesTableContainer,
    items,
    [
      { label: "Зона", render: (item) => `<span class="mono bright">${escapeHTML(stringifyValue(item.Number))}</span>` },
      { label: "Назва", render: (item) => escapeHTML(stringifyValue(item.Name)) },
      { label: "Тип", render: (item) => `<span class="dim">${escapeHTML(stringifyValue(item.SensorType))}</span>` },
      { label: "Стан", render: (item) => renderStatusPill(zoneSeverity(item.Status), item.Status || "—") },
      { label: "Група", render: (item) => escapeHTML(stringifyValue(item.GroupName || item.GroupNumber || "—")) },
    ],
    "Для цього об'єкта зони відсутні",
  );
}

function renderContactsTable(items) {
  renderDataTable(
    elements.contactsTableContainer,
    items,
    [
      { label: "Пріоритет", render: (item) => `<span class="mono bright">${escapeHTML(stringifyValue(item.Priority))}</span>` },
      { label: "ПІБ", render: (item) => escapeHTML(stringifyValue(item.Name)) },
      { label: "Посада", render: (item) => `<span class="dim">${escapeHTML(stringifyValue(item.Position))}</span>` },
      { label: "Телефон", render: (item) => `<span class="mono">${escapeHTML(stringifyValue(item.Phone))}</span>` },
      { label: "Кодове слово", render: (item) => escapeHTML(stringifyValue(item.CodeWord || "—")) },
    ],
    "Для цього об'єкта відповідальні не задані",
  );
}

function renderEventsTable(container, items, objectScoped) {
  renderDataTable(
    container,
    items,
    [
      { label: "Час", render: (item) => `<span class="mono dim">${escapeHTML(formatDate(item.Time))}</span>` },
      { label: "Об'єкт", render: (item) => escapeHTML(stringifyValue(item.ObjectName || item.ObjectNumber || "—")) },
      { label: "Тип події", render: (item) => escapeHTML(stringifyValue(item.TypeText || item.TypeCode || "—")) },
      { label: "Зона", render: (item) => `<span class="mono">${escapeHTML(stringifyValue(item.ZoneNumber || "—"))}</span>` },
      { label: "Опис", render: (item) => `<span class="dim">${escapeHTML(stringifyValue(item.Details || "—"))}</span>` },
      { label: "Рівень", render: (item) => renderStatusPill(item.VisualSeverity, severityLabel(item.VisualSeverity)) },
    ],
    "Події відсутні",
    objectScoped
      ? null
      : (item) => {
          if (item && item.ObjectID) {
            loadObjectDetails(item.ObjectID);
          }
        },
  );
}

function renderAlarmsTable(container, items) {
  renderDataTable(
    container,
    items,
    [
      { label: "Час", render: (item) => `<span class="mono dim">${escapeHTML(formatDate(item.Time))}</span>` },
      { label: "Об'єкт", render: (item) => escapeHTML(stringifyValue(item.ObjectName || item.ObjectNumber || "—")) },
      { label: "Адреса", render: (item) => `<span class="dim">${escapeHTML(stringifyValue(item.Address || "—"))}</span>` },
      { label: "Тип", render: (item) => escapeHTML(stringifyValue(item.TypeText || item.TypeCode || "—")) },
      { label: "Зона", render: (item) => `<span class="mono">${escapeHTML(stringifyValue(item.ZoneName || item.ZoneNumber || "—"))}</span>` },
      { label: "Деталі", render: (item) => `<span class="dim">${escapeHTML(stringifyValue(item.Details || "—"))}</span>` },
      { label: "Рівень", render: (item) => renderStatusPill(item.VisualSeverity, severityLabel(item.VisualSeverity)) },
    ],
    "Активних тривог немає",
    (item) => {
      if (item && item.ObjectID) {
        loadObjectDetails(item.ObjectID);
      }
    },
    (item) => String(item.VisualSeverity || "").toLowerCase() === "critical",
  );
}

function renderDataTable(container, items, columns, emptyText, onRowClick = null, rowAlarmPredicate = null) {
  if (!Array.isArray(items) || items.length === 0) {
    renderErrorState(container, emptyText);
    return;
  }

  const table = document.createElement("table");
  table.className = "vt";
  const thead = document.createElement("thead");
  const headerRow = document.createElement("tr");
  columns.forEach((column) => {
    const th = document.createElement("th");
    th.textContent = column.label;
    headerRow.appendChild(th);
  });
  thead.appendChild(headerRow);

  const tbody = document.createElement("tbody");
  items.forEach((item) => {
    const row = document.createElement("tr");
    if (rowAlarmPredicate && rowAlarmPredicate(item)) {
      row.classList.add("alarm");
    }
    if (onRowClick) {
      row.style.cursor = "pointer";
      row.addEventListener("click", () => onRowClick(item));
    }
    columns.forEach((column) => {
      const td = document.createElement("td");
      td.innerHTML = column.render(item);
      row.appendChild(td);
    });
    tbody.appendChild(row);
  });
  table.appendChild(thead);
  table.appendChild(tbody);
  container.replaceChildren(table);
}

function renderErrorState(container, message) {
  const block = document.createElement("div");
  block.className = "table-empty";
  block.textContent = message || "Сталася помилка";
  container.replaceChildren(block);
}

function buildEmptyTableRow(colspan, message) {
  const row = document.createElement("tr");
  const cell = document.createElement("td");
  cell.colSpan = colspan;
  cell.className = "dim";
  cell.textContent = message;
  row.appendChild(cell);
  return row;
}

function clearSelectedObject() {
  state.selectedObjectID = null;
  state.selectedObject = null;
  elements.currentObjectNumber.textContent = "—";
  elements.currentObjectName.textContent = "Оберіть об'єкт";
  elements.currentObjectAddress.textContent = "—";
  elements.currentObjectSource.textContent = "—";
  elements.currentObjectStatusText.textContent = "Не вибрано";
  elements.currentObjectStatusBadge.className = "obj-status";
  elements.objectEmptyState.classList.remove("hidden");
  elements.objectWorkspace.classList.add("hidden");
  elements.statusSelectedObject.textContent = "—";
  renderObjectList();
}

function activateMainTab(tabName) {
  state.activeMainTab = tabName;
  elements.mainTabs.forEach((button) => {
    button.classList.toggle("active", button.dataset.mainTab === tabName);
  });
  Object.entries(elements.mainPanels).forEach(([name, panel]) => {
    panel.classList.toggle("active", name === tabName);
  });
}

function guardStatusCaption(status) {
  switch (String(status || "").toLowerCase()) {
    case "guarded":
      return "Під охороною";
    case "disarmed":
      return "Без охорони";
    default:
      return "—";
  }
}

function connectionStatusCaption(status) {
  switch (String(status || "").toLowerCase()) {
    case "online":
      return "На зв'язку";
    case "offline":
      return "Немає зв'язку";
    default:
      return "—";
  }
}

function monitoringStatusCaption(status) {
  switch (String(status || "").toLowerCase()) {
    case "active":
      return "Активний";
    case "blocked":
      return "Заблокований";
    case "debug":
      return "Стенди";
    default:
      return "—";
  }
}

function activateObjectTab(tabName) {
  state.activeObjectTab = tabName;
  elements.objectTabButtons.forEach((button) => {
    button.classList.toggle("active", button.dataset.objectTab === tabName);
  });
  Object.entries(elements.objectPanels).forEach(([name, panel]) => {
    panel.classList.toggle("active", name === tabName);
  });
}

async function fetchJSON(url) {
  const response = await fetch(url, {
    headers: {
      Accept: "application/json",
    },
  });
  let payload = null;
  try {
    payload = await response.json();
  } catch {
    payload = null;
  }
  if (!response.ok) {
    const message = payload && payload.error ? payload.error : `HTTP ${response.status}`;
    throw new Error(message);
  }
  return payload;
}

function renderStatusPill(rawSeverity, label) {
  const severity = String(rawSeverity || "unknown").toLowerCase();
  return `<span class="status-pill ${escapeHTML(severity)}">${escapeHTML(stringifyValue(label))}</span>`;
}

function severityLabel(severity) {
  switch (String(severity || "").toLowerCase()) {
    case "normal":
      return "Норма";
    case "info":
      return "Інфо";
    case "warning":
      return "Попередження";
    case "critical":
      return "Критично";
    default:
      return "Невідомо";
  }
}

function severityClassFromSummary(item) {
  const text = String(item.StatusText || "").toLowerCase();
  if (text.includes("пожеж") || text.includes("трив")) {
    return "critical";
  }
  if (text.includes("несправ") || text.includes("offline") || text.includes("зв'яз")) {
    return "warning";
  }
  if (text.includes("норм") || text.includes("охорон")) {
    return "normal";
  }
  return "unknown";
}

function zoneSeverity(statusText) {
  const text = String(statusText || "").toLowerCase();
  if (text.includes("трив") || text.includes("пож")) {
    return "critical";
  }
  if (text.includes("несправ") || text.includes("fault")) {
    return "warning";
  }
  if (text.includes("норм")) {
    return "normal";
  }
  return "unknown";
}

function dotClassForSummary(item) {
  const severity = severityClassFromSummary(item);
  switch (severity) {
    case "critical":
      return "dot-red";
    case "warning":
      return "dot-orange";
    case "normal":
      return "dot-green";
    default:
      return "dot-gray";
  }
}

function setMeta(element, text) {
  element.textContent = text;
}

function stringifyValue(value) {
  if (value === null || value === undefined || value === "") {
    return "—";
  }
  return String(value);
}

function formatDate(raw) {
  if (!raw) {
    return "—";
  }
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) {
    return stringifyValue(raw);
  }
  return new Intl.DateTimeFormat("uk-UA", {
    dateStyle: "short",
    timeStyle: "medium",
  }).format(date);
}

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

init();
