package main

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

func extractTarball(path string, dest string, stripComponents int) bool {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println(ColorRed + "error:", ColorReset + "cannot open tarball:", path)
		return false
	}
	defer f.Close()

	var tr *tar.Reader

	switch {
		case strings.HasSuffix(path, ".tar.gz") || strings.HasSuffix(path, ".tgz"):
			gr, err := gzip.NewReader(f)
			if err != nil {
				fmt.Println(ColorRed + "error:", ColorReset + "cannot read gzip:", err)
				return false
			}
			defer gr.Close()
			tr = tar.NewReader(gr)

		case strings.HasSuffix(path, ".tar.bz2"):
			tr = tar.NewReader(bzip2.NewReader(f))

		case strings.HasSuffix(path, ".tar.zst"):
			zr, err := zstd.NewReader(f)
			if err != nil {
				fmt.Println(ColorRed + "error:", ColorReset + "cannot read zstd:", err)
				return false
			}
			defer zr.Close()
			tr = tar.NewReader(zr)

		case strings.HasSuffix(path, ".tar.xz"):
			xr, err := xz.NewReader(f)
			if err != nil {
				fmt.Println(ColorRed + "error:", ColorReset + "cannot read xz:", err)
				return false
			}
			tr = tar.NewReader(xr)

		default:
			fmt.Println(ColorRed + "error:", ColorReset + "unsupported format:", path)
			return false
	}

	for {
		header, err := tr.Next()
		if err == io.EOF { break }
		if err != nil {
			fmt.Println(ColorRed + "error:", ColorReset + "extraction error:", err)
			return false
		}

		// Strip components
		parts := strings.SplitN(header.Name, "/", stripComponents+1)
		if len(parts) <= stripComponents {
			continue
		}
		relPath := parts[stripComponents]
		if relPath == "" { continue }

		target := filepath.Join(dest, relPath)

		switch header.Typeflag {
			case tar.TypeDir:
				os.MkdirAll(target, os.FileMode(header.Mode))
				os.Chtimes(target, header.AccessTime, header.ModTime)
			case tar.TypeReg:
				os.MkdirAll(filepath.Dir(target), 0755)
				outFile, err := os.Create(target)
				if err != nil {
					fmt.Println(ColorRed + "error:", ColorReset + "cannot create file:", target)
					return false
				}
				io.Copy(outFile, tr)
				outFile.Close()
				os.Chmod(target, os.FileMode(header.Mode))
				// Préserver les timestamps
				os.Chtimes(target, header.AccessTime, header.ModTime)
			case tar.TypeSymlink:
				os.Symlink(header.Linkname, target)
		}
	}
	return true
}
