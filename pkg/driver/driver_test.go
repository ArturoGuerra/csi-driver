// Copyright 2019 Hewlett Packard Enterprise Development LP
package driver

import (
	"os"
	"testing"

	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/kubernetes-csi/csi-test/pkg/sanity"

	"github.com/hpe-storage/common-host-libs/chapi"
	"github.com/hpe-storage/common-host-libs/concurrent"
	"github.com/hpe-storage/common-host-libs/storageprovider"
	"github.com/hpe-storage/common-host-libs/storageprovider/fake"
	"github.com/hpe-storage/common-host-libs/util"
	"github.com/hpe-storage/csi-driver/pkg/flavor"
)

func TestPluginSuite(t *testing.T) {
	socket := "/tmp/csi.sock"
	endpoint := "unix://" + socket
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove unix domain socket file %s, error: %s", socket, err)
	}

	util.OpenLogFile("csi-test.log", 10, 4, 90, true)
	defer util.CloseLogFile()

	// driver := realDriver(t, endpoint)
	// secretsFile := "csi-secrets.yaml"
	driver := fakeDriver(endpoint)
	secretsFile := "fake-csi-secrets.yaml"
	driver.grpc = NewNonBlockingGRPCServer()
	// start node, controller and identity servers on same endpoint for tests
	go driver.grpc.Start(driver.endpoint, driver, driver, driver)
	defer driver.Stop(true)

	config := &sanity.Config{
		StagingPath: "./csi-mnt",
		TargetPath:  "./csi-mnt-stage",
		Address:     endpoint,
		SecretsFile: secretsFile,
	}

	sanity.Test(t, config)
}

// nolint: deadcode
func realDriver(t *testing.T, endpoint string) *Driver {
	driver, err := NewDriver("test-driver", "0.1", endpoint, flavor.Kubernetes, true, false, "", "")

	if err != nil {
		t.Fatal("Failed to initialize driver")
	}

	return driver
}

func fakeDriver(endpoint string) *Driver {
	driver := &Driver{
		name:              "test-driver",
		version:           "0.1",
		endpoint:          endpoint,
		storageProviders:  make(map[string]storageprovider.StorageProvider),
		chapiDriver:       &chapi.FakeDriver{},
		requestCache:      make(map[string]interface{}),
		requestCacheMutex: concurrent.NewMapMutex(),
	}

	driver.storageProviders["fake"] = fake.NewFakeStorageProvider()

	driver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	})

	driver.AddNodeServiceCapabilities([]csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
	})

	driver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER,
	})

	return driver
}
