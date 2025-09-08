// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build android
// +build android

package fs

import (
	"context"
	"errors"
	"log"
	"runtime"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/shirou/gopsutil/v4/mem"
	"github.com/syncthing/notify"
)

// androidWatcher implements an Android-specific watcher with resource-conscious optimizations
type androidWatcher struct {
	fs              *BasicFilesystem
	ctx             context.Context
	cancel          context.CancelFunc
	backendChan     chan notify.EventInfo
	outChan         chan<- Event
	errChan         chan<- error
	ignore          Matcher
	roots           []string
	bufferSize      int
	fileCount       int
	folderName      string
	overflowTracker *overflowTracker
	watchMetrics    *watchMetrics

	mut sync.Mutex
}

// newAndroidWatcher creates a new Android watcher
func newAndroidWatcher(fs *BasicFilesystem, name string, ignore Matcher, ctx context.Context, ignorePerms bool, fileCount int) (*androidWatcher, error) {
	// Count files in the folder to determine optimal buffer size
	// fileCount is already provided as a parameter

	// Use platform-specific buffer size optimized for Android
	bufferSize := getAndroidOptimalBufferSize(fileCount)

	// Initialize overflow tracking for adaptive buffer management
	overflowTracker := newOverflowTracker()

	// Adjust buffer size based on system resources and folder characteristics
	bufferSize = overflowTracker.getOptimalBufferSize(fileCount)

	backendChan := make(chan notify.EventInfo, bufferSize)

	// Use platform-specific event mask
	eventMask := notify.All
	if ignorePerms {
		eventMask &^= notify.Write
	}

	watchPath, roots, err := fs.watchPaths(name)
	if err != nil {
		return nil, err
	}

	absShouldIgnore := func(absPath string) bool {
		if !utf8.ValidString(absPath) {
			return true
		}

		rel, err := fs.unrootedChecked(absPath, roots)
		if err != nil {
			return true
		}
		return ignore.Match(rel).CanSkipDir()
	}

	err = notify.WatchWithFilter(watchPath, backendChan, absShouldIgnore, eventMask)
	if err != nil {
		notify.Stop(backendChan)
		// Check for inotify limits
		if reachedMaxUserWatches(err) {
			err = errors.New("failed to set up inotify handler. Please increase inotify limits, see https://docs.syncthing.net/users/faq.html#inotify-limits")
		}
		return nil, err
	}

	// Create context with cancel
	ctx, cancel := context.WithCancel(ctx)

	w := &androidWatcher{
		fs:              fs,
		ctx:             ctx,
		cancel:          cancel,
		backendChan:     backendChan,
		outChan:         make(chan Event),
		errChan:         make(chan error),
		ignore:          ignore,
		roots:           roots,
		bufferSize:      bufferSize,
		fileCount:       fileCount,
		folderName:      name,
		overflowTracker: overflowTracker,
		watchMetrics:    newWatchMetrics(),
	}

	// Apply Android-specific optimizations
	w.optimizeForAndroid()

	return w, nil
}

