// Copyright (C) 2016 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !(solaris && !cgo) && !(darwin && !cgo) && !(darwin && kqueue) && !(android && amd64)
// +build !solaris cgo
// +build !darwin cgo
// +build !darwin !kqueue
// +build !android !amd64

package fs

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/syncthing/notify"
	"github.com/syncthing/syncthing/lib/build"
)

// Notify does not block on sending to channel, so the channel must be buffered.
// The actual number is magic.
// Not meant to be changed, but must be changeable for tests
var backendBuffer = 500

// For Windows systems with large filesets, we use a larger buffer to prevent
// event overflow which can cause file change notifications to be missed.
func init() {
	if build.IsWindows {
		// Use a larger buffer on Windows to handle large filesets better
		backendBuffer = 2000
	}
}

// overflowTracker keeps track of buffer overflow events for adaptive management
type overflowTracker struct {
	mu                   sync.Mutex
	count                int
	lastOverflow         time.Time
	frequency            time.Duration
	adaptiveBuffer       int
	minBufferSize        int
	maxBufferSize        int
	resizeFactor         float64
	consecutiveOverflows int
	overflowHistory      []time.Time
	avgOverflowInterval  time.Duration
	overflowRate         float64
}

// newOverflowTracker creates a new overflow tracker with default configuration
func newOverflowTracker() *overflowTracker {
	return &overflowTracker{
		count:          0,
		lastOverflow:   time.Time{},
		frequency:      0,
		adaptiveBuffer: 1000,
		minBufferSize:  500,
		maxBufferSize:  10000,
		resizeFactor:   1.5,
		overflowHistory: make([]time.Time, 0, 100), // Keep last 100 overflow timestamps
	}
}

// newOverflowTrackerWithConfig creates a new overflow tracker with custom configuration
func newOverflowTrackerWithConfig(minBufferSize, maxBufferSize, adaptiveBuffer int) *overflowTracker {
	ot := newOverflowTracker()
	ot.minBufferSize = minBufferSize
	ot.maxBufferSize = maxBufferSize
	ot.adaptiveBuffer = adaptiveBuffer
	ot.resizeFactor = float64(maxBufferSize) / float64(minBufferSize) / 10.0
	if ot.resizeFactor < 1.1 {
		ot.resizeFactor = 1.1
	}
	return ot
}

// recordOverflow records an overflow event and updates tracking metrics
func (ot *overflowTracker) recordOverflow() {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	now := time.Now()
	ot.count++
	ot.consecutiveOverflows++

	// Add to overflow history
	ot.overflowHistory = append(ot.overflowHistory, now)
	
	// Keep only the last 100 timestamps
	if len(ot.overflowHistory) > 100 {
		ot.overflowHistory = ot.overflowHistory[1:]
	}

	if !ot.lastOverflow.IsZero() {
		// Calculate the time between overflows
		ot.frequency = now.Sub(ot.lastOverflow)
	}

	ot.lastOverflow = now
	
	// Calculate average overflow interval
	ot.calculateAvgOverflowInterval()
	
	// Calculate overflow rate (overflows per minute)
	ot.calculateOverflowRate()
}

// shouldIncreaseBuffer determines if we should increase the buffer size based on overflow patterns
func (ot *overflowTracker) shouldIncreaseBuffer() bool {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	// If we have frequent overflows (less than 30 seconds between them) and we haven't maxed out the buffer
	return ot.frequency > 0 && ot.frequency < 30*time.Second && ot.adaptiveBuffer < ot.maxBufferSize && ot.consecutiveOverflows > 2
}

// shouldDecreaseBuffer determines if we should decrease the buffer size based on inactivity
func (ot *overflowTracker) shouldDecreaseBuffer(lastEvent time.Time) bool {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	// If we haven't had an overflow for more than 5 minutes and buffer is more than 2x the minimum
	inactivityDuration := time.Since(ot.lastOverflow)
	eventInactivity := time.Since(lastEvent)
	
	return inactivityDuration > 5*time.Minute && 
		   eventInactivity > 10*time.Minute && 
		   ot.adaptiveBuffer > ot.minBufferSize*2
}

// getSystemPressure calculates a normalized system pressure value (0.0 to 1.0)
func (ot *overflowTracker) getSystemPressure() float64 {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	// Calculate pressure based on overflow rate, buffer utilization, and consecutive overflows
	ratePressure := ot.overflowRate / 10.0 // Normalize to 0-1 range (assuming 10 overflows/min is high)
	bufferPressure := float64(ot.adaptiveBuffer-ot.minBufferSize) / float64(ot.maxBufferSize-ot.minBufferSize)
	overflowPressure := float64(ot.consecutiveOverflows) / 20.0 // Normalize to 0-1 range (assuming 20 consecutive is high)

	// Weighted average
	pressure := (ratePressure*0.4 + bufferPressure*0.3 + overflowPressure*0.3)
	
	// Clamp to 0-1 range
	if pressure > 1.0 {
		pressure = 1.0
	}
	if pressure < 0.0 {
		pressure = 0.0
	}
	
	return pressure
}

