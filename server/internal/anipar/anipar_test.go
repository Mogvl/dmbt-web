package anipar

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fansubFixtures maps the fansub name (as passed to Parse) to the testdata
// base name. Input titles come from <base>.titles and expected outputs from
// <base>.json (title -> list of expected results — the CSVs contain duplicate
// lines that map to multiple numbered snapshots).
var fansubFixtures = []struct {
	fansub string
	base   string
}{
	{"Kirara Fantasia", "kirara_fantasia"},
	{"ANi", "ani"},
	{"LoliHouse", "lolihouse"},
	{"Prejudice-Studio", "prejudice_studio"},
	{"三明治摆烂组", "三明治摆烂组"},
	{"喵萌奶茶屋", "喵萌奶茶屋"},
	{"桜都字幕组", "桜都字幕组"},
	{"绿茶字幕组", "绿茶字幕组"},
	{"雪飄工作室(FLsnow)", "雪飄工作室_flsnow"},
}

func loadFixture(t *testing.T, base string) (titles []string, expected map[string][][]byte) {
	t.Helper()
	dir := "testdata"
	titlesBytes, err := os.ReadFile(filepath.Join(dir, base+".titles"))
	if err != nil {
		t.Fatalf("read titles: %v", err)
	}
	for _, line := range strings.Split(string(titlesBytes), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		titles = append(titles, line)
	}

	jsonBytes, err := os.ReadFile(filepath.Join(dir, base+".json"))
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	raw := map[string][]json.RawMessage{}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		t.Fatalf("unmarshal expected: %v", err)
	}
	expected = make(map[string][][]byte, len(raw))
	for k, v := range raw {
		list := make([][]byte, 0, len(v))
		for _, item := range v {
			list = append(list, item)
		}
		expected[k] = list
	}
	return titles, expected
}

func TestFixtures(t *testing.T) {
	for _, fc := range fansubFixtures {
		fc := fc
		t.Run(fc.fansub, func(t *testing.T) {
			titles, expected := loadFixture(t, fc.base)
			var failures, passes int
			for i, title := range titles {
				list, ok := expected[title]
				if !ok || len(list) == 0 {
					t.Errorf("[%s] line %d: no expected snapshot for %q", fc.fansub, i, title)
					continue
				}
				wantJSON := list[0]
				expected[title] = list[1:]

				got := Parse(title, fc.fansub)
				if got == nil {
					failures++
					t.Errorf("[%s] line %d: parse(%q) returned nil, want %s", fc.fansub, i, title, string(wantJSON))
					continue
				}
				gotJSON, err := json.Marshal(got)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				var gotAny, wantAny any
				if err := json.Unmarshal(gotJSON, &gotAny); err != nil {
					t.Fatalf("unmarshal got: %v", err)
				}
				if err := json.Unmarshal(wantJSON, &wantAny); err != nil {
					t.Fatalf("unmarshal want: %v", err)
				}
				if !reflect.DeepEqual(gotAny, wantAny) {
					failures++
					t.Errorf("[%s] line %d: mismatch for %q\n  got:  %s\n  want: %s", fc.fansub, i, title, string(gotJSON), string(wantJSON))
					continue
				}
				passes++
			}
			if len(titles) == 0 {
				t.Fatalf("no titles loaded for %s", fc.base)
			}
			t.Logf("%s: %d/%d passed", fc.fansub, passes, passes+failures)
		})
	}
}

// TestDebugFansub covers the inline-snapshot case from debug.test.ts.
func TestDebugFansub(t *testing.T) {
	title := `[雪飄工作室][アイカツプラネット！ミララボ/Aikatsu_Planet!-Mirror_Labo/偶像活動星球！镜中练功房][S2E01（总第13集）][繁](檢索:偶活/愛活)`
	got := Parse(title, Fansub雪飄工作室)
	gotJSON, _ := json.Marshal(got)
	want := `{"episode":{"number":1,"type":"总第13集"},"fansub":{"alias":"雪飄工作室","name":"雪飄工作室(FLsnow)"},"search":["偶活","愛活"],"season":{"number":2},"subtitle":{"languages":["繁"]},"title":"アイカツプラネット！ミララボ","titles":["Aikatsu_Planet!-Mirror_Labo","偶像活動星球！镜中练功房"]}`
	var gotAny, wantAny any
	json.Unmarshal(gotJSON, &gotAny)
	json.Unmarshal([]byte(want), &wantAny)
	if !reflect.DeepEqual(gotAny, wantAny) {
		t.Errorf("debug case mismatch\n  got:  %s\n  want: %s", string(gotJSON), want)
	}
}

// TestANiSwap covers the ANi two-title swap behavior.
func TestANiSwap(t *testing.T) {
	title := `[ANi] Classroom of the Elite S2 -  歡迎來到實力至上主義的教室 第二季 - 02 [1080P][Baha][WEB-DL][AAC AVC][CHT][MP4]`
	got := Parse(title, FansubANi)
	gotJSON, _ := json.Marshal(got)
	want := `{"fansub":{"name":"ANi"},"file":{"extension":"MP4","audio":{"codec":"AAC"},"video":{"codec":"AVC","resolution":"1080p"}},"subtitle":{"languages":["繁"]},"source":"WEB-DL","platform":"Baha","episode":{"number":2},"season":{"number":2},"title":"歡迎來到實力至上主義的教室","titles":["Classroom of the Elite"]}`
	var gotAny, wantAny any
	json.Unmarshal(gotJSON, &gotAny)
	json.Unmarshal([]byte(want), &wantAny)
	if !reflect.DeepEqual(gotAny, wantAny) {
		t.Errorf("ani swap mismatch\n  got:  %s\n  want: %s", string(gotJSON), want)
	}
}
