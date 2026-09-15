package report

import (
	"testing"

	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/scan"
)

func TestHumanSize(t *testing.T) {
	i18n.Set(i18n.EN)
	t.Cleanup(func() { i18n.Set(i18n.RU) })
	cases := map[int64]string{
		0:                               "0 B",
		512:                             "512 B",
		int64(config.KB):                "1.0 KB",
		int64(15 * config.MB):           "15.0 MB",
		int64(1.5 * float64(config.GB)): "1.5 GB",
		int64(2 * config.TB):            "2.0 TB",
	}
	for n, want := range cases {
		if got := HumanSize(n); got != want {
			t.Errorf("HumanSize(%d)=%q want %q", n, got, want)
		}
	}
	i18n.Set(i18n.RU)
	if got := HumanSize(int64(15 * config.MB)); got != "15.0 МБ" {
		t.Errorf("RU units: %q", got)
	}
}

func TestTopItems(t *testing.T) {
	entries := []scan.Entry{
		{Rel: "small", Size: 1}, {Rel: "big", Size: 100, Files: 3}, {Rel: "mid", Size: 50},
	}
	top := topItems(entries, 2)
	if len(top) != 2 || top[0].Rel != "big" || top[0].Files != 3 || top[1].Rel != "mid" {
		t.Errorf("topItems wrong: %+v", top)
	}
	if got := topItems(entries, 10); len(got) != 3 {
		t.Errorf("n larger than input must return all: %d", len(got))
	}
	if got := topItems(nil, 5); len(got) != 0 {
		t.Errorf("nil input: %+v", got)
	}
}
