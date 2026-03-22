//go:build windows

package dialogs

import (
	"fmt"
	"image/color"
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

const (
	ccRGBInit  = 0x00000001
	ccFullOpen = 0x00000002
)

type chooseColor struct {
	lStructSize    uint32
	hwndOwner      uintptr
	hInstance      uintptr
	rgbResult      uint32
	lpCustColors   *uint32
	flags          uint32
	lCustData      uintptr
	lpfnHook       uintptr
	lpTemplateName *uint16
}

var (
	comdlg32                 = syscall.NewLazyDLL("comdlg32.dll")
	procChooseColorW         = comdlg32.NewProc("ChooseColorW")
	procCommDlgExtendedError = comdlg32.NewProc("CommDlgExtendedError")

	// Користувацькі кольори, які Win32-діалог показує в нижній секції.
	win32CustomColors [16]uint32
)

func showEventColorPicker(
	win fyne.Window,
	title string,
	description string,
	current color.NRGBA,
	onPicked func(color.NRGBA),
) {
	if onPicked == nil {
		return
	}

	cc := chooseColor{
		lStructSize:  uint32(unsafe.Sizeof(chooseColor{})),
		hwndOwner:    0,
		rgbResult:    nrgbaToColorRef(current),
		lpCustColors: &win32CustomColors[0],
		flags:        ccRGBInit | ccFullOpen,
	}

	ret, _, callErr := procChooseColorW.Call(uintptr(unsafe.Pointer(&cc)))
	if ret != 0 {
		onPicked(colorRefToNRGBA(cc.rgbResult))
		return
	}

	extErr, _, _ := procCommDlgExtendedError.Call()
	if extErr == 0 {
		// Користувач скасував вибір кольору.
		return
	}

	// Показуємо помилку лише для реального збою виклику Win32 API.
	err := fmt.Errorf("win32 ChooseColorW failed (code=%d, syscall=%v)", extErr, callErr)
	dialog.ShowError(err, win)
}

func nrgbaToColorRef(c color.NRGBA) uint32 {
	return uint32(c.R) | (uint32(c.G) << 8) | (uint32(c.B) << 16)
}

func colorRefToNRGBA(colorRef uint32) color.NRGBA {
	return color.NRGBA{
		R: uint8(colorRef & 0xFF),
		G: uint8((colorRef >> 8) & 0xFF),
		B: uint8((colorRef >> 16) & 0xFF),
		A: 255,
	}
}
