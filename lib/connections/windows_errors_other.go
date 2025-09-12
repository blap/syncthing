// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build !windows

package connections

// categorizeWindowsError is a placeholder function for non-Windows platforms
// It always returns ErrorCategoryUnknown since Windows-specific errors
// cannot occur on non-Windows platforms
func categorizeWindowsError(err error) ErrorCategory {
	return ErrorCategoryUnknown
}