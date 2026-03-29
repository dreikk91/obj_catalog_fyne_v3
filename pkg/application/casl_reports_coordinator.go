package application

import (
	"context"

	"obj_catalog_fyne_v3/pkg/ui/dialogs"
)

type caslReportsProvider interface {
	GetStatisticReport(ctx context.Context, name string, limit int) ([]map[string]any, error)
}

func (a *Application) resolveCASLReportsProvider() (caslReportsProvider, bool) {
	provider := a.getDataProvider()
	if provider == nil {
		return nil, false
	}
	reporter, ok := provider.(caslReportsProvider)
	if !ok {
		return nil, false
	}
	return reporter, true
}

func (a *Application) openCASLReportsDialog() {
	reporter, ok := a.resolveCASLReportsProvider()
	if !ok {
		dialogs.ShowInfoDialog(
			a.mainWindow,
			"Недоступно",
			"CASL-звіти недоступні. Перевірте, що CASL Cloud увімкнений у налаштуваннях.",
		)
		return
	}
	dialogs.ShowCASLReportsDialog(a.mainWindow, reporter)
}
