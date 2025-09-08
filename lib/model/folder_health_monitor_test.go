// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"context"
	"iter"
	"net"
	"testing"
	"time"

	"github.com/syncthing/syncthing/internal/db"
	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/connections"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/stats"
	"github.com/syncthing/syncthing/lib/ur/contract"
	"github.com/syncthing/syncthing/lib/versioner"
)

func TestFolderHealthMonitor(t *testing.T) {
	// Create a test configuration
	cfg := config.Configuration{
		Folders: []config.FolderConfiguration{
			{
				ID:   "test-folder",
				Path: "/tmp/test-folder",
				Devices: []config.FolderDeviceConfiguration{
					{DeviceID: protocol.LocalDeviceID},
				},
			},
		},
	}

	// Create a mock config wrapper
	wrapper := createMockConfigWrapper(cfg)

	// Create a mock model
	mockModel := &mockModel{}

	// Create a mock event logger
	evLogger := events.NewLogger()

	// Create the folder health monitor
	fhm := NewFolderHealthMonitor(wrapper, mockModel, evLogger)

	// Start the folder health monitor service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the service in a goroutine
	go func() {
		_ = fhm.Serve(ctx)
	}()

	// Give the service some time to start
	time.Sleep(100 * time.Millisecond)

	// Verify that the folder health monitor was initialized correctly
	if fhm == nil {
		t.Fatal("FolderHealthMonitor should not be nil")
	}

	// Verify that the folder tickers map is initialized
	if fhm.folderTickers == nil {
		t.Fatal("folderTickers should not be nil")
	}

	// Verify that the last health status map is initialized
	if fhm.lastHealthStatus == nil {
		t.Fatal("lastHealthStatus should not be nil")
	}

	// Verify that the performance stats map is initialized
	if fhm.performanceStats == nil {
		t.Fatal("performanceStats should not be nil")
	}

	// Test the GetFolderHealthStatus method
	_, exists := fhm.GetFolderHealthStatus("test-folder")
	if exists {
		t.Error("Expected folder health status to not exist for test-folder")
	}

	// Test the GetAllFoldersHealthStatus method
	allStatus := fhm.GetAllFoldersHealthStatus()
	if len(allStatus) != 0 {
		t.Errorf("Expected empty health status map, got %d entries", len(allStatus))
	}

	// Test the GetFolderPerformanceStats method
	_, exists = fhm.GetFolderPerformanceStats("test-folder")
	if exists {
		t.Error("Expected folder performance stats to not exist for test-folder")
	}

	// Test the GetAllFoldersPerformanceStats method
	allPerfStats := fhm.GetAllFoldersPerformanceStats()
	if len(allPerfStats) != 0 {
		t.Errorf("Expected empty performance stats map, got %d entries", len(allPerfStats))
	}
}

// mockModel implements the Model interface for testing
type mockModel struct{}

func (m *mockModel) Serve(ctx context.Context) error {
	return nil
}

func (m *mockModel) String() string {
	return "mockModel"
}

func (m *mockModel) AddConnection(conn protocol.Connection, hello protocol.Hello) {
	// No-op for testing
}

func (m *mockModel) OnHello(deviceID protocol.DeviceID, addr net.Addr, hello protocol.Hello) error {
	// No-op for testing
	return nil
}

func (m *mockModel) DeviceStatistics() (map[protocol.DeviceID]stats.DeviceStatistics, error) {
	// No-op for testing
	return nil, nil
}

func (m *mockModel) SetConnectionsService(service connections.Service) {
	// No-op for testing
}

func (m *mockModel) Index(conn protocol.Connection, idx *protocol.Index) error {
	// No-op for testing
	return nil
}

func (m *mockModel) IndexUpdate(conn protocol.Connection, idxUp *protocol.IndexUpdate) error {
	// No-op for testing
	return nil
}

func (m *mockModel) Request(conn protocol.Connection, req *protocol.Request) (protocol.RequestResponse, error) {
	// No-op for testing
	return nil, nil
}

func (m *mockModel) ClusterConfig(conn protocol.Connection, config *protocol.ClusterConfig) error {
	// No-op for testing
	return nil
}

func (m *mockModel) Closed(conn protocol.Connection, err error) {
	// No-op for testing
}

func (m *mockModel) DownloadProgress(conn protocol.Connection, p *protocol.DownloadProgress) error {
	// No-op for testing
	return nil
}

func (m *mockModel) AllGlobalFiles(folder string) (iter.Seq[db.FileMetadata], func() error) {
	// No-op for testing
	return func(yield func(db.FileMetadata) bool) {}, nil
}

func (m *mockModel) Availability(folder string, file protocol.FileInfo, block protocol.BlockInfo) ([]Availability, error) {
	// No-op for testing
	return nil, nil
}

func (m *mockModel) ResetFolder(folder string) error {
	// No-op for testing
	return nil
}

func (m *mockModel) DelayScan(folder string, next time.Duration) {
	// No-op for testing
}

func (m *mockModel) ScanFolder(folder string) error {
	// No-op for testing
	return nil
}

func (m *mockModel) ScanFolders() map[string]error {
	// No-op for testing
	return nil
}

