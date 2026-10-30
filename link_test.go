package curseforge

import "testing"

func TestParseLink(t *testing.T) {
	t.Parallel()
	if ref, err := ParseLink("curseforge://install?addonId=309243&fileId=555"); err != nil || ref != (Ref{Project: 309243, File: 555}) {
		t.Fatalf("%+v %v", ref, err)
	}
	for _, bad := range []string{
		"", "curseforge://install", "curseforge://install?addonId=1", "curseforge://install?addonId=0&fileId=2",
		"curseforge://install?addonId=-1&fileId=2", "curseforge://install?addonId=x&fileId=2",
		"curseforge://install/../x?addonId=1&fileId=2", "curseforge://user@install?addonId=1&fileId=2",
		"curseforge://install?addonId=1&fileId=2#f", "curseforge://other?addonId=1&fileId=2", "https://install?addonId=1&fileId=2",
	} {
		if _, err := ParseLink(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}

// FuzzParseLink holds that an accepted install link names a positive project and file.
func FuzzParseLink(f *testing.F) {
	f.Add("curseforge://install?addonId=309243&fileId=555")
	f.Add("curseforge://install?addonId=-1&fileId=555")
	f.Add("curseforge://install/../x?addonId=1&fileId=2")
	f.Add("curseforge://user@install?addonId=1&fileId=2")
	f.Add("")
	f.Fuzz(func(t *testing.T, raw string) {
		ref, err := ParseLink(raw)
		if err == nil && (ref.Project <= 0 || ref.File <= 0) {
			t.Fatalf("bad ref %+v from %q", ref, raw)
		}
	})
}
