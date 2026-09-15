package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ByteSize is a size in bytes. Thresholds use binary units (1 MB = 1 048 576
// bytes) because that is what Windows Explorer shows to the tool's users.
type ByteSize int64

const (
	KB ByteSize = 1 << 10
	MB ByteSize = 1 << 20
	GB ByteSize = 1 << 30
	TB ByteSize = 1 << 40
)

var units = []struct {
	suffix string
	size   ByteSize
}{
	{"TB", TB}, {"GB", GB}, {"MB", MB}, {"KB", KB},
	{"T", TB}, {"G", GB}, {"M", MB}, {"K", KB},
	{"B", 1},
}

// ParseByteSize parses "100MB", "15 mb", "1.5G", "512", "2K". Units are binary.
func ParseByteSize(s string) (ByteSize, error) {
	orig := s
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return 0, fmt.Errorf("empty size")
	}
	mult := ByteSize(1)
	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			mult = u.size
			s = strings.TrimSpace(strings.TrimSuffix(s, u.suffix))
			break
		}
	}
	if s == "" {
		return 0, fmt.Errorf("size %q has no number", orig)
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		if n < 0 {
			return 0, fmt.Errorf("size %q is negative", orig)
		}
		return ByteSize(n) * mult, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < 0 {
		return 0, fmt.Errorf("invalid size %q", orig)
	}
	return ByteSize(f * float64(mult)), nil
}

// String renders the size with one decimal in the largest fitting binary unit.
func (b ByteSize) String() string {
	switch {
	case b >= TB:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", int64(b))
	}
}

// UnmarshalYAML accepts either a bare integer (bytes) or a string with a unit.
func (b *ByteSize) UnmarshalYAML(node *yaml.Node) error {
	v, err := ParseByteSize(node.Value)
	if err != nil {
		return err
	}
	*b = v
	return nil
}

// MarshalYAML writes the human form so a dumped config stays readable.
func (b ByteSize) MarshalYAML() (any, error) { return b.String(), nil }

// MarshalJSON writes plain bytes; report consumers should not parse units.
func (b ByteSize) MarshalJSON() ([]byte, error) { return json.Marshal(int64(b)) }

// UnmarshalJSON accepts plain bytes or a string with a unit.
func (b *ByteSize) UnmarshalJSON(data []byte) error {
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		*b = ByteSize(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	v, err := ParseByteSize(s)
	if err != nil {
		return err
	}
	*b = v
	return nil
}
