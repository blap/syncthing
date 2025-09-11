// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/thejerf/suture/v4"
)

// FolderPerformanceStats represents performance statistics for a folder
type FolderPerformanceStats struct {
	LastCheckTime    time.Time     `json:"lastCheckTime"`
	CheckDuration    time.Duration `json:"checkDuration"`
	CPUUsagePercent  float64       `json:"cpuUsagePercent"`
	MemoryUsageBytes uint64        `json:"memoryUsageBytes"`
	CheckCount       int           `json:"checkCount"`
	FailedCheckCount int           `json:"failedCheckCount"`
	AvgCheckDuration time.Duration `json:"avgCheckDuration"`
	LastError        string        `json:"lastError"`
}

const (
	// Default health check interval for active folders
	defaultHealthCheckInterval = 30 * time.Second

	// Health check interval for inactive folders
	inactiveHealthCheckInterval = 5 * time.Minute

	// Health check interval for paused folders
	pausedHealthCheckInterval = 30 * time.Minute

	// Memory optimization thresholds
	memoryOptimizationThreshold = 1024 * 1024 * 1024 // 1GB
)

// FolderHealthMonitor monitors the health of folders periodically
type FolderHealthMonitor struct {
	suture.Service
	ctx      context.Context
	cancel   context.CancelFunc
	cfg      config.Wrapper
	model    Model
	evLogger events.Logger

	// Map of folder ID to health check ticker
	folderTickers map[string]*time.Ticker
	tickersMut    sync.RWMutex

	// Map of folder ID to last health status
	lastHealthStatus map[string]config.FolderHealthStatus
	healthStatusMut  sync.RWMutex

	// Performance monitoring
	performanceStats map[string]FolderPerformanceStats
	perfStatsMut     sync.RWMutex

	// Memory optimization
	memoryLimiter *MemoryLimiter
}

// NewFolderHealthMonitor creates a new folder health monitor
func NewFolderHealthMonitor(cfg config.Wrapper, model Model, evLogger events.Logger) *FolderHealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	fhm := &FolderHealthMonitor{
		ctx:              ctx,
		cancel:           cancel,
		cfg:              cfg,
		model:            model,
		evLogger:         evLogger,
		folderTickers:    make(map[string]*time.Ticker),
		lastHealthStatus: make(map[string]config.FolderHealthStatus),
		performanceStats: make(map[string]FolderPerformanceStats),
		memoryLimiter:    NewMemoryLimiter(),
	}

	// Subscribe to configuration changes to update monitored folders
	cfg.Subscribe(fhm)

	return fhm
}

// Serve implements suture.Service
func (fhm *FolderHealthMonitor) Serve(ctx context.Context) error {
	// Initialize health monitoring for existing folders
	fhm.initializeFolderMonitoring()

	// Wait for context cancellation
	<-fhm.ctx.Done()

	// Cleanup
	fhm.cleanup()

	return nil
}

// initializeFolderMonitoring sets up health monitoring for all folders in the configuration
func (fhm *FolderHealthMonitor) initializeFolderMonitoring() {
	folders := fhm.cfg.Folders()

	for folderID := range folders {
		fhm.startMonitoringFolder(folderID)
	}
}

// startMonitoringFolder starts health monitoring for a specific folder
func (fhm *FolderHealthMonitor) startMonitoringFolder(folderID string) {
	fhm.tickersMut.Lock()
	defer fhm.tickersMut.Unlock()

	// Stop existing ticker if any
	if ticker, exists := fhm.folderTickers[folderID]; exists {
		ticker.Stop()
	}

	// Get folder configuration
	folder, ok := fhm.cfg.Folder(folderID)
	if !ok {
		return
	}

	// Determine check interval based on folder state
	interval := fhm.getHealthCheckInterval(folder, folderID)

	// Create new ticker
	ticker := time.NewTicker(interval)
	fhm.folderTickers[folderID] = ticker

	// Start monitoring goroutine
	go fhm.monitorFolderHealth(folderID, ticker)

	slog.Debug("Started health monitoring for folder", "folder", folderID, "interval", interval)
}