// increaseBuffer increases the buffer size based on system pressure
func (ot *overflowTracker) increaseBuffer() int {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	// Get adaptive resize factor based on system pressure
	factor := ot.getAdaptiveResizeFactor()
	
	// Increase buffer by the adaptive factor
	ot.adaptiveBuffer = int(float64(ot.adaptiveBuffer) * factor)
	
	// Cap at maximum buffer size
	if ot.adaptiveBuffer > ot.maxBufferSize {
		ot.adaptiveBuffer = ot.maxBufferSize
	}

	return ot.adaptiveBuffer
}

// decreaseBuffer decreases the buffer size based on system pressure
func (ot *overflowTracker) decreaseBuffer() int {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	// Decrease buffer by resize factor
	ot.adaptiveBuffer = int(float64(ot.adaptiveBuffer) / ot.resizeFactor)
	
	// Ensure it doesn't go below minimum buffer size
	if ot.adaptiveBuffer < ot.minBufferSize {
		ot.adaptiveBuffer = ot.minBufferSize
	}

	return ot.adaptiveBuffer
}

// getOptimalBufferSize calculates an optimal buffer size based on folder size
func (ot *overflowTracker) getOptimalBufferSize(fileCount int) int {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	// Calculate buffer size based on file count with logarithmic scaling
	// This prevents extremely large buffers for huge folders
	optimalSize := int(float64(ot.minBufferSize) * (1 + (float64(fileCount) / 1000.0)))
	
	// Apply logarithmic scaling for very large folders
	if fileCount > 10000 {
		optimalSize = int(float64(ot.minBufferSize) * (1 + (10 * (1 + (float64(fileCount) / 100000.0)))))
	}
	
	// Clamp between min and max buffer sizes
	if optimalSize < ot.minBufferSize {
		optimalSize = ot.minBufferSize
	}
	if optimalSize > ot.maxBufferSize {
		optimalSize = ot.maxBufferSize
	}

	return optimalSize
}

// updateBufferSizeBasedOnResources dynamically adjusts buffer size based on system resources
func (ot *overflowTracker) updateBufferSizeBasedOnResources(fileCount int) int {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	// Get the optimal buffer size for this folder
	optimalSize := ot.getOptimalBufferSize(fileCount)
	
	// Get current system pressure
	pressure := ot.getSystemPressure()
	
	// Adjust buffer size based on pressure
	if pressure > 0.7 {
		// High pressure - increase buffer toward optimal size
		ot.adaptiveBuffer = int(float64(ot.adaptiveBuffer)*(1-pressure) + float64(optimalSize)*pressure)
	} else if pressure < 0.3 {
		// Low pressure - potentially decrease buffer
		ot.adaptiveBuffer = int(float64(ot.adaptiveBuffer)*0.9 + float64(optimalSize)*0.1)
	} else {
		// Moderate pressure - slowly adjust toward optimal
		ot.adaptiveBuffer = int(float64(ot.adaptiveBuffer)*0.95 + float64(optimalSize)*0.05)
	}
	
	// Clamp between min and max buffer sizes
	if ot.adaptiveBuffer < ot.minBufferSize {
		ot.adaptiveBuffer = ot.minBufferSize
	}
	if ot.adaptiveBuffer > ot.maxBufferSize {
		ot.adaptiveBuffer = ot.maxBufferSize
	}

	return ot.adaptiveBuffer
}

// getAdaptiveResizeFactor calculates a resize factor based on system pressure
func (ot *overflowTracker) getAdaptiveResizeFactor() float64 {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	pressure := ot.getSystemPressure()
	
	// Return different factors based on pressure
	if pressure > 0.8 {
		return 2.0 // High pressure - aggressive resize
	} else if pressure > 0.6 {
		return 1.5 // Medium-high pressure
	} else if pressure > 0.4 {
		return 1.2 // Medium pressure
	} else {
		return 1.1 // Low pressure - conservative resize
	}
}

// getBufferSize returns the current adaptive buffer size
func (ot *overflowTracker) getBufferSize() int {
	ot.mu.Lock()
	defer ot.mu.Unlock()
	return ot.adaptiveBuffer
}

// resetConsecutiveOverflows resets the consecutive overflow counter
func (ot *overflowTracker) resetConsecutiveOverflows() {
	ot.mu.Lock()
	defer ot.mu.Unlock()
	ot.consecutiveOverflows = 0
}

