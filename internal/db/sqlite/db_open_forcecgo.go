// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build forcecgo

package sqlite

import (
	_ "github.com/mattn/go-sqlite3" // register sqlite3 database driver
	"github.com/syncthing/syncthing/lib/build"
)

const (
	dbDriver      = "sqlite3"
	commonOptions = "_fk=true&_rt=true&_cache_size=-65536&_sync=1&_txlock=immediate"
)

func init() {
	// This tag indicates we're using the CGO-enabled SQLite driver
	// but we need to make sure CGO is actually enabled for it to work
	build.AddTag("cgo-sqlite")

	// Add a tag to indicate this is a special build that avoids the console panic
	build.AddTag("forcecgo-build")
}
