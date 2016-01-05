package doc

/*
This package uses "go/doc" to extract documentation for packages
Used by swagger to put comments on types, operation and determine which types are enums
*/

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"stash.ovh.net/xander/gogadget/wosk"
)

type constantInfo struct {
	ListC []string
	Imprt string
}

// Infos structure contains extrax
type Infos struct {
	FunctionsDoc    map[string]string
	TypesDoc        map[string]string
	StructFieldsDoc map[string]map[string]string
	Constants       map[string]constantInfo
}

// handy interface to loop untill self feeding generation is over
type stopper func() bool

var gopath string

func init() {
	gopath = os.Getenv("GOPATH")
	if gopath == "" {
		panic("gopath not defined")
	}
}

// LoadDoc takes all route, determine source file where handlers are
// and extracts documentation to docInfos.FunctionsDoc[handlerName]
func LoadDoc(routes []*wosk.Route) *Infos {

	var sourceDirs = map[string]bool{}
	var allDone stopper = func() bool {
		for _, treated := range sourceDirs {
			if !treated {
				return false
			}
		}
		return true
	}

	docInfos := new(Infos)
	docInfos.FunctionsDoc = map[string]string{}
	docInfos.TypesDoc = map[string]string{}
	docInfos.Constants = map[string]constantInfo{}
	docInfos.StructFieldsDoc = map[string]map[string]string{}

	for _, ruut := range routes {
		if !ruut.GetHandler().IsValid() || ruut.GetHandler().IsNil() {
			fmt.Fprintf(os.Stderr, "This handler violates Disney's correction policy (it's Fucking Goofy) -> %+v\nSKIPPING ROUTE!!\n\n", ruut)
			continue
		}
		ptr := ruut.GetHandler().Pointer()
		filename, _ := runtime.FuncForPC(ptr).FileLine(ptr)
		dir, _ := filepath.Split(filename)
		sourceDirs[dir] = false
	}

	for !allDone() {
		for dir := range sourceDirs {
			if sourceDirs[dir] {
				continue
			}
			treatSourcedir(dir, docInfos, sourceDirs)
			sourceDirs[dir] = true
		}
	}

	return docInfos
}

func treatSourcedir(dir string, docInfos *Infos, sourceDirs map[string]bool) {

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	for _, pkg := range pkgs {
		importpath := dir + "/" + pkg.Name
		d := doc.New(pkg, importpath, doc.AllDecls)
		for _, imp := range d.Imports {
			if !sourceDirs[imp] {
				if _, err := os.Stat(gopath + "/src/" + imp); err == nil {
					sourceDirs[imp] = true
					// if you want to load all docs from all packages, you can uncomment next line
					// then maybe we would need to prefix everything with pkg
					sourceDirs[gopath+"/src/"+imp] = false
				}
			}
		}

		for _, astFunc := range d.Funcs {
			docInfos.FunctionsDoc[pkg.Name+"."+astFunc.Name] = strings.Replace(astFunc.Doc, "\n", ". ", -1)
		}
		for _, astTypes := range d.Types {

			a := astTypes.Decl
			for _, tspec := range a.Specs {
				switch tspec.(type) {
				case *ast.TypeSpec:

					switch tspec.(*ast.TypeSpec).Type.(type) {
					case *ast.StructType:
						ss := tspec.(*ast.TypeSpec).Type.(*ast.StructType)
						for _, f := range ss.Fields.List {
							name := ""
							for _, i := range f.Names {
								name += i.Name
							}
							if docInfos.StructFieldsDoc[astTypes.Name] == nil {
								docInfos.StructFieldsDoc[astTypes.Name] = map[string]string{}
							}
							docInfos.StructFieldsDoc[astTypes.Name][name] = f.Doc.Text()
							// could have picked f.Comment.Text() too
						}
					default:
						continue
					}
				default:
					continue
				}
			}

			docInfos.TypesDoc[astTypes.Name] = strings.Replace(astTypes.Doc, "\n", ". ", -1)
			if len(astTypes.Consts) > 0 {
				var c1 constantInfo
				pkg := strings.Split(d.ImportPath, "/")
				p := strings.Replace(d.ImportPath, "/"+pkg[len(pkg)-1], "", 1)
				c1.Imprt = strings.Replace(p, gopath, "", 0)
				for _, v := range astTypes.Consts {
					c1.ListC = append(c1.ListC, v.Names...)
				}
				docInfos.Constants[d.Name+"."+astTypes.Name] = c1
			}

		}
	}

}
