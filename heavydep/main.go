package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"

	"github.com/ttacon/heavydep"
)

/*
ROADMAP:
1) clean up dot interaction
2) decent validation
3) allow -v (verbose) error Logging
4) doc.go
5) _tests
6) versions of heavydep.WeightedImportsFor* that take limiting paramater (i.e. only want top 5)
*/

var (
	pkg           = flag.String("pkg", "", "pkg to inspect")
	recursiveDeps = flag.Bool("r", false, "recursively investigate dependencies")
	topDeps       = flag.Int("n", 0, "list most common 'n' dependencies")
	printGraph    = flag.Bool("g", false, "make a svg of the weighted dependency graph")
)

func main() {
	flag.Parse()

	// TODO(ttacon): validate that it is a go package
	if len(*pkg) == 0 {
		heavydep.Log("you're mean, actually provide a file")
		return
	}

	var imps []*heavydep.WeightedImport

	if !*recursiveDeps {
		imps = heavydep.WeightedImportsForPkg(*pkg)
	} else {
		imps = heavydep.WeightedImportsForPkgRec(*pkg)
	}

	if len(imps) == 0 {
		heavydep.Log("package %q has no third-party dependencies\n", *pkg)
		return
	}

	display(imps, *topDeps)
	printDot(*printGraph, imps)
}

func display(imps []*heavydep.WeightedImport, maxDisplay int) {
	// TODO(ttacon): "column" like output/align weights
	sort.Sort(heavydep.ByWeight(imps))
	for i, val := range imps {
		if maxDisplay > 0 && i == maxDisplay {
			break
		}
		fmt.Printf("[%s] %d\n", val.Name, val.Weight)
	}
}

func printDot(doIt bool, imps []*heavydep.WeightedImport) {
	// TODO(ttacon): open dot file in os.TempDir() and actually
	// execute dot to produce desired file at specified location
	if !doIt {
		return
	}

	buf := bytes.Buffer{}

	buf.WriteString("digraph G {\n")

	for _, imp := range imps {
		for depender, weight := range imp.ImpEdges {
			buf.WriteString("\t\t\"" + depender + "\" -> " + imp.Name + " [label=\"" + strconv.Itoa(weight) + "\"];\n")
		}
	}
	buf.WriteString("}\n")

	err := ioutil.WriteFile("tmp.dot", buf.Bytes(), os.ModePerm)
	if err != nil {
		heavydep.Log("couldn't write file, err: ", err)
	}
}
