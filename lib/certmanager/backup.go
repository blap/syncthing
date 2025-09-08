// Copyright (C) 2024 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package certmanager handles certificate backup and restore functionality
package certmanager

import (
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/syncthing/syncthing/lib/locations"
)

// BackupService handles certificate backup and restore operations
type BackupService struct{}

// NewBackupService creates a new backup service
func NewBackupService() *BackupService {
	return &BackupService{}
}

// BackupCertificates creates a backup of the current certificates
func (bs *BackupService) BackupCertificates() error {
	certFile := locations.Get(locations.CertFile)
	keyFile := locations.Get(locations.KeyFile)
	httpsCertFile := locations.Get(locations.HTTPSCertFile)
	httpsKeyFile := locations.Get(locations.HTTPSKeyFile)
	
	backupDir := filepath.Join(locations.GetBaseDir(locations.ConfigBaseDir), "cert-backups")
	
	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	// Generate timestamp for backup
	timestamp := time.Now().Format("20060102-150405")
	
	// Backup device certificates
	if err := bs.backupFile(certFile, filepath.Join(backupDir, fmt.Sprintf("cert-%s.pem", timestamp))); err != nil {
		return fmt.Errorf("failed to backup cert file: %w", err)
	}
	
	if err := bs.backupFile(keyFile, filepath.Join(backupDir, fmt.Sprintf("key-%s.pem", timestamp))); err != nil {
		return fmt.Errorf("failed to backup key file: %w", err)
	}
	
	// Backup HTTPS certificates if they exist and are different
	if httpsCertFile != certFile {
		if _, err := os.Stat(httpsCertFile); err == nil {
			if err := bs.backupFile(httpsCertFile, filepath.Join(backupDir, fmt.Sprintf("https-cert-%s.pem", timestamp))); err != nil {
				return fmt.Errorf("failed to backup https cert file: %w", err)
			}
		}
	}
	
	if httpsKeyFile != keyFile {
		if _, err := os.Stat(httpsKeyFile); err == nil {
			if err := bs.backupFile(httpsKeyFile, filepath.Join(backupDir, fmt.Sprintf("https-key-%s.pem", timestamp))); err != nil {
				return fmt.Errorf("failed to backup https key file: %w", err)
			}
		}
	}
	
	slog.Info("Certificate backup completed", "backupDir", backupDir, "timestamp", timestamp)
	return nil
}

// backupFile copies a file to a backup location
func (bs *BackupService) backupFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer srcFile.Close()
	
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer dstFile.Close()
	
	// Copy file contents
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file from %s to %s: %w", src, dst, err)
	}
	
	// Copy file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to get source file info %s: %w", src, err)
	}
	
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set permissions on destination file %s: %w", dst, err)
	}
	
	slog.Debug("File backed up", "src", src, "dst", dst)
	return nil
}

// RestoreCertificates restores certificates from a backup
func (bs *BackupService) RestoreCertificates(timestamp string) error {
	backupDir := filepath.Join(locations.GetBaseDir(locations.ConfigBaseDir), "cert-backups")
	
	certFile := locations.Get(locations.CertFile)
	keyFile := locations.Get(locations.KeyFile)
	httpsCertFile := locations.Get(locations.HTTPSCertFile)
	httpsKeyFile := locations.Get(locations.HTTPSKeyFile)
	
	// Restore device certificates
	if err := bs.restoreFile(filepath.Join(backupDir, fmt.Sprintf("cert-%s.pem", timestamp)), certFile); err != nil {
		return fmt.Errorf("failed to restore cert file: %w", err)
	}
	
	if err := bs.restoreFile(filepath.Join(backupDir, fmt.Sprintf("key-%s.pem", timestamp)), keyFile); err != nil {
		return fmt.Errorf("failed to restore key file: %w", err)
	}
	
	// Restore HTTPS certificates if they exist in the backup
	httpsCertBackup := filepath.Join(backupDir, fmt.Sprintf("https-cert-%s.pem", timestamp))
	if _, err := os.Stat(httpsCertBackup); err == nil {
		if err := bs.restoreFile(httpsCertBackup, httpsCertFile); err != nil {
			return fmt.Errorf("failed to restore https cert file: %w", err)
		}
	}
	
	httpsKeyBackup := filepath.Join(backupDir, fmt.Sprintf("https-key-%s.pem", timestamp))
	if _, err := os.Stat(httpsKeyBackup); err == nil {
		if err := bs.restoreFile(httpsKeyBackup, httpsKeyFile); err != nil {
			return fmt.Errorf("failed to restore https key file: %w", err)
		}
	}
	
	slog.Info("Certificate restore completed", "backupDir", backupDir, "timestamp", timestamp)
	return nil
}

