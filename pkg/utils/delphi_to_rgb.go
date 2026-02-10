package utils

import (
	"image/color"
)

// // DelphiToNRGBA конвертує число Delphi (BGR) у color.NRGBA
// func DelphiToNRGBA(delphiColorCode int) color.NRGBA {
// 	return color.NRGBA{
// 		R: uint8(delphiColorCode & 0xFF),
// 		G: uint8((delphiColorCode >> 8) & 0xFF),
// 		B: uint8((delphiColorCode >> 16) & 0xFF),
// 		A: 255, // Повна непрозорість
// 	}
// }

// // SelectColorNRGBA повертає кольори як NRGBA об'єкти
// func SelectColorNRGBA(colorValue int) (text, row color.NRGBA) {
// 	colorMapping := map[int][2]int{
// 		1:  {8454143, 183},      // Alarm | text: rgba(255,255,128,255) row: rgba(183,0,0,255)
// 		2:  {0, 4568826},        // tech alarm | text: rgba(0,0,0,255) row: rgba(250,182,69,255)
// 		5:  {0, 10789024},       // restore | text: rgba(0,0,0,255) row: rgba(160,160,164,255)
// 		6:  {0, 16777215},       // info | text: rgba(0,0,0,255) row: rgba(255,255,255,255)
// 		7:  {3368448, 16777215}, // PartArmedOn | text: rgba(0,102,51,255) row: rgba(255,255,255,255)
// 		8:  {3368448, 16777215}, // PartArmedOn | text: rgba(0,102,51,255) row: rgba(255,255,255,255)
// 		9:  {0, 10789024},       // restore | text: rgba(0,0,0,255) row: rgba(160,160,164,255)
// 		10: {16777215, 7314531}, // ArmedOn | text: rgba(255,255,255,255) row: rgba(99,156,111,255)
// 		12: {0, 8454143},        // ConnFailed | text: rgba(0,0,0,255) row: rgba(255,255,128,255)
// 		13: {0, 10789024},       // restore | text: rgba(0,0,0,255) row: rgba(160,160,164,255)
// 		14: {16777215, 32896},   // PartArmedOff | text: rgba(255,255,255,255) row: rgba(128,128,0,255)
// 		17: {0, 10789024},       // restore | text: rgba(0,0,0,255) row: rgba(160,160,164,255)
// 		18: {16777215, 32896},   // PartArmedOff | text: rgba(255,255,255,255) row: rgba(128,128,0,255)
// 	}

// 	colors, exists := colorMapping[colorValue]
// 	if !exists {
// 		return color.NRGBA{0, 0, 0, 255}, color.NRGBA{0, 0, 0, 255}
// 	}

// 	return DelphiToNRGBA(colors[0]), DelphiToNRGBA(colors[1])
// }

func SelectColorNRGBA(colorValue int) (text, row color.NRGBA) {
	colorMapping := map[int]struct {
		text color.NRGBA
		row  color.NRGBA
	}{
		1: { // Alarm
			text: color.NRGBA{R: 255, G: 255, B: 128, A: 255}, // rgb(255,255,128)
			row:  color.NRGBA{R: 183, G: 0, B: 0, A: 255},    // rgb(183,0,0)
		},
		2: { // Tech alarm
			text: color.NRGBA{R: 0, G: 0, B: 0, A: 255},      // rgb(0,0,0)
			row:  color.NRGBA{R: 250, G: 182, B: 69, A: 255}, // rgb(250,182,69)
		},
		5: { // Restore
			text: color.NRGBA{R: 0, G: 0, B: 0, A: 255},        // rgb(0,0,0)
			row:  color.NRGBA{R: 160, G: 160, B: 164, A: 255}, // rgb(160,160,164)
		},
		6: { // Info
			text: color.NRGBA{R: 0, G: 0, B: 0, A: 255},        // rgb(0,0,0)
			row:  color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // rgb(255,255,255)
		},
		7: { // PartArmedOn
			text: color.NRGBA{R: 0, G: 102, B: 51, A: 255},    // rgb(0,102,51)
			row:  color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // rgb(255,255,255)
		},
		8: { // PartArmedOn
			text: color.NRGBA{R: 0, G: 102, B: 51, A: 255},    // rgb(0,102,51)
			row:  color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // rgb(255,255,255)
		},
		9: { // Restore
			text: color.NRGBA{R: 0, G: 0, B: 0, A: 255},        // rgb(0,0,0)
			row:  color.NRGBA{R: 160, G: 160, B: 164, A: 255}, // rgb(160,160,164)
		},
		10: { // ArmedOn
			text: color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // rgb(255,255,255)
			row:  color.NRGBA{R: 99, G: 156, B: 111, A: 255},  // rgb(99,156,111)
		},
		12: { // ConnFailed
			text: color.NRGBA{R: 0, G: 0, B: 0, A: 255},        // rgb(0,0,0)
			row:  color.NRGBA{R: 255, G: 255, B: 128, A: 255}, // rgb(255,255,128)
		},
		13: { // Restore
			text: color.NRGBA{R: 0, G: 0, B: 0, A: 255},        // rgb(0,0,0)
			row:  color.NRGBA{R: 160, G: 160, B: 164, A: 255}, // rgb(160,160,164)
		},
		14: { // PartArmedOff
			text: color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // rgb(255,255,255)
			row:  color.NRGBA{R: 128, G: 128, B: 0, A: 255},   // rgb(128,128,0)
		},
		17: { // Restore
			text: color.NRGBA{R: 0, G: 0, B: 0, A: 255},        // rgb(0,0,0)
			row:  color.NRGBA{R: 160, G: 160, B: 164, A: 255}, // rgb(160,160,164)
		},
		18: { // PartArmedOff
			text: color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // rgb(255,255,255)
			row:  color.NRGBA{R: 128, G: 128, B: 0, A: 255},   // rgb(128,128,0)
		},
	}

	if c, ok := colorMapping[colorValue]; ok {
		return c.text, c.row
	}

	// default (Delphi 0,0 → чорний текст на білому)
	return color.NRGBA{R: 0, G: 0, B: 0, A: 255},   // rgb(0,0,0)
		color.NRGBA{R: 255, G: 255, B: 255, A: 255} // rgb(255, 255, 255)
}


