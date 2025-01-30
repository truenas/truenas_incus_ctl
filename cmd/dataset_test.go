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

var api *core.Session
var dsText = MemoryRawa{}

func setupTest(t *testing.T) {
	if api == nil {
		api = &core.MockSession{ Source: &dsText }
		api.Login()
	}
}

func TestDatasetCreate(t *testing.T) {
	setupTest(t)
	dsText.Current = nil
	// createOrUpdateDataset()
	// verify json response
	// verify dsText.Current
}