// restoreFile copies a backup file to the original location
func (bs *BackupService) restoreFile(src, dst string) error {
	// Check if backup file exists
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("backup file %s does not exist", src)
	}
	
	// Create backup of current file if it exists
	if _, err := os.Stat(dst); err == nil {
		backupPath := dst + ".restore-backup"
		if err := bs.backupFile(dst, backupPath); err != nil {
			slog.Warn("Failed to create restore backup", "file", dst, "error", err)
		} else {
			slog.Debug("Created restore backup", "file", backupPath)
		}
	}
	
	// Restore the file
	if err := bs.backupFile(src, dst); err != nil {
		return fmt.Errorf("failed to restore file from %s to %s: %w", src, dst, err)
	}
	
	slog.Debug("File restored", "src", src, "dst", dst)
	return nil
}

// ListBackups returns a list of available certificate backups
func (bs *BackupService) ListBackups() ([]string, error) {
	backupDir := filepath.Join(locations.GetBaseDir(locations.ConfigBaseDir), "cert-backups")
	
	// Check if backup directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return []string{}, nil
	}
	
	// Read directory contents
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}
	
	// Extract timestamps from filenames
	timestamps := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		name := entry.Name()
		// Look for files matching our backup pattern
		if len(name) >= 15 && name[len(name)-4:] == ".pem" {
			// Extract timestamp from filename (format: cert-20060102-150405.pem)
			if len(name) >= 20 {
				timestamp := name[len(name)-19 : len(name)-4] // Extract 20060102-150405
				timestamps = append(timestamps, timestamp)
			}
		}
	}
	
	return timestamps, nil
}

// DeleteBackup removes a specific certificate backup
func (bs *BackupService) DeleteBackup(timestamp string) error {
	backupDir := filepath.Join(locations.GetBaseDir(locations.ConfigBaseDir), "cert-backups")
	
	// Delete all files with the specified timestamp
	files := []string{
		filepath.Join(backupDir, fmt.Sprintf("cert-%s.pem", timestamp)),
		filepath.Join(backupDir, fmt.Sprintf("key-%s.pem", timestamp)),
		filepath.Join(backupDir, fmt.Sprintf("https-cert-%s.pem", timestamp)),
		filepath.Join(backupDir, fmt.Sprintf("https-key-%s.pem", timestamp)),
	}
	
	deleted := 0
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			if !os.IsNotExist(err) {
				slog.Warn("Failed to delete backup file", "file", file, "error", err)
			}
		} else {
			deleted++
			slog.Debug("Deleted backup file", "file", file)
		}
	}
	
	if deleted == 0 {
		return fmt.Errorf("no backup files found for timestamp %s", timestamp)
	}
	
	slog.Info("Backup deleted", "timestamp", timestamp, "filesDeleted", deleted)
	return nil
}

// ValidateBackup checks if a backup is valid
func (bs *BackupService) ValidateBackup(timestamp string) error {
	backupDir := filepath.Join(locations.GetBaseDir(locations.ConfigBaseDir), "cert-backups")
	
	// Check if required backup files exist
	requiredFiles := []string{
		filepath.Join(backupDir, fmt.Sprintf("cert-%s.pem", timestamp)),
		filepath.Join(backupDir, fmt.Sprintf("key-%s.pem", timestamp)),
	}
	
	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("required backup file %s does not exist", file)
		}
	}
	
	// Try to load the certificate
	certFile := filepath.Join(backupDir, fmt.Sprintf("cert-%s.pem", timestamp))
	keyFile := filepath.Join(backupDir, fmt.Sprintf("key-%s.pem", timestamp))
	
	if _, err := tls.LoadX509KeyPair(certFile, keyFile); err != nil {
		return fmt.Errorf("backup certificate is invalid: %w", err)
	}
	
	slog.Debug("Backup validation successful", "timestamp", timestamp)
	return nil
}