// stopMonitoringFolder stops health monitoring for a specific folder
func (fhm *FolderHealthMonitor) stopMonitoringFolder(folderID string) {
	fhm.tickersMut.Lock()
	defer fhm.tickersMut.Unlock()

	if ticker, exists := fhm.folderTickers[folderID]; exists {
		ticker.Stop()
		delete(fhm.folderTickers, folderID)
		slog.Debug("Stopped health monitoring for folder", "folder", folderID)
	}
}

// monitorFolderHealth performs periodic health checks for a folder
func (fhm *FolderHealthMonitor) monitorFolderHealth(folderID string, ticker *time.Ticker) {
	for {
		select {
		case <-fhm.ctx.Done():
			return
		case <-ticker.C:
			fhm.performHealthCheck(folderID)
		}
	}
}

// performHealthCheck performs a health check for a specific folder with performance monitoring
func (fhm *FolderHealthMonitor) performHealthCheck(folderID string) {
	startTime := time.Now()

	// Collect initial system stats
	initialCPU, _ := cpu.Percent(0, false)
	initialMem, initialMemErr := mem.VirtualMemory()

	// Get folder configuration
	folder, ok := fhm.cfg.Folder(folderID)
	if !ok {
		// Folder no longer exists, stop monitoring
		fhm.stopMonitoringFolder(folderID)
		return
	}

	// Perform health check
	healthStatus := fhm.checkFolderHealth(folder)

	// If folder is unhealthy, try to automatically resolve common issues
	if !healthStatus.Healthy {
		// Try to create missing marker file
		markerPath := filepath.Join(folder.Path, folder.MarkerName)
		if _, err := os.Stat(markerPath); os.IsNotExist(err) {
			if err := os.WriteFile(markerPath, []byte{}, 0o644); err != nil {
				slog.Warn("Failed to create missing marker file",
					"folder", folderID,
					"error", err)
			} else {
				slog.Info("Created missing marker file",
					"folder", folderID,
					"path", markerPath)
				// Recheck health after creating marker
				healthStatus = fhm.checkFolderHealth(folder)
			}
		}
	}

	// Check for predictive issues
	fhm.checkPredictiveIssues(folderID, folder)

	// Collect final system stats
	finalCPU, _ := cpu.Percent(0, false)
	finalMem, finalMemErr := mem.VirtualMemory()

	// Calculate performance metrics
	checkDuration := time.Since(startTime)
	cpuUsage := 0.0
	if len(initialCPU) > 0 && len(finalCPU) > 0 {
		cpuUsage = (initialCPU[0] + finalCPU[0]) / 2
	}

	// Calculate memory usage with proper error handling
	var memUsage uint64
	if initialMemErr != nil || finalMemErr != nil {
		// If we can't get memory stats, use 0 as default
		memUsage = 0
		slog.Debug("Failed to collect memory statistics for folder health check",
			"folder", folderID,
			"initialError", initialMemErr,
			"finalError", finalMemErr)
	} else if initialMem != nil {
		// Calculate the difference in memory usage
		if finalMem.Used > initialMem.Used {
			memUsage = finalMem.Used - initialMem.Used
		} else {
			memUsage = 0 // Memory usage decreased or stayed the same
		}
	} else {
		// If initialMem is nil but finalMem is not, use finalMem.Used
		memUsage = finalMem.Used
	}

	// Update performance stats
	fhm.updatePerformanceStats(folderID, checkDuration, cpuUsage, memUsage, healthStatus)

	// Store the health status
	fhm.healthStatusMut.Lock()
	fhm.lastHealthStatus[folderID] = healthStatus
	fhm.healthStatusMut.Unlock()

	// Check if health status changed
	if fhm.hasHealthStatusChanged(folderID, healthStatus) {
		// Emit health status change event
		fhm.evLogger.Log(events.FolderHealthChanged, map[string]interface{}{
			"folder":       folderID,
			"healthStatus": healthStatus,
		})

		// Log health issues
		if !healthStatus.Healthy {
			for _, issue := range healthStatus.Issues {
				slog.Warn("Folder health issue detected",
					"folder", folderID,
					"issue", issue)
			}
		}
	}

	// Apply memory optimization if needed
	fhm.applyMemoryOptimization(folderID, memUsage)

	// Update monitoring interval based on folder state
	fhm.updateMonitoringInterval(folderID, folder)
}

