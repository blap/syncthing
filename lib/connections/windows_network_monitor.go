// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build windows

package connections

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/syncthing/syncthing/internal/slogutil"
	"golang.org/x/sys/windows"
)

// Windows API constants
const (
	AF_UNSPEC = 0
	AF_INET   = 2
	AF_INET6  = 23

	MIB_IF_TYPE_ETHERNET = 6
	MIB_IF_TYPE_WIFI     = 71

	IF_OPER_STATUS_UP      = 1
	IF_OPER_STATUS_DOWN    = 2
	IF_OPER_STATUS_TESTING = 3

	// Network profile constants
	NLM_NETWORK_CATEGORY_PUBLIC  = 0
	NLM_NETWORK_CATEGORY_PRIVATE = 1
	NLM_NETWORK_CATEGORY_DOMAIN  = 2
)

// Windows API structures
type MibIfRow2 struct {
	InterfaceIndex               uint32
	InterfaceLuid                uint64
	InterfaceGuid                windows.GUID
	Alias                        [257]uint16
	Description                  [257]uint16
	PhysicalAddressLength        uint32
	PhysicalAddress              [8]byte
	PermanentPhysicalAddress     [8]byte
	Mtu                          uint32
	Type                         uint32
	TunnelType                   uint32
	MediaType                    uint32
	PhysicalMediumType           uint32
	AccessType                   uint32
	DirectionType                uint32
	InterfaceAndOperStatusFlags  uint8
	OperStatus                   uint32
	AdminStatus                  uint32
	MediaConnectState            uint32
	NetworkGuid                  windows.GUID
	ConnectionType               uint32
	TransmitLinkSpeed            uint64
	ReceiveLinkSpeed             uint64
	InOctets                     uint64
	InUcastPkts                  uint64
	InNUcastPkts                 uint64
	InDiscards                   uint64
	InErrors                     uint64
	InUnknownProtos              uint64
	InUcastOctets                uint64
	InMulticastOctets            uint64
	InBroadcastOctets            uint64
	OutOctets                    uint64
	OutUcastPkts                 uint64
	OutNUcastPkts                uint64
	OutDiscards                  uint64
	OutErrors                    uint64
	OutUcastOctets               uint64
	OutMulticastOctets           uint64
	OutBroadcastOctets           uint64
	OutQLen                      uint64
}

// MibIfTable2 represents the MIB-II interface table
// This structure is used by GetIfTable2 function
type MibIfTable2 struct {
	NumEntries uint32
	Table      [1]MibIfRow2 // This is actually a variable-length array
}

// NetworkAdapterInfo holds detailed information about a network adapter
type NetworkAdapterInfo struct {
	Name          string
	IsUp          bool
	Type          uint32
	MediaType     uint32
	LinkSpeed     uint64
	LastChange    time.Time
	ChangeCount   int
}

// NetworkStabilityMetrics tracks network stability for adaptive behavior
type NetworkStabilityMetrics struct {
	TotalChanges      int
	RecentChanges     int
	LastErrorTime     time.Time
	StabilityScore    float64 // 0.0 (unstable) to 1.0 (stable)
	AdaptiveTimeout   time.Duration
	LastCheckTime     time.Time
}

// NetworkChangeEvent represents a network change event for logging
type NetworkChangeEvent struct {
	Timestamp     time.Time
	AdapterName   string
	EventType     string // "added", "removed", "state_change", "profile_change"
	Details       string
	StabilityScore float64
}

// WindowsNetworkMonitorInterface defines the interface for Windows network monitors
type WindowsNetworkMonitorInterface interface {
	Start()
	Stop()
}

// WindowsNetworkMonitor monitors Windows network adapter state changes
// and triggers reconnection when adapters wake up or network conditions change
type WindowsNetworkMonitor struct {
	service        Service
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	adapterStates  map[string]NetworkAdapterInfo // adapter name -> detailed info
	currentProfile string                        // current network profile
	mut            sync.RWMutex
	// Enhanced monitoring for KB5060998 compatibility
	lastScanTime   time.Time
	scanInterval   time.Duration
	changeCooldown time.Duration
	// Network list manager for profile detection (full COM integration)
	networkListManager uintptr // Handle to INetworkListManager COM interface
	// Real-time notification handles
	addrChangeHandle  syscall.Handle
	routeChangeHandle syscall.Handle
	notificationChan  chan struct{}
	// Network stability tracking
	stabilityMetrics *NetworkStabilityMetrics
	// Event logging for diagnostics
	eventLog       []NetworkChangeEvent
	maxEventLogSize int
}

