// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build linux
// +build linux

package fs

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/shirou/gopsutil/v4/mem"
	"github.com/syncthing/notify"
)

// linuxWatcher implements a Linux-specific watcher using inotify
type linuxWatcher struct {
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

// newLinuxWatcher creates a new Linux watcher
func newLinuxWatcher(fs *BasicFilesystem, name string, ignore Matcher, ctx context.Context, ignorePerms bool, fileCount int) (*linuxWatcher, error) {
	// Count files in the folder to determine optimal buffer size
	// fileCount is already provided as a parameter

	// Use platform-specific buffer size
	bufferSize := getLinuxOptimalBufferSize(fileCount)

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
		return nil, err
	}

	// Create context with cancel
	ctx, cancel := context.WithCancel(ctx)

	w := &linuxWatcher{
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

	// Apply Linux-specific optimizations
	w.optimizeForLinux()

	return w, nil
}

// watchLoop runs the main event loop for the Linux watcher
func (w *linuxWatcher) watchLoop() {
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
			w.watchMetrics.getMetrics()
			log.Println(w.fs.Type(), w.fs.URI(), "Watch: Stopped. Final metrics")
			return
		}
	}
}

// updatePrometheusMetrics periodically updates Prometheus metrics
func (w *linuxWatcher) updatePrometheusMetrics() {
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

// getLinuxOptimalBufferSize returns the optimal buffer size for Linux
func getLinuxOptimalBufferSize(fileCount int) int {
	// Linux inotify benefits from larger buffers for large directories
	baseSize := 1000

	// Adjust based on folder size
	if fileCount > 50000 {
		// Very large folder
		return baseSize * 4
	} else if fileCount > 10000 {
		// Large folder
		return baseSize * 2
	} else if fileCount < 1000 {
		// Small folder
		return baseSize / 2
	}

	return baseSize
}

// Linux-specific optimizations for inotify
func (w *linuxWatcher) optimizeForLinux() {
	// Linux-specific optimizations can be added here
	// For example, adjusting buffer sizes based on Linux performance characteristics

	// Get system memory information
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		// If system has plenty of memory, we can use larger buffers
		if memInfo.Available > 2*1024*1024*1024 { // More than 2GB available
			w.bufferSize = int(float64(w.bufferSize) * 1.5)
		} else if memInfo.Available < 512*1024*1024 { // Less than 512MB available
			w.bufferSize = int(float64(w.bufferSize) * 0.75)
		}
	}

	// Adjust based on number of CPU cores
	numCPU := runtime.NumCPU()
	if numCPU > 8 {
		// High core count systems can handle more events
		w.bufferSize = int(float64(w.bufferSize) * 1.2)
	} else if numCPU < 4 {
		// Low core count systems need smaller buffers to avoid overwhelming
		w.bufferSize = int(float64(w.bufferSize) * 0.8)
	}
}
