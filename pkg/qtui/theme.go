//go:build qt

package qtui

// NativeWindowsStyleSheet contains base Qt styles for the application.
const NativeWindowsStyleSheet = `
	/* === Status indicator cards === */
	QFrame[class="status-card"] {
		border: 1px solid #d8d8d8;
		border-radius: 6px;
		padding: 8px 12px;
		min-width: 140px;
	}

	/* === Object card sections (QGroupBox) === */
	QGroupBox {
		font-weight: 600;
		color: #1a73e8;
		border: 1px solid #e0e0e0;
		border-radius: 6px;
		margin-top: 12px;
		padding-top: 18px;
		background: #fafafa;
	}
	QGroupBox::title {
		subcontrol-origin: margin;
		subcontrol-position: top left;
		padding: 2px 10px;
		color: #1a73e8;
		font-weight: 600;
	}

	/* === Card fields === */
	QLineEdit[readOnly="true"] {
		border: 1px solid #e8e8e8;
		background: #ffffff;
		padding: 3px 6px;
		border-radius: 3px;
	}

	/* === Scroll area for card === */
	QScrollArea {
		border: none;
	}
`