func (m *mockModel) ScanFolderSubdirs(folder string, subs []string) error {
	// No-op for testing
	return nil
}

func (m *mockModel) State(folder string) (string, time.Time, error) {
	// No-op for testing
	return "", time.Time{}, nil
}

func (m *mockModel) FolderErrors(folder string) ([]FileError, error) {
	// No-op for testing
	return nil, nil
}

func (m *mockModel) WatchError(folder string) error {
	// No-op for testing
	return nil
}

func (m *mockModel) Override(folder string) {
	// No-op for testing
}

func (m *mockModel) Revert(folder string) {
	// No-op for testing
}

func (m *mockModel) BringToFront(folder, file string) {
	// No-op for testing
}

func (m *mockModel) LoadIgnores(folder string) ([]string, []string, error) {
	// No-op for testing
	return nil, nil, nil
}

func (m *mockModel) CurrentIgnores(folder string) ([]string, []string, error) {
	// No-op for testing
	return nil, nil, nil
}

func (m *mockModel) SetIgnores(folder string, content []string) error {
	// No-op for testing
	return nil
}

func (m *mockModel) GetFolderVersions(folder string) (map[string][]versioner.FileVersion, error) {
	// No-op for testing
	return nil, nil
}

func (m *mockModel) RestoreFolderVersions(folder string, versions map[string]time.Time) (map[string]error, error) {
	// No-op for testing
	return nil, nil
}

func (m *mockModel) LocalFiles(folder string, device protocol.DeviceID) (iter.Seq[protocol.FileInfo], func() error) {
	// No-op for testing
	return func(yield func(protocol.FileInfo) bool) {}, nil
}

func (m *mockModel) LocalFilesSequenced(folder string, device protocol.DeviceID, startSet int64) (iter.Seq[protocol.FileInfo], func() error) {
	// No-op for testing
	return func(yield func(protocol.FileInfo) bool) {}, nil
}

func (m *mockModel) LocalSize(folder string, device protocol.DeviceID) (db.Counts, error) {
	// No-op for testing
	return db.Counts{}, nil
}

func (m *mockModel) GlobalSize(folder string) (db.Counts, error) {
	// No-op for testing
	return db.Counts{}, nil
}

func (m *mockModel) NeedSize(folder string, device protocol.DeviceID) (db.Counts, error) {
	// No-op for testing
	return db.Counts{}, nil
}

func (m *mockModel) ReceiveOnlySize(folder string) (db.Counts, error) {
	// No-op for testing
	return db.Counts{}, nil
}

func (m *mockModel) Sequence(folder string, device protocol.DeviceID) (int64, error) {
	// No-op for testing
	return 0, nil
}

func (m *mockModel) RemoteSequences(folder string) (map[protocol.DeviceID]int64, error) {
	// No-op for testing
	return nil, nil
}

func (m *mockModel) NeedFolderFiles(folder string, page, perpage int) ([]protocol.FileInfo, []protocol.FileInfo, []protocol.FileInfo, error) {
	// No-op for testing
	return nil, nil, nil, nil
}

func (m *mockModel) RemoteNeedFolderFiles(folder string, device protocol.DeviceID, page, perpage int) ([]protocol.FileInfo, error) {
	// No-op for testing
	return nil, nil
}

func (m *mockModel) LocalChangedFolderFiles(folder string, page, perpage int) ([]protocol.FileInfo, error) {
	// No-op for testing
	return nil, nil
}

func (m *mockModel) FolderProgressBytesCompleted(folder string) int64 {
	// No-op for testing
	return 0
}

func (m *mockModel) CurrentFolderFile(folder string, file string) (protocol.FileInfo, bool, error) {
	// No-op for testing
	return protocol.FileInfo{}, false, nil
}

func (m *mockModel) CurrentGlobalFile(folder string, file string) (protocol.FileInfo, bool, error) {
	// No-op for testing
	return protocol.FileInfo{}, false, nil
}

func (m *mockModel) Completion(device protocol.DeviceID, folder string) (FolderCompletion, error) {
	// No-op for testing
	return FolderCompletion{}, nil
}

func (m *mockModel) ConnectionStats() map[string]interface{} {
	// No-op for testing
	return nil
}

func (m *mockModel) FolderStatistics() (map[string]stats.FolderStatistics, error) {
	// No-op for testing
	return nil, nil
}

func (m *mockModel) UsageReportingStats(report *contract.Report, version int, preview bool) {
	// No-op for testing
}

func (m *mockModel) ConnectedTo(remoteID protocol.DeviceID) bool {
	// No-op for testing
	return false
}

func (m *mockModel) PendingDevices() (map[protocol.DeviceID]db.ObservedDevice, error) {
	// No-op for testing
	return nil, nil
}

func (m *mockModel) PendingFolders(device protocol.DeviceID) (map[string]db.PendingFolder, error) {
	// No-op for testing
	return nil, nil
}

func (m *mockModel) DismissPendingDevice(device protocol.DeviceID) error {
	// No-op for testing
	return nil
}

func (m *mockModel) DismissPendingFolder(device protocol.DeviceID, folder string) error {
	// No-op for testing
	return nil
}

