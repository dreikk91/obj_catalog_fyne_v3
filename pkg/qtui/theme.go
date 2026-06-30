//go:build qt

package qtui

const (
	qtPrimaryColor    = "#4585BC"
	qtBorderColor     = "#B1BFCD"
	qtMutedTextColor  = "#607D8B"
	qtSurfaceColor    = "#F8F9FA"
	qtAltSurfaceColor = "#EFF3F7"
)

// NativeWindowsStyleSheet contains base Qt styles for the application.
const NativeWindowsStyleSheet = `
	/* === Operations-console shell === */
	QMainWindow {
		background: #E7EDF2;
	}
	QMenuBar {
		background: #172B3A;
		color: #F8FAFC;
		padding: 2px;
	}
	QMenuBar::item {
		padding: 5px 10px;
		background: transparent;
	}
	QMenuBar::item:selected {
		background: #2D4A5E;
	}
	QStatusBar {
		background: #172B3A;
		color: #E2E8F0;
		border-top: 1px solid #0F1F2B;
	}
	QStatusBar QLabel {
		color: #E2E8F0;
		padding: 3px 8px;
	}

	QTabWidget::pane {
		border: 1px solid #AEBECD;
		background: #FFFFFF;
	}
	QTabBar::tab {
		background: #DCE5EC;
		color: #405466;
		border: 1px solid #B8C6D1;
		border-bottom: 0;
		padding: 7px 14px;
		min-width: 72px;
		font-weight: 600;
	}
	QTabBar::tab:selected {
		background: #FFFFFF;
		color: #1E6FA8;
		border-top: 3px solid #1E78B4;
		padding-top: 5px;
	}
	QTabBar::tab:hover:!selected {
		background: #EAF0F4;
	}

	QTableView, QTreeView {
		background: #FFFFFF;
		alternate-background-color: #F3F6F8;
		gridline-color: #D7E0E7;
		selection-background-color: #1E78B4;
		selection-color: #FFFFFF;
		border: 1px solid #B8C6D1;
	}
	QHeaderView::section {
		background: #E5ECF1;
		color: #263F50;
		border: 0;
		border-right: 1px solid #C3CFD8;
		border-bottom: 1px solid #AEBECD;
		padding: 5px 7px;
		font-weight: 700;
	}

	QPushButton {
		min-height: 24px;
		padding: 3px 10px;
	}

	/* === Status indicator cards === */
	QFrame[class="status-card"] {
		border: 1px solid ` + qtBorderColor + `;
		border-radius: 6px;
		padding: 8px 12px;
		min-width: 140px;
	}

	/* === Object card sections (QGroupBox) === */
	QGroupBox {
		font-weight: 600;
		color: #254B62;
		border: 1px solid ` + qtBorderColor + `;
		border-radius: 4px;
		margin-top: 12px;
		padding-top: 18px;
		background: ` + qtSurfaceColor + `;
	}
	QGroupBox::title {
		subcontrol-origin: margin;
		subcontrol-position: top left;
		padding: 2px 10px;
		color: #254B62;
		font-weight: 700;
	}

	/* === Card fields === */
	QLineEdit[readOnly="true"] {
		border: 1px solid ` + qtBorderColor + `;
		background: #ffffff;
		padding: 3px 6px;
		border-radius: 3px;
	}

	/* === Scroll area for card === */
	QScrollArea {
		border: none;
	}
`