func SelectColorNRGBADark(colorValue int) (text, row color.NRGBA) {
	colorMapping := map[int]struct {
		text color.NRGBA
		row  color.NRGBA
	}{
		1: { // Alarm
			text: color.NRGBA{R: 255, G: 200, B: 200, A: 255}, // rgb(255,200,200)
			row:  color.NRGBA{R: 90, G: 20, B: 20, A: 255},    // rgb(90,20,20)
		},
		2: { // Tech alarm
			text: color.NRGBA{R: 255, G: 220, B: 170, A: 255}, // rgb(255,220,170)
			row:  color.NRGBA{R: 80, G: 60, B: 30, A: 255},    // rgb(80,60,30)
		},
		5: { // Restore
			text: color.NRGBA{R: 210, G: 210, B: 215, A: 255}, // rgb(210,210,215)
			row:  color.NRGBA{R: 45, G: 45, B: 50, A: 255},    // rgb(45,45,50)
		},
		6: { // Info
			text: color.NRGBA{R: 220, G: 220, B: 220, A: 255}, // rgb(220,220,220)
			row:  color.NRGBA{R: 30, G: 30, B: 30, A: 255},    // rgb(30,30,30)
		},
		7: { // PartArmedOn
			text: color.NRGBA{R: 170, G: 255, B: 210, A: 255}, // rgb(170,255,210)
			row:  color.NRGBA{R: 20, G: 70, B: 45, A: 255},    // rgb(20,70,45)
		},
		8: { // PartArmedOn
			text: color.NRGBA{R: 170, G: 255, B: 210, A: 255}, // rgb(170,255,210)
			row:  color.NRGBA{R: 20, G: 70, B: 45, A: 255},    // rgb(20,70,45)
		},
		9: { // Restore
			text: color.NRGBA{R: 210, G: 210, B: 215, A: 255}, // rgb(210,210,215)
			row:  color.NRGBA{R: 45, G: 45, B: 50, A: 255},    // rgb(45,45,50)
		},
		10: { // ArmedOn
			text: color.NRGBA{R: 200, G: 255, B: 225, A: 255}, // rgb(200,255,225)
			row:  color.NRGBA{R: 25, G: 85, B: 60, A: 255},    // rgb(25,85,60)
		},
		12: { // ConnFailed
			text: color.NRGBA{R: 255, G: 245, B: 180, A: 255}, // rgb(255,245,180)
			row:  color.NRGBA{R: 85, G: 75, B: 30, A: 255},    // rgb(85,75,30)
		},
		13: { // Restore
			text: color.NRGBA{R: 210, G: 210, B: 215, A: 255}, // rgb(210,210,215)
			row:  color.NRGBA{R: 45, G: 45, B: 50, A: 255},    // rgb(45,45,50)
		},
		14: { // PartArmedOff
			text: color.NRGBA{R: 255, G: 240, B: 180, A: 255}, // rgb(255,240,180)
			row:  color.NRGBA{R: 90, G: 85, B: 30, A: 255},    // rgb(90,85,30)
		},
		17: { // Restore
			text: color.NRGBA{R: 210, G: 210, B: 215, A: 255}, // rgb(210,210,215)
			row:  color.NRGBA{R: 45, G: 45, B: 50, A: 255},    // rgb(45,45,50)
		},
		18: { // PartArmedOff
			text: color.NRGBA{R: 255, G: 240, B: 180, A: 255}, // rgb(255,240,180)
			row:  color.NRGBA{R: 90, G: 85, B: 30, A: 255},    // rgb(90,85,30)
		},
	}

	if c, ok := colorMapping[colorValue]; ok {
		return c.text, c.row
	}

	// default (dark)
	return color.NRGBA{R: 220, G: 220, B: 220, A: 255}, // rgb(220,220,220)
		color.NRGBA{R: 30, G: 30, B: 30, A: 255}        // rgb(30,30,30)
}


