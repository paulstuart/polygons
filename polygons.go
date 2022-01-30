package polygons

import (
	"log"
	"math"
	"sort"
	"unsafe"

	"github.com/paulstuart/geo"

	"github.com/tidwall/rtree"
)

type Point = geo.Point
type GeoType = geo.GeoType

// Finder is a collection polygons with associated integer ids
// The ids do not have to be unique (multiple polygons can share an id)
// Searching for a polygon containing a point is optimized by creating
// an RTree of bounding boxes, and then searching candidates within
// matching bounding boxes
type Finder struct {
	tree   rtree.RTree
	polys  []PPoints
	ids    map[int]int // key is the polygon index, value is the id
	sorted PolyPoints
}

// NewFinder returns a Finder for finding a containing polygon
func NewFinder() *Finder {
	return &Finder{
		ids: make(map[int]int),
	}
}

func (py *Finder) Sort() {
	var size int
	for _, pp := range py.polys {
		size += len(pp)
	}
	py.sorted = make(PolyPoints, size)
	for i, pp := range py.polys {
		for _, p := range pp {
			ppt := PolyPoint{p, i}
			py.sorted = append(py.sorted, ppt)
		}
	}

	sort.Slice(py.sorted, func(i, j int) bool {
		return py.sorted[i].Less(py.sorted[j])
	})

}

// Add a polygon to be searched
func (py *Finder) Add(id int, pp PPoints) {
	idx := len(py.polys)
	a, b := pp.BBox()
	py.tree.Insert(a, b, idx)
	py.polys = append(py.polys, pp)
	py.ids[idx] = id
}

// Search returns the id of the polygon that contains the given point
// If polygons are searchable, it returns the id of the closest polygon
// and the distance away
//
// If not found and no search index, it returns -1
func (py *Finder) Search(pt [2]float64) (int, float64) {
	// there may be many bboxen that contain the point,
	// but only one polygon should actually contain it
	found := -1
	//var possible []int
	point := Pair{pt[0], pt[1]}
	py.tree.Search(pt, pt, func(min, max [2]float64, data interface{}) bool {
		idx, ok := data.(int)
		if !ok {
			log.Printf("expected type %T but got type %T", idx, data)
			return true // keep searching
		}
		if py.polys[idx].Contains(point) {
			found = idx
			return false
		}
		//	possible = append(possible, idx)
		return true
	})
	if found >= 0 {
		return py.ids[found], 0
	}
	if len(py.sorted) > 0 {
		//closest := math.MaxFloat64
		gpt := geo.GeoPoint(pt[0], pt[1])
		//	for _, idx := range possible {
		if i, dist := geo.Closest(py.sorted, gpt, 10.0); i < len(py.sorted) {
			return py.ids[found], dist
		}
		//	}
	}
	return found, 0
	//	iter func(min, max [2]float64, data interface{}) bool,
}

type Pair [2]float64

func (p Pair) Less(x Pair) bool {
	if p[0] < x[0] {
		return true
	} else if p[0] > x[0] {
		return false
	} else {
		// lon is secondary sort
		return p[1] < x[1]
	}
}

func (p Pair) Point() geo.Point {
	return geo.Point{geo.GeoType(p[0]), geo.GeoType(p[1])}
}

// Define Infinite (Using INT_MAX caused overflow problems)
const farOut = math.MaxFloat64

type PolyPoint struct {
	P Pair
	I int
}

func (p PolyPoint) Less(x PolyPoint) bool {
	if p.P[0] < x.P[0] {
		return true
	} else if p.P[0] > x.P[0] {
		return false
	} else {
		// lon is secondary sort
		return p.P[1] < x.P[1]
	}
}

type PolyPoints []PolyPoint

type PPoints []Pair
type BBox [2]Pair

func (pp PolyPoints) Len() int {
	return len(pp)
}

func (pp PolyPoints) IndexPoint(i int) Point {
	p := pp[i].P
	lat := GeoType(p[0])
	lon := GeoType(p[1])
	return Point{lat, lon}
	//	return Point{GeoType(p[0]), GeoType(p[1])}
	//	return Point{GeoType(p[0]), GeoType(p[1])}
}

const psize = int(unsafe.Sizeof(PolyPoint{}))

