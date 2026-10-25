package ts

import (
	"fmt"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// Transpiler provides in-memory TypeScript transpilation and bundling via pure-Go esbuild.
type Transpiler struct{}

// NewTranspiler creates a new in-memory Transpiler instance.
func NewTranspiler() *Transpiler {
	return &Transpiler{}
}

// Transpile converts raw TypeScript source code into standard JavaScript in-memory (~2ms).
func (t *Transpiler) Transpile(source string) (string, error) {
	result := api.Transform(source, api.TransformOptions{
		Loader:            api.LoaderTS,
		Target:            api.ES2020,
		Format:            api.FormatCommonJS,
		Sourcemap:         api.SourceMapInline,
		MinifyWhitespace:  false,
		MinifyIdentifiers: false,
		MinifySyntax:      false,
	})

	if len(result.Errors) > 0 {
		var errMsgs []string
		for _, e := range result.Errors {
			errMsgs = append(errMsgs, fmt.Sprintf("%s:%d:%d: %s", e.Location.File, e.Location.Line, e.Location.Column, e.Text))
		}
		return "", fmt.Errorf("typescript transpilation failed:\n%s", strings.Join(errMsgs, "\n"))
	}

	return string(result.Code), nil
}

// Bundle resolves imports and bundles a multi-file TypeScript plugin into a single runnable JS script.
func (t *Transpiler) Bundle(entryFilePath string) (string, error) {
	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entryFilePath},
		Bundle:      true,
		Write:       false,
		Target:      api.ES2020,
		Format:      api.FormatIIFE,
		Platform:    api.PlatformNeutral,
		Sourcemap:   api.SourceMapInline,
	})

	if len(result.Errors) > 0 {
		var errMsgs []string
		for _, e := range result.Errors {
			if e.Location != nil {
				errMsgs = append(errMsgs, fmt.Sprintf("%s:%d:%d: %s", e.Location.File, e.Location.Line, e.Location.Column, e.Text))
			} else {
				errMsgs = append(errMsgs, e.Text)
			}
		}
		return "", fmt.Errorf("typescript bundle failed:\n%s", strings.Join(errMsgs, "\n"))
	}

	if len(result.OutputFiles) == 0 {
		return "", fmt.Errorf("bundle produced no output files")
	}

	return string(result.OutputFiles[0].Contents), nil
}
