package cmd

import (
	"testing"

	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"truenas/admin-tool/core"

	"github.com/spf13/cobra"
)

const TESTING_DATASET_FILE = "ds_testing.tsv"

var api *core.Session

func setupTest(t *testing.T) {
	_ = os.Remove(TESTING_DATASET_FILE)
	SetMockDatasetFileName(TESTING_DATASET_FILE)
	if api == nil {
		api = &core.MockSession{}
		api.Login()
	}
}

func TestDatasetCreate(t *testing.T) {
	setupTest(t)
}


