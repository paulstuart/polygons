package polygons

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"

	"github.com/paulstuart/geo"

	"github.com/paulstuart/rtree"
	"golang.org/x/exp/constraints"
)

// Searcher is read-only collection polygons with associated integer ids
// The ids do not have to be unique (multiple polygons can share an id)
// Searching for a polygon containing a point is optimized by creating
// an RTree of bounding boxes, and then searching candidates within
// matching bounding boxes
// TODO: make `int` a [T]
type Searcher[T constraints.Unsigned] struct {
	IDs    map[int]T // Key lookup -> Value (external ID)
	Polys  []PPoints // the actual
	Sorted PolyPoints
	Tree   rtree.ReadOnly[T]
}

func NewSearcher[T constraints.Unsigned](f *Finder[T]) Searcher[T] {
	ids := make(map[int]T, len(f.ids))
	for k, v := range f.ids {
		ids[k] = T(v)
	}
	reply := Searcher[T]{
		Polys:  f.polys,
		IDs:    ids,
		Sorted: f.sorted,
		Tree:   rtree.NewReadOnly(f.tree),
	}
	fmt.Println("SEARCH TREE SIZE:", reply.Tree.Len())
	return reply
}

// Search returns the id of the polygon that contains the given point
// If polygons are searchable, it returns the id of the closest polygon
// and the distance away
//
// If not found return -1
func (s *Searcher[T]) Search(pt [2]float64) (int, float64) {
	// there may be many bboxen that contain the point,
	// but only one polygon should actually contain it
	var found bool
	var idx T
	fmt.Println("COUNT:", len(s.IDs))
	point := Pair{pt[0], pt[1]}
	s.Tree.Search(pt, pt, func(min, max [2]float64, what T) bool {
		fmt.Printf("NEW CHECK: %v\n", what)
		if s.Polys[what].Contains(point) {
			idx = what
			found = true
			return false
		}
		return true
	})
	if found {
		return int(s.IDs[int(idx)]), 0
	}
	if len(s.Sorted) > 0 {
		gpt := geo.GeoPoint(pt[0], pt[1])
		if i, dist := geo.Closest(s.Sorted, gpt, 10.0); i < len(s.Sorted) {
			return int(s.IDs[int(idx)]), dist
		}
	}
	return -1, 0
}

func Echo[T any](in T) T {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(in)
	if err != nil {
		panic(err)
	}
	var out T
	if err := gob.NewDecoder(&buf).Decode(&out); err != nil {
		panic(err)
	}
	return out
}

func SaveJSON[T any](filename string, in T) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(in); err != nil {
		return err
	}
	return f.Close()
}
