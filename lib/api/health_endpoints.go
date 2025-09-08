package api

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/locations"
	"github.com/syncthing/syncthing/lib/model"
)

// getSystemHealth returns overall system health status including folders, connections, and certificates
func (s *service) getSystemHealth(w http.ResponseWriter, r *http.Request) {
	// Cast model to HealthMonitoringModel to access health monitoring methods
	healthModel, ok := s.model.(model.HealthMonitoringModel)
	if !ok {
		// If the model doesn't implement HealthMonitoringModel, return empty health status
		folderHealth := make(map[string]config.FolderHealthStatus)
		connectionsHealth := make(map[string]interface{})
		certificatesHealth := make(map[string]interface{})
		healthStatus := map[string]interface{}{
			"folders":      folderHealth,
			"connections":  connectionsHealth,
			"certificates": certificatesHealth,
			"timestamp":    time.Now(),
		}
		sendJSON(w, healthStatus)
		return
	}

	// Get folder health status
	folderHealth := healthModel.GetAllFoldersHealthStatus()
	
	// Get connection health status
	connectionsHealth := s.connectionsService.ConnectionStatus()
	
	// Get certificate health status
	certificatesHealth := s.getCertificatesHealthData()
	
	// Combine all health information
	healthStatus := map[string]interface{}{
		"folders":      folderHealth,
		"connections":  connectionsHealth,
		"certificates": certificatesHealth,
		"timestamp":    time.Now(),
	}
	
	sendJSON(w, healthStatus)
}

// getConnectionsHealth returns health status of all connections
func (s *service) getConnectionsHealth(w http.ResponseWriter, r *http.Request) {
	healthStatus := s.connectionsService.ConnectionStatus()
	sendJSON(w, healthStatus)
}

// getCertificatesHealth returns health status of all certificates
func (s *service) getCertificatesHealth(w http.ResponseWriter, r *http.Request) {
	healthStatus := s.getCertificatesHealthData()
	sendJSON(w, healthStatus)
}

// getCertificatesHealthData returns certificate health data
func (s *service) getCertificatesHealthData() map[string]interface{} {
	healthData := make(map[string]interface{})
	
	// Check device certificate
	deviceCertFile := locations.Get(locations.CertFile)
	deviceKeyFile := locations.Get(locations.KeyFile)
	deviceCertHealth := s.checkCertificateHealth(deviceCertFile, deviceKeyFile, "device")
	healthData["device"] = deviceCertHealth
	
	// Check HTTPS certificate if different from device certificate
	httpsCertFile := locations.Get(locations.HTTPSCertFile)
	httpsKeyFile := locations.Get(locations.HTTPSKeyFile)
	if httpsCertFile != deviceCertFile {
		httpsCertHealth := s.checkCertificateHealth(httpsCertFile, httpsKeyFile, "https")
		healthData["https"] = httpsCertHealth
	}
	
	return healthData
}

// checkCertificateHealth checks the health of a certificate
func (s *service) checkCertificateHealth(certFile, keyFile, certType string) map[string]interface{} {
	health := make(map[string]interface{})
	health["type"] = certType
	health["file"] = certFile
	
	// Check if certificate file exists
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		health["status"] = "missing"
		health["error"] = "Certificate file not found"
		return health
	}
	
	// Check if key file exists
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		health["status"] = "missing"
		health["error"] = "Key file not found"
		return health
	}
	
	// Try to load the certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		health["status"] = "invalid"
		health["error"] = fmt.Sprintf("Failed to load certificate: %v", err)
		return health
	}
	
	// Parse certificate
	if len(cert.Certificate) == 0 {
		health["status"] = "invalid"
		health["error"] = "No certificates in certificate file"
		return health
	}
	
	parsedCert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		health["status"] = "invalid"
		health["error"] = fmt.Sprintf("Failed to parse certificate: %v", err)
		return health
	}
	
	// Check expiration
	now := time.Now()
	timeUntilExpiry := parsedCert.NotAfter.Sub(now)
	
	health["status"] = "valid"
	health["subject"] = parsedCert.Subject.String()
	health["notBefore"] = parsedCert.NotBefore
	health["notAfter"] = parsedCert.NotAfter
	health["timeUntilExpiry"] = timeUntilExpiry.String()
	
	// Check if expiring soon
	if timeUntilExpiry <= 0 {
		health["status"] = "expired"
		health["alert"] = "Certificate has expired"
	} else if timeUntilExpiry <= 30*24*time.Hour {
		health["status"] = "expiring"
		health["alert"] = fmt.Sprintf("Certificate expires in %v", timeUntilExpiry)
	}
	
	return health
}

// getSystemAlerts returns all active system alerts including predictive alerts
func (s *service) getSystemAlerts(w http.ResponseWriter, r *http.Request) {
	// Cast model to HealthMonitoringModel to access health monitoring methods
	healthModel, ok := s.model.(model.HealthMonitoringModel)
	if !ok {
		// If the model doesn't implement HealthMonitoringModel, return empty alerts
		alerts := make([]map[string]interface{}, 0)
		sendJSON(w, map[string]interface{}{
			"alerts":    alerts,
			"timestamp": time.Now(),
		})
		return
	}

	// Get folder performance stats for predictive alerts
	folderPerfStats := healthModel.GetAllFoldersPerformanceStats()
	
	// Get connection health for predictive alerts
	connectionsHealth := s.connectionsService.ConnectionStatus()
	
	// Collect alerts
	alerts := make([]map[string]interface{}, 0)
	
	// Check for folder performance issues
	for folderID, stats := range folderPerfStats {
		// Check for performance degradation
		if stats.AvgCheckDuration > time.Second*5 {
			alerts = append(alerts, map[string]interface{}{
				"type":      "performance_degradation",
				"severity":  "warning",
				"folder":    folderID,
				"message":   fmt.Sprintf("Folder health checks are taking long: %v", stats.AvgCheckDuration),
				"timestamp": stats.LastCheckTime,
			})
		}
		
		// Check for high failure rate
		if stats.CheckCount > 0 {
			failureRate := float64(stats.FailedCheckCount) / float64(stats.CheckCount)
			if failureRate > 0.1 {
				alerts = append(alerts, map[string]interface{}{
					"type":      "high_failure_rate",
					"severity":  "warning",
					"folder":    folderID,
					"message":   fmt.Sprintf("High folder failure rate: %.2f%%", failureRate*100),
					"timestamp": stats.LastCheckTime,
				})
			}
		}
	}
	
	// Check for connection stability issues
	for deviceID, connHealth := range connectionsHealth {
		if connHealth.Error != nil {
			alerts = append(alerts, map[string]interface{}{
				"type":      "unstable_connection",
				"severity":  "warning",
				"device":    deviceID,
				"message":   fmt.Sprintf("Unstable connection: %v", *connHealth.Error),
				"timestamp": time.Now(),
			})
		}
	}
	
	sendJSON(w, map[string]interface{}{
		"alerts":    alerts,
		"timestamp": time.Now(),
	})
}