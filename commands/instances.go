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
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/arduino/arduino-cli/commands/cmderrors"
	"github.com/arduino/arduino-cli/commands/internal/instances"
	"github.com/arduino/arduino-cli/internal/arduino/cores"
	"github.com/arduino/arduino-cli/internal/arduino/cores/packageindex"
	"github.com/arduino/arduino-cli/internal/arduino/cores/packagemanager"
	"github.com/arduino/arduino-cli/internal/arduino/globals"
	"github.com/arduino/arduino-cli/internal/arduino/libraries"
	"github.com/arduino/arduino-cli/internal/arduino/libraries/librariesindex"
	"github.com/arduino/arduino-cli/internal/arduino/libraries/librariesmanager"
	"github.com/arduino/arduino-cli/internal/arduino/resources"
	"github.com/arduino/arduino-cli/internal/arduino/sketch"
	"github.com/arduino/arduino-cli/internal/arduino/utils"
	"github.com/arduino/arduino-cli/internal/cli/configuration"
	"github.com/arduino/arduino-cli/internal/i18n"
	rpc "github.com/arduino/arduino-cli/rpc/cc/arduino/cli/commands/v1"
	paths "github.com/arduino/go-paths-helper"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func installTool(pm *packagemanager.PackageManager, tool *cores.ToolRelease, downloadCB rpc.DownloadProgressCB, taskCB rpc.TaskProgressCB) error {
	pme, release := pm.NewExplorer()
	defer release()
	taskCB(&rpc.TaskProgress{Name: tr("Downloading missing tool %s", tool)})
	if err := pme.DownloadToolRelease(tool, nil, downloadCB); err != nil {
		return fmt.Errorf(tr("downloading %[1]s tool: %[2]s"), tool, err)
	}
	taskCB(&rpc.TaskProgress{Completed: true})
	if err := pme.InstallTool(tool, taskCB, true); err != nil {
		return fmt.Errorf(tr("installing %[1]s tool: %[2]s"), tool, err)
	}
	return nil
}

// Create a new Instance ready to be initialized, supporting directories are also created.
func (s *arduinoCoreServerImpl) Create(ctx context.Context, req *rpc.CreateRequest) (*rpc.CreateResponse, error) {
	var userAgent []string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		userAgent = md.Get("user-agent")
	}
	if len(userAgent) == 0 {
		userAgent = []string{"gRPCClientUnknown/0.0.0"}
	}

	// Setup downloads directory
	downloadsDir := configuration.DownloadsDir(configuration.Settings)
	if downloadsDir.NotExist() {
		err := downloadsDir.MkdirAll()
		if err != nil {
			return nil, &cmderrors.PermissionDeniedError{Message: tr("Failed to create downloads directory"), Cause: err}
		}
	}

	// Setup data directory
	dataDir := configuration.DataDir(configuration.Settings)
	packagesDir := configuration.PackagesDir(configuration.Settings)
	if packagesDir.NotExist() {
		err := packagesDir.MkdirAll()
		if err != nil {
			return nil, &cmderrors.PermissionDeniedError{Message: tr("Failed to create data directory"), Cause: err}
		}
	}

	inst, err := instances.Create(dataDir, packagesDir, downloadsDir, userAgent...)
	if err != nil {
		return nil, err
	}
	return &rpc.CreateResponse{Instance: inst}, nil
}

// InitStreamResponseToCallbackFunction returns a gRPC stream to be used in Init that sends
// all responses to the callback function.
func InitStreamResponseToCallbackFunction(ctx context.Context, cb func(r *rpc.InitResponse) error) rpc.ArduinoCoreService_InitServer {
	return streamResponseToCallback(ctx, cb)
}

