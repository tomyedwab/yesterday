package packages

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/nexushub/processes"
	"github.com/tomyedwab/yesterday/nexushub/types"
)

type PackageManager struct {
	DB         *sqlx.DB
	pkgDir     string
	installDir string
}

func NewPackageManager() (*PackageManager, error) {
	pkgDir := os.Getenv("PKG_DIR")
	if pkgDir == "" {
		pkgDir = "/usr/local/etc/nexushub/packages"
	}

	installDir := os.Getenv("INSTALL_DIR")
	if installDir == "" {
		installDir = "/usr/local/etc/nexushub/install"
	}

	db := sqlx.MustConnect("sqlite3", path.Join(installDir, "packages.db"))
	err := PackageDBInit(db)
	if err != nil {
		return nil, err
	}

	return &PackageManager{
		DB:         db,
		pkgDir:     pkgDir,
		installDir: installDir,
	}, nil
}

func (pm *PackageManager) GetPkgDir() string {
	return pm.pkgDir
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
	pkg, err := PackageDBGetByInstanceID(pm.DB, instanceID)
	if err != nil {
		return false
	}
	if pkg == nil {
		return false
	}
	if pm.installDir == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join(pm.installDir, instanceID, "app", "bin", "app")); err != nil {
		return false
	}
	return true
}

func (pm *PackageManager) GetPackageByInstanceID(id string) (*Package, error) {
	return PackageDBGetByInstanceID(pm.DB, id)
}

func (pm *PackageManager) GetPackageByHash(hash string) (*Package, error) {
	return PackageDBGetByHash(pm.DB, hash)
}

func (pm *PackageManager) GetActivePackages() ([]*Package, error) {
	return PackageDBGetActive(pm.DB)
}

func (pm *PackageManager) InstallPackage(name, hash, instanceID string) error {
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

	// Read manifest.json in the newly-installed directory
	manifestPath := filepath.Join(pm.installDir, instanceID, "app", "manifest.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest types.PackageManifest
	err = json.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		return err
	}

	subscriptionsMap := make(map[string]bool)
	for _, subscription := range manifest.Subscriptions {
		subscriptionsMap[subscription] = true
	}

	err = PackageDBInsert(pm.DB, instanceID, hash, manifest.Name, manifest.Version, subscriptionsMap)
	return err
}

func (pm *PackageManager) SetPackageActive(instanceID string) error {
	return PackageDBUpdateTTL(pm.DB, instanceID)
}

func (pm *PackageManager) GetAppInstances() ([]processes.AppInstance, error) {
	packages, err := PackageDBGetActive(pm.DB)
	if err != nil {
		return nil, err
	}
	ret := make([]processes.AppInstance, len(packages))
	for i, pkg := range packages {
		ret[i] = processes.AppInstance{
			InstanceID:    pkg.InstanceID,
			HostName:      "",
			PkgPath:       filepath.Join(pm.installDir, pkg.InstanceID),
			Subscriptions: pkg.Subscriptions,
		}
	}
	return ret, nil
}
