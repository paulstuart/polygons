package polygons

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"testing"

	"golang.org/x/exp/constraints"
)

const (
	AlaLat = 37.77033688841509
	AlaLon = -122.25697282612731
)

// CountyGeo is adapted from github.com/paulstuart/go-counties
type CountyGeo struct {
	GeoID int     `json:"geoid"`
	Name  string  `json:"name"`
	Full  string  `json:"fullname"`
	State string  `json:"state"`
	BBox  BBox    `json:"bbox"`
	Poly  PPoints `json:"polygon"`
}

type polyTest struct {
	pt     Pair
	poly   PPoints
	inside bool
}

const (
	sampleCounties = "testdata/ca_counties.gob.gz"
	allCounties    = "../go-counties/county_geo.gob.gz"
)

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || !errors.Is(err, fs.ErrNotExist)
}

func TestDownsize(t *testing.T) {
	if exists(sampleCounties) {
		t.Skip("already downsized: " + sampleCounties)
	}
	var cg []CountyGeo
	if err := GobLoad(allCounties, &cg); err != nil {
		t.Fatal(err)
	}
	var ca []CountyGeo
	for _, c := range cg {
		if c.State == "CA" {
			ca = append(ca, c)
		}
	}
	if err := GobDump(sampleCounties, ca); err != nil {
		t.Fatal(err)
	}
}

func TestPolygons(t *testing.T) {
	tests := []polyTest{
		{
			Pair{20, 20},
			PPoints{{0, 0}, {10, 0}, {10, 10}, {0, 10}},
			false,
		},
		{
			Pair{5, 5},
			PPoints{{0, 0}, {10, 0}, {10, 10}, {0, 10}},
			true,
		},
		{
			Pair{3, 3},
			PPoints{{0, 0}, {5, 5}, {5, 0}},
			true,
		},
		{
			Pair{-1, 10},
			PPoints{{0, 0}, {10, 0}, {10, 10}, {0, 10}},
			false,
		},
	}

	for _, tt := range tests {
		inside := tt.poly.Contains(tt.pt)
		if tt.inside != inside {
			t.Errorf("want: %t have: %t", tt.inside, inside)
		}
	}
}

func loadPolygon(t *testing.T, filename string) PPoints {
	t.Helper()
	f, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	var poly PPoints
	if err := json.NewDecoder(f).Decode(&poly); err != nil {
		t.Fatal(err)
	}
	return poly
}

func loadPolygons(t *testing.T, filename string) []PPoints {
	t.Helper()
	f, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	var poly []PPoints
	if err := json.NewDecoder(f).Decode(&poly); err != nil {
		t.Fatal(err)
	}
	return poly
}

func TestBeaverton(t *testing.T) {
	poly := loadPolygon(t, "testdata/washington.json")
	const (
		lat = 45.481300
		lon = -122.743996
	)
	pt := Pair{lon, lat}
	inside := poly.Contains(pt)
	t.Logf("INSIDE: %t", inside)
}

func TestAlameda(t *testing.T) {
	polys := loadPolygons(t, "testdata/alameda.json")
	for _, poly := range polys {
		pt := Pair{AlaLon, AlaLat}
		inside := poly.Contains(pt)
		t.Logf("INSIDE: %t", inside)
	}
}

type Helper interface {
	Helper()
	Fatal(...interface{})
}

func Prep[T constraints.Unsigned](t Helper) *Finder[T] {
	t.Helper()
	const filename = sampleCounties
	var cg []CountyGeo
	if err := GobLoad(filename, &cg); err != nil {
		t.Fatal(err)
	}
	sort.Slice(cg, func(i, j int) bool {
		return cg[i].GeoID < cg[j].GeoID
	})
	pg := NewFinder[T]()

	for _, c := range cg {
		pg.Add(c.GeoID, c.Poly)
	}
	fmt.Printf("PG SIZE:%d\n", pg.tree.Len())
	// fmt.Printf("PG COUNT:%d\n", pg.tree.Count())
	return pg
}

/*
func TestSearchers(t *testing.T) {
	f := Prep[uint](t)
	f.Sort()
	s := NewSearcher(f)
	pt := Pair{AlaLon, AlaLat}
	id, dist := s.Search(pt)
	if id < 0 {
		t.Fatal("not found")
	}
	t.Logf("ID: %d DIST: %f", id, dist)
	twin := Echo(s)
	id, dist = twin.Search(pt)
	if id < 0 {
		t.Fatal("not found")
	}
	if err := s.Equal(twin); err != nil {
		t.Errorf("not very equal: %v", err)
	}
	t.Logf("TWIN ID: %d DIST: %f", id, dist)
	now := time.Now()
	id, dist = twin.Search(pt)
	if id < 0 {
		t.Fatal("twin not found")
	}
	elapsed := time.Since(now)
	t.Logf("TWIN ID: %d DIST: %f (%s)", id, dist, &elapsed)
	//	SaveJSON("testdata/searcher.json", s)

}
*/

func TestBBox(t *testing.T) {
	pg := Prep[uint](t)
	pg.Sort()
	pt := Pair{AlaLon, AlaLat}
	id, dist := pg.Search(pt)
	if id < 0 {
		t.Fatal("not found")
	}
	t.Logf("REAL LEN: %d", len(pg.sorted))
	t.Logf("poly search size: %d", pg.sorted.Size())
	t.Logf("ID: %d DIST: %f", id, dist)
}

func BenchmarkBBox(b *testing.B) {
	pg := Prep[uint](b)
	pt := Pair{AlaLon, AlaLat}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pg.Search(pt)
	}
}
