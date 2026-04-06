//go:build windows

package dialogs

import (
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	ofnFileMustExist = 0x00001000
	ofnPathMustExist = 0x00000800
	ofnExplorer      = 0x00080000
	ofnNoChangeDir   = 0x00000008
)

type openFileName struct {
	LStructSize       uint32
	HwndOwner         windows.Handle
	HInstance         windows.Handle
	LpstrFilter       *uint16
	LpstrCustomFilter *uint16
	NMaxCustFilter    uint32
	NFilterIndex      uint32
	LpstrFile         *uint16
	NMaxFile          uint32
	LpstrFileTitle    *uint16
	NMaxFileTitle     uint32
	LpstrInitialDir   *uint16
	LpstrTitle        *uint16
	Flags             uint32
	NFileOffset       uint16
	NFileExtension    uint16
	LpstrDefExt       *uint16
	LCustData         uintptr
	LpfnHook          uintptr
	LpTemplateName    *uint16
	PvReserved        unsafe.Pointer
	DwReserved        uint32
	FlagsEx           uint32
}

var (
	comdlg32DLL         = windows.NewLazySystemDLL("comdlg32.dll")
	getOpenFileNameProc = comdlg32DLL.NewProc("GetOpenFileNameW")
)

func pickCASLImageFilePath(title string) (string, error) {
	titlePtr, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return "", err
	}
	filter := utf16.Encode([]rune("Image Files\x00*.jpg;*.jpeg;*.png;*.webp;*.bmp;*.gif\x00All Files\x00*.*\x00\x00"))

	buffer := make([]uint16, 4096)
	dialog := openFileName{
		LStructSize: uint32(unsafe.Sizeof(openFileName{})),
		LpstrFilter: &filter[0],
		LpstrFile:   &buffer[0],
		NMaxFile:    uint32(len(buffer)),
		LpstrTitle:  titlePtr,
		Flags:       ofnExplorer | ofnPathMustExist | ofnFileMustExist | ofnNoChangeDir,
	}

	ret, _, callErr := getOpenFileNameProc.Call(uintptr(unsafe.Pointer(&dialog)))
	if ret == 0 {
		if callErr != windows.ERROR_SUCCESS && callErr != nil {
			return "", callErr
		}
		return "", nil
	}
	return windows.UTF16ToString(buffer), nil
}