func (m *mockModel) GlobalDirectoryTree(folder, prefix string, levels int, dirsOnly bool) ([]*TreeEntry, error) {
	// No-op for testing
	return nil, nil
}

func (m *mockModel) RequestGlobal(ctx context.Context, deviceID protocol.DeviceID, folder, name string, blockNo int, offset int64, size int, hash []byte, fromTemporary bool) ([]byte, error) {
	// No-op for testing
	return nil, nil
}

// createMockConfigWrapper creates a mock config wrapper for testing
func createMockConfigWrapper(cfg config.Configuration) config.Wrapper {
	// This is a simplified mock implementation
	// In a real test, you would use a proper mock or test implementation
	return &mockConfigWrapper{cfg: cfg}
}

type mockConfigWrapper struct {
	cfg config.Configuration
}

func (w *mockConfigWrapper) ConfigPath() string {
	// Return empty string for testing purposes
	return ""
}

func (w *mockConfigWrapper) Serve(ctx context.Context) error {
	return nil
}

func (w *mockConfigWrapper) String() string {
	return "mockConfigWrapper"
}

func (w *mockConfigWrapper) Config() config.Configuration {
	return w.cfg
}

func (w *mockConfigWrapper) RawCopy() config.Configuration {
	return w.cfg
}

func (w *mockConfigWrapper) Subscribe(c config.Committer) config.Configuration {
	// Return empty configuration for testing purposes
	return config.Configuration{}
}

func (w *mockConfigWrapper) Unsubscribe(c config.Committer) {
	// No-op for testing
}

func (w *mockConfigWrapper) Modify(config.ModifyFunction) (config.Waiter, error) {
	// No-op for testing
	return nil, nil
}

func (w *mockConfigWrapper) Folder(id string) (config.FolderConfiguration, bool) {
	for _, folder := range w.cfg.Folders {
		if folder.ID == id {
			return folder, true
		}
	}
	return config.FolderConfiguration{}, false
}

func (w *mockConfigWrapper) FolderList() []config.FolderConfiguration {
	return w.cfg.Folders
}

func (w *mockConfigWrapper) Folders() map[string]config.FolderConfiguration {
	// Return empty map for testing purposes
	folderMap := make(map[string]config.FolderConfiguration)
	for _, folder := range w.cfg.Folders {
		folderMap[folder.ID] = folder
	}
	return folderMap
}

func (w *mockConfigWrapper) Device(id protocol.DeviceID) (config.DeviceConfiguration, bool) {
	// No-op for testing
	return config.DeviceConfiguration{}, false
}

func (w *mockConfigWrapper) DeviceList() []config.DeviceConfiguration {
	// No-op for testing
	return nil
}

func (w *mockConfigWrapper) DefaultDevice() config.DeviceConfiguration {
	// No-op for testing
	return config.DeviceConfiguration{}
}

func (w *mockConfigWrapper) DefaultFolder() config.FolderConfiguration {
	// No-op for testing
	return config.FolderConfiguration{}
}

func (w *mockConfigWrapper) DefaultIgnores() config.Ignores {
	// No-op for testing
	return config.Ignores{}
}

func (w *mockConfigWrapper) FolderPasswords(device protocol.DeviceID) map[string]string {
	// No-op for testing
	return make(map[string]string)
}

func (w *mockConfigWrapper) Devices() map[protocol.DeviceID]config.DeviceConfiguration {
	// Return empty map for testing purposes
	return make(map[protocol.DeviceID]config.DeviceConfiguration)
}

func (w *mockConfigWrapper) GUI() config.GUIConfiguration {
	// Return empty GUI configuration for testing purposes
	return config.GUIConfiguration{}
}

func (w *mockConfigWrapper) IgnoredDevice(id protocol.DeviceID) bool {
	// No-op for testing
	return false
}

func (w *mockConfigWrapper) IgnoredDevices() []config.ObservedDevice {
	// Return empty slice for testing purposes
	return []config.ObservedDevice{}
}

func (w *mockConfigWrapper) IgnoredFolder(device protocol.DeviceID, folder string) bool {
	// No-op for testing
	return false
}

func (w *mockConfigWrapper) LDAP() config.LDAPConfiguration {
	// Return empty LDAP configuration for testing purposes
	return config.LDAPConfiguration{}
}

func (w *mockConfigWrapper) MyID() protocol.DeviceID {
	// Return empty device ID for testing purposes
	return protocol.EmptyDeviceID
}

func (w *mockConfigWrapper) Options() config.OptionsConfiguration {
	// Return empty options configuration for testing purposes
	return config.OptionsConfiguration{}
}

func (w *mockConfigWrapper) RemoveDevice(id protocol.DeviceID) (config.Waiter, error) {
	// No-op for testing
	return nil, nil
}

func (w *mockConfigWrapper) RemoveFolder(id string) (config.Waiter, error) {
	// No-op for testing
	return nil, nil
}

func (w *mockConfigWrapper) RequiresRestart() bool {
	// Return false for testing purposes
	return false
}

func (w *mockConfigWrapper) Save() error {
	// No-op for testing
	return nil
}