// calculateAvgOverflowInterval calculates the average time between overflows
func (ot *overflowTracker) calculateAvgOverflowInterval() {
	if len(ot.overflowHistory) >= 2 {
		totalDuration := time.Duration(0)
		for i := 1; i < len(ot.overflowHistory); i++ {
			totalDuration += ot.overflowHistory[i].Sub(ot.overflowHistory[i-1])
		}
		ot.avgOverflowInterval = totalDuration / time.Duration(len(ot.overflowHistory)-1)
	}
}

// calculateOverflowRate calculates the overflow rate (overflows per minute)
func (ot *overflowTracker) calculateOverflowRate() {
	if len(ot.overflowHistory) >= 2 {
		first := ot.overflowHistory[0]
		last := ot.overflowHistory[len(ot.overflowHistory)-1]
		duration := last.Sub(first)
		if duration > 0 {
			overflowsPerMinute := float64(len(ot.overflowHistory)-1) / (duration.Minutes())
			ot.overflowRate = overflowsPerMinute
		}
	}
}

// watchMetrics tracks performance metrics for file watching
type watchMetrics struct {
	mu              sync.Mutex
	eventsProcessed int64
	eventsDropped   int64
	overflows       int64
	startTime       time.Time
	lastEvent       time.Time
}

// newWatchMetrics creates a new watch metrics tracker
func newWatchMetrics() *watchMetrics {
	return &watchMetrics{
		startTime: time.Now(),
	}
}

// recordEvent records that an event was processed
func (wm *watchMetrics) recordEvent() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.eventsProcessed++
	wm.lastEvent = time.Now()
}

// recordDroppedEvent records that an event was dropped
func (wm *watchMetrics) recordDroppedEvent() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.eventsDropped++
}

// recordOverflow records a buffer overflow
func (wm *watchMetrics) recordOverflow() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.overflows++
}

// getMetrics returns current metrics
func (wm *watchMetrics) getMetrics() (eventsProcessed, eventsDropped, overflows int64, uptime, timeSinceLastEvent time.Duration) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	now := time.Now()
	uptime = now.Sub(wm.startTime)
	timeSinceLastEvent = now.Sub(wm.lastEvent)

	return wm.eventsProcessed, wm.eventsDropped, wm.overflows, uptime, timeSinceLastEvent
}

// logMetrics periodically logs metrics for monitoring
func (wm *watchMetrics) logMetrics(fs *BasicFilesystem, name string) {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			eventsProcessed, eventsDropped, overflows, uptime, timeSinceLastEvent := wm.getMetrics()
			l.Debugln(fs.Type(), fs.URI(), "Watch metrics for", name, "- Processed:", eventsProcessed,
				"Dropped:", eventsDropped, "Overflows:", overflows,
				"Uptime:", uptime.Truncate(time.Second),
				"Idle:", timeSinceLastEvent.Truncate(time.Second))
		}
	}()
}

