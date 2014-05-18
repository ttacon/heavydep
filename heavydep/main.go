package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strconv"
	"strings"

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
	dotFile       = flag.String("g", "", "filename for the svg output (dependency graph)")
)

func main() {
	flag.Parse()

	var (
		dotFileName string
		dotFileType string
	)

	// TODO(ttacon): validate that it is a go package
	if len(*pkg) == 0 {
		heavydep.Log("you're mean, actually provide a file")
		return
	}

	if len(*dotFile) > 0 {
		dotFileName = *dotFile
		dotFileType = getImageFileType(*dotFile)
		if dotFileType == "" {
			heavydep.Log("the file type of %q isn't a supported type", dotFileName)
			return
		}
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
	if len(dotFileName) > 0 {
		printDot(imps, dotFileName, dotFileType)
	}
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

func printDot(imps []*heavydep.WeightedImport, dotFileName, dotFileType string) {
	// TODO(ttacon): open dot file in os.TempDir() and actually
	// execute dot to produce desired file at specified location
	buf := bytes.Buffer{}

	buf.WriteString("digraph G {\n")

	for _, imp := range imps {
		for depender, weight := range imp.ImpEdges {
			buf.WriteString("\t\t\"" + depender + "\" -> " + imp.Name + " [label=\"" + strconv.Itoa(weight) + "\"];\n")
		}
	}
	buf.WriteString("}\n")

	_, err := exec.LookPath("dot")
	if err != nil {
		heavydep.Log("dot doesn't seemed to be installed (or it's not on your path) - not writing %q\n", dotFileName)
		return
	}

	cmd := exec.Command("dot", "-T"+dotFileType, "-o"+dotFileName)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		heavydep.Log("failed to retrieve a stdin pipe to the dot, exiting...")
		return
	}

	bufLen := buf.Len()
	written, err := io.Copy(stdin, &buf)
	if err != nil {
		heavydep.Log("failed to write all bytes from buf to dot, err: ", err)
		return
	}

	if written != int64(bufLen) {
		heavydep.Log("wrote fewer bytes than expected, wrote %d, expected to write %d\n", written, bufLen)
		return
	}

	err = stdin.Close()
	if err != nil {
		heavydep.Log("couldn't close stdin pipe to dot, err: ", err)
		return
	}

	err = cmd.Run()
	if err != nil {
		heavydep.Log("failed to run dot, err: ", err)
	}
}

func getImageFileType(filename string) string {
	pieces := strings.Split(filename, ".")
	if len(pieces) != 2 {
		return ""
	}
	extension := pieces[1]
	switch extension {
	case "png":
		fallthrough
	case "jpeg":
		fallthrough
	case "bmp":
		fallthrough
	case "gif":
		fallthrough
	case "jpg":
		fallthrough
	case "pdf":
		return extension
	default:
		return ""
	}
}
