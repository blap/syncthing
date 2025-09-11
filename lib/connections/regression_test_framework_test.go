// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/syncthing/syncthing/internal/db"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/stats"
)

// TestRegressionTestFramework tests the basic functionality of the regression testing framework
func TestRegressionTestFramework(t *testing.T) {
	// Create test environment
	ctx := context.Background()

	// Create mock config
	cfg := createTestConfig()

	// Create mock model
	model := &regressionTestModel{t: t}

	// Create regression test framework
	framework := NewRegressionTestFramework(cfg, events.NoopLogger, model)

	// Test framework creation
	if framework == nil {
		t.Fatal("Failed to create regression test framework")
	}

	// Test basic functionality
	t.Run("FrameworkCreation", func(t *testing.T) {
		if framework.cfg == nil {
			t.Error("Framework config should not be nil")
		}
		if framework.registry == nil {
			t.Error("Framework registry should not be nil")
		}
		if framework.model == nil {
			t.Error("Framework model should not be nil")
		}
	})

	// Test regression test suite
	t.Run("RegressionTestSuite", func(t *testing.T) {
		suite := NewRegressionTestSuite(framework)
		if suite == nil {
			t.Fatal("Failed to create regression test suite")
		}

		// Add standard test cases
		suite.AddStandardTestCases()

		// Check that test cases were added
		if len(suite.testCases) == 0 {
			t.Error("Expected test cases to be added")
		}

		// Add custom test case
		customTestCase := ConnectionPathTestCase{
			Name:         "Custom_Test_Case",
			Address:      "tcp://127.0.0.1:0",
			ExpectedType: "tcp",
			Timeout:      5 * time.Second,
			ValidationFn: func(conn protocol.Connection) error {
				return nil // Always pass for testing
			},
		}
		suite.AddTestCase(customTestCase)

		// Verify test case was added
		if len(suite.testCases) != 4 {
			t.Errorf("Expected 4 test cases, got %d", len(suite.testCases))
		}
	})

	// Test result generation
	t.Run("ResultGeneration", func(t *testing.T) {
		// Create some test results
		results := []ConnectionPathTestResult{
			{
				TestName:      "Test1",
				Success:       true,
				ConnectionType: "tcp",
				Latency:       10 * time.Millisecond,
				Throughput:    100.5,
				Timestamp:     time.Now(),
			},
			{
				TestName:  "Test2",
				Success:   false,
				Error:     fmt.Errorf("connection failed"),
				Timestamp: time.Now(),
			},
		}

		// Generate report
		report := framework.GenerateRegressionReport(results)
		if report == "" {
			t.Error("Expected non-empty report")
		}

		// Check that report contains expected information
		if len(report) < 100 {
			t.Errorf("Expected detailed report, got short report: %s", report)
		}
	})

	// Test running regression tests
	t.Run("RunRegressionTests", func(t *testing.T) {
		// Create test cases
		testCases := []ConnectionPathTestCase{
			{
				Name:         "Test_Case_1",
				Address:      "tcp://127.0.0.1:0",
				ExpectedType: "tcp",
				Timeout:      5 * time.Second,
				ValidationFn: func(conn protocol.Connection) error {
					return nil
				},
			},
		}

		// Run tests
		results := framework.RunRegressionTests(ctx, testCases)

		// Check results
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Success {
			t.Error("Expected test to succeed")
		}
	})
}

// TestConnectionPathTestCase tests the ConnectionPathTestCase struct
func TestConnectionPathTestCase(t *testing.T) {
	testCase := ConnectionPathTestCase{
		Name:         "Test_Case",
		Address:      "tcp://127.0.0.1:0",
		ExpectedType: "tcp",
		Timeout:      10 * time.Second,
		ValidationFn: func(conn protocol.Connection) error {
			return nil
		},
	}

	if testCase.Name != "Test_Case" {
		t.Errorf("Expected name Test_Case, got %s", testCase.Name)
	}

	if testCase.Address != "tcp://127.0.0.1:0" {
		t.Errorf("Expected address tcp://127.0.0.1:0, got %s", testCase.Address)
	}

	if testCase.ExpectedType != "tcp" {
		t.Errorf("Expected type tcp, got %s", testCase.ExpectedType)
	}

	if testCase.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %s", testCase.Timeout)
	}

	// Test validation function
	mockConn := &mockConnection{}
	err := testCase.ValidationFn(mockConn)
	if err != nil {
		t.Errorf("Expected validation to pass, got error: %v", err)
	}
}

// TestConnectionPathTestResult tests the ConnectionPathTestResult struct
func TestConnectionPathTestResult(t *testing.T) {
	result := ConnectionPathTestResult{
		TestName:      "Test_Result",
		Success:       true,
		ConnectionType: "tcp",
		Latency:       5 * time.Millisecond,
		Throughput:    50.0,
		Timestamp:     time.Now(),
	}

	if result.TestName != "Test_Result" {
		t.Errorf("Expected test name Test_Result, got %s", result.TestName)
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if result.ConnectionType != "tcp" {
		t.Errorf("Expected connection type tcp, got %s", result.ConnectionType)
	}

	if result.Latency != 5*time.Millisecond {
		t.Errorf("Expected latency 5ms, got %s", result.Latency)
	}

	if result.Throughput != 50.0 {
		t.Errorf("Expected throughput 50.0, got %f", result.Throughput)
	}

	// Check that timestamp was set
	if result.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

// regressionTestModel implements the Model interface for testing regression
type regressionTestModel struct {
	t *testing.T
}

func (m *regressionTestModel) OnHello(remoteID protocol.DeviceID, addr net.Addr, hello protocol.Hello) error {
	m.t.Logf("Device received hello from %s at %s", remoteID, addr)
	return nil
}

func (m *regressionTestModel) AddConnection(conn protocol.Connection, hello protocol.Hello) {
	m.t.Logf("Device added connection to %s", conn.DeviceID())
}

func (m *regressionTestModel) DeviceStatistics() (map[protocol.DeviceID]stats.DeviceStatistics, error) {
	return make(map[protocol.DeviceID]stats.DeviceStatistics), nil
}

func (m *regressionTestModel) SetConnectionsService(service Service) {
	// Not needed for this test
}

func (m *regressionTestModel) Index(conn protocol.Connection, idx *protocol.Index) error {
	m.t.Logf("Device received index from %s", conn.DeviceID())
	return nil
}

func (m *regressionTestModel) IndexUpdate(conn protocol.Connection, idxUp *protocol.IndexUpdate) error {
	m.t.Logf("Device received index update from %s", conn.DeviceID())
	return nil
}

func (m *regressionTestModel) Request(conn protocol.Connection, req *protocol.Request) (protocol.RequestResponse, error) {
	m.t.Logf("Device received request from %s", conn.DeviceID())
	return nil, nil
}

func (m *regressionTestModel) ClusterConfig(conn protocol.Connection, config *protocol.ClusterConfig) error {
	m.t.Logf("Device received cluster config from %s", conn.DeviceID())
	return nil
}

func (m *regressionTestModel) Closed(conn protocol.Connection, err error) {
	m.t.Logf("Device connection to %s closed: %v", conn.DeviceID(), err)
}

func (m *regressionTestModel) DownloadProgress(conn protocol.Connection, p *protocol.DownloadProgress) error {
	m.t.Logf("Device received download progress from %s", conn.DeviceID())
	return nil
}

func (m *regressionTestModel) GlobalSize(_ string) (db.Counts, error) {
	return db.Counts{}, nil
}

func (m *regressionTestModel) UsageReportingStats(_ interface{}, _ int, _ bool) {
	// Not needed for this test
}