// countFilesInDirectory counts the number of files in a directory recursively
func countFilesInDirectory(fs *BasicFilesystem, dir string) (int, error) {
	count := 0
	err := fs.Walk(dir, func(path string, info FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count, err
}

// checkLargeFolder analyzes a folder and provides recommendations if it's large
func checkLargeFolder(fs *BasicFilesystem, name string) {
	// Count files in the folder
	fileCount, err := countFilesInDirectory(fs, name)
	if err != nil {
		l.Debugln(fs.Type(), fs.URI(), "Watch: Could not count files in", name, "-", err)
		return
	}

	// If the folder has many files, provide recommendations
	if fileCount > 10000 {
		l.Debugln(fs.Type(), fs.URI(), "Watch: Folder", name, "contains", fileCount, "files which may cause performance issues.",
			"Consider excluding temporary files, build artifacts, or using more specific folder paths.")
	} else if fileCount > 5000 {
		l.Debugln(fs.Type(), fs.URI(), "Watch: Folder", name, "contains", fileCount, "files.",
			"Monitor performance and consider exclusions if issues occur.")
	} else if fileCount > 1000 {
		l.Debugln(fs.Type(), fs.URI(), "Watch: Folder", name, "contains", fileCount, "files.")
	}
}

func (f *BasicFilesystem) Watch(name string, ignore Matcher, ctx context.Context, ignorePerms bool) (<-chan Event, <-chan error, error) {
	watchPath, roots, err := f.watchPaths(name)
	if err != nil {
		return nil, nil, err
	}

	// Proactively check if this is a large folder and provide recommendations
	checkLargeFolder(f, name)

	outChan := make(chan Event)
	backendChan := make(chan notify.EventInfo, backendBuffer)

	eventMask := subEventMask
	if !ignorePerms {
		eventMask |= permEventMask
	}

	absShouldIgnore := func(absPath string) bool {
		if !utf8.ValidString(absPath) {
			return true
		}

		rel, err := f.unrootedChecked(absPath, roots)
		if err != nil {
			return true
		}
		return ignore.Match(rel).CanSkipDir()
	}
	err = notify.WatchWithFilter(watchPath, backendChan, absShouldIgnore, eventMask)
	if err != nil {
		notify.Stop(backendChan)
		// Add Windows-specific error messages
		if build.IsWindows && isWindowsWatchingError(err) {
			l.Debugln(f.Type(), f.URI(), "Watch: Windows file watching limitation encountered. Consider excluding large directories or using manual scans.")
		}
		if reachedMaxUserWatches(err) {
			err = errors.New("failed to set up inotify handler. Please increase inotify limits, see https://docs.syncthing.net/users/faq.html#inotify-limits")
		}
		return nil, nil, err
	}

	errChan := make(chan error)
	go f.watchLoop(ctx, name, roots, backendChan, outChan, errChan, ignore)

	return outChan, errChan, nil
}

// isWindowsWatchingError checks if an error is a Windows-specific watching error
func isWindowsWatchingError(err error) bool {
	// Common Windows file watching errors
	errorString := err.Error()
	windowsErrors := []string{
		"parameter is incorrect",
		"operation was cancelled",
		"access is denied",
		"file system does not support file change notifications",
	}

	for _, winErr := range windowsErrors {
		if strings.Contains(strings.ToLower(errorString), winErr) {
			return true
		}
	}

	return false
}

func (f *BasicFilesystem) watchLoop(ctx context.Context, name string, roots []string, backendChan chan notify.EventInfo, outChan chan<- Event, errChan chan<- error, ignore Matcher) {
	// Initialize overflow tracking for adaptive buffer management
	overflowTracker := newOverflowTracker()

	// Initialize metrics tracking
	metrics := newWatchMetrics()
	metrics.logMetrics(f, name) // Start periodic logging

	for {
		// Detect channel overflow
		if len(backendChan) == backendBuffer {
		outer:
			for {
				select {
				case <-backendChan:
					metrics.recordDroppedEvent() // Record dropped events
				default:
					break outer
				}
			}
			// Record the overflow for adaptive management
			overflowTracker.recordOverflow()
			metrics.recordOverflow() // Record for metrics

			// When next scheduling a scan, do it on the entire folder as events have been lost.
			outChan <- Event{Name: name, Type: NonRemove}
			l.Debugln(f.Type(), f.URI(), "Watch: Event overflow, send \".\"")
			// Log a warning when buffer overflows to help with debugging
			l.Debugln(f.Type(), f.URI(), "Watch: Event buffer overflow detected. Consider increasing buffer size or reducing file change frequency.")

			// Check if we should increase the buffer size based on overflow patterns
			if overflowTracker.shouldIncreaseBuffer() {
				newSize := overflowTracker.increaseBuffer()
				l.Debugln(f.Type(), f.URI(), "Watch: Increasing adaptive buffer size to", newSize, "due to frequent overflows")
			}
		}

		select {
		case ev := <-backendChan:
			evPath := ev.Path()

			if !utf8.ValidString(evPath) {
				l.Debugln(f.Type(), f.URI(), "Watch: Ignoring invalid UTF-8")
				continue
			}

			relPath, err := f.unrootedChecked(evPath, roots)
			if err != nil {
				select {
				case errChan <- err:
					l.Debugln(f.Type(), f.URI(), "Watch: Sending error", err)
				case <-ctx.Done():
				}
				notify.Stop(backendChan)
				l.Debugln(f.Type(), f.URI(), "Watch: Stopped due to", err)
				return
			}

			if ignore.Match(relPath).IsIgnored() {
				l.Debugln(f.Type(), f.URI(), "Watch: Ignoring", relPath)
				continue
			}
			evType := f.eventType(ev.Event())
			select {
			case outChan <- Event{Name: relPath, Type: evType}:
				metrics.recordEvent() // Record processed event
				l.Debugln(f.Type(), f.URI(), "Watch: Sending", relPath, evType)
			case <-ctx.Done():
				notify.Stop(backendChan)
				l.Debugln(f.Type(), f.URI(), "Watch: Stopped")
				return
			}
		case <-ctx.Done():
			notify.Stop(backendChan)
			// Log final metrics when stopping
			eventsProcessed, eventsDropped, overflows, _, _ := metrics.getMetrics()
			l.Debugln(f.Type(), f.URI(), "Watch: Stopped. Final metrics - Processed:", eventsProcessed,
				"Dropped:", eventsDropped, "Overflows:", overflows)
			return
		}
	}
}

func (*BasicFilesystem) eventType(notifyType notify.Event) EventType {
	if notifyType&rmEventMask != 0 {
		return Remove
	}
	return NonRemove
}
