package copy

import (
	"strings"
	"testing"

	"github.com/cloudberry-contrib/cbcopy/option"
	"github.com/cloudberry-contrib/cbcopy/utils"
	"github.com/spf13/pflag"
)

func setupTableModeFlags(truncate, appendFlag, skipExisting, metadataOnly, globalMetadataOnly bool) *pflag.FlagSet {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Bool(option.TRUNCATE, false, "")
	flags.Bool(option.APPEND, false, "")
	flags.Bool(option.SKIP_EXISTING, false, "")
	flags.Bool(option.METADATA_ONLY, false, "")
	flags.Bool(option.GLOBAL_METADATA_ONLY, false, "")

	boolStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}
	_ = flags.Set(option.TRUNCATE, boolStr(truncate))
	_ = flags.Set(option.APPEND, boolStr(appendFlag))
	_ = flags.Set(option.SKIP_EXISTING, boolStr(skipExisting))
	_ = flags.Set(option.METADATA_ONLY, boolStr(metadataOnly))
	_ = flags.Set(option.GLOBAL_METADATA_ONLY, boolStr(globalMetadataOnly))

	utils.SetCmdFlags(flags)
	return flags
}

func TestValidateTableMode(t *testing.T) {
	const mutexMsg = "One and only one of the following flags must be specified: --truncate, --append, --skip-existing"

	cases := []struct {
		name                                                                        string
		truncate, appendFlag, skipExisting, metadataOnly, globalMetadataOnly        bool
		wantErrSubstr                                                               string
	}{
		{"truncate only", true, false, false, false, false, ""},
		{"append only", false, true, false, false, false, ""},
		{"skip-existing only", false, false, true, false, false, ""},

		{"truncate + append", true, true, false, false, false, mutexMsg},
		{"truncate + skip-existing", true, false, true, false, false, mutexMsg},
		{"append + skip-existing", false, true, true, false, false, mutexMsg},
		{"all three", true, true, true, false, false, mutexMsg},
		{"none", false, false, false, false, false, mutexMsg},

		{"metadata-only bypass even with none", false, false, false, true, false, ""},
		{"global-metadata-only bypass even with none", false, false, false, false, true, ""},
		{"metadata-only bypass with conflicting flags", true, true, true, true, false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flags := setupTableModeFlags(tc.truncate, tc.appendFlag, tc.skipExisting, tc.metadataOnly, tc.globalMetadataOnly)
			v := NewModeValidator(flags)

			err := v.validateTableMode()

			if tc.wantErrSubstr == "" {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErrSubstr)
			}
			if !strings.Contains(err.Error(), tc.wantErrSubstr) {
				t.Fatalf("expected error containing %q, got %q", tc.wantErrSubstr, err.Error())
			}
		})
	}
}