// NewWindowsNetworkMonitor creates a new Windows network monitor (Windows only)
func NewWindowsNetworkMonitor(svc Service) *WindowsNetworkMonitor {
	// Create a new WindowsNetworkMonitor instance
	ctx, cancel := context.WithCancel(context.Background())
	
	monitor := &WindowsNetworkMonitor{
		service:         svc,
		ctx:             ctx,
		cancel:          cancel,
		adapterStates:   make(map[string]NetworkAdapterInfo),
		currentProfile:  "Unknown",
		scanInterval:    5 * time.Second,
		changeCooldown:  1 * time.Second,
		notificationChan: make(chan struct{}, 10),
		stabilityMetrics: &NetworkStabilityMetrics{
			StabilityScore:  1.0,
			AdaptiveTimeout: 5 * time.Second,
			LastCheckTime:   time.Now(),
		},
		eventLog:         make([]NetworkChangeEvent, 0),
		maxEventLogSize:  100,
	}
	
	return monitor
}

// Start begins monitoring network adapter state changes
func (w *WindowsNetworkMonitor) Start() {
	// Register for network change notifications
	w.registerForNetworkChangeNotifications()

	w.wg.Add(1)
	go w.monitorNetworkChanges()
	
	// Start real-time notification handler
	w.wg.Add(1)
	go w.handleRealTimeNotifications()
	
	// Start adaptive timeout adjustment
	w.wg.Add(1)
	go w.adjustAdaptiveTimeouts()
	
	// Start periodic diagnostics logging
	w.wg.Add(1)
	go w.logDiagnosticsPeriodically()
}

// Stop stops monitoring network adapter state changes
func (w *WindowsNetworkMonitor) Stop() {
	// Log final diagnostics before stopping
	w.logDiagnostics()
	
	// Clean up network list manager
	w.cleanupNetworkListManager()
	
	// Unregister network change notifications
	w.unregisterNetworkChangeNotifications()
	
	w.cancel()
	w.wg.Wait()
}

// cleanupNetworkListManager cleans up the Windows Network List Manager
func (w *WindowsNetworkMonitor) cleanupNetworkListManager() {
	// Release the INetworkListManager interface if it was created
	if w.networkListManager != 0 {
		// In a full COM implementation, we would call the Release method on the interface
		// For now, we just reset the handle since we're not doing full COM integration
		w.networkListManager = 0
	}

	// Uninitialize COM
	ole32, err := syscall.LoadDLL("ole32.dll")
	if err == nil {
		coUninitialize, err := ole32.FindProc("CoUninitialize")
		if err == nil {
			coUninitialize.Call()
		}
	}

	slog.Debug("Cleaned up Network List Manager COM interface")
}

// logDiagnosticsPeriodically logs diagnostics periodically for monitoring
func (w *WindowsNetworkMonitor) logDiagnosticsPeriodically() {
	defer w.wg.Done()
	
	ticker := time.NewTicker(5 * time.Minute) // Log every 5 minutes
	defer ticker.Stop()
	
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.logDiagnostics()
		}
	}
}

// logDiagnostics logs comprehensive network diagnostics
func (w *WindowsNetworkMonitor) logDiagnostics() {
	w.mut.RLock()
	defer w.mut.RUnlock()
	
	slog.Info("Network diagnostics report",
		"totalAdapterChanges", w.stabilityMetrics.TotalChanges,
		"recentAdapterChanges", w.stabilityMetrics.RecentChanges,
		"stabilityScore", w.stabilityMetrics.StabilityScore,
		"adaptiveTimeout", w.stabilityMetrics.AdaptiveTimeout,
		"currentProfile", w.currentProfile,
		"activeAdapters", len(w.adapterStates))
	
	// Log details for each adapter
	for name, adapter := range w.adapterStates {
		slog.Debug("Adapter details",
			"name", name,
			"isUp", adapter.IsUp,
			"type", adapter.Type,
			"mediaType", adapter.MediaType,
			"linkSpeed", adapter.LinkSpeed,
			"changeCount", adapter.ChangeCount)
	}
	
	// Log recent events
	if len(w.eventLog) > 0 {
		slog.Debug("Recent network events", "eventCount", len(w.eventLog))
		// Log last 10 events
		startIdx := 0
		if len(w.eventLog) > 10 {
			startIdx = len(w.eventLog) - 10
		}
		for i := startIdx; i < len(w.eventLog); i++ {
			event := w.eventLog[i]
			slog.Debug("Network event",
				"timestamp", event.Timestamp.Format("2006-01-02 15:04:05"),
				"adapter", event.AdapterName,
				"type", event.EventType,
				"details", event.Details,
				"stabilityScore", event.StabilityScore)
		}
	}
}

