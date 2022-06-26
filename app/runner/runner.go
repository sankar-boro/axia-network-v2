// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package runner

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	"github.com/sankar-boro/axia/app"
	"github.com/sankar-boro/axia/app/process"
	"github.com/sankar-boro/axia/node"
	"github.com/sankar-boro/axia/vms/rpcchainvm/grpcutils"

	appplugin "github.com/sankar-boro/axia/app/plugin"
)

// Run an Axia node.
// If specified in the config, serves a hashicorp plugin that can be consumed by
// the daemon (see axia/main).
func Run(config Config, nodeConfig node.Config) {
	nodeApp := process.NewApp(nodeConfig) // Create node wrapper
	if config.PluginMode {                // Serve as a plugin
		plugin.Serve(&plugin.ServeConfig{
			HandshakeConfig: appplugin.Handshake,
			Plugins: map[string]plugin.Plugin{
				appplugin.Name: appplugin.New(nodeApp),
			},
			GRPCServer: grpcutils.NewDefaultServer, // A non-nil value here enables gRPC serving for this plugin
			Logger: hclog.New(&hclog.LoggerOptions{
				Level: hclog.Error,
			}),
		})
		return
	}

	fmt.Println(process.Header)

	exitCode := app.Run(nodeApp)
	os.Exit(exitCode)
}
