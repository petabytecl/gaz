package lockblockingio_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/petabytecl/gaz/linters/lockblockingio"
)

func TestLockBlockingIO(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, lockblockingio.Analyzer, "a")
}
