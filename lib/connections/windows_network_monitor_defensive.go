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
	"strings"
	"sync"
	"time"

	"github.com/syncthing/syncthing/internal/slogutil"
)

// DefensiveWindowsNetworkMonitor is a defensive implementation of Windows network monitoring
// that avoids unsafe CGO calls and uses only safe Go standard library functions
type DefensiveWindowsNetworkMonitor struct {
	service        Service
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	adapterStates  map[string]NetworkAdapterInfo
	currentProfile string
	mut            sync.RWMutex
	
	// Enhanced monitoring
	lastScanTime   time.Time
	scanInterval   time.Duration
	changeCooldown time.Duration
	
	// Network stability tracking
	stabilityMetrics *NetworkStabilityMetrics
	
	// Event logging
	eventLog       []NetworkChangeEvent
	maxEventLogSize int
	
	// Feature flags
	enableRealtimeNotifications bool
	enableCOMInitialization     bool
}

// NewDefensiveWindowsNetworkMonitor creates a new defensive Windows network monitor
func NewDefensiveWindowsNetworkMonitor(svc Service) *DefensiveWindowsNetworkMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	monitor := &DefensiveWindowsNetworkMonitor{
		service:        svc,
		ctx:            ctx,
		cancel:         cancel,
		adapterStates:  make(map[string]NetworkAdapterInfo),
		currentProfile: "Unknown",
		scanInterval:   10 * time.Second,
		changeCooldown: 2 * time.Second,
		stabilityMetrics: &NetworkStabilityMetrics{
			StabilityScore:  1.0,
			AdaptiveTimeout: 5 * time.Second,
		},
		eventLog:                    make([]NetworkChangeEvent, 0, 100),
		maxEventLogSize:             100,
		enableRealtimeNotifications: false, // Disabled by default for safety
		enableCOMInitialization:     false, // Disabled by default for safety
	}

	// Initialize the current network profile using safe methods
	monitor.currentProfile = monitor.GetNetworkProfile()

	return monitor
}

// Start begins monitoring network adapter state changes
func (w *DefensiveWindowsNetworkMonitor) Start() {
	// Only start the main monitoring goroutine
	w.wg.Add(1)
	go w.monitorNetworkChanges()
}

// Stop stops monitoring network adapter state changes
func (w *DefensiveWindowsNetworkMonitor) Stop() {
	// Log final diagnostics before stopping
	w.logDiagnostics()
	
	w.cancel()
	
	// Add a timeout to prevent hanging
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// All goroutines finished
	case <-time.After(5 * time.Second):
		// Timeout - force exit
		slog.Warn("DefensiveWindowsNetworkMonitor.Stop timed out waiting for goroutines")
	}
}

// monitorNetworkChanges periodically checks for network adapter state changes
func (w *DefensiveWindowsNetworkMonitor) monitorNetworkChanges() {
	defer w.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered from panic in monitorNetworkChanges", "panic", r)
		}
	}()

	// Initial scan to populate adapter states
	w.scanNetworkAdapters()

	// Use a ticker for periodic checks
	ticker := time.NewTicker(w.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			// Safely check for network changes
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("Recovered from panic in checkForNetworkChanges", "panic", r)
						w.logNetworkEvent("", "check_panic", fmt.Sprintf("Recovered from panic: %v", r))
					}
				}()
				w.checkForNetworkChanges()
			}()
			
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

// scanNetworkAdapters gets the current state of all network adapters safely
func (w *DefensiveWindowsNetworkMonitor) scanNetworkAdapters() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered from panic in scanNetworkAdapters", "panic", r)
			w.logNetworkEvent("", "scan_panic", fmt.Sprintf("Recovered from panic: %v", r))
		}
	}()

	w.mut.Lock()
	defer w.mut.Unlock()

	// Update last scan time
	w.lastScanTime = time.Now()

	// Get list of network interfaces using Go's standard library (safe)
	interfaces, err := w.getNetworkInterfacesSafely()
	if err != nil {
		slog.Debug("Failed to get network interfaces", slogutil.Error(err))
		w.logNetworkEvent("", "scan_error", fmt.Sprintf("Failed to get network interfaces: %v", err))
		return
	}

	// Update adapter states with detailed information
	for _, iface := range interfaces {
		name := iface.Name
		isUp := iface.IsUp

		// Get existing adapter info or create new one
		adapterInfo, exists := w.adapterStates[name]
		if !exists {
			// New adapter
			adapterInfo = NetworkAdapterInfo{
				Name:        name,
				IsUp:        isUp,
				Type:        0, // Not available in safe implementation
				MediaType:   0, // Not available in safe implementation
				LinkSpeed:   0, // Not available in safe implementation
				LastChange:  time.Now(),
				ChangeCount: 0,
			}
			w.logNetworkEvent(name, "adapter_added", fmt.Sprintf("New adapter detected, isUp: %v", isUp))
		} else {
			// Existing adapter, check for changes
			hasChanged := adapterInfo.IsUp != isUp
			
			if hasChanged {
				details := fmt.Sprintf("State changed from %v to %v", adapterInfo.IsUp, isUp)
				
				adapterInfo.IsUp = isUp
				adapterInfo.LastChange = time.Now()
				adapterInfo.ChangeCount++
				
				w.logNetworkEvent(name, "adapter_state_change", details)
			}
		}

		w.adapterStates[name] = adapterInfo
	}
}