// Init loads installed libraries and Platforms in CoreInstance with specified ID,
// a gRPC status error is returned if the CoreInstance doesn't exist.
// All responses are sent through responseCallback, can be nil to ignore all responses.
// Failures don't stop the loading process, in case of loading failure the Platform or library
// is simply skipped and an error gRPC status is sent to responseCallback.
func (s *arduinoCoreServerImpl) Init(req *rpc.InitRequest, stream rpc.ArduinoCoreService_InitServer) error {
	instance := req.GetInstance()
	if !instances.IsValid(instance) {
		return &cmderrors.InvalidInstanceError{}
	}

	// Setup callback functions
	var responseCallback func(*rpc.InitResponse) error
	if stream != nil {
		syncSend := NewSynchronizedSend(stream.Send)
		responseCallback = syncSend.Send
	} else {
		responseCallback = func(*rpc.InitResponse) error { return nil }
	}
	responseError := func(st *status.Status) {
		responseCallback(&rpc.InitResponse{
			Message: &rpc.InitResponse_Error{
				Error: st.Proto(),
			},
		})
	}
	taskCallback := func(msg *rpc.TaskProgress) {
		responseCallback(&rpc.InitResponse{
			Message: &rpc.InitResponse_InitProgress{
				InitProgress: &rpc.InitResponse_Progress{
					TaskProgress: msg,
				},
			},
		})
	}
	downloadCallback := func(msg *rpc.DownloadProgress) {
		responseCallback(&rpc.InitResponse{
			Message: &rpc.InitResponse_InitProgress{
				InitProgress: &rpc.InitResponse_Progress{
					DownloadProgress: msg,
				},
			},
		})
	}

	// Try to extract profile if specified
	var profile *sketch.Profile
	if req.GetProfile() != "" {
		sk, err := sketch.New(paths.New(req.GetSketchPath()))
		if err != nil {
			return &cmderrors.InvalidArgumentError{Cause: err}
		}
		p, err := sk.GetProfile(req.GetProfile())
		if err != nil {
			return err
		}
		profile = p
		responseCallback(&rpc.InitResponse{
			Message: &rpc.InitResponse_Profile{
				Profile: profile.ToRpc(),
			},
		})
	}

	// Perform first-update of indexes if needed
	defaultIndexURL, _ := utils.URLParse(globals.DefaultIndexURL)
	allPackageIndexUrls := []*url.URL{defaultIndexURL}
	if profile == nil {
		for _, u := range configuration.Settings.GetStringSlice("board_manager.additional_urls") {
			URL, err := utils.URLParse(u)
			if err != nil {
				e := &cmderrors.InitFailedError{
					Code:   codes.InvalidArgument,
					Cause:  fmt.Errorf(tr("Invalid additional URL: %v", err)),
					Reason: rpc.FailedInstanceInitReason_FAILED_INSTANCE_INIT_REASON_INVALID_INDEX_URL,
				}
				responseError(e.GRPCStatus())
				continue
			}
			allPackageIndexUrls = append(allPackageIndexUrls, URL)
		}
	}
	if err := firstUpdate(context.Background(), req.GetInstance(), downloadCallback, allPackageIndexUrls); err != nil {
		e := &cmderrors.InitFailedError{
			Code:   codes.InvalidArgument,
			Cause:  err,
			Reason: rpc.FailedInstanceInitReason_FAILED_INSTANCE_INIT_REASON_INDEX_DOWNLOAD_ERROR,
		}
		responseError(e.GRPCStatus())
	}

	{
		// We need to rebuild the PackageManager currently in use by this instance
		// in case this is not the first Init on this instances, that might happen
		// after reinitializing an instance after installing or uninstalling a core.
		// If this is not done the information of the uninstall core is kept in memory,
		// even if it should not.
		pm, err := instances.GetPackageManager(instance)
		if err != nil {
			return err
		}
		pmb, commitPackageManager := pm.NewBuilder()

		// Load packages index
		for _, URL := range allPackageIndexUrls {
			if URL.Scheme == "file" {
				_, err := pmb.LoadPackageIndexFromFile(paths.New(URL.Path))
				if err != nil {
					e := &cmderrors.InitFailedError{
						Code:   codes.FailedPrecondition,
						Cause:  fmt.Errorf(tr("Loading index file: %v", err)),
						Reason: rpc.FailedInstanceInitReason_FAILED_INSTANCE_INIT_REASON_INDEX_LOAD_ERROR,
					}
					responseError(e.GRPCStatus())
				}
				continue
			}

			if err := pmb.LoadPackageIndex(URL); err != nil {
				e := &cmderrors.InitFailedError{
					Code:   codes.FailedPrecondition,
					Cause:  fmt.Errorf(tr("Loading index file: %v", err)),
					Reason: rpc.FailedInstanceInitReason_FAILED_INSTANCE_INIT_REASON_INDEX_LOAD_ERROR,
				}
				responseError(e.GRPCStatus())
			}
		}

		loadBuiltinTools := func() []error {
			builtinPackage := pmb.GetOrCreatePackage("builtin")
			return pmb.LoadToolsFromPackageDir(builtinPackage, pmb.PackagesDir.Join("builtin", "tools"))
		}

		// Load Platforms
		if profile == nil {
			for _, err := range pmb.LoadHardware() {
				s := &cmderrors.PlatformLoadingError{Cause: err}
				responseError(s.GRPCStatus())
			}
		} else {
			// Load platforms from profile
			errs := pmb.LoadHardwareForProfile(
				profile, true, downloadCallback, taskCallback,
			)
			for _, err := range errs {
				s := &cmderrors.PlatformLoadingError{Cause: err}
				responseError(s.GRPCStatus())
			}

			// Load "builtin" tools
			_ = loadBuiltinTools()
		}

		// We load hardware before verifying builtin tools are installed
		// otherwise we wouldn't find them and reinstall them each time
		// and they would never get reloaded.

		builtinToolsToInstall := []*cores.ToolRelease{}
		for name, tool := range pmb.GetOrCreatePackage("builtin").Tools {
			latest := tool.LatestRelease()
			if latest == nil {
				e := &cmderrors.InitFailedError{
					Code:   codes.Internal,
					Cause:  fmt.Errorf(tr("can't find latest release of tool %s", name)),
					Reason: rpc.FailedInstanceInitReason_FAILED_INSTANCE_INIT_REASON_TOOL_LOAD_ERROR,
				}
				responseError(e.GRPCStatus())
			} else if !latest.IsInstalled() {
				builtinToolsToInstall = append(builtinToolsToInstall, latest)
			}
		}

		// Install builtin tools if necessary
		if len(builtinToolsToInstall) > 0 {
			for _, toolRelease := range builtinToolsToInstall {
				if err := installTool(pmb.Build(), toolRelease, downloadCallback, taskCallback); err != nil {
					e := &cmderrors.InitFailedError{
						Code:   codes.Internal,
						Cause:  err,
						Reason: rpc.FailedInstanceInitReason_FAILED_INSTANCE_INIT_REASON_TOOL_LOAD_ERROR,
					}
					responseError(e.GRPCStatus())
				}
			}

			// We installed at least one builtin tool after loading hardware
			// so we must reload again otherwise we would never found them.
			for _, err := range loadBuiltinTools() {
				s := &cmderrors.PlatformLoadingError{Cause: err}
				responseError(s.GRPCStatus())
			}
		}

		commitPackageManager()
	}

	pme, release, err := instances.GetPackageManagerExplorer(instance)
	if err != nil {
		return err
	}
	defer release()

	for _, err := range pme.LoadDiscoveries() {
		s := &cmderrors.PlatformLoadingError{Cause: err}
		responseError(s.GRPCStatus())
	}

	// Create library manager and add libraries directories
	lmb := librariesmanager.NewBuilder()

	// Load libraries
	for _, pack := range pme.GetPackages() {
		for _, platform := range pack.Platforms {
			if platformRelease := pme.GetInstalledPlatformRelease(platform); platformRelease != nil {
				lmb.AddLibrariesDir(librariesmanager.LibrariesDir{
					PlatformRelease: platformRelease,
					Path:            platformRelease.GetLibrariesDir(),
					Location:        libraries.PlatformBuiltIn,
				})
			}
		}
	}

	indexFileName, err := globals.LibrariesIndexResource.IndexFileName()
	if err != nil {
		// should never happen
		panic("failed getting libraries index file name: " + err.Error())
	}
	indexFile := pme.IndexDir.Join(indexFileName)

	logrus.WithField("index", indexFile).Info("Loading libraries index file")
	li, err := librariesindex.LoadIndex(indexFile)
	if err != nil {
		s := status.Newf(codes.FailedPrecondition, tr("Loading index file: %v"), err)
		responseError(s)
		li = librariesindex.EmptyIndex
	}
	instances.SetLibrariesIndex(instance, li)

	if profile == nil {
		// Add directories of libraries bundled with IDE
		if bundledLibsDir := configuration.IDEBuiltinLibrariesDir(configuration.Settings); bundledLibsDir != nil {
			lmb.AddLibrariesDir(librariesmanager.LibrariesDir{
				Path:     bundledLibsDir,
				Location: libraries.IDEBuiltIn,
			})
		}

		// Add libraries directory from config file
		lmb.AddLibrariesDir(librariesmanager.LibrariesDir{
			Path:     configuration.LibrariesDir(configuration.Settings),
			Location: libraries.User,
		})
	} else {
		// Load libraries required for profile
		for _, libraryRef := range profile.Libraries {
			uid := libraryRef.InternalUniqueIdentifier()
			libRoot := configuration.ProfilesCacheDir(configuration.Settings).Join(uid)
			libDir := libRoot.Join(libraryRef.Library)

			if !libDir.IsDir() {
				// Download library
				taskCallback(&rpc.TaskProgress{Name: tr("Downloading library %s", libraryRef)})
				libRelease, err := li.FindRelease(libraryRef.Library, libraryRef.Version)
				if err != nil {
					taskCallback(&rpc.TaskProgress{Name: tr("Library %s not found", libraryRef)})
					err := &cmderrors.LibraryNotFoundError{Library: libraryRef.Library}
					responseError(err.GRPCStatus())
					continue
				}
				if err := libRelease.Resource.Download(pme.DownloadDir, nil, libRelease.String(), downloadCallback, ""); err != nil {
					taskCallback(&rpc.TaskProgress{Name: tr("Error downloading library %s", libraryRef)})
					e := &cmderrors.FailedLibraryInstallError{Cause: err}
					responseError(e.GRPCStatus())
					continue
				}
				taskCallback(&rpc.TaskProgress{Completed: true})

				// Install library
				taskCallback(&rpc.TaskProgress{Name: tr("Installing library %s", libraryRef)})
				if err := libRelease.Resource.Install(pme.DownloadDir, libRoot, libDir); err != nil {
					taskCallback(&rpc.TaskProgress{Name: tr("Error installing library %s", libraryRef)})
					e := &cmderrors.FailedLibraryInstallError{Cause: err}
					responseError(e.GRPCStatus())
					continue
				}
				taskCallback(&rpc.TaskProgress{Completed: true})
			}

			lmb.AddLibrariesDir(librariesmanager.LibrariesDir{
				Path:     libRoot,
				Location: libraries.User,
			})
		}
	}

	lm, libsLoadingWarnings := lmb.Build()
	_ = instances.SetLibraryManager(instance, lm) // should never fail
	for _, status := range libsLoadingWarnings {
		logrus.WithError(status.Err()).Warnf("Error loading library")
		// TODO: report as warning: responseError(err)
	}

	// Refreshes the locale used, this will change the
	// language of the CLI if the locale is different
	// after started.
	i18n.Init(configuration.Settings.GetString("locale"))

	return nil
}