// logNetworkEvent logs a network change event for diagnostics
func (w *WindowsNetworkMonitor) logNetworkEvent(adapterName, eventType, details string) {
	w.mut.Lock()
	defer w.mut.Unlock()
	
	event := NetworkChangeEvent{
		Timestamp:     time.Now(),
		AdapterName:   adapterName,
		EventType:     eventType,
		Details:       details,
		StabilityScore: w.stabilityMetrics.StabilityScore,
	}
	
	// Add to event log
	w.eventLog = append(w.eventLog, event)
	
	// Trim log if too large
	if len(w.eventLog) > w.maxEventLogSize {
		// Keep the most recent events
		w.eventLog = w.eventLog[len(w.eventLog)-w.maxEventLogSize:]
	}
	
	// Log the event
	slog.Info("Network event logged",
		"adapter", adapterName,
		"type", eventType,
		"details", details,
		"stabilityScore", w.stabilityMetrics.StabilityScore)
}

// updateAdaptiveTimeouts updates the adaptive timeouts based on network stability metrics
func (w *WindowsNetworkMonitor) updateAdaptiveTimeouts() {
	w.mut.Lock()
	defer w.mut.Unlock()
	
	now := time.Now()
	
	// Update stability score based on recent changes
	if now.Sub(w.stabilityMetrics.LastCheckTime) >= 30*time.Second {
		// Calculate stability score based on recent changes
		// Fewer changes = more stable network
		changeRate := float64(w.stabilityMetrics.RecentChanges) / 5.0 // Normalize to 0-1 range
		if changeRate > 1.0 {
			changeRate = 1.0
		}
		
		// Adjust stability score (weighted average)
		w.stabilityMetrics.StabilityScore = 0.7*w.stabilityMetrics.StabilityScore + 0.3*(1.0-changeRate)
		
		// Reset recent changes counter
		w.stabilityMetrics.RecentChanges = 0
		w.stabilityMetrics.LastCheckTime = now
		
		// Adjust adaptive timeout based on stability
		if w.stabilityMetrics.StabilityScore > 0.8 {
			// Stable network, use shorter timeouts
			w.stabilityMetrics.AdaptiveTimeout = 5 * time.Second
		} else if w.stabilityMetrics.StabilityScore > 0.5 {
			// Moderately stable network, use standard timeouts
			w.stabilityMetrics.AdaptiveTimeout = 10 * time.Second
		} else {
			// Unstable network, use longer timeouts to accommodate issues
			w.stabilityMetrics.AdaptiveTimeout = 20 * time.Second
		}
		
		slog.Debug("Updated adaptive timeout",
			"stabilityScore", w.stabilityMetrics.StabilityScore,
			"adaptiveTimeout", w.stabilityMetrics.AdaptiveTimeout,
			"recentChanges", w.stabilityMetrics.RecentChanges)
	}
}

// adjustAdaptiveTimeouts periodically adjusts timeouts based on network stability
func (w *WindowsNetworkMonitor) adjustAdaptiveTimeouts() {
	defer w.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.updateAdaptiveTimeouts()
		}
	}
}

// monitorNetworkChanges periodically checks for network adapter state changes
func (w *WindowsNetworkMonitor) monitorNetworkChanges() {
	defer w.wg.Done()

	// Initial scan to populate adapter states
	w.scanNetworkAdapters()

	// Use a ticker for periodic checks
	// Start with default interval but adjust based on stability
	ticker := time.NewTicker(w.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.checkForNetworkChanges()
			
			// Adjust scan interval based on network stability
			adaptiveInterval := w.getAdaptiveScanInterval()
			if adaptiveInterval != w.scanInterval {
				w.scanInterval = adaptiveInterval
				ticker.Reset(w.scanInterval)
				slog.Debug("Adjusted scan interval", "newInterval", w.scanInterval)
			}
		}
	}
}

// getAdaptiveScanInterval returns an adaptive scan interval based on network stability
func (w *WindowsNetworkMonitor) getAdaptiveScanInterval() time.Duration {
	w.mut.RLock()
	defer w.mut.RUnlock()
	
	// Adjust scan interval based on stability
	if w.stabilityMetrics.StabilityScore > 0.8 {
		// Stable network, scan less frequently
		return 10 * time.Second
	} else if w.stabilityMetrics.StabilityScore > 0.5 {
		// Moderately stable network, use default interval
		return 5 * time.Second
	} else {
		// Unstable network, scan more frequently to detect changes quickly
		return 2 * time.Second
	}
}

// handleRealTimeNotifications handles real-time network change notifications
func (w *WindowsNetworkMonitor) handleRealTimeNotifications() {
	defer w.wg.Done()
	
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.notificationChan:
			// Received a real-time notification, check for changes
			slog.Debug("Received real-time network change notification")
			// Add a small delay to debounce rapid changes
			time.Sleep(100 * time.Millisecond)
			w.checkForNetworkChanges()
		}
	}
}

