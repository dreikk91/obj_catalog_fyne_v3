package architecture_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

type listedPackage struct {
	ImportPath string
	Module     *struct {
		Path string
	}
}

func TestQtDoesNotDependOnFyne(t *testing.T) {
	assertModuleAbsent(t, "fyne.io/fyne/v2", "-tags=qt", "./cmd/obj-catalog-qt")
}

func TestFyneDoesNotDependOnQt(t *testing.T) {
	assertModuleAbsent(t, "github.com/mappu/miqt", "./pkg/application", "./pkg/ui")
}

func assertModuleAbsent(t *testing.T, forbiddenModule string, args ...string) {
	t.Helper()
	commandArgs := append([]string{"list", "-deps", "-json"}, args...)
	command := exec.Command("go", commandArgs...)
	command.Dir = repositoryRoot(t)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("go %v: %v", commandArgs, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	var forbiddenImports []string
	for {
		var pkg listedPackage
		if err := decoder.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("decode go list output: %v", err)
		}
		if pkg.Module != nil && pkg.Module.Path == forbiddenModule {
			forbiddenImports = append(forbiddenImports, pkg.ImportPath)
		}
	}
	if len(forbiddenImports) == 0 {
		return
	}
	sort.Strings(forbiddenImports)
	t.Fatalf("forbidden module %s is reachable through: %v", forbiddenModule, forbiddenImports)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate architecture test")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