// watchLoop runs the main event loop for the Android watcher
func (w *androidWatcher) watchLoop() {
	// Initialize metrics tracking
	w.watchMetrics.logMetrics(w.fs, w.folderName) // Start periodic logging

	// Start Prometheus metrics updates
	metricsUpdateTicker := time.NewTicker(1 * time.Minute)
	go func() {
		for range metricsUpdateTicker.C {
			w.watchMetrics.updatePrometheusMetrics(w.fs, w.overflowTracker)
		}
	}()
	defer metricsUpdateTicker.Stop()

	lastProcessedEvent := time.Now()

	// Periodically re-evaluate buffer size based on system resources
	resourceCheckTicker := time.NewTicker(10 * time.Minute)
	defer resourceCheckTicker.Stop()

	for {
		// Detect channel overflow
		if len(w.backendChan) == cap(w.backendChan) {
		outer:
			for {
				select {
				case <-w.backendChan:
					w.watchMetrics.recordDroppedEvent() // Record dropped events
				default:
					break outer
				}
			}
			// Record the overflow for adaptive management
			w.overflowTracker.recordOverflow()
			w.watchMetrics.recordOverflow() // Record for metrics

			// When next scheduling a scan, do it on the entire folder as events have been lost.
			w.outChan <- Event{Name: w.folderName, Type: NonRemove}
			log.Println(w.fs.Type(), w.fs.URI(), "Watch: Event overflow, send \".\"")
			// Log a warning when buffer overflows to help with debugging
			log.Println(w.fs.Type(), w.fs.URI(), "Watch: Event buffer overflow detected. Consider increasing buffer size or reducing file change frequency.")

			// Check if we should increase the buffer size based on overflow patterns
			if w.overflowTracker.shouldIncreaseBuffer() {
				newSize := w.overflowTracker.increaseBuffer()
				metricBufferResizes.WithLabelValues(w.fs.URI()).Inc()
				log.Println(w.fs.Type(), w.fs.URI(), "Watch: Increasing adaptive buffer size to", newSize, "due to frequent overflows")
			}
		}

		// Check if we should decrease the buffer size based on low activity
		if w.overflowTracker.shouldDecreaseBuffer(lastProcessedEvent) {
			newSize := w.overflowTracker.decreaseBuffer()
			metricBufferResizes.WithLabelValues(w.fs.URI()).Inc()
			log.Println(w.fs.Type(), w.fs.URI(), "Watch: Decreasing adaptive buffer size to", newSize, "due to low activity")
		}

		select {
		case <-resourceCheckTicker.C:
			// Periodically re-evaluate buffer size based on system resources
			newSize := w.overflowTracker.updateBufferSizeBasedOnResources(w.fileCount)
			if newSize != cap(w.backendChan) {
				metricBufferResizes.WithLabelValues(w.fs.URI()).Inc()
				log.Println(w.fs.Type(), w.fs.URI(), "Watch: Adjusted buffer size to", newSize, "based on system resources")
			}

		case ev := <-w.backendChan:
			evPath := ev.Path()
			lastProcessedEvent = time.Now()

			if !utf8.ValidString(evPath) {
				log.Println(w.fs.Type(), w.fs.URI(), "Watch: Ignoring invalid UTF-8")
				continue
			}

			relPath, err := w.fs.unrootedChecked(evPath, w.roots)
			if err != nil {
				select {
				case w.errChan <- err:
					log.Println(w.fs.Type(), w.fs.URI(), "Watch: Sending error", err)
				case <-w.ctx.Done():
				}
				notify.Stop(w.backendChan)
				log.Println(w.fs.Type(), w.fs.URI(), "Watch: Stopped due to", err)
				return
			}

			if w.ignore.Match(relPath).IsIgnored() {
				log.Println(w.fs.Type(), w.fs.URI(), "Watch: Ignoring", relPath)
				continue
			}
			evType := w.fs.eventType(ev.Event())
			select {
			case w.outChan <- Event{Name: relPath, Type: evType}:
				w.watchMetrics.recordEvent() // Record processed event
				log.Println(w.fs.Type(), w.fs.URI(), "Watch: Sending", relPath, evType)
			case <-w.ctx.Done():
				notify.Stop(w.backendChan)
				log.Println(w.fs.Type(), w.fs.URI(), "Watch: Stopped")
				return
			}
		case <-w.ctx.Done():
			notify.Stop(w.backendChan)
			// Log final metrics when stopping
			_, _, overflows, _, _ := w.watchMetrics.getMetrics()
			log.Println(w.fs.Type(), w.fs.URI(), "Watch: Stopped. Final metrics - Overflows:", overflows)
			return
		}
	}
}

// updatePrometheusMetrics periodically updates Prometheus metrics
func (w *androidWatcher) updatePrometheusMetrics() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.watchMetrics.updatePrometheusMetrics(w.fs, w.overflowTracker)
		case <-w.ctx.Done():
			return
		}
	}
}

// getAndroidOptimalBufferSize returns the optimal buffer size for Android
// Android devices often have limited resources, so we use smaller buffers
func getAndroidOptimalBufferSize(fileCount int) int {
	// Android inotify benefits from smaller buffers to conserve resources
	baseSize := 500

	// Adjust based on folder size, but keep it conservative for Android
	if fileCount > 50000 {
		// Very large folder
		return baseSize * 3
	} else if fileCount > 10000 {
		// Large folder
		return baseSize * 2
	} else if fileCount < 1000 {
		// Small folder
		return baseSize
	}

	return baseSize
}

// Android-specific optimizations for resource-constrained environments
func (w *androidWatcher) optimizeForAndroid() {
	// Android-specific optimizations to conserve system resources

	// Get system memory information
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		// If system has limited memory, use smaller buffers
		if memInfo.Available < 512*1024*1024 { // Less than 512MB available
			w.bufferSize = int(float64(w.bufferSize) * 0.75)
		} else if memInfo.Available > 2*1024*1024*1024 { // More than 2GB available
			w.bufferSize = int(float64(w.bufferSize) * 1.25)
		}
	}

	// Adjust based on number of CPU cores (Android devices often have fewer cores)
	numCPU := runtime.NumCPU()
	if numCPU <= 2 {
		// Low core count systems need smaller buffers to avoid overwhelming
		w.bufferSize = int(float64(w.bufferSize) * 0.75)
	} else if numCPU >= 8 {
		// High core count systems can handle more events
		w.bufferSize = int(float64(w.bufferSize) * 1.1)
	}

	// Always cap the buffer size on Android to conserve resources
	if w.bufferSize > 5000 {
		w.bufferSize = 5000
	}
}