func (pp PolyPoints) Size() int {
	return psize * len(pp)
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Given three collinear PPoints p, q, r, the function checks if
// PPoint q lies on line segment 'pr'
func onSegment(p, q, r Pair) bool {
	return (q[0] <= max(p[0], r[0]) && q[0] >= min(p[0], r[0]) &&
		q[1] <= max(p[1], r[1]) && q[1] >= min(p[1], r[1]))
}

// To find orientation of ordered triplet (p, q, r).
// The function returns following values
// 0 --> p, q and r are collinear
// 1 --> Clockwise
// 2 --> Counterclockwise
func orientation(p, q, r Pair) int {
	val := (q[1]-p[1])*(r[0]-q[0]) -
		(q[0]-p[0])*(r[1]-q[1])

	if val == 0 {
		return 0 // collinear
	}
	if val > 0 {
		return 1
	}
	return 2 // clock or counterclock wise
}

// The function that returns true if line segment 'p1q1'
// and 'p2q2' intersect.
func doIntersect(p1, q1, p2, q2 Pair) bool {
	// Find the four orientations needed for general and
	// special cases
	o1 := orientation(p1, q1, p2)
	o2 := orientation(p1, q1, q2)
	o3 := orientation(p2, q2, p1)
	o4 := orientation(p2, q2, q1)

	// General case
	if o1 != o2 && o3 != o4 {
		return true
	}

	// Special Cases
	// p1, q1 and p2 are collinear and p2 lies on segment p1q1
	if o1 == 0 && onSegment(p1, p2, q1) {
		return true
	}

	// p1, q1 and p2 are collinear and q2 lies on segment p1q1
	if o2 == 0 && onSegment(p1, q2, q1) {
		return true
	}

	// p2, q2 and p1 are collinear and p1 lies on segment p2q2
	if o3 == 0 && onSegment(p2, p1, q2) {
		return true
	}

	// p2, q2 and q1 are collinear and q1 lies on segment p2q2
	if o4 == 0 && onSegment(p2, q1, q2) {
		return true
	}

	return false // Doesn't fall in any of the above cases
}

// ContainsPoint returns true if the point is within the polygon
func (pps PPoints) ContainsPoint(pt geo.Point) bool {
	pair := Pair{float64(pt.Lat), float64(pt.Lon)}
	return pps.Contains(pair)
}

// Returns true if the PPoint p lies inside the polygon[] with n vertices
func (pps PPoints) Contains(p Pair) bool {
	// There must be at least 3 vertices in polygon[]
	if len(pps) < 3 {
		return false
	}
	// Create a line segment from p to ~infinity
	extreme := Pair{farOut, p[1]}

	// Count intersections of the above line with sides of polygon
	var count, i int
	for {
		next := (i + 1) % len(pps)

		// Check if the line segment from 'p' to 'extreme' intersects
		// with the line segment from 'polygon[i]' to 'polygon[next]'
		if doIntersect(pps[i], pps[next], p, extreme) {
			// If the point 'p' is collinear with line segment 'i-next',
			// then return if it lies on segment
			if orientation(pps[i], p, pps[next]) == 0 {
				return onSegment(pps[i], p, pps[next])
			}
			count++
		}
		i = next
		if i == 0 {
			break
		}
	}

	// Return true if count is odd, false otherwise
	return (count & 1) == 1
}

func (pp PPoints) BBox() (Pair, Pair) {
	const max = math.MaxFloat64
	var xMax, yMax, xMin, yMin float64 = -max, -max, max, max

	for _, pt := range pp {

		if pt[0] > xMax {
			xMax = pt[0]
		}
		if pt[1] > yMax {
			yMax = pt[1]
		}
		if pt[0] < xMin {
			xMin = pt[0]
		}
		if pt[1] < yMin {
			yMin = pt[1]
		}

	}
	return Pair{xMin, yMin}, Pair{xMax, yMax}
}

/*
func (pp PPoints) BBox() (Pair, Pair) {

	var xMax, yMax, xMin, yMin float64

	for _, pt := range pp {

		if pt[0] > xMax || xMax == 0.0 {
			xMax = pt[0]
		}
		if pt[1] > yMax || yMax == 0.0 {
			yMax = pt[1]
		}
		if pt[0] < xMin || xMin == 0.0 {
			xMin = pt[0]
		}
		if pt[1] < yMin || yMin == 0.0 {
			yMin = pt[1]
		}

	}
	return Pair{xMin, yMin}, Pair{xMax, yMax}
}
*/
