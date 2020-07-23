package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/go-akka/configuration"
	"github.com/go-akka/configuration/hocon"
)

var (
	inputFile = flag.String("hocon", "", "Hocon fomat input file")
)

func init() {
	flag.Parse()
}

func main() {
	// save the current directory and chdir back to it when done
	if curDir, err := os.Getwd(); err != nil {
		panic(err)
	} else {
		defer os.Chdir(curDir)
	}

	// Split directory and filename
	confDir, confFile := path.Split(*inputFile)
	os.Chdir(confDir)

	// Read all file content
	content, err := ioutil.ReadFile(confFile)
	if err != nil {
		log.Fatalf("unable to open file for read: %v", err)
	}

	// Load HOCON file.
	cfg := configuration.ParseString(string(content), myIncludeCallback).Root()

	// Extract object
	res := visitNode(cfg)

	// Encode as json
	if err := json.NewEncoder(os.Stdout).Encode(res); err != nil {
		log.Fatalf("unable to encode json : %v", err)
	}
}

// -----------------------------------------------------------------------------

func visitNode(node *hocon.HoconValue) interface{} {
	if node.IsArray() {
		nodes := node.GetArray()

		res := make([]interface{}, len(nodes))
		for i, n := range nodes {
			res[i] = visitNode(n)
		}

		return res
	}

	if node.IsObject() {
		obj := node.GetObject()

		res := map[string]interface{}{}
		keys := obj.GetKeys()
		for _, k := range keys {
			res[k] = visitNode(obj.GetKey(k))
		}

		return res
	}

	if node.IsString() {
		return node.GetString()
	}

	if node.IsEmpty() {
		return nil
	}

	return nil
}

func myIncludeCallback(filename string) *hocon.HoconRoot {
	if files, err := filepath.Glob(filename); err != nil {
		panic(err)
	} else if len(files) == 0 {
		log.Printf("[WARN] [%s] does not match any file", filename)
		return hocon.Parse("", nil)
	} else {
		var root = hocon.Parse("", nil)
		for _, f := range files {
			log.Printf("Loading configurations from file [%s]", f)
			if data, err := ioutil.ReadFile(f); err != nil {
				panic(err)
			} else {
				node := hocon.Parse(string(data), myIncludeCallback)
				if node != nil {
					root.Value().GetObject().Merge(node.Value().GetObject())
					// merge substitutions
					subs := make([]*hocon.HoconSubstitution, 0)
					for _, s := range root.Substitutions() {
						subs = append(subs, s)
					}
					for _, s := range node.Substitutions() {
						subs = append(subs, s)
					}
					root = hocon.NewHoconRoot(root.Value(), subs...)
				}
			}
		}
		return root
	}
}