// applyMemoryOptimization applies memory optimization techniques when usage is high
func (fhm *FolderHealthMonitor) applyMemoryOptimization(folderID string, memUsage uint64) {
	if memUsage > memoryOptimizationThreshold {
		slog.Debug("High memory usage detected, applying optimization",
			"folder", folderID,
			"memoryBytes", memUsage)

		// Trigger garbage collection to free unused memory
		runtime.GC()

		// Apply memory limiting to the folder if it supports it
		if f, ok := fhm.model.(*model).folderRunners.Get(folderID); ok {
			if memoryLimiter, ok := f.(interface{ SetMemoryLimit(limit int64) }); ok {
				// Set a reasonable memory limit based on available system memory
				if sysMem, err := mem.VirtualMemory(); err == nil {
					// Limit to 10% of available memory, but not less than 100MB
					limit := int64(sysMem.Available / 10)
					if limit < 100*1024*1024 {
						limit = 100 * 1024 * 1024
					}
					memoryLimiter.SetMemoryLimit(limit)
				}
			}
		}
	}
}

// checkPredictiveIssues checks for potential future issues based on performance trends
func (fhm *FolderHealthMonitor) checkPredictiveIssues(folderID string, folder config.FolderConfiguration) {
	fhm.perfStatsMut.RLock()
	defer fhm.perfStatsMut.RUnlock()

	stats, exists := fhm.performanceStats[folderID]
	if !exists {
		return
	}

	// Check for performance degradation trends
	if stats.CheckCount >= 5 { // Need at least 5 data points
		// Check if average check duration is increasing significantly
		if stats.AvgCheckDuration > time.Second*5 {
			slog.Warn("Folder performance degradation detected",
				"folder", folderID,
				"avgDuration", stats.AvgCheckDuration)

			// Emit predictive alert event
			fhm.evLogger.Log(events.Failure, map[string]interface{}{
				"folder":  folderID,
				"type":    "performance_degradation",
				"message": fmt.Sprintf("Folder %s health check duration is increasing: %v", folderID, stats.AvgCheckDuration),
			})
		}

		// Check if failure rate is increasing
		failureRate := float64(stats.FailedCheckCount) / float64(stats.CheckCount)
		if failureRate > 0.1 { // More than 10% failure rate
			slog.Warn("High folder failure rate detected",
				"folder", folderID,
				"failureRate", failureRate)

			// Emit predictive alert event
			fhm.evLogger.Log(events.Failure, map[string]interface{}{
				"folder":  folderID,
				"type":    "high_failure_rate",
				"message": fmt.Sprintf("Folder %s has high failure rate: %.2f%%", folderID, failureRate*100),
			})
		}
	}

	// Check for resource usage issues and apply throttling if needed
	if folder.ThrottlingEnabled {
		// Check CPU usage
		if stats.CPUUsagePercent > float64(folder.MaxCPUUsagePercent) {
			slog.Warn("High CPU usage detected for folder, throttling may be needed",
				"folder", folderID,
				"cpuPercent", stats.CPUUsagePercent,
				"maxAllowed", folder.MaxCPUUsagePercent)

			// Emit throttling alert event
			fhm.evLogger.Log(events.Failure, map[string]interface{}{
				"folder":  folderID,
				"type":    "high_cpu_usage",
				"message": fmt.Sprintf("Folder %s is using high CPU: %.2f%% (max allowed: %d%%)", folderID, stats.CPUUsagePercent, folder.MaxCPUUsagePercent),
			})
		}

		// Check memory usage (convert MB to bytes)
		maxMemoryBytes := uint64(folder.MaxMemoryUsageMB) * 1024 * 1024

		// Add sanity check to prevent extremely high memory usage values from being reported
		if stats.MemoryUsageBytes > maxMemoryBytes && stats.MemoryUsageBytes < 100*1024*1024*1024 { // Less than 100GB
			slog.Warn("High memory usage detected for folder, throttling may be needed",
				"folder", folderID,
				"memoryBytes", stats.MemoryUsageBytes,
				"maxAllowedMB", folder.MaxMemoryUsageMB)

			// Emit throttling alert event
			fhm.evLogger.Log(events.Failure, map[string]interface{}{
				"folder":  folderID,
				"type":    "high_memory_usage",
				"message": fmt.Sprintf("Folder %s is using high memory: %d bytes (%d MB) (max allowed: %d MB)", folderID, stats.MemoryUsageBytes, stats.MemoryUsageBytes/1024/1024, folder.MaxMemoryUsageMB),
			})
		}
	} else if stats.CPUUsagePercent > 80 {
		slog.Warn("High CPU usage detected for folder",
			"folder", folderID,
			"cpuPercent", stats.CPUUsagePercent)

		// Emit predictive alert event
		fhm.evLogger.Log(events.Failure, map[string]interface{}{
			"folder":  folderID,
			"type":    "high_cpu_usage",
			"message": fmt.Sprintf("Folder %s is using high CPU: %.2f%%", folderID, stats.CPUUsagePercent),
		})
	}

	// Add sanity check for memory usage reporting
	if stats.MemoryUsageBytes > 1024*1024*1024 && stats.MemoryUsageBytes < 100*1024*1024*1024 { // Between 1GB and 100GB
		slog.Warn("High memory usage detected for folder",
			"folder", folderID,
			"memoryBytes", stats.MemoryUsageBytes)

		// Emit predictive alert event
		fhm.evLogger.Log(events.Failure, map[string]interface{}{
			"folder":  folderID,
			"type":    "high_memory_usage",
			"message": fmt.Sprintf("Folder %s is using high memory: %d bytes", folderID, stats.MemoryUsageBytes),
		})
	}
}

