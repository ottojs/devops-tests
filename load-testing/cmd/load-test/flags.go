package main

import (
	"flag"
	"fmt"
	"os"
)

// Command line options
type Options struct {
	ConfigFile string
	JSONOutput bool
}

// Parses command line flags and returns options
func parseFlags() *Options {
	opts := &Options{}

	flag.StringVar(&opts.ConfigFile, "config", "", "Path to JSON config file")
	flag.BoolVar(&opts.JSONOutput, "json", false, "Output results in JSON format")
	flag.Parse()

	if opts.ConfigFile == "" {
		fmt.Println("Error: No config file provided. Use -config flag to specify a configuration file.")
		flag.Usage()
		os.Exit(exitError)
	}

	return opts
}