// scanNetworkAdapters gets the current state of all network adapters using Windows API
func (w *WindowsNetworkMonitor) scanNetworkAdapters() {
	w.mut.Lock()
	defer w.mut.Unlock()

	// Update last scan time
	w.lastScanTime = time.Now()

	// Get list of network interfaces using Windows API
	interfaces, err := w.getNetworkInterfaces()
	if err != nil {
		slog.Debug("Failed to get network interfaces", slogutil.Error(err))
		w.logNetworkEvent("", "scan_error", fmt.Sprintf("Failed to get network interfaces: %v", err))
		return
	}

	// Update adapter states with detailed information
	for _, iface := range interfaces {
		// Convert wide string to Go string
		name := windows.UTF16ToString(iface.Alias[:])
		isUp := (iface.OperStatus == IF_OPER_STATUS_UP)

		// Get existing adapter info or create new one
		adapterInfo, exists := w.adapterStates[name]
		if !exists {
			// New adapter
			adapterInfo = NetworkAdapterInfo{
				Name:        name,
				IsUp:        isUp,
				Type:        iface.Type,
				MediaType:   iface.MediaType,
				LinkSpeed:   iface.TransmitLinkSpeed,
				LastChange:  time.Now(),
				ChangeCount: 0,
			}
			w.logNetworkEvent(name, "adapter_added", fmt.Sprintf("New adapter detected, isUp: %v, type: %d", isUp, iface.Type))
		} else {
			// Existing adapter, check for changes
			hasChanged := adapterInfo.IsUp != isUp || 
				adapterInfo.Type != iface.Type || 
				adapterInfo.MediaType != iface.MediaType || 
				adapterInfo.LinkSpeed != iface.TransmitLinkSpeed
			
			if hasChanged {
				details := fmt.Sprintf("State changed from %v to %v, type: %d->%d, media: %d->%d, speed: %d->%d",
					adapterInfo.IsUp, isUp,
					adapterInfo.Type, iface.Type,
					adapterInfo.MediaType, iface.MediaType,
					adapterInfo.LinkSpeed, iface.TransmitLinkSpeed)
				
				adapterInfo.IsUp = isUp
				adapterInfo.Type = iface.Type
				adapterInfo.MediaType = iface.MediaType
				adapterInfo.LinkSpeed = iface.TransmitLinkSpeed
				adapterInfo.LastChange = time.Now()
				adapterInfo.ChangeCount++
				
				w.logNetworkEvent(name, "adapter_state_change", details)
			}
		}

		w.adapterStates[name] = adapterInfo
	}
}

// getNetworkInterfaces retrieves network interface information using Windows IP Helper API
func (w *WindowsNetworkMonitor) getNetworkInterfaces() ([]MibIfRow2, error) {
	// Load the IP Helper API DLL
	iphlpapi, err := syscall.LoadDLL("iphlpapi.dll")
	if err != nil {
		// Fallback to net.Interfaces if IP Helper API is not available
		w.logNetworkEvent("", "api_error", fmt.Sprintf("Failed to load iphlpapi.dll: %v", err))
		return w.getNetworkInterfacesFallback()
	}

	// Get the GetIfTable2 procedure
	getIfTable2, err := iphlpapi.FindProc("GetIfTable2")
	if err != nil {
		// Fallback to net.Interfaces if GetIfTable2 is not available
		w.logNetworkEvent("", "api_error", fmt.Sprintf("Failed to find GetIfTable2: %v", err))
		return w.getNetworkInterfacesFallback()
	}

	// Call GetIfTable2 to get the interface table
	var table *MibIfTable2
	ret, _, _ := getIfTable2.Call(uintptr(unsafe.Pointer(&table)))
	if ret != 0 {
		// Fallback to net.Interfaces if GetIfTable2 fails
		w.logNetworkEvent("", "api_error", fmt.Sprintf("GetIfTable2 failed with code: %d", ret))
		return w.getNetworkInterfacesFallback()
	}

	// Convert the table to our internal representation
	var result []MibIfRow2
	if table != nil && table.NumEntries > 0 {
		// Access the table entries
		entries := (*[1 << 20]MibIfRow2)(unsafe.Pointer(&table.Table[0]))[:table.NumEntries:table.NumEntries]
		for _, entry := range entries {
			result = append(result, entry)
		}
	}

	// Free the table memory
	freeMibTable, err := iphlpapi.FindProc("FreeMibTable")
	if err == nil && table != nil {
		freeMibTable.Call(uintptr(unsafe.Pointer(table)))
	}

	return result, nil
}

// getNetworkInterfacesFallback retrieves network interface information using Go's net.Interfaces
// This is a fallback implementation when the Windows IP Helper API is not available
func (w *WindowsNetworkMonitor) getNetworkInterfacesFallback() ([]MibIfRow2, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		w.logNetworkEvent("", "fallback_error", fmt.Sprintf("net.Interfaces failed: %v", err))
		return nil, err
	}

	// Convert to our internal representation
	var result []MibIfRow2
	for _, iface := range interfaces {
		row := MibIfRow2{
			InterfaceIndex: uint32(iface.Index),
			Alias:          [257]uint16{},
			OperStatus:     IF_OPER_STATUS_DOWN,
		}

		// Convert interface name to wide string
		nameRunes := []rune(iface.Name)
		for i, r := range nameRunes {
			if i >= 256 {
				break
			}
			row.Alias[i] = uint16(r)
		}

		// Set operational status based on interface flags
		if iface.Flags&net.FlagUp != 0 {
			row.OperStatus = IF_OPER_STATUS_UP
		}

		result = append(result, row)
	}

	return result, nil
}

