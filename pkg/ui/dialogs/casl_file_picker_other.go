//go:build !windows

package dialogs

func pickCASLImageFilePath(title string) (string, error) {
	return "", nil
}
