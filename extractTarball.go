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
		parts := strings.SplitN(header.Name, "/", stripComponents+1)
		if len(parts) <= stripComponents {
			continue
		}
		relPath := parts[stripComponents]
		if relPath == "" { continue }

		target := filepath.Join(dest, relPath)
		if !filepath.IsLocal(relPath) {
			fmt.Println(ColorRed+"fatal:", ColorReset+"malicious archive detected")
			fmt.Println("zip slip detected")
			fmt.Println("exiting now, no changes will be done to the system")
			fmt.Println("system protected")
			return false
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return false
		}
		realDir, err := filepath.EvalSymlinks(filepath.Dir(target))
		if err != nil {
			return false
		}
		realDest, err := filepath.EvalSymlinks(dest)
		if err != nil {
			return false
		}
		rel, err := filepath.Rel(realDest, realDir)
		if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
			fmt.Println(ColorRed+"fatal:", ColorReset+"malicious archive detected")
			fmt.Println("symlink escape detected")
			fmt.Println("exiting now, no changes will be done to the system")
			fmt.Println("system protected")
			return false
		}
		switch header.Typeflag {
			case tar.TypeDir:
				os.MkdirAll(target, os.FileMode(header.Mode))
				os.Chtimes(target, header.AccessTime, header.ModTime)
			case tar.TypeReg:
				os.Remove(target)
				outFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
				if err != nil {
					fmt.Println(ColorRed + "error:", ColorReset + "cannot create file:", target)
					return false
				}
				io.Copy(outFile, tr)
				outFile.Close()
				os.Chmod(target, os.FileMode(header.Mode))
				os.Chtimes(target, header.AccessTime, header.ModTime)
			case tar.TypeSymlink:
				if !filepath.IsAbs(header.Linkname) && !filepath.IsLocal(filepath.Join(filepath.Dir(relPath), header.Linkname)) {
					fmt.Println(ColorRed+"fatal:", ColorReset+"malicious archive detected")
					fmt.Println("symlink escape detected")
					fmt.Println("exiting now, no changes will be done to the system")
					fmt.Println("system protected")
					return false
				}
				os.Remove(target)
				if err := os.Symlink(header.Linkname, target); err != nil {
					return false
				}
		}
	}
	return true
}
