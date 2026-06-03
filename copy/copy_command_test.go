package copy

import (
	"strings"
	"testing"

	"github.com/cloudberry-contrib/cbcopy/option"
	"github.com/cloudberry-contrib/cbcopy/utils"
	"github.com/spf13/pflag"
)

// setupCopyCommandFlags registers the flags FormMasterHelperAddress and
// formatCopy{To,From}Command read via utils.MustGetFlagString.
func setupCopyCommandFlags(t *testing.T, srcHost, destHost, dataPortRange string) {
	t.Helper()
	fs := pflag.NewFlagSet("copy-command-test", pflag.ContinueOnError)
	fs.String(option.SOURCE_HOST, "", "")
	fs.String(option.DEST_HOST, "", "")
	fs.String(option.DATA_PORT_RANGE, "", "")
	if err := fs.Parse([]string{
		"--" + option.SOURCE_HOST, srcHost,
		"--" + option.DEST_HOST, destHost,
		"--" + option.DATA_PORT_RANGE, dataPortRange,
	}); err != nil {
		t.Fatalf("flag parse: %v", err)
	}
	utils.SetCmdFlags(fs)
}

func TestFormMasterHelperAddress_PicksHostByConnectionMode(t *testing.T) {
	setupCopyCommandFlags(t, "src.example.com", "dest.example.com", "50000-60000")
	ports := []HelperPortInfo{{Content: -1, Port: 51234}}

	cases := []struct {
		mode   string
		wantIP string
	}{
		{option.ConnectionModePush, "dest.example.com"},
		{option.ConnectionModePull, "src.example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			cb := CopyBase{ConnectionMode: tc.mode}
			gotPort, gotIP := cb.FormMasterHelperAddress(ports)
			if gotIP != tc.wantIP {
				t.Errorf("ip = %q, want %q", gotIP, tc.wantIP)
			}
			if gotPort != "51234" {
				t.Errorf("port = %q, want %q", gotPort, "51234")
			}
		})
	}
}

// TestCopyOnMasterCommandFormat asserts that the SQL emitted by
// CopyOnMaster respects --connection-mode in both directions. Push and
// pull must be mirror images: under pull, src master --listen sends, and
// dest master dials src with --host/--port. The negative checks
// (wantNotContain) guard against accidental "both branches build the same
// string" regressions.
func TestCopyOnMasterCommandFormat(t *testing.T) {
	setupCopyCommandFlags(t, "src.example.com", "dest.example.com", "50000-60000")
	table := option.Table{Schema: "public", Name: "t1"}
	ports := []HelperPortInfo{{Content: -1, Port: 51234}}
	const cmdId = "cmd-abc"

	newCopy := func(mode string) *CopyOnMaster {
		return &CopyOnMaster{CopyBase: CopyBase{
			WorkerId:       0,
			ConnectionMode: mode,
			CompArg:        "--no-compression",
		}}
	}

	cases := []struct {
		name           string
		mode           string
		direction      string // "to" or "from"
		wantContain    []string
		wantNotContain []string
	}{
		{
			name:      "push CopyTo: src master dials dest",
			mode:      option.ConnectionModePush,
			direction: "to",
			wantContain: []string{
				"COPY public.t1 TO PROGRAM",
				"--seg-id -1",
				"--host dest.example.com",
				"--port 51234",
				"--direction send",
				"IGNORE EXTERNAL PARTITIONS",
			},
			wantNotContain: []string{"--listen", "--data-port-range"},
		},
		{
			name:      "push CopyFrom: dest master listens",
			mode:      option.ConnectionModePush,
			direction: "from",
			wantContain: []string{
				"COPY public.t1 FROM PROGRAM",
				"--listen",
				"--seg-id -1",
				"--cmd-id cmd-abc",
				"--data-port-range 50000-60000",
				"--direction receive",
			},
			wantNotContain: []string{"--host", "--port"},
		},
		{
			name:      "pull CopyTo: src master listens",
			mode:      option.ConnectionModePull,
			direction: "to",
			wantContain: []string{
				"COPY public.t1 TO PROGRAM",
				"--listen",
				"--seg-id -1",
				"--cmd-id cmd-abc",
				"--data-port-range 50000-60000",
				"--direction send",
				"IGNORE EXTERNAL PARTITIONS",
			},
			wantNotContain: []string{"--host", "--port"},
		},
		{
			name:      "pull CopyFrom: dest master dials src",
			mode:      option.ConnectionModePull,
			direction: "from",
			wantContain: []string{
				"COPY public.t1 FROM PROGRAM",
				"--seg-id -1",
				"--host src.example.com",
				"--port 51234",
				"--direction receive",
			},
			wantNotContain: []string{"--listen", "--data-port-range"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newCopy(tc.mode)
			var got string
			switch tc.direction {
			case "to":
				got = c.formatCopyToCommand(table, ports, cmdId)
			case "from":
				got = c.formatCopyFromCommand(table, ports, cmdId)
			default:
				t.Fatalf("unknown direction %q", tc.direction)
			}
			for _, want := range tc.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("expected to contain %q, got:\n%s", want, got)
				}
			}
			for _, notWant := range tc.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("expected NOT to contain %q, got:\n%s", notWant, got)
				}
			}
		})
	}
}
