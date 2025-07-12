package packages

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type PackageManager struct {
	pkgDir     string
	installDir string
}

func NewPackageManager() *PackageManager {
	pkgDir := os.Getenv("PKG_DIR")
	if pkgDir == "" {
		pkgDir = "/usr/local/etc/nexushub/packages"
	}
	installDir := os.Getenv("INSTALL_DIR")
	if installDir == "" {
		installDir = "/usr/local/etc/nexushub/install"
	}
	return &PackageManager{
		pkgDir:     pkgDir,
		installDir: installDir,
	}
}

func (pm *PackageManager) GetInstallDir() string {
	return pm.installDir
}

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func (pm *PackageManager) IsInstalled(instanceID string) bool {
	if pm.installDir == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join(pm.installDir, instanceID, "app", "bin", "app")); err != nil {
		return false
	}
	return true
}

func (pm *PackageManager) InstallPackage(name, instanceID string) error {
	libkrunPath := filepath.Join(pm.pkgDir, "github_com__tomyedwab__yesterday__libkrun.zip")
	err := Unzip(libkrunPath, filepath.Join(pm.installDir, instanceID))
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(pm.installDir, instanceID, "db"), 0755)
	if err != nil {
		return err
	}

	pkgPath := filepath.Join(pm.pkgDir, name) + ".zip"
	err = Unzip(pkgPath, filepath.Join(pm.installDir, instanceID, "app"))
	if err != nil {
		return err
	}

	return nil
}