// updatePerformanceStats updates the performance statistics for a folder
func (fhm *FolderHealthMonitor) updatePerformanceStats(folderID string, duration time.Duration, cpuUsage float64, memUsage uint64, healthStatus config.FolderHealthStatus) {
	fhm.perfStatsMut.Lock()
	defer fhm.perfStatsMut.Unlock()

	stats, exists := fhm.performanceStats[folderID]
	if !exists {
		stats = FolderPerformanceStats{
			CheckCount:       0,
			FailedCheckCount: 0,
			AvgCheckDuration: 0,
		}
	}

	stats.LastCheckTime = time.Now()
	stats.CheckDuration = duration
	stats.CPUUsagePercent = cpuUsage
	stats.MemoryUsageBytes = memUsage
	stats.CheckCount++

	if !healthStatus.Healthy {
		stats.FailedCheckCount++
		if len(healthStatus.Issues) > 0 {
			stats.LastError = healthStatus.Issues[0]
		}
	}

	// Calculate average check duration
	totalDuration := time.Duration(stats.AvgCheckDuration.Nanoseconds()*int64(stats.CheckCount-1)) + duration
	stats.AvgCheckDuration = time.Duration(totalDuration.Nanoseconds() / int64(stats.CheckCount))

	fhm.performanceStats[folderID] = stats
}

// checkFolderHealth performs a health check on a folder configuration
func (fhm *FolderHealthMonitor) checkFolderHealth(folder config.FolderConfiguration) config.FolderHealthStatus {
	startTime := time.Now()

	// Perform the actual health check
	err := folder.CheckPath()

	// Calculate check duration
	checkDuration := time.Since(startTime)

	// Create health status
	healthStatus := config.FolderHealthStatus{
		Healthy:       err == nil,
		CheckTime:     time.Now(),
		LastChecked:   time.Now(),
		CheckDuration: checkDuration,
	}

	if err != nil {
		healthStatus.Issues = append(healthStatus.Issues, err.Error())
	}

	return healthStatus
}

// getHealthCheckInterval determines the appropriate health check interval based on folder state
func (fhm *FolderHealthMonitor) getHealthCheckInterval(folder config.FolderConfiguration, folderID string) time.Duration {
	// Check if folder has custom health check interval
	if folder.HealthCheckIntervalS > 0 {
		return time.Duration(folder.HealthCheckIntervalS) * time.Second
	}

	// Default behavior based on folder state
	if folder.Paused {
		return pausedHealthCheckInterval
	}

	// Check last performance stats to determine if folder is active
	fhm.perfStatsMut.RLock()
	stats, exists := fhm.performanceStats[folderID]
	fhm.perfStatsMut.RUnlock()

	if exists && stats.LastCheckTime.After(time.Now().Add(-10*time.Minute)) {
		// Folder has been recently active
		return defaultHealthCheckInterval
	}

	// Folder is likely inactive
	return inactiveHealthCheckInterval
}

