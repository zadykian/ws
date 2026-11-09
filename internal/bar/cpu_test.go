package bar

import (
	"testing"
	"testing/fstest"
)

func TestReadCPUAndUsage(t *testing.T) {
	fsys := fstest.MapFS{"proc/stat": {Data: []byte("cpu  100 0 100 700 100 0 0 0 0 0\ncpu0 1 2 3 4 5\n")}}
	before, err := readCPU(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if before != (cpuTimes{busy: 200, total: 1000}) {
		t.Fatalf("readCPU = %+v, want 200 busy of 1000", before)
	}
	fsys["proc/stat"] = &fstest.MapFile{Data: []byte("cpu  130 0 120 740 110 0 0 0 0 0\n")}
	now, err := readCPU(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := usage(before, now); !ok || got != 50 {
		t.Errorf("usage = %d, %v; want 50, true", got, ok)
	}
	if _, ok := usage(now, now); ok {
		t.Error("usage of no time passed is a reading")
	}
}

func TestReadCPURefuses(t *testing.T) {
	for name, data := range map[string]string{
		"another first line": "intr 1 2 3 4 5 6\n",
		"a word for a count": "cpu 1 2 x 4 5 6\n",
		"too few fields":     "cpu 1 2 3\n",
	} {
		if _, err := readCPU(fstest.MapFS{"proc/stat": {Data: []byte(data)}}); err == nil {
			t.Errorf("%s: readCPU took %q", name, data)
		}
	}
	if _, err := readCPU(fstest.MapFS{}); err == nil {
		t.Error("readCPU took a missing /proc/stat")
	}
}
