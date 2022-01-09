package polygons

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"os"
)

// GobDump saves the object in a gzipped GOB encoded file
func GobDump(filename string, obj interface{}) error {
	if obj == nil {
		return fmt.Errorf("nil object of type %T", obj)
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	w := gzip.NewWriter(f)
	err = gob.NewEncoder(w).Encode(obj)
	if err != nil {
		w.Close()
		return fmt.Errorf("error encoding to %q -- %w", filename, err)
	}
	if err := w.Close(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// GobLoad populates the object from a gzipped GOB encoded file
func GobLoad(filename string, obj interface{}) error {
	if obj == nil {
		return fmt.Errorf("nil object of type %T", obj)
	}
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer r.Close()
	dec := gob.NewDecoder(r)
	return dec.Decode(obj)
}