// Destroy deletes an instance.
func (s *arduinoCoreServerImpl) Destroy(ctx context.Context, req *rpc.DestroyRequest) (*rpc.DestroyResponse, error) {
	if ok := instances.Delete(req.GetInstance()); !ok {
		return nil, &cmderrors.InvalidInstanceError{}
	}
	return &rpc.DestroyResponse{}, nil
}

// UpdateLibrariesIndex updates the library_index.json
func UpdateLibrariesIndex(ctx context.Context, req *rpc.UpdateLibrariesIndexRequest, downloadCB rpc.DownloadProgressCB) (*rpc.UpdateLibrariesIndexResponse_Result, error) {
	logrus.Info("Updating libraries index")

	pme, release, err := instances.GetPackageManagerExplorer(req.GetInstance())
	if err != nil {
		return nil, err
	}
	indexDir := pme.IndexDir
	release()

	index := globals.LibrariesIndexResource
	result := func(status rpc.IndexUpdateReport_Status) *rpc.UpdateLibrariesIndexResponse_Result {
		return &rpc.UpdateLibrariesIndexResponse_Result{
			LibrariesIndex: &rpc.IndexUpdateReport{
				IndexUrl: globals.LibrariesIndexResource.URL.String(),
				Status:   status,
			},
		}
	}

	// Create the index directory if it doesn't exist
	if err := indexDir.MkdirAll(); err != nil {
		return result(rpc.IndexUpdateReport_STATUS_FAILED), &cmderrors.PermissionDeniedError{Message: tr("Could not create index directory"), Cause: err}
	}

	// Check if the index file is already up-to-date
	indexFileName, _ := index.IndexFileName()
	if info, err := indexDir.Join(indexFileName).Stat(); err == nil {
		ageSecs := int64(time.Since(info.ModTime()).Seconds())
		if ageSecs < req.GetUpdateIfOlderThanSecs() {
			return result(rpc.IndexUpdateReport_STATUS_ALREADY_UP_TO_DATE), nil
		}
	}

	// Perform index update
	if err := globals.LibrariesIndexResource.Download(indexDir, downloadCB); err != nil {
		return nil, err
	}

	return result(rpc.IndexUpdateReport_STATUS_UPDATED), nil
}