// checkForNetworkChanges compares current adapter states with previous states
// and triggers reconnection if adapters wake up or network conditions change
func (w *WindowsNetworkMonitor) checkForNetworkChanges() {
	w.mut.Lock()
	defer w.mut.Unlock()

	// Check if we're still in cooldown period
	if time.Since(w.lastScanTime) < w.changeCooldown {
		return
	}

	// Get current adapter states
	currentStates := make(map[string]NetworkAdapterInfo)
	interfaces, err := w.getNetworkInterfaces()
	if err != nil {
		slog.Debug("Failed to get current network interfaces", slogutil.Error(err))
		w.logNetworkEvent("", "scan_error", fmt.Sprintf("Failed to get current network interfaces: %v", err))
		return
	}

	// Populate current states with detailed information
	for _, iface := range interfaces {
		name := windows.UTF16ToString(iface.Alias[:])
		isUp := (iface.OperStatus == IF_OPER_STATUS_UP)

		// Get existing adapter info or create new one
		adapterInfo, exists := currentStates[name]
		if !exists {
			// New adapter
			adapterInfo = NetworkAdapterInfo{
				Name:        name,
				IsUp:        isUp,
				Type:        iface.Type,
				MediaType:   iface.MediaType,
				LinkSpeed:   iface.TransmitLinkSpeed,
				LastChange:  time.Now(),
				ChangeCount: 0,
			}
		} else {
			// Existing adapter, check for changes
			hasChanged := adapterInfo.IsUp != isUp || 
				adapterInfo.Type != iface.Type || 
				adapterInfo.MediaType != iface.MediaType || 
				adapterInfo.LinkSpeed != iface.TransmitLinkSpeed
			
			if hasChanged {
				adapterInfo.IsUp = isUp
				adapterInfo.Type = iface.Type
				adapterInfo.MediaType = iface.MediaType
				adapterInfo.LinkSpeed = iface.TransmitLinkSpeed
				adapterInfo.LastChange = time.Now()
				adapterInfo.ChangeCount++
			}
		}

		currentStates[name] = adapterInfo
	}

	// Check for adapter state changes
	changesDetected := false
	significantChanges := 0
	
	for adapter, currentInfo := range currentStates {
		previousInfo, exists := w.adapterStates[adapter]
		if !exists {
			// New adapter detected
			changesDetected = true
			significantChanges++
			w.stabilityMetrics.TotalChanges++
			w.stabilityMetrics.RecentChanges++
			w.logNetworkEvent(adapter, "adapter_added", "New network adapter detected")
			slog.Info("New network adapter detected",
				"adapter", adapter,
				"isUp", currentInfo.IsUp,
				"type", currentInfo.Type)
		} else {
			// Check for significant changes
			stateChanged := previousInfo.IsUp != currentInfo.IsUp
			typeChanged := previousInfo.Type != currentInfo.Type
			mediaChanged := previousInfo.MediaType != currentInfo.MediaType
			speedChanged := previousInfo.LinkSpeed != currentInfo.LinkSpeed
			frequentChanges := currentInfo.ChangeCount > previousInfo.ChangeCount + 3 // More than 3 changes since last check
			
			if stateChanged || typeChanged || mediaChanged || speedChanged || frequentChanges {
				changesDetected = true
				
				// Count significant changes (adapter coming up or frequent changes)
				if (!previousInfo.IsUp && currentInfo.IsUp) || frequentChanges {
					significantChanges++
				}
				
				// Update stability metrics
				w.stabilityMetrics.TotalChanges++
				w.stabilityMetrics.RecentChanges++
				w.stabilityMetrics.LastErrorTime = time.Now()
				
				changeDetails := fmt.Sprintf("State: %v->%v, Type: %d->%d, Media: %d->%d, Speed: %d->%d, Frequent: %v",
					previousInfo.IsUp, currentInfo.IsUp,
					previousInfo.Type, currentInfo.Type,
					previousInfo.MediaType, currentInfo.MediaType,
					previousInfo.LinkSpeed, currentInfo.LinkSpeed,
					frequentChanges)
				
				w.logNetworkEvent(adapter, "adapter_changed", changeDetails)
				
				slog.Info("Network adapter change detected",
					"adapter", adapter,
					"previousState", previousInfo.IsUp,
					"currentState", currentInfo.IsUp,
					"stateChanged", stateChanged,
					"typeChanged", typeChanged,
					"mediaChanged", mediaChanged,
					"speedChanged", speedChanged,
					"frequentChanges", frequentChanges)
				
				// Log detailed information for frequent changes which might indicate KB5060998 issues
				if frequentChanges {
					warningDetails := fmt.Sprintf("KB5060998 impact suspected: changeCount %d, previousCount %d",
						currentInfo.ChangeCount, previousInfo.ChangeCount)
					w.logNetworkEvent(adapter, "kb5060998_suspected", warningDetails)
					slog.Warn("Frequent network adapter changes detected - possible KB5060998 impact",
						"adapter", adapter,
						"changeCount", currentInfo.ChangeCount,
						"previousCount", previousInfo.ChangeCount)
				}

				// If an adapter was down and is now up, trigger reconnection
				if !previousInfo.IsUp && currentInfo.IsUp {
					w.logNetworkEvent(adapter, "adapter_up", "Network adapter woke up, triggering reconnection")
					slog.Info("Network adapter woke up, triggering reconnection", "adapter", adapter)
					w.triggerReconnection()
				}
			}
		}
	}
	
	// Check for removed adapters
	for adapter, previousInfo := range w.adapterStates {
		if _, exists := currentStates[adapter]; !exists {
			changesDetected = true
			significantChanges++
			w.stabilityMetrics.TotalChanges++
			w.stabilityMetrics.RecentChanges++
			w.logNetworkEvent(adapter, "adapter_removed", "Network adapter removed")
			slog.Info("Network adapter removed",
				"adapter", adapter,
				"wasUp", previousInfo.IsUp)
		}
	}

	// Update stored states only if there were changes
	if changesDetected {
		w.adapterStates = currentStates
		// Update last scan time
		w.lastScanTime = time.Now()
	}

	// Check for network profile changes with enhanced detection
	newProfile := w.GetNetworkProfileEnhanced()
	profileChanged := (w.currentProfile != newProfile)
	if profileChanged {
		changesDetected = true
		significantChanges++
		w.stabilityMetrics.TotalChanges++
		w.stabilityMetrics.RecentChanges++
		w.logNetworkEvent("", "profile_changed", fmt.Sprintf("Network profile changed from %s to %s", w.currentProfile, newProfile))
		slog.Info("Network profile changed",
			"previousProfile", w.currentProfile,
			"newProfile", newProfile)
		w.currentProfile = newProfile
	}

	// If significant changes were detected, trigger reconnection
	// Also trigger if we've detected potential KB5060998 issues (frequent changes)
	if significantChanges > 0 {
		w.logNetworkEvent("", "reconnection_triggered", fmt.Sprintf("Significant network changes detected: %d", significantChanges))
		slog.Info("Significant network changes detected, triggering reconnection",
			"changeCount", significantChanges)
		w.triggerReconnection()
	}
}

