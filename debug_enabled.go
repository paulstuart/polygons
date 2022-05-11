//go:build debug

package polygons

import (
	"log"
)

func debugf(text string, args ...interface{}) {
	log.Printf(text, args...)
}
