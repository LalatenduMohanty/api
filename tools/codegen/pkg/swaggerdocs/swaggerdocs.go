package swaggerdocs

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io/ioutil"
	"reflect"

	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

const (
	// headerContent is a header boilerplate applied to swagger generated files.
	headerContent = `// This file contains a collection of methods that can be used from go-restful to
// generate Swagger API documentation for its models. Please read this PR for more
// information on the implementation: https://github.com/emicklei/go-restful/pull/215
//
// TODOs are ignored from the parser (e.g. TODO(andronat):... || TODO:...) if and only if
// they are on one line! For multiple line or blocks that you want to ignore use ---.
// Any context after a --- is ignored.
//
// Those methods can be generated by using hack/update-swagger-docs.sh

// AUTO-GENERATED FUNCTIONS START HERE
`

	// footerContent is a footer boilerplate applied to swagger generated files.
	footerContent = "// AUTO-GENERATED FUNCTIONS END HERE\n"

	// DefaultOutputFileName is the default output file name for the generated swagger docs.
	DefaultOutputFileName = "zz_generated.swagger_doc_generated.go"

	// typesGlob is a glob used to find all types files within a group version package.
	typesGlob = "types*.go"
)

// verifySwaggerDocs reads the existing swagger documentation and verifies that the content
// is up to date.
func verifySwaggerDocs(packageName, filePath string, docsForTypes []kruntime.KubeTypes, enforceComments bool) error {
	// Verify that every field has a doc string.
	buf := bytes.NewBuffer(nil)
	rc, err := kruntime.VerifySwaggerDocsExist(docsForTypes, buf)
	if err != nil {
		return fmt.Errorf("could not verify existing docs: %w", err)
	}
	if rc > 0 {
		if enforceComments {
			return fmt.Errorf("missing swagger docs for the following %d fields:\n%s", rc, buf.String())
		} else {
			klog.Warningf("Existing swagger docs are missing %d entries:\n%s", rc, buf.String())
		}
	}

	// Verify that the existing data matches the generated data.
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading existing swagger docs file: %w", err)
	}

	// This mutates docsForTypes so must run after the VerifySwaggerDocsExist step.
	generatedData, err := generateSwaggerDocs(packageName, docsForTypes)
	if err != nil {
		return fmt.Errorf("error generating swagger docs: %w", err)
	}

	if !reflect.DeepEqual(data, generatedData) {
		return errors.New("swagger docs are out of date, please regenerate the swagger docs")
	}

	return nil
}

// generateSwaggerDocs generates swagger documentation and writes it to the output
// file path given.
func generateSwaggerDocs(packageName string, docsForTypes []kruntime.KubeTypes) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	buf.WriteString(fmt.Sprintf("package %s\n", packageName))
	buf.WriteString(headerContent)

	if err := kruntime.WriteSwaggerDocFunc(docsForTypes, buf); err != nil {
		return nil, fmt.Errorf("error generating swagger docs for types: %w", err)
	}

	buf.WriteString(footerContent)

	// This formats the output as if we ran gofmt over the file.
	formattedOut, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not format output data: %w", err)
	}

	return formattedOut, nil
}