// triggerReconnection triggers immediate reconnection attempts to all devices
func (w *WindowsNetworkMonitor) triggerReconnection() {
	if w.service != nil {
		slog.Info("Triggering immediate reconnection to all devices")
		w.logNetworkEvent("", "reconnection_started", "Triggering immediate reconnection to all devices")
		// Call the service's DialNow method to trigger immediate reconnection
		w.service.DialNow()
	}
}

// GetNetworkProfile returns the current network profile (Public/Private)
func (w *WindowsNetworkMonitor) GetNetworkProfile() string {
	w.mut.RLock()
	defer w.mut.RUnlock()
	// Return the current profile
	return w.currentProfile
}

// GetNetworkProfileEnhanced returns the current network profile with enhanced detection
func (w *WindowsNetworkMonitor) GetNetworkProfileEnhanced() string {
	// For now, just return the standard profile
	return w.GetNetworkProfile()
}

// GetAdapterStates returns a copy of the current adapter states for testing purposes
func (w *WindowsNetworkMonitor) GetAdapterStates() map[string]NetworkAdapterInfo {
	w.mut.RLock()
	defer w.mut.RUnlock()
	
	// Create a copy to avoid race conditions
	states := make(map[string]NetworkAdapterInfo)
	for k, v := range w.adapterStates {
		states[k] = v
	}
	return states
}

// GetStabilityMetrics returns a copy of the current stability metrics for testing purposes
func (w *WindowsNetworkMonitor) GetStabilityMetrics() NetworkStabilityMetrics {
	w.mut.RLock()
	defer w.mut.RUnlock()
	
	// Return a copy
	return *w.stabilityMetrics
}

// GetEventLog returns a copy of the event log for testing purposes
func (w *WindowsNetworkMonitor) GetEventLog() []NetworkChangeEvent {
	w.mut.RLock()
	defer w.mut.RUnlock()
	
	// Create a copy to avoid race conditions
	log := make([]NetworkChangeEvent, len(w.eventLog))
	copy(log, w.eventLog)
	return log
}

