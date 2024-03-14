// This file is part of arduino-cli.
//
// Copyright 2020 ARDUINO SA (http://www.arduino.cc/)
//
// This software is released under the GNU General Public License version 3,
// which covers the main part of arduino-cli.
// The terms of this license can be found at:
// https://www.gnu.org/licenses/gpl-3.0.en.html
//
// You can be released from the requirements of the above licenses by purchasing
// a commercial license. Buying such a license is mandatory if you want to
// modify or otherwise use the software for commercial activities involving the
// Arduino software without disclosing the source code of your own applications.
// To purchase a commercial license, send an email to license@arduino.cc.

package commands

import (
	"context"

	"github.com/arduino/arduino-cli/commands/internal/instances"
	"github.com/arduino/arduino-cli/internal/arduino/cores"
	"github.com/arduino/arduino-cli/internal/arduino/cores/packagemanager"
	rpc "github.com/arduino/arduino-cli/rpc/cc/arduino/cli/commands/v1"
)

// PlatformUpgrade FIXMEDOC
func PlatformUpgrade(ctx context.Context, srv rpc.ArduinoCoreServiceServer, req *rpc.PlatformUpgradeRequest, downloadCB rpc.DownloadProgressCB, taskCB rpc.TaskProgressCB) (*rpc.PlatformUpgradeResponse, error) {
	upgrade := func() (*cores.PlatformRelease, error) {
		pme, release, err := instances.GetPackageManagerExplorer(req.GetInstance())
		if err != nil {
			return nil, err
		}
		defer release()

		// Extract all PlatformReference to platforms that have updates
		ref := &packagemanager.PlatformReference{
			Package:              req.GetPlatformPackage(),
			PlatformArchitecture: req.GetArchitecture(),
		}
		platform, err := pme.DownloadAndInstallPlatformUpgrades(ref, downloadCB, taskCB, req.GetSkipPostInstall(), req.GetSkipPreUninstall())
		if err != nil {
			return platform, err
		}

		return platform, nil
	}

	var rpcPlatform *rpc.Platform
	platformRelease, err := upgrade()
	if platformRelease != nil {
		rpcPlatform = &rpc.Platform{
			Metadata: PlatformToRPCPlatformMetadata(platformRelease.Platform),
			Release:  PlatformReleaseToRPC(platformRelease),
		}
	}
	if err != nil {
		return &rpc.PlatformUpgradeResponse{Platform: rpcPlatform}, err
	}
	if err := srv.Init(&rpc.InitRequest{Instance: req.GetInstance()}, InitStreamResponseToCallbackFunction(ctx, nil)); err != nil {
		return nil, err
	}

	return &rpc.PlatformUpgradeResponse{Platform: rpcPlatform}, nil
}