// getNetworkInterfacesSafely retrieves network interface information using Go's standard library
func (w *DefensiveWindowsNetworkMonitor) getNetworkInterfacesSafely() ([]DefensiveNetworkInterface, error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered from panic in getNetworkInterfacesSafely", "panic", r)
			w.logNetworkEvent("", "get_interfaces_panic", fmt.Sprintf("Recovered from panic: %v", r))
		}
	}()

	// Use Go's standard library net.Interfaces which is safe
	interfaces, err := net.Interfaces()
	if err != nil {
		w.logNetworkEvent("", "interfaces_error", fmt.Sprintf("net.Interfaces failed: %v", err))
		return nil, err
	}

	// Convert to our internal representation
	var result []DefensiveNetworkInterface
	for _, iface := range interfaces {
		defensiveIface := DefensiveNetworkInterface{
			Name:  iface.Name,
			IsUp:  iface.Flags&net.FlagUp != 0,
			Index: iface.Index,
		}
		result = append(result, defensiveIface)
	}

	return result, nil
}

// DefensiveNetworkInterface represents a network interface in a safe way
type DefensiveNetworkInterface struct {
	Name  string
	IsUp  bool
	Index int
}

// GetNetworkProfile returns the current network profile safely
func (w *DefensiveWindowsNetworkMonitor) GetNetworkProfile() string {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered from panic in GetNetworkProfile", "panic", r)
			w.logNetworkEvent("", "profile_panic", fmt.Sprintf("Recovered from panic: %v", r))
		}
	}()

	// Try to get the network profile using safe heuristics
	profile := w.getNetworkProfileSafely()
	if profile == "" {
		// Fallback to a default value
		w.logNetworkEvent("", "profile_fallback", "Using default network profile")
		return "Unknown"
	}
	return profile
}

// getNetworkProfileSafely retrieves the network profile using safe heuristics
func (w *DefensiveWindowsNetworkMonitor) getNetworkProfileSafely() string {
	// Use heuristic-based detection based on adapter states
	w.mut.RLock()
	defer w.mut.RUnlock()
	
	// Count different types of adapters
	wiredAdapters := 0
	wirelessAdapters := 0
	
	for _, adapter := range w.adapterStates {
		if adapter.IsUp {
			// In a safe implementation, we can't determine the exact type
			// So we'll use some heuristics based on common naming patterns
			name := adapter.Name
			switch {
			case containsAny(name, []string{"ethernet", "eth", "lan"}):
				wiredAdapters++
			case containsAny(name, []string{"wi-fi", "wifi", "wireless"}):
				wirelessAdapters++
			default:
				// Default assumption
				wirelessAdapters++
			}
		}
	}
	
	// Make a determination based on adapters
	if wiredAdapters > wirelessAdapters {
		return "Domain" // Wired connections often indicate domain networks
	} else if wirelessAdapters > 0 {
		return "Private" // Wireless connections often indicate private networks
	}
	
	// Default fallback
	return "Public"
}

// containsAny checks if a string contains any of the substrings (case insensitive)
func containsAny(s string, substrings []string) bool {
	// Convert to lowercase for case-insensitive comparison
	s = strings.ToLower(s)
	for _, substr := range substrings {
		substr = strings.ToLower(substr)
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// handleRealTimeNotifications handles real-time network change notifications (stub implementation)
func (w *DefensiveWindowsNetworkMonitor) handleRealTimeNotifications() {
	defer w.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered from panic in handleRealTimeNotifications", "panic", r)
		}
	}()
	
	// This is a stub implementation - real-time notifications are disabled for safety
	// In a full implementation, this would handle real-time notifications
	
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			// Sleep and periodically check
			time.Sleep(1 * time.Second)
		}
	}
}

// adjustAdaptiveTimeouts periodically adjusts timeouts based on network stability
func (w *DefensiveWindowsNetworkMonitor) adjustAdaptiveTimeouts() {
	defer w.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered from panic in adjustAdaptiveTimeouts", "panic", r)
		}
	}()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("Recovered from panic in updateAdaptiveTimeouts", "panic", r)
					}
				}()
				w.updateAdaptiveTimeouts()
			}()
		}
	}
}