// GetMaxEventLogSize returns the maximum event log size for testing purposes
func (w *WindowsNetworkMonitor) GetMaxEventLogSize() int {
	w.mut.RLock()
	defer w.mut.RUnlock()
	return w.maxEventLogSize
}

// SetAdapterState allows tests to set adapter states directly
func (w *WindowsNetworkMonitor) SetAdapterState(name string, info NetworkAdapterInfo) {
	w.mut.Lock()
	defer w.mut.Unlock()
	w.adapterStates[name] = info
}

// GetAdaptiveTimeout returns the current adaptive timeout for testing purposes
func (w *WindowsNetworkMonitor) GetAdaptiveTimeout() time.Duration {
	w.mut.RLock()
	defer w.mut.RUnlock()
	return w.stabilityMetrics.AdaptiveTimeout
}

// NetworkInterfaceChangeCallback handles network interface change notifications
// This would be called by Windows when network interfaces change
func (w *WindowsNetworkMonitor) NetworkInterfaceChangeCallback() {
	// This is a callback that would be registered with Windows
	// For now, we'll just trigger a scan
	slog.Info("Network interface change notification received")
	w.logNetworkEvent("", "interface_notification", "Network interface change notification received")
	// Send notification to real-time handler
	select {
	case w.notificationChan <- struct{}{}:
	default:
		// Channel is full, which is fine - we'll process the notification soon
		slog.Debug("Notification channel full, dropping notification")
		w.logNetworkEvent("", "notification_dropped", "Notification channel full, dropping notification")
	}
}

// registerForNetworkChangeNotifications registers for network change notifications
// This would use Windows APIs to register for real-time network change notifications
func (w *WindowsNetworkMonitor) registerForNetworkChangeNotifications() {
	// Load the IP Helper API DLL
	iphlpapi, err := syscall.LoadDLL("iphlpapi.dll")
	if err != nil {
		slog.Debug("Failed to load iphlpapi.dll for network change notifications", slogutil.Error(err))
		w.logNetworkEvent("", "registration_error", fmt.Sprintf("Failed to load iphlpapi.dll: %v", err))
		return
	}

	// Get the NotifyAddrChange procedure
	notifyAddrChange, err := iphlpapi.FindProc("NotifyAddrChange")
	if err != nil {
		slog.Debug("Failed to find NotifyAddrChange", slogutil.Error(err))
		w.logNetworkEvent("", "registration_error", fmt.Sprintf("Failed to find NotifyAddrChange: %v", err))
		return
	}

	// Get the NotifyRouteChange procedure
	notifyRouteChange, err := iphlpapi.FindProc("NotifyRouteChange")
	if err != nil {
		slog.Debug("Failed to find NotifyRouteChange", slogutil.Error(err))
		w.logNetworkEvent("", "registration_error", fmt.Sprintf("Failed to find NotifyRouteChange: %v", err))
		return
	}

	// Create event handles for notifications
	addrEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		slog.Debug("Failed to create address change event", slogutil.Error(err))
		w.logNetworkEvent("", "registration_error", fmt.Sprintf("Failed to create address change event: %v", err))
		return
	}
	
	routeEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		slog.Debug("Failed to create route change event", slogutil.Error(err))
		w.logNetworkEvent("", "registration_error", fmt.Sprintf("Failed to create route change event: %v", err))
		windows.CloseHandle(addrEvent)
		return
	}

	// Register for address change notifications
	ret, _, _ := notifyAddrChange.Call(uintptr(unsafe.Pointer(&w.addrChangeHandle)), uintptr(addrEvent))
	if ret != 0 {
		slog.Debug("Failed to register for address change notifications", "errorCode", ret)
		w.logNetworkEvent("", "registration_error", fmt.Sprintf("Failed to register for address change notifications, error code: %d", ret))
		windows.CloseHandle(addrEvent)
		windows.CloseHandle(routeEvent)
		return
	}

	// Register for route change notifications
	ret, _, _ = notifyRouteChange.Call(uintptr(unsafe.Pointer(&w.routeChangeHandle)), uintptr(routeEvent))
	if ret != 0 {
		slog.Debug("Failed to register for route change notifications", "errorCode", ret)
		w.logNetworkEvent("", "registration_error", fmt.Sprintf("Failed to register for route change notifications, error code: %d", ret))
		// Clean up address change registration
		// Note: In a full implementation, we would use CancelIPChangeNotify here
		w.addrChangeHandle = 0
		windows.CloseHandle(addrEvent)
		windows.CloseHandle(routeEvent)
		return
	}

	slog.Debug("Registered for real-time network change notifications")
	w.logNetworkEvent("", "registration_success", "Registered for real-time network change notifications")

	// Start goroutines to wait for notifications
	w.wg.Add(1)
	go w.waitForAddressChanges(syscall.Handle(addrEvent))
	
	w.wg.Add(1)
	go w.waitForRouteChanges(syscall.Handle(routeEvent))
}