// UpdateIndex FIXMEDOC
func UpdateIndex(ctx context.Context, req *rpc.UpdateIndexRequest, downloadCB rpc.DownloadProgressCB) (*rpc.UpdateIndexResponse_Result, error) {
	if !instances.IsValid(req.GetInstance()) {
		return nil, &cmderrors.InvalidInstanceError{}
	}

	report := func(indexURL *url.URL, status rpc.IndexUpdateReport_Status) *rpc.IndexUpdateReport {
		return &rpc.IndexUpdateReport{
			IndexUrl: indexURL.String(),
			Status:   status,
		}
	}

	indexpath := configuration.DataDir(configuration.Settings)

	urls := []string{globals.DefaultIndexURL}
	if !req.GetIgnoreCustomPackageIndexes() {
		urls = append(urls, configuration.Settings.GetStringSlice("board_manager.additional_urls")...)
	}

	failed := false
	result := &rpc.UpdateIndexResponse_Result{}
	for _, u := range urls {
		URL, err := url.Parse(u)
		if err != nil {
			logrus.Warnf("unable to parse additional URL: %s", u)
			msg := fmt.Sprintf("%s: %v", tr("Unable to parse URL"), err)
			downloadCB.Start(u, tr("Downloading index: %s", u))
			downloadCB.End(false, msg)
			failed = true
			result.UpdatedIndexes = append(result.GetUpdatedIndexes(), report(URL, rpc.IndexUpdateReport_STATUS_FAILED))
			continue
		}

		logrus.WithField("url", URL).Print("Updating index")

		if URL.Scheme == "file" {
			path := paths.New(URL.Path)
			if URL.Scheme == "file" && runtime.GOOS == "windows" && len(URL.Path) > 1 {
				// https://github.com/golang/go/issues/32456
				// Parsed local file URLs on Windows are returned with a leading / so we remove it
				path = paths.New(URL.Path[1:])
			}
			if _, err := packageindex.LoadIndexNoSign(path); err != nil {
				msg := fmt.Sprintf("%s: %v", tr("Invalid package index in %s", path), err)
				downloadCB.Start(u, tr("Downloading index: %s", filepath.Base(URL.Path)))
				downloadCB.End(false, msg)
				failed = true
				result.UpdatedIndexes = append(result.GetUpdatedIndexes(), report(URL, rpc.IndexUpdateReport_STATUS_FAILED))
			} else {
				result.UpdatedIndexes = append(result.GetUpdatedIndexes(), report(URL, rpc.IndexUpdateReport_STATUS_SKIPPED))
			}
			continue
		}

		// Check if the index is up-to-date
		indexResource := resources.IndexResource{URL: URL}
		indexFileName, err := indexResource.IndexFileName()
		if err != nil {
			downloadCB.Start(u, tr("Downloading index: %s", filepath.Base(URL.Path)))
			downloadCB.End(false, tr("Invalid index URL: %s", err))
			failed = true
			result.UpdatedIndexes = append(result.GetUpdatedIndexes(), report(URL, rpc.IndexUpdateReport_STATUS_FAILED))
			continue
		}
		indexFile := indexpath.Join(indexFileName)
		if info, err := indexFile.Stat(); err == nil {
			ageSecs := int64(time.Since(info.ModTime()).Seconds())
			if ageSecs < req.GetUpdateIfOlderThanSecs() {
				result.UpdatedIndexes = append(result.GetUpdatedIndexes(), report(URL, rpc.IndexUpdateReport_STATUS_ALREADY_UP_TO_DATE))
				continue
			}
		}

		if strings.HasSuffix(URL.Host, "arduino.cc") && strings.HasSuffix(URL.Path, ".json") {
			indexResource.SignatureURL, _ = url.Parse(u) // should not fail because we already parsed it
			indexResource.SignatureURL.Path += ".sig"
		}
		if err := indexResource.Download(indexpath, downloadCB); err != nil {
			failed = true
			result.UpdatedIndexes = append(result.GetUpdatedIndexes(), report(URL, rpc.IndexUpdateReport_STATUS_FAILED))
		} else {
			result.UpdatedIndexes = append(result.GetUpdatedIndexes(), report(URL, rpc.IndexUpdateReport_STATUS_UPDATED))
		}
	}

	if failed {
		return result, &cmderrors.FailedDownloadError{Message: tr("Some indexes could not be updated.")}
	}
	return result, nil
}

