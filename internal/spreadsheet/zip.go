// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package spreadsheet

import (
	"archive/zip"
	"io"
	"time"
)

// zipEpoch is the fixed modification time stamped on every generated entry.
// A fixed timestamp keeps repeated exports of the same result byte-identical,
// which makes the export comparable and testable.
var zipEpoch = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// writeZipEntry stores one XML part with the fixed timestamp and deflate
// compression.
func writeZipEntry(archive *zip.Writer, name, body string) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetModTime(zipEpoch)
	entry, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.WriteString(entry, body)
	return err
}
