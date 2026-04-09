package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"

	"obj_catalog_fyne_v3/pkg/models"
	apptheme "obj_catalog_fyne_v3/pkg/theme"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

func TestBuildCaseHistoryEventLine_LightThemePreservesContrastBackground(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	app.Settings().SetTheme(apptheme.NewLightTheme(12))

	line := buildCaseHistoryEventLine(models.Event{
		SC1:     11,
		Type:    models.EventDisarm,
		Details: "Тестова подія",
	})

	outer, ok := line.(*fyne.Container)
	if !ok {
		t.Fatalf("expected outer padded container, got %T", line)
	}
	if len(outer.Objects) != 1 {
		t.Fatalf("expected 1 object in outer container, got %d", len(outer.Objects))
	}

	stack, ok := outer.Objects[0].(*fyne.Container)
	if !ok {
		t.Fatalf("expected stacked container, got %T", outer.Objects[0])
	}
	if len(stack.Objects) != 2 {
		t.Fatalf("expected background and content in stack, got %d objects", len(stack.Objects))
	}

	bg, ok := stack.Objects[0].(*canvas.Rectangle)
	if !ok {
		t.Fatalf("expected rectangle background, got %T", stack.Objects[0])
	}
	if got, want := color.NRGBAModel.Convert(bg.FillColor).(color.NRGBA), (color.NRGBA{R: 70, G: 120, B: 170, A: 255}); got != want {
		t.Fatalf("unexpected background color: got %+v want %+v", got, want)
	}

	content, ok := stack.Objects[1].(*fyne.Container)
	if !ok {
		t.Fatalf("expected padded text container, got %T", stack.Objects[1])
	}
	if len(content.Objects) != 1 {
		t.Fatalf("expected 1 text object, got %d", len(content.Objects))
	}

	txt, ok := content.Objects[0].(*canvas.Text)
	if !ok {
		t.Fatalf("expected canvas text, got %T", content.Objects[0])
	}
	if got, want := color.NRGBAModel.Convert(txt.Color).(color.NRGBA), (color.NRGBA{R: 255, G: 255, B: 255, A: 255}); got != want {
		t.Fatalf("unexpected text color: got %+v want %+v", got, want)
	}
}

func TestBuildCaseHistoryEventList_LimitsHeightAndUsesVerticalScrollForLongHistory(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	app.Settings().SetTheme(apptheme.NewLightTheme(12))

	events := make([]models.Event, 0, 7)
	for i := 0; i < 7; i++ {
		events = append(events, models.Event{
			ID:      i + 1,
			Type:    models.EventOperatorAction,
			Details: "Подія " + string(rune('A'+i)),
		})
	}

	obj := buildCaseHistoryEventList(viewmodels.WorkAreaCaseHistoryGroup{
		Root:   events[0],
		Events: events,
		Title:  "CASL кейс",
	})

	scroll, ok := obj.(*container.Scroll)
	if !ok {
		t.Fatalf("expected vertical scroll for long case history, got %T", obj)
	}
	if scroll.MinSize().Height <= 0 {
		t.Fatalf("expected positive scroll min height, got %f", scroll.MinSize().Height)
	}
}

func TestBuildCaseHistoryEventList_ShortHistoryDoesNotWrapIntoScroll(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	app.Settings().SetTheme(apptheme.NewLightTheme(12))

	events := []models.Event{
		{ID: 1, Type: models.EventBurglary, Details: "Початок"},
		{ID: 2, Type: models.EventOperatorAction, Details: "Оператор"},
		{ID: 3, Type: models.EventRestore, Details: "Відновлення"},
	}

	obj := buildCaseHistoryEventList(viewmodels.WorkAreaCaseHistoryGroup{
		Root:   events[0],
		Events: events,
		Title:  "CASL кейс",
	})

	if _, ok := obj.(*container.Scroll); ok {
		t.Fatalf("did not expect scroll for short case history")
	}
}
