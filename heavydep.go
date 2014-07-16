package heavydep

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
)

var (
	gopath = os.Getenv("GOPATH")
)

// Weighed Import represents a third-party dependency,
// the number of times it is relied as well as how much
// it relies on other third-party dependencies.
type WeightedImport struct {
	Name     string
	Weight   int
	ImpEdges map[string]int
	depender string
}

// WeightedImportsForPkgRec provides the same functionality as
// WeightedImportsForPkg but it recursively investigates each
// third-party dependency to see how heavily it depends on other
// third-party dependencies. The returned slice is sorted from
// the most relied upon dependency (heaviest) to the least depended
// on.
func WeightedImportsForPkgRec(pkg string) []*WeightedImport {
	var (
		imps          = WeightedImportsForPkg(pkg)
		investigated  = make(map[string]struct{})
		toInvestigate = make([]string, len(imps))
		all           = make([]*WeightedImport, len(imps))
	)

	investigated[pkg] = struct{}{}

	for i, imp := range imps {
		toInvestigate[i] = imp.Name
		all[i] = imp
	}

	for len(toInvestigate) > 0 {
		curr := toInvestigate[0]
		toInvestigate = toInvestigate[1:]
		imps = WeightedImportsForPkg(strings.Trim(curr, "\""))
		for _, imp := range imps {
			if _, ok := investigated[imp.Name]; !ok {
				toInvestigate = append(toInvestigate, imp.Name)
			}
			all = append(all, imp)
		}
		investigated[curr] = struct{}{}
	}

	// condense weightDeps
	depMap := make(map[string]*WeightedImport)
	for _, imp := range all {
		if dep, ok := depMap[imp.Name]; ok {
			dep.Weight += imp.Weight
			dep.ImpEdges[imp.depender] = imp.Weight
		} else {
			imp.ImpEdges = map[string]int{
				imp.depender: imp.Weight,
			}
			depMap[imp.Name] = imp
		}
	}

	var toSort = make([]*WeightedImport, len(depMap))
	i := 0
	for _, dep := range depMap {
		toSort[i] = dep
		i++
	}
	return toSort
}

// WeightedImportsForPkg computes the third-party imports that the given
// package depends on. By "weighted import" we simply mean that we
// calculate how many files in a given package depend on that
// third-party dependency. The returns slice is sorted from most
// relied upon dependency (heaviest) to least depended on.
func WeightedImportsForPkg(pkg string) []*WeightedImport {
	dir, err := os.Open(path.Join(gopath, "src", pkg))
	if err != nil {
		// TODO(ttacon): for now let's supress these
		// kinds of errors and just set a flag notifying
		// the user that there were errors, later we can
		// add a flag which can Log these errors all at once
		// Log("ruh roh, err: ", err, " pkg: ", pkg)
		return nil
	}
	defer dir.Close()

	files, err := dir.Readdir(0)
	if err != nil {
		Log("couldn't read dir, err: ", err)
		return nil
	}

	var imports []*WeightedImport
	var importMap = make(map[string]*WeightedImport)
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".go") || strings.HasSuffix(file.Name(), "_test.go") {
			continue
		}
		imps := importsForFile(path.Join("src", pkg, file.Name()))
		for _, imp := range imps {
			if _, ok := importMap[imp]; ok {
				importMap[imp].Weight++
			} else {
				newImp := &WeightedImport{
					Name:     imp,
					Weight:   1,
					depender: pkg,
				}
				importMap[imp] = newImp
				imports = append(imports, newImp)
			}
		}
	}

	imports = filterOutStdlib(imports)
	sort.Sort(ByWeight(imports))

	return imports
}

func isStdLib(imp string) bool {
	cleaned := strings.Trim(imp, "\"")
	pieces := strings.Split(cleaned, "/")
	if len(pieces) == 0 {
		return true
	}

	if _, ok := stdlib[pieces[0]]; ok {
		return true
	}
	return false
}

func filterOutStdlib(imps []*WeightedImport) []*WeightedImport {
	var ret []*WeightedImport

	for _, imp := range imps {
		if isStdLib(imp.Name) {
			continue
		}
		ret = append(ret, imp)
	}

	return ret
}

// ByWeight sorts a slice of WeightedImports by the imports weight.
type ByWeight []*WeightedImport

func (b ByWeight) Len() int           { return len(b) }
func (b ByWeight) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByWeight) Less(i, j int) bool { return b[i].Weight > b[j].Weight }

func importsForFile(fileName string) []string {
	f, err := os.Open(path.Join(gopath, fileName))
	if err != nil {
		Log("ruh roh bad file, err: ", err)
		return nil
	}
	defer f.Close()

	src, err := ioutil.ReadAll(f)
	if err != nil {
		Log("couldn't read your silly file, err: ", err)
		return nil
	}

	fset := token.NewFileSet()
	fi, err := parser.ParseFile(fset, fileName, src, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		Log("failed parsing file, err: ", err)
		return nil
	}

	var imports = make([]string, len(fi.Imports))
	for i, imp := range fi.Imports {
		imports[i] = imp.Path.Value
	}
	return imports
}

// TODO(ttacon): Log(err) should be valid too...
func Log(message string, args ...interface{}) {
	if message[len(message)-1] != '\n' {
		message += "\n"
	}
	message = "[heavydep] " + message
	if !strings.Contains(message, "%") {
		fmt.Println(message)
	}
	fmt.Printf(message, args...)
}
