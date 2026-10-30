package curseforge

import (
	"slices"
	"testing"
)

func TestNewestPrefersStableOnTheReleaseChannel(t *testing.T) {
	t.Parallel()
	files := []File{
		{ID: 1, IsAvailable: true, ReleaseType: ReleaseStable, FileDate: "2026-01-01", GameVersions: []string{"1.5"}},
		{ID: 2, IsAvailable: true, ReleaseType: ReleaseStable, FileDate: "2026-02-01", GameVersions: []string{"1.6"}},
		{ID: 3, IsAvailable: true, ReleaseType: ReleaseBeta, FileDate: "2026-03-01"},
		{ID: 4, IsAvailable: false, ReleaseType: ReleaseStable, FileDate: "2026-04-01"},
		{ID: 5, IsAvailable: true, ReleaseType: ReleaseAlpha, FileDate: "2026-05-01"},
	}
	for _, c := range []struct {
		versions  []string
		max, want int
	}{
		{nil, ReleaseStable, 2},
		{[]string{"1.5"}, ReleaseStable, 1},
		{nil, ReleaseBeta, 3},
		{nil, ReleaseAlpha, 5},
	} {
		if f, ok := Newest(files, c.versions, c.max); !ok || f.ID != c.want {
			t.Errorf("%v max %d: got %d %v, want %d", c.versions, c.max, f.ID, ok, c.want)
		}
	}
	if f, ok := Newest(files[2:3], nil, ReleaseStable); !ok || f.ID != 3 {
		t.Errorf("with no stable file the newest other one stands in: %d %v", f.ID, ok)
	}
	if _, ok := Newest(nil, nil, ReleaseStable); ok {
		t.Error("no files, no newest")
	}
}

func TestFileHelpers(t *testing.T) {
	t.Parallel()
	f := File{
		Hashes:       []Hash{{Value: "CD", Algo: HashMD5}, {Value: "AB", Algo: HashSHA1}},
		Dependencies: []Dependency{{ModID: 11, RelationType: RelationRequired}, {ModID: 12, RelationType: RelationOptional}, {ModID: 0, RelationType: RelationRequired}},
	}
	if f.SHA1() != "ab" || !slices.Equal(f.Required(), []int{11}) {
		t.Fatalf("%q %v", f.SHA1(), f.Required())
	}
	if (File{}).SHA1() != "" || (Mod{}).Restricted() {
		t.Fatal("empty values are neither hashed nor restricted")
	}
}