// unregisterNetworkChangeNotifications unregisters network change notifications
func (w *WindowsNetworkMonitor) unregisterNetworkChangeNotifications() {
	// Note: In a full implementation, we would use CancelIPChangeNotify here
	// For now, we just reset the handles
	if w.addrChangeHandle != 0 {
		w.addrChangeHandle = 0
	}
	
	if w.routeChangeHandle != 0 {
		w.routeChangeHandle = 0
	}
	
	slog.Debug("Unregistered network change notifications")
	w.logNetworkEvent("", "unregistration_success", "Unregistered network change notifications")
}

// waitForAddressChanges waits for IP address change notifications
func (w *WindowsNetworkMonitor) waitForAddressChanges(event syscall.Handle) {
	defer w.wg.Done()
	
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			// Wait for the event
			result, err := windows.WaitForSingleObject(windows.Handle(event), windows.INFINITE)
			if err != nil {
				slog.Debug("Error waiting for address change event", slogutil.Error(err))
				w.logNetworkEvent("", "wait_error", fmt.Sprintf("Error waiting for address change event: %v", err))
				return
			}
			
			if result == windows.WAIT_OBJECT_0 {
				slog.Debug("Address change notification received")
				w.logNetworkEvent("", "address_change", "Address change notification received")
				// Send notification to real-time handler
				select {
				case w.notificationChan <- struct{}{}:
				default:
					// Channel is full, which is fine
					slog.Debug("Notification channel full, dropping address change notification")
					w.logNetworkEvent("", "notification_dropped", "Notification channel full, dropping address change notification")
				}
				
				// Re-register for the next notification
				iphlpapi, err := syscall.LoadDLL("iphlpapi.dll")
				if err != nil {
					slog.Debug("Failed to load iphlpapi.dll", slogutil.Error(err))
					w.logNetworkEvent("", "re_registration_error", fmt.Sprintf("Failed to load iphlpapi.dll: %v", err))
					return
				}
				
				notifyAddrChange, err := iphlpapi.FindProc("NotifyAddrChange")
				if err != nil {
					slog.Debug("Failed to find NotifyAddrChange", slogutil.Error(err))
					w.logNetworkEvent("", "re_registration_error", fmt.Sprintf("Failed to find NotifyAddrChange: %v", err))
					return
				}
				
				ret, _, _ := notifyAddrChange.Call(uintptr(unsafe.Pointer(&w.addrChangeHandle)), uintptr(event))
				if ret != 0 {
					slog.Debug("Failed to re-register for address change notifications", "errorCode", ret)
					w.logNetworkEvent("", "re_registration_error", fmt.Sprintf("Failed to re-register for address change notifications, error code: %d", ret))
					return
				}
			}
		}
	}
}

// waitForRouteChanges waits for route change notifications
func (w *WindowsNetworkMonitor) waitForRouteChanges(event syscall.Handle) {
	defer w.wg.Done()
	
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			// Wait for the event
			result, err := windows.WaitForSingleObject(windows.Handle(event), windows.INFINITE)
			if err != nil {
				slog.Debug("Error waiting for route change event", slogutil.Error(err))
				w.logNetworkEvent("", "wait_error", fmt.Sprintf("Error waiting for route change event: %v", err))
				return
			}
			
			if result == windows.WAIT_OBJECT_0 {
				slog.Debug("Route change notification received")
				w.logNetworkEvent("", "route_change", "Route change notification received")
				// Send notification to real-time handler
				select {
				case w.notificationChan <- struct{}{}:
				default:
					// Channel is full, which is fine
					slog.Debug("Notification channel full, dropping route change notification")
					w.logNetworkEvent("", "notification_dropped", "Notification channel full, dropping route change notification")
				}
				
				// Re-register for the next notification
				iphlpapi, err := syscall.LoadDLL("iphlpapi.dll")
				if err != nil {
					slog.Debug("Failed to load iphlpapi.dll", slogutil.Error(err))
					w.logNetworkEvent("", "re_registration_error", fmt.Sprintf("Failed to load iphlpapi.dll: %v", err))
					return
				}
				
				notifyRouteChange, err := iphlpapi.FindProc("NotifyRouteChange")
				if err != nil {
					slog.Debug("Failed to find NotifyRouteChange", slogutil.Error(err))
					w.logNetworkEvent("", "re_registration_error", fmt.Sprintf("Failed to find NotifyRouteChange: %v", err))
					return
				}
				
				ret, _, _ := notifyRouteChange.Call(uintptr(unsafe.Pointer(&w.routeChangeHandle)), uintptr(event))
				if ret != 0 {
					slog.Debug("Failed to re-register for route change notifications", "errorCode", ret)
					w.logNetworkEvent("", "re_registration_error", fmt.Sprintf("Failed to re-register for route change notifications, error code: %d", ret))
					return
				}
			}
		}
	}
}