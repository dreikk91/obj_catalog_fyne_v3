# Qt Designer files

These `.ui` files are editable Qt Designer prototypes for the MIQT UI.

They are not loaded by the current Go runtime yet. The active UI is still built in Go under `pkg/qtui`.

Files:

- `main_window.ui` - main window shell with splitters, object list, work area, bottom tabs, menu, toolbar and status bar.
- `object_list_panel.ui` - object list panel with search, status/source filters and the object table.
- `alarm_panel.ui` - compact grouped alarm ribbon/table/history panel.
- `event_log_panel.ui` - event log table with period/source/severity/context filters and pause controls.
- `work_area_panel.ui` - object work area tabs: card, zones, contacts, journal and export.
