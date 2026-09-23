package desktop

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

// Disk history (ADR-0203): one JSON line per calendar day in
// %LOCALAPPDATA%\PiCode\disk-history.jsonl, written by every full scan and
// by the shell's daily background scan, read by the Management window.

// Sample is one day's measurement.
type Sample struct {
	Day          string           `json:"day"` // YYYY-MM-DD, local time
	At           time.Time        `json:"at"`
	WinFreeBytes int64            `json:"winFree"`
	WinSizeBytes int64            `json:"winSize"`
	DiskBytes    int64            `json:"disk"` // the disk file, allocated
	HeldBytes    int64            `json:"held"`
	UsedBytes    int64            `json:"used"` // inside the distro
	Caches       map[string]int64 `json:"caches,omitempty"`
}

// HistoryMax bounds the file: a little over a year of days.
const HistoryMax = 400

// HistoryPath sits next to the hold file and the installed tools.
func HistoryPath() string {
	return filepath.Join(filepath.Dir(HoldPath()), "disk-history.jsonl")
}

// ReadHistory returns the samples oldest first. A missing file is empty; a
// line that does not parse is skipped.
func ReadHistory(path string) []Sample {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []Sample
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		var s Sample
		if json.Unmarshal(sc.Bytes(), &s) == nil && dayRe.MatchString(s.Day) {
			out = append(out, s)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Day < out[j].Day })
	return out
}

// RecordSample writes s as its day's line — replacing that day's earlier
// line — and keeps the newest HistoryMax days. Written to a temp file and
// renamed, so a reader never sees half a file.
func RecordSample(path string, s Sample) error {
	days := map[string]Sample{}
	for _, old := range ReadHistory(path) {
		days[old.Day] = old
	}
	days[s.Day] = s
	var all []Sample
	for _, v := range days {
		all = append(all, v)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Day < all[j].Day })
	if len(all) > HistoryMax {
		all = all[len(all)-HistoryMax:]
	}
	var buf bytes.Buffer
	for _, v := range all {
		line, err := json.Marshal(v)
		if err != nil {
			return err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// A temp file of its own (two writers never share one), then a rename
	// retried briefly: on Windows a reader or a scanner holding the target
	// open without delete sharing makes the replace fail for a moment.
	f, err := os.CreateTemp(filepath.Dir(path), "disk-history-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err := f.Write(buf.Bytes()); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	for i := 0; ; i++ {
		err = os.Rename(tmp, path)
		if err == nil || i == 4 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		os.Remove(tmp)
	}
	return err
}

// dayRe is the only shape a Day may have; anything else in the file (it is
// the person's to edit) is not a sample.
var dayRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// Growth is how much one cache changed between two samples.
type Growth struct {
	ID    string `json:"id"`
	Bytes int64  `json:"bytes"` // positive: grew
	From  string `json:"from"`  // day of the baseline sample
}

// GrowthSince compares the newest sample with the newest one at least
// `days` days older, per cache, largest growth first. With no such baseline
// it answers nothing — a trend needs two points.
func GrowthSince(samples []Sample, days int) []Growth {
	if len(samples) < 2 {
		return nil
	}
	last := samples[len(samples)-1]
	lastDay, err := time.Parse("2006-01-02", last.Day)
	if err != nil {
		return nil
	}
	cutoff := lastDay.AddDate(0, 0, -days).Format("2006-01-02")
	var base *Sample
	for i := len(samples) - 2; i >= 0; i-- {
		if samples[i].Day <= cutoff {
			base = &samples[i]
			break
		}
	}
	if base == nil {
		return nil
	}
	var out []Growth
	for id, now := range last.Caches {
		// A cache the baseline did not measure (its stage failed, or it
		// did not exist yet) has no growth to report — never its whole
		// size as "grew".
		was, measured := base.Caches[id]
		if !measured {
			continue
		}
		if d := now - was; d != 0 {
			out = append(out, Growth{ID: id, Bytes: d, From: base.Day})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Bytes > out[j].Bytes })
	return out
}
