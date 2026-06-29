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
		color: ` + qtPrimaryColor + `;
		border: 1px solid ` + qtBorderColor + `;
		border-radius: 6px;
		margin-top: 12px;
		padding-top: 18px;
		background: ` + qtSurfaceColor + `;
	}
	QGroupBox::title {
		subcontrol-origin: margin;
		subcontrol-position: top left;
		padding: 2px 10px;
		color: ` + qtPrimaryColor + `;
		font-weight: 600;
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
