package polygons

import (
	"fmt"

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
	ids    map[int]T // Key lookup -> Value (external ID)
	polys  []PPoints // the actual
	sorted PolyPoints
	tree   rtree.ReadOnly[T]
}

func NewSearcher[T constraints.Unsigned](f *Finder) Searcher[T] {
	ids := make(map[int]T, len(f.ids))
	for k, v := range f.ids {
		ids[k] = T(v)
	}
	return Searcher[T]{
		polys: f.polys,
		ids:   ids,
		//tree:  rtree.NewReadOnly[T](f.tree),
	}
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
	fmt.Println("COUNT:", len(s.ids))
	point := Pair{pt[0], pt[1]}
	s.tree.Search(pt, pt, func(min, max [2]float64, what T) bool {
		fmt.Printf("CHECK: %v\n", what)
		if s.polys[what].Contains(point) {
			idx = what
			return false
		}
		return true
	})
	if found {
		return int(s.ids[int(idx)]), 0
	}
	if len(s.sorted) > 0 {
		gpt := geo.GeoPoint(pt[0], pt[1])
		if i, dist := geo.Closest(s.sorted, gpt, 10.0); i < len(s.sorted) {
			return int(s.ids[int(idx)]), dist
		}
	}
	return -1, 0
}
