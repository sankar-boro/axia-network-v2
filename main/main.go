// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/sankar-boro/axia-network-v2/app/runner"
	"github.com/sankar-boro/axia-network-v2/config"
	"github.com/sankar-boro/axia-network-v2/version"
	"github.com/spf13/pflag"
)

func main() {
	fs := config.BuildFlagSet()
	v, err := config.BuildViper(fs, os.Args[1:])

	if errors.Is(err, pflag.ErrHelp) {
		os.Exit(0)
	}

	if err != nil {
		fmt.Printf("couldn't configure flags: %s\n", err)
		os.Exit(1)
	}

	runnerConfig, err := config.GetRunnerConfig(v)
	if err != nil {
		fmt.Printf("couldn't load process config: %s\n", err)
		os.Exit(1)
	}

	if runnerConfig.DisplayVersionAndExit {
		fmt.Print(version.String)
		os.Exit(0)
	}

	nodeConfig, err := config.GetNodeConfig(v, runnerConfig.BuildDir)
	if err != nil {
		fmt.Printf("couldn't load node config: %s\n", err)
		os.Exit(1)
	}

	runner.Run(runnerConfig, nodeConfig)
}
