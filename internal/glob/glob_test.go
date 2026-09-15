package glob

import "testing"

func TestMatch(t *testing.T) {
	cases := []struct {
		pattern, name string
		want          bool
	}{
		{"Thumbs.db", "thumbs.db", true},
		{"*.BAK", "report.bak", true},
		{"~$*", "~$Отчёт.docx", true},
		{"*.tmp", "file.tmp.txt", false},
		{"[", "x", false}, // invalid pattern never matches
	}
	for _, c := range cases {
		if got := Match(c.pattern, c.name); got != c.want {
			t.Errorf("Match(%q,%q)=%v want %v", c.pattern, c.name, got, c.want)
		}
	}
}

func TestMatchAny(t *testing.T) {
	pats := []string{"node_modules", "Старое/", "Проект B/Фото/*.jpg", "*.log"}
	cases := []struct {
		rel, name string
		want      bool
	}{
		{"x/node_modules", "node_modules", true},
		{"старое/архив.zip", "архив.zip", true},
		{"Старое", "Старое", true},
		{"Проект B/Фото/IMG_1.JPG", "IMG_1.JPG", true},
		{"Проект B/Фото/sub/IMG_1.jpg", "IMG_1.jpg", false},
		{"a/b/app.log", "app.log", true},
		{"a/b/app.txt", "app.txt", false},
	}
	for _, c := range cases {
		if got := MatchAny(pats, c.rel, c.name); got != c.want {
			t.Errorf("MatchAny(%q,%q)=%v want %v", c.rel, c.name, got, c.want)
		}
	}
}