// logDiagnosticsPeriodically logs diagnostics periodically for monitoring
func (w *DefensiveWindowsNetworkMonitor) logDiagnosticsPeriodically() {
	defer w.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered from panic in logDiagnosticsPeriodically", "panic", r)
		}
	}()
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("Recovered from panic in logDiagnostics", "panic", r)
					}
				}()
				w.logDiagnostics()
			}()
		}
	}
}

// checkForNetworkChanges compares current adapter states with previous states
func (w *DefensiveWindowsNetworkMonitor) checkForNetworkChanges() {
	w.mut.Lock()
	defer w.mut.Unlock()

	// Check if we're still in cooldown period
	if time.Since(w.lastScanTime) < w.changeCooldown {
		return
	}

	// Get current adapter states
	currentStates := make(map[string]NetworkAdapterInfo)
	interfaces, err := w.getNetworkInterfacesSafely()
	if err != nil {
		slog.Debug("Failed to get current network interfaces", slogutil.Error(err))
		w.logNetworkEvent("", "scan_error", fmt.Sprintf("Failed to get current network interfaces: %v", err))
		return
	}

	// Populate current states
	for _, iface := range interfaces {
		name := iface.Name
		isUp := iface.IsUp

		// Get existing adapter info or create new one
		adapterInfo, exists := currentStates[name]
		if !exists {
			// New adapter
			adapterInfo = NetworkAdapterInfo{
				Name:        name,
				IsUp:        isUp,
				Type:        0,
				MediaType:   0,
				LinkSpeed:   0,
				LastChange:  time.Now(),
				ChangeCount: 0,
			}
		} else {
			// Existing adapter, check for changes
			hasChanged := adapterInfo.IsUp != isUp
			
			if hasChanged {
				adapterInfo.IsUp = isUp
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
				"isUp", currentInfo.IsUp)
		} else {
			// Check for significant changes
			stateChanged := previousInfo.IsUp != currentInfo.IsUp
			frequentChanges := currentInfo.ChangeCount > previousInfo.ChangeCount + 3 // More than 3 changes since last check
			
			if stateChanged || frequentChanges {
				changesDetected = true
				
				// Count significant changes (adapter coming up or frequent changes)
				if (!previousInfo.IsUp && currentInfo.IsUp) || frequentChanges {
					significantChanges++
				}
				
				// Update stability metrics
				w.stabilityMetrics.TotalChanges++
				w.stabilityMetrics.RecentChanges++
				w.stabilityMetrics.LastErrorTime = time.Now()
				
				changeDetails := fmt.Sprintf("State: %v->%v, Frequent: %v",
					previousInfo.IsUp, currentInfo.IsUp, frequentChanges)
				
				w.logNetworkEvent(adapter, "adapter_changed", changeDetails)
				
				slog.Info("Network adapter change detected",
					"adapter", adapter,
					"previousState", previousInfo.IsUp,
					"currentState", currentInfo.IsUp,
					"stateChanged", stateChanged,
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
	if significantChanges > 0 {
		w.logNetworkEvent("", "reconnection_triggered", fmt.Sprintf("Significant network changes detected: %d", significantChanges))
		slog.Info("Significant network changes detected, triggering reconnection",
			"changeCount", significantChanges)
		w.triggerReconnection()
	}
}

// triggerReconnection triggers immediate reconnection attempts to all devices
func (w *DefensiveWindowsNetworkMonitor) triggerReconnection() {
	if w.service != nil {
		slog.Info("Triggering immediate reconnection to all devices")
		w.logNetworkEvent("", "reconnection_started", "Triggering immediate reconnection to all devices")
		// Call the service's DialNow method to trigger immediate reconnection
		w.service.DialNow()
	}
}

// GetNetworkProfileEnhanced returns the current network profile with enhanced detection
func (w *DefensiveWindowsNetworkMonitor) GetNetworkProfileEnhanced() string {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered from panic in GetNetworkProfileEnhanced", "panic", r)
			w.logNetworkEvent("", "enhanced_profile_panic", fmt.Sprintf("Recovered from panic: %v", r))
		}
	}()

	// Try to get the network profile using enhanced safe heuristics
	profile := w.getNetworkProfileSafely()
	if profile == "" {
		// Fallback to standard method
		w.logNetworkEvent("", "enhanced_profile_fallback", "Using standard network profile")
		return w.GetNetworkProfile()
	}
	return profile
}

// getAdaptiveScanInterval returns an adaptive scan interval based on network stability
func (w *DefensiveWindowsNetworkMonitor) getAdaptiveScanInterval() time.Duration {
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

// updateAdaptiveTimeouts updates the adaptive timeouts based on network stability metrics
func (w *DefensiveWindowsNetworkMonitor) updateAdaptiveTimeouts() {
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

// logDiagnostics logs comprehensive network diagnostics
func (w *DefensiveWindowsNetworkMonitor) logDiagnostics() {
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
func (w *DefensiveWindowsNetworkMonitor) logNetworkEvent(adapterName, eventType, details string) {
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