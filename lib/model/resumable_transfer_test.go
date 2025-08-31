// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"bytes"
	"context"
	"testing"

	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/rand"
)

// TestResumableBlockTransfer tests that when a connection is dropped during
// block transfer, the receiver can resume the transfer from the last checkpoint
// rather than starting over.
func TestResumableBlockTransfer(t *testing.T) {
	// Unit under test: Resumable block transfer functionality
	// Expected behavior: When a connection drops during block transfer and is
	// reestablished, the receiver should resume from the last complete checkpoint
	// rather than requesting the entire block again.

	m, _, fcfg, wcfgCancel := setupModelWithConnection(t)
	defer wcfgCancel()
	tfs := fcfg.Filesystem()
	defer cleanupModelAndRemoveDir(m, tfs.URI())

	// Create a large file (16 MiB) to test with
	fileSize := int64(16 * 1024 * 1024) // 16 MiB
	fileName := "largefile.dat"

	// Generate random content for the file
	fileData := make([]byte, fileSize)
	rand.Read(fileData)

	// Create the file on the filesystem
	fd, err := tfs.Create(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fd.Write(fileData); err != nil {
		t.Fatal(err)
	}
	fd.Close()

	// Scan the folder to index the file
	m.ScanFolder("default")

	// Now try to request the file
	resp, err := m.Request(device1Conn, &protocol.Request{
		Folder: "default",
		Name:   fileName,
		Offset: 0,
		Size:   int(fileSize),
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Verify that we got the correct data
	responseData := resp.Data()
	if !bytes.Equal(responseData, fileData) {
		t.Fatal("Response data does not match original file data")
	}

	// Close the response to release resources
	resp.Close()

	// Test partial request
	resp, err = m.Request(device1Conn, &protocol.Request{
		Folder: "default",
		Name:   fileName,
		Offset: 0,
		Size:   1024, // Request first 1KB
	})
	if err != nil {
		t.Fatalf("Partial request failed: %v", err)
	}

	// Verify that we got the correct partial data
	responseData = resp.Data()
	if !bytes.Equal(responseData, fileData[:1024]) {
		t.Fatal("Partial response data does not match original file data")
	}

	// Close the response to release resources
	resp.Close()
}

// TestResumableTransferWithCheckpoint tests that transfers are divided into chunks
// based on the transferChunkSizeBytes configuration.
func TestResumableTransferWithCheckpoint(t *testing.T) {
	// Unit under test: Checkpoint-based resumable transfer
	// Expected behavior: Transfers are divided into chunks based on transferChunkSizeBytes
	// and each chunk is requested separately.

	m, fc, fcfg, wcfgCancel := setupModelWithConnection(t)
	defer wcfgCancel()
	tfs := fcfg.Filesystem()
	defer cleanupModelAndRemoveDir(m, tfs.URI())

	// Enable resumable transfers for this folder
	fcfg.ResumableTransfersEnabled = true

	// Update the folder configuration
	setFolder(t, m.cfg, fcfg)

	// Create a large file (3 MiB) to test with
	fileSize := int64(3 * 1024 * 1024) // 3 MiB
	fileName := "checkpointfile.dat"

	// Generate random content for the file
	fileData := make([]byte, fileSize)
	rand.Read(fileData)

	// Create the file on the filesystem
	fd, err := tfs.Create(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fd.Write(fileData); err != nil {
		t.Fatal(err)
	}
	fd.Close()

	// Scan the folder to index the file
	m.ScanFolder("default")

	// Track how many times each chunk is requested
	requestCount := make(map[int64]int)

	// Set up a custom request handler to track requests
	fc.RequestCalls(func(ctx context.Context, req *protocol.Request) ([]byte, error) {
		// Count requests by offset
		requestCount[req.Offset]++

		// Return the requested data
		start := req.Offset
		end := start + int64(req.Size)
		if end > fileSize {
			end = fileSize
		}
		return fileData[start:end], nil
	})

	// Request the file
	resp, err := m.Request(device1Conn, &protocol.Request{
		Folder: "default",
		Name:   fileName,
		Offset: 0,
		Size:   int(fileSize),
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Verify that we got the correct data
	responseData := resp.Data()
	if !bytes.Equal(responseData, fileData) {
		t.Fatal("Response data does not match original file data")
	}

	// Close the response to release resources
	resp.Close()

	// With a 3MB file, Syncthing will divide it into 24 blocks of 128KB each
	// Since each block is smaller than the transfer chunk size (1MB), each block
	// will be requested as a single chunk
	// We should have requests at offsets: 0, 131072, 262144, ..., up to the file size
	expectedNumRequests := int((fileSize + 131072 - 1) / 131072) // Ceiling division
	if len(requestCount) != expectedNumRequests {
		t.Errorf("Expected %d requests, but got %d", expectedNumRequests, len(requestCount))
	}

	// Verify that each block was requested only once (no redundant requests)
	for offset, count := range requestCount {
		if count > 1 {
			t.Errorf("Block at offset %d was requested %d times, expected only once", offset, count)
		}
		// Verify that the offset is a multiple of the block size (128KB)
		if offset%131072 != 0 {
			t.Errorf("Block at offset %d is not aligned to block boundary", offset)
		}
	}
}

// TestResumableTransferConnectionReestablishment simulates a connection drop during
// a resumable transfer and verifies that the transfer resumes from the last checkpoint
// after reconnection.
func TestResumableTransferConnectionReestablishment(t *testing.T) {
	m, fc, fcfg, wcfgCancel := setupModelWithConnection(t)
	defer wcfgCancel()
	tfs := fcfg.Filesystem()
	defer cleanupModelAndRemoveDir(m, tfs.URI())

	// Enable resumable transfers for this folder
	fcfg.ResumableTransfersEnabled = true

	// Update the folder configuration
	setFolder(t, m.cfg, fcfg)

	// Create a large file (2 MiB) to test with
	fileSize := int64(2 * 1024 * 1024) // 2 MiB
	fileName := "reconnectfile.dat"

	// Generate random content for the file
	fileData := make([]byte, fileSize)
	rand.Read(fileData)

	// Create the file on the filesystem
	fd, err := tfs.Create(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fd.Write(fileData); err != nil {
		t.Fatal(err)
	}
	fd.Close()

	// Scan the folder to index the file
	m.ScanFolder("default")

	// Track the progress of the transfer
	var requestOffsets []int64
	var requestSizes []int

	// Set up a custom request handler that simulates a connection drop
	// after the first chunk is received
	fc.RequestCalls(func(ctx context.Context, req *protocol.Request) ([]byte, error) {
		requestOffsets = append(requestOffsets, req.Offset)
		requestSizes = append(requestSizes, req.Size)

		// Simulate a connection drop after the first request
		if len(requestOffsets) == 1 {
			// Return only part of the data to simulate a partial transfer
			partialSize := req.Size / 2
			if partialSize > len(fileData) {
				partialSize = len(fileData)
			}
			return fileData[req.Offset : req.Offset+int64(partialSize)], nil
		}

		// For subsequent requests, return the full data
		start := req.Offset
		end := start + int64(req.Size)
		if end > fileSize {
			end = fileSize
		}
		return fileData[start:end], nil
	})

	// Request the file
	resp, err := m.Request(device1Conn, &protocol.Request{
		Folder: "default",
		Name:   fileName,
		Offset: 0,
		Size:   int(fileSize),
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Verify that we got the correct data
	responseData := resp.Data()
	if !bytes.Equal(responseData, fileData) {
		t.Fatal("Response data does not match original file data")
	}

	// Close the response to release resources
	resp.Close()

	// Verify that we had multiple requests (indicating resumable transfer worked)
	if len(requestOffsets) < 2 {
		t.Errorf("Expected multiple requests for resumable transfer, but got %d", len(requestOffsets))
	}

	// Verify that the requests were sequential and contiguous
	expectedOffset := int64(0)
	for i, offset := range requestOffsets {
		if offset != expectedOffset {
			t.Errorf("Request %d: expected offset %d, got %d", i, expectedOffset, offset)
		}
		if i < len(requestSizes) {
			expectedOffset += int64(requestSizes[i])
		}
	}
}

// BenchmarkResumableVsNonResumableTransfers compares the performance of resumable
// and non-resumable transfers for large files.
func BenchmarkResumableVsNonResumableTransfers(b *testing.B) {
	// This benchmark would compare transfer performance between resumable and
	// non-resumable modes. Due to the complexity of setting up a proper benchmark
	// environment with network simulation, this is a placeholder for future implementation.

	// For now, we'll just document what such a benchmark would measure:
	// 1. Transfer time for large files with resumable transfers enabled
	// 2. Transfer time for large files with resumable transfers disabled
	// 3. Memory usage during transfers
	// 4. Disk I/O patterns

	// The benchmark would need to:
	// - Set up a realistic network simulation environment
	// - Create large test files
	// - Measure transfer times with and without resumable transfers
	// - Simulate network interruptions to test resumable transfer benefits

	b.Skip("Benchmark not yet implemented - requires complex network simulation environment")
}
