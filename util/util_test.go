package util

import (
	"github.com/juho05/log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetSeverity(log.NONE)
	os.Exit(m.Run())
}