// func SelectColorNRGBADark(colorValue int) (text, row color.NRGBA) {
// 	colorMapping := map[int]struct {
// 		text color.NRGBA
// 		row  color.NRGBA
// 	}{
// 		1: { // Alarm
// 			text: color.NRGBA{R: 255, G: 220, B: 220, A: 255}, // rgba(255,220,220,255)
// 			row:  color.NRGBA{R: 140, G: 30, B: 30, A: 255},   // rgba(140,30,30,255)
// 		},
// 		2: { // Tech alarm
// 			text: color.NRGBA{R: 255, G: 240, B: 210, A: 255}, // rgba(255,240,210,255)
// 			row:  color.NRGBA{R: 160, G: 110, B: 30, A: 255},  // rgba(160,110,30,255)
// 		},
// 		5: { // Restore
// 			text: color.NRGBA{R: 210, G: 215, B: 220, A: 255}, // rgba(210,215,220,255)
// 			row:  color.NRGBA{R: 75, G: 80, B: 90, A: 255},    // rgba(75,80,90,255)
// 		},
// 		6: { // Info
// 			text: color.NRGBA{R: 220, G: 220, B: 220, A: 255}, // rgba(220,220,220,255)
// 			row:  color.NRGBA{R: 45, G: 45, B: 48, A: 255},    // rgba(45,45,48,255)
// 		},
// 		7: { // PartArmedOn
// 			text: color.NRGBA{R: 180, G: 235, B: 210, A: 255}, // rgba(180,235,210,255)
// 			row:  color.NRGBA{R: 40, G: 70, B: 55, A: 255},    // rgba(40,70,55,255)
// 		},
// 		8: { // PartArmedOn
// 			text: color.NRGBA{R: 180, G: 235, B: 210, A: 255}, // rgba(180,235,210,255)
// 			row:  color.NRGBA{R: 40, G: 70, B: 55, A: 255},    // rgba(40,70,55,255)
// 		},
// 		9: { // Restore
// 			text: color.NRGBA{R: 210, G: 215, B: 220, A: 255}, // rgba(210,215,220,255)
// 			row:  color.NRGBA{R: 75, G: 80, B: 90, A: 255},    // rgba(75,80,90,255)
// 		},
// 		10: { // ArmedOn
// 			text: color.NRGBA{R: 200, G: 245, B: 220, A: 255}, // rgba(200,245,220,255)
// 			row:  color.NRGBA{R: 55, G: 110, B: 85, A: 255},   // rgba(55,110,85,255)
// 		},
// 		12: { // ConnFailed
// 			text: color.NRGBA{R: 255, G: 245, B: 210, A: 255}, // rgba(255,245,210,255)
// 			row:  color.NRGBA{R: 125, G: 105, B: 35, A: 255},  // rgba(125,105,35,255)
// 		},
// 		13: { // Restore
// 			text: color.NRGBA{R: 210, G: 215, B: 220, A: 255}, // rgba(210,215,220,255)
// 			row:  color.NRGBA{R: 75, G: 80, B: 90, A: 255},    // rgba(75,80,90,255)
// 		},
// 		14: { // PartArmedOff
// 			text: color.NRGBA{R: 240, G: 240, B: 200, A: 255}, // rgba(240,240,200,255)
// 			row:  color.NRGBA{R: 100, G: 95, B: 40, A: 255},   // rgba(100,95,40,255)
// 		},
// 		17: { // Restore
// 			text: color.NRGBA{R: 210, G: 215, B: 220, A: 255}, // rgba(210,215,220,255)
// 			row:  color.NRGBA{R: 75, G: 80, B: 90, A: 255},    // rgba(75,80,90,255)
// 		},
// 		18: { // PartArmedOff
// 			text: color.NRGBA{R: 240, G: 240, B: 200, A: 255}, // rgba(240,240,200,255)
// 			row:  color.NRGBA{R: 100, G: 95, B: 40, A: 255},   // rgba(100,95,40,255)
// 		},
// 	}

// 	if c, ok := colorMapping[colorValue]; ok {
// 		return c.text, c.row
// 	}

// 	// default dark
// 	return color.NRGBA{R: 220, G: 220, B: 220, A: 255}, // rgba(220,220,220,255)
// 		color.NRGBA{R: 45, G: 45, B: 48, A: 255} // rgba(45,45,48,255)
// }
