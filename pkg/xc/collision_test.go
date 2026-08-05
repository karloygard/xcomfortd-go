package xc

import (
	"os"
	"strings"
	"testing"
)

type noopHandler struct{}

func (noopHandler) StatusValue(*Datapoint, int)                {}
func (noopHandler) StatusBool(*Datapoint, bool)                 {}
func (noopHandler) StatusShutter(*Datapoint, ShutterStatus)     {}
func (noopHandler) Event(*Datapoint, Event)                     {}
func (noopHandler) Wheel(*Datapoint, interface{})               {}
func (noopHandler) Valve(*Datapoint, int)                       {}
func (noopHandler) ValueEvent(*Datapoint, Event, interface{})   {}
func (noopHandler) Value(*Datapoint, interface{})               {}
func (noopHandler) Battery(*Device, int)                        {}
func (noopHandler) Power(*Device, interface{})                  {}
func (noopHandler) InternalTemperature(*Device, int)            {}
func (noopHandler) Rssi(*Device, int)                           {}
func (noopHandler) DPLChanged()                                 {}

func newTestInterface() *Interface {
	i := &Interface{}
	i.Init(noopHandler{}, false)
	return i
}

// TestOldBehaviour_MergedFileCausesCollision demonstrates the ORIGINAL bug:
// one merged multi-floor file has datapoint numbers restarting at 1 per
// floor. Loading that single file (as every interface previously did,
// identically) into one Interface silently loses the first floor's entry
// for any colliding number - it's overwritten during parsing of ONE file,
// long before any per-interface separation could matter.
func TestOldBehaviour_MergedFileCausesCollision(t *testing.T) {
	merged := "tmp_merged.txt"
	content := "1\tFloorA_Light1 \t1000001\t16\t0\t0\t0\t#000#\t\n" +
		"1\tFloorB_Shutter1 \t2000002\t27\t0\t0\t0\t#000#\t\n"
	writeTmp(t, merged, content)

	iface := newTestInterface()
	if err := iface.ReadFile(merged); err != nil {
		t.Fatal(err)
	}

	dp := iface.Datapoint(1)
	if dp == nil {
		t.Fatal("expected a datapoint at number 1")
	}

	// BUG (documented, pre-fix): only the LAST entry in the file survives.
	// FloorA_Light1 (serial 1000001) is silently lost even though it was
	// listed first - this is the exact collision that made an entire
	// floor's devices vanish while later-floor entries kept working.
	if dp.device.serialNumber != 2000002 {
		t.Fatalf("got serial %d, want 2000002 (FloorB_Shutter1 wins, "+
			"FloorA_Light1/1000001 was silently overwritten) - "+
			"this IS the bug we are documenting", dp.device.serialNumber)
	}
	t.Logf("Confirmed pre-fix collision: datapoint 1 resolves to serial %d "+
		"(FloorB_Shutter1), FloorA_Light1 (1000001) is unreachable despite "+
		"being a real, distinct physical device", dp.device.serialNumber)
}

// TestNewBehaviour_PerHostFilesAvoidCollision demonstrates the FIX: giving
// each Interface only its own floor's file (as datapointFileFor now does
// in main.go when multiple --file values are supplied) means datapoint
// number 1 correctly resolves to a DIFFERENT, correct device on each
// interface - no collision, both floors' devices are reachable.
func TestNewBehaviour_PerHostFilesAvoidCollision(t *testing.T) {
	fileA := "tmp_floora.txt"
	fileB := "tmp_floorb.txt"
	writeTmp(t, fileA, "1\tFloorA_Light1 \t1000001\t16\t0\t0\t0\t#000#\t\n")
	writeTmp(t, fileB, "1\tFloorB_Shutter1 \t2000002\t27\t0\t0\t0\t#000#\t\n")

	ifaceA := newTestInterface()
	if err := ifaceA.ReadFile(fileA); err != nil {
		t.Fatal(err)
	}

	ifaceB := newTestInterface()
	if err := ifaceB.ReadFile(fileB); err != nil {
		t.Fatal(err)
	}

	dpA := ifaceA.Datapoint(1)
	dpB := ifaceB.Datapoint(1)

	if dpA == nil || dpA.device.serialNumber != 1000001 {
		t.Fatalf("Floor A interface: datapoint 1 should resolve to "+
			"FloorA_Light1 (1000001), got %v", dpA)
	}
	if dpB == nil || dpB.device.serialNumber != 2000002 {
		t.Fatalf("Floor B interface: datapoint 1 should resolve to "+
			"FloorB_Shutter1 (2000002), got %v", dpB)
	}

	t.Logf("FIX CONFIRMED: Floor A interface datapoint 1 = serial %d "+
		"(FloorA_Light1), Floor B interface datapoint 1 = serial %d "+
		"(FloorB_Shutter1) - both correctly distinct, no collision",
		dpA.device.serialNumber, dpB.device.serialNumber)
}

func writeTmp(t *testing.T, name, content string) {
	t.Helper()
	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(name) })
}

var _ = strings.TrimSpace
