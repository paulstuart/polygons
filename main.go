package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

var Separator = "\t"

func main() {
	var fields string
	var header bool
	flag.BoolVar(&header, "header", false, "skip header")
	flag.StringVar(&fields, "fields", "", "field list, any dashed range is returned as a json polygon")
	flag.Parse()
	if len(flag.Args()) < 1 {
		log.Fatalf("usage: %s <filename>\n", os.Args[0])
	}
	args := flag.Args()
	f, err := os.Open(args[0])
	if err != nil {
		log.Fatal(err)
	}
	r := csv.NewReader(f)
	w := os.Stdout
	p, cols, err := polyWriter(fields, w)
	if err != nil {
		log.Fatal(err)
	}
	for {
		record, err := r.Read()
		if err != nil {
			if err != io.EOF {
				log.Fatal(err)
			}
			if len(record) > 0 {
				p(record)
			}
			fmt.Fprintln(w)
			return
		}
		if header {
			//fmt.Print("HEADER:")
			header = false
			for i, name := range cols {
				//fmt.Printf("CHECK %d/%d: %s\n", i+1, len(cols), name[:1])
				if strings.ContainsAny(name[:1], "0123456789") {
					idx, err := strconv.Atoi(name)
					if err != nil {
						log.Fatalf("column %d has bad index %q: %v\n", i, name, err)
					}
					idx--
					cols[i] = record[idx]
				}
			}
			fmt.Fprintln(w, strings.Join(cols, Separator))
			continue
		}
		p(record)
	}
}

type colWriter func([]string) error

func polyWriter(fields string, w io.Writer) (colWriter, []string, error) {
	if w == nil {
		w = os.Stdout
	}
	var columns []colWriter
	var names []string
	flist := strings.Split(fields, ",")
	for i, field := range flist {
		if idx := strings.Index(field, ":"); idx > 0 {
			//fmt.Printf("for %d/%d: %s\n", i+1, len(flist), field[idx+1:])
			names = append(names, field[idx+1:])
			field = field[:idx]
		} else {
			names = append(names, field)
		}
		between := strings.Split(field, "-")
		at := strings.TrimSpace(between[0])
		col, err := strconv.Atoi(at)
		if err != nil {
			return nil, nil, fmt.Errorf("field %d (%s) is invalid: %w", i, at, err)
		}
		// expect columns to use 1-based offset, so adjust accordingly
		col--
		//comma := i > 0
		if len(between) == 1 {
			fn := func(ss []string) error {
				if len(ss) < col {
					return fmt.Errorf("column %d is greater than list size of %d", col, len(ss))
				}
				_, err := fmt.Fprint(w, ss[col])
				return err
			}
			columns = append(columns, fn)
		} else {
			upto := strings.TrimSpace(between[1])
			last, err := strconv.Atoi(upto)
			if err != nil {
				return nil, nil, fmt.Errorf("field %d (%s) is invalid: %w", i, upto, err)
			}
			// expect columns to use 1-based offset, so adjust accordingly
			last--
			fn := func(ss []string) error {
				if len(ss) < col {
					return fmt.Errorf("column %d is greater than list size of %d", last, len(ss))
				}
				span := ss[col:last]
				_, err := fmt.Fprint(w, poly(span))
				return err
			}
			columns = append(columns, fn)
		}
	}
	//fmt.Println("NAMES:", names)
	return func(ss []string) error {
		//fmt.Println("LINE:", ss)
		for i, cw := range columns {
			if i > 0 {
				if _, err := fmt.Fprint(w, Separator); err != nil {
					return err
				}
			}
			if err := cw(ss); err != nil {
				return err
			}
		}
		fmt.Fprintln(w)
		return nil
	}, names, nil
}

func poly(ss []string) string {
	sb := &strings.Builder{}
	sb.WriteString(`"[`)
	for i, s := range ss {
		if s == "" {
			break
		}
		if i%2 == 1 {
			sb.WriteString(",")
			sb.WriteString(s)
			sb.WriteString("]")
		} else {
			if i > 1 {
				sb.WriteString(",")
			}
			sb.WriteString("[")
			sb.WriteString(s)
		}
	}
	// close the polygon if necessary
	if len(ss) > 3 && ((ss[0] != ss[len(ss)-2]) || ss[1] != ss[len(ss)-1]) {
		sb.WriteString(",[")
		sb.WriteString(ss[0])
		sb.WriteString(",")
		sb.WriteString(ss[1])
		sb.WriteString("]")
	}
	sb.WriteString(`]"`)
	return sb.String()
}

/*
// if data given as lon,lat
func polyInverse(ss []string) string {
	var lon string
	sb := &strings.Builder{}
	sb.WriteString(`'[`)
	for i, s := range ss {
		if s == "" {
			break
		}
		if i%2 == 1 {
			sb.WriteString(",")
			sb.WriteString(s)
			sb.WriteString("]")
		} else {
			if i > 1 {
				sb.WriteString(",")
			}
			sb.WriteString("[")
			sb.WriteString(s)
		}
	}
	sb.WriteString(`]'`)
	return sb.String()
}
*/
