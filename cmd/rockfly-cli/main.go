package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	CLIVersion = "1.0.0"
)

func printUsage() {
	fmt.Printf("Rockfly Plugin Developer Tooling (CLI v%s)\n\n", CLIVersion)
	fmt.Println("Usage:")
	fmt.Println("  rockfly-cli new-plugin --name <Name> [--lang ts|go] [--dir <Path>]")
	fmt.Println("  rockfly-cli check [--dir <Path>]")
	fmt.Println("  rockfly-cli build [--dir <Path>] [--out <Path>]")
	fmt.Println("  rockfly-cli test [--dir <Path>]")
	fmt.Println("  rockfly-cli version")
	fmt.Println()
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "version", "--version", "-v":
		fmt.Printf("rockfly-cli version %s\n", CLIVersion)

	case "new-plugin":
		fs := flag.NewFlagSet("new-plugin", flag.ExitOnError)
		name := fs.String("name", "", "Name of the plugin")
		lang := fs.String("lang", "ts", "Language runtime: 'ts' (TypeScript) or 'go' (WebAssembly)")
		dir := fs.String("dir", "", "Target directory (defaults to plugin name)")
		_ = fs.Parse(os.Args[2:])

		if *name == "" && len(fs.Args()) > 0 {
			*name = fs.Args()[0]
		}
		if *name == "" {
			fmt.Println("Error: --name is required for new-plugin")
			os.Exit(1)
		}

		if err := ScaffoldPlugin(*name, *lang, *dir); err != nil {
			fmt.Printf("Scaffold error: %v\n", err)
			os.Exit(1)
		}
		target := *dir
		if target == "" {
			target = *name
		}
		fmt.Printf("Successfully created %s plugin %q in %s/\n", *lang, *name, target)

	case "check":
		fs := flag.NewFlagSet("check", flag.ExitOnError)
		dir := fs.String("dir", ".", "Plugin directory to validate")
		_ = fs.Parse(os.Args[2:])

		dur, err := CheckPlugin(*dir)
		if err != nil {
			fmt.Printf("Check FAILED: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Check PASSED for plugin in %s (verified in %v)\n", *dir, dur)

	case "build":
		fs := flag.NewFlagSet("build", flag.ExitOnError)
		dir := fs.String("dir", ".", "Plugin directory to build")
		out := fs.String("out", "", "Output distribution directory")
		_ = fs.Parse(os.Args[2:])

		artifact, dur, err := BuildPlugin(*dir, *out)
		if err != nil {
			fmt.Printf("Build FAILED: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Build SUCCESS: artifact generated at %s in %v\n", artifact, dur)

	case "test":
		fs := flag.NewFlagSet("test", flag.ExitOnError)
		dir := fs.String("dir", ".", "Plugin directory to test")
		_ = fs.Parse(os.Args[2:])

		dur, err := TestPlugin(*dir)
		if err != nil {
			fmt.Printf("Test FAILED: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("All tests PASSED in %v\n", dur)

	default:
		fmt.Printf("Unknown command %q\n\n", command)
		printUsage()
		os.Exit(1)
	}
}