// updateMonitoringInterval updates the monitoring interval based on folder health
func (fhm *FolderHealthMonitor) updateMonitoringInterval(folderID string, folder config.FolderConfiguration) {
	fhm.tickersMut.Lock()
	defer fhm.tickersMut.Unlock()

	ticker, exists := fhm.folderTickers[folderID]
	if !exists {
		return
	}

	// Determine new interval
	newInterval := fhm.getHealthCheckInterval(folder, folderID)

	// Only update if interval has changed significantly
	if currentInterval := ticker.C; currentInterval != nil {
		// We can't directly check the ticker's duration, so we'll reset if needed
		ticker.Reset(newInterval)
		slog.Debug("Updated health check interval",
			"folder", folderID,
			"interval", newInterval)
	}
}

// hasHealthStatusChanged checks if the health status has changed
func (fhm *FolderHealthMonitor) hasHealthStatusChanged(folderID string, newStatus config.FolderHealthStatus) bool {
	fhm.healthStatusMut.RLock()
	defer fhm.healthStatusMut.RUnlock()

	oldStatus, exists := fhm.lastHealthStatus[folderID]
	if !exists {
		return true
	}

	// Compare health status
	return oldStatus.Healthy != newStatus.Healthy
}

// cleanup stops all monitoring and releases resources
func (fhm *FolderHealthMonitor) cleanup() {
	fhm.tickersMut.Lock()
	defer fhm.tickersMut.Unlock()

	for _, ticker := range fhm.folderTickers {
		ticker.Stop()
	}

	fhm.folderTickers = make(map[string]*time.Ticker)
}

// CommitConfiguration implements config.Committer
func (fhm *FolderHealthMonitor) CommitConfiguration(from, to config.Configuration) bool {
	// Handle folder additions/removals
	fromFolders := make(map[string]config.FolderConfiguration)
	toFolders := make(map[string]config.FolderConfiguration)

	for _, folder := range from.Folders {
		fromFolders[folder.ID] = folder
	}

	for _, folder := range to.Folders {
		toFolders[folder.ID] = folder
	}

	// Stop monitoring removed folders
	for id := range fromFolders {
		if _, exists := toFolders[id]; !exists {
			fhm.stopMonitoringFolder(id)
		}
	}

	// Start monitoring new folders
	for id := range toFolders {
		if _, exists := fromFolders[id]; !exists {
			fhm.startMonitoringFolder(id)
		}
	}

	return true
}

// String implements fmt.Stringer
func (fhm *FolderHealthMonitor) String() string {
	return "FolderHealthMonitor"
}

// GetAllFoldersHealthStatus returns the health status of all folders
func (fhm *FolderHealthMonitor) GetAllFoldersHealthStatus() map[string]config.FolderHealthStatus {
	fhm.healthStatusMut.RLock()
	defer fhm.healthStatusMut.RUnlock()

	// Create a copy of the map to avoid race conditions
	result := make(map[string]config.FolderHealthStatus, len(fhm.lastHealthStatus))
	for k, v := range fhm.lastHealthStatus {
		result[k] = v
	}
	return result
}

// GetFolderHealthStatus returns the health status of a specific folder
func (fhm *FolderHealthMonitor) GetFolderHealthStatus(folderID string) (config.FolderHealthStatus, bool) {
	fhm.healthStatusMut.RLock()
	defer fhm.healthStatusMut.RUnlock()

	status, exists := fhm.lastHealthStatus[folderID]
	return status, exists
}

// GetAllFoldersPerformanceStats returns performance statistics for all folders
func (fhm *FolderHealthMonitor) GetAllFoldersPerformanceStats() map[string]FolderPerformanceStats {
	fhm.perfStatsMut.RLock()
	defer fhm.perfStatsMut.RUnlock()

	// Create a copy of the map to avoid race conditions
	result := make(map[string]FolderPerformanceStats, len(fhm.performanceStats))
	for k, v := range fhm.performanceStats {
		result[k] = v
	}
	return result
}

// GetFolderPerformanceStats returns performance statistics for a specific folder
func (fhm *FolderHealthMonitor) GetFolderPerformanceStats(folderID string) (FolderPerformanceStats, bool) {
	fhm.perfStatsMut.RLock()
	defer fhm.perfStatsMut.RUnlock()

	stats, exists := fhm.performanceStats[folderID]
	return stats, exists
}
