// Package models - структура зони (шлейфу) пожежної сигналізації
package models

// ZoneStatus визначає стан зони
type ZoneStatus string

const (
	ZoneNormal ZoneStatus = "normal" // Норма
	ZoneFire   ZoneStatus = "fire"   // Пожежа
	ZoneBreak  ZoneStatus = "break"  // Обрив
	ZoneShort  ZoneStatus = "short"  // Коротке замикання
)

// Zone представляє зону (шлейф) пожежної сигналізації
type Zone struct {
	Number     int        // Номер зони (1, 2, 3...)
	Name       string     // Назва (напр. "Склад 1 поверх")
	SensorType string     // Тип датчиків (напр. "Димові")
	Status     ZoneStatus // Поточний стан
	IsBypassed bool       // Чи відключена зона
}

// GetStatusDisplay повертає текстовий опис статусу зони
func (z *Zone) GetStatusDisplay() string {
	if z.IsBypassed {
		return "ВІДКЛЮЧЕНО"
	}
	switch z.Status {
	case ZoneNormal:
		return "НОРМА"
	case ZoneFire:
		return "ПОЖЕЖА"
	case ZoneBreak:
		return "ОБРИВ"
	case ZoneShort:
		return "КЗ"
	default:
		return "—"
	}
}