// firstUpdate downloads libraries and packages indexes if they don't exist.
// This ideally is only executed the first time the CLI is run.
func firstUpdate(ctx context.Context, instance *rpc.Instance, downloadCb func(msg *rpc.DownloadProgress), externalPackageIndexes []*url.URL) error {
	// Gets the data directory to verify if library_index.json and package_index.json exist
	dataDir := configuration.DataDir(configuration.Settings)
	libraryIndex := dataDir.Join("library_index.json")

	if libraryIndex.NotExist() {
		// The library_index.json file doesn't exists, that means the CLI is run for the first time
		// so we proceed with the first update that downloads the file
		req := &rpc.UpdateLibrariesIndexRequest{Instance: instance}
		if _, err := UpdateLibrariesIndex(ctx, req, downloadCb); err != nil {
			return err
		}
	}

	for _, URL := range externalPackageIndexes {
		if URL.Scheme == "file" {
			continue
		}
		packageIndexFileName, err := (&resources.IndexResource{URL: URL}).IndexFileName()
		if err != nil {
			return &cmderrors.FailedDownloadError{
				Message: tr("Error downloading index '%s'", URL),
				Cause:   &cmderrors.InvalidURLError{}}
		}
		packageIndexFile := dataDir.Join(packageIndexFileName)
		if packageIndexFile.NotExist() {
			// The index file doesn't exists, that means the CLI is run for the first time,
			// or the 3rd party package index URL has just been added. Similarly to the
			// library update we download that file and all the other package indexes from
			// additional_urls
			req := &rpc.UpdateIndexRequest{Instance: instance}
			if _, err := UpdateIndex(ctx, req, downloadCb); err != nil {
				return err
			}
			break
		}
	}

	return nil
}
