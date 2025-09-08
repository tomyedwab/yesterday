package packages

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

// TODO: Move to a configurable setting
const TTLInterval time.Duration = time.Minute * 5

type Package struct {
	InstanceID        string          `db:"instance_id"`
	PackageHash       string          `db:"package_hash"`
	Name              string          `db:"name"`
	Version           string          `db:"version"`
	SubscriptionsJson []byte          `db:"subscriptions"`
	Subscriptions     map[string]bool `db:"-"`
	ActiveTtl         time.Time       `db:"active_ttl"`
}

const packageSchema = `
CREATE TABLE IF NOT EXISTS package_v1 (
	instance_id STRING PRIMARY KEY NOT NULL,
	package_hash STRING NOT NULL,
	name STRING NOT NULL,
	version STRING NOT NULL,
	subscriptions JSONB NOT NULL,
	active_ttl TIMESTAMP
);
`

const getPackageByInstanceIDV1Sql = `
SELECT instance_id, package_hash, name, version, subscriptions, active_ttl FROM package_v1 WHERE instance_id = $1;
`

const getPackageByHashV1Sql = `
SELECT instance_id, package_hash, name, version, subscriptions, active_ttl FROM package_v1 WHERE package_hash = $1;
`

const getActivePackagesV1Sql = `
SELECT instance_id, package_hash, name, version, subscriptions, active_ttl FROM package_v1 WHERE active_ttl > CURRENT_TIMESTAMP;
`

const insertPackageV1Sql = `
INSERT INTO package_v1 (instance_id, package_hash, name, version, subscriptions, active_ttl)
VALUES ($1, $2, $3, $4, $5, $6);
`

const updatePackageV1Sql = `
UPDATE package_v1 SET active_ttl = $1 WHERE instance_id = $2;
`

func PackageDBInit(db *sqlx.DB) error {
	_, err := db.Exec(packageSchema)
	return err
}

func PackageDBGetByInstanceID(db *sqlx.DB, instanceID string) (*Package, error) {
	var pkg Package
	err := db.Get(&pkg, getPackageByInstanceIDV1Sql, instanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	err = json.Unmarshal(pkg.SubscriptionsJson, &pkg.Subscriptions)
	if err != nil {
		return nil, err
	}
	return &pkg, err
}

func PackageDBGetByHash(db *sqlx.DB, hash string) (*Package, error) {
	var pkg Package
	err := db.Get(&pkg, getPackageByHashV1Sql, hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	err = json.Unmarshal(pkg.SubscriptionsJson, &pkg.Subscriptions)
	if err != nil {
		return nil, err
	}
	return &pkg, err
}

func PackageDBGetActive(db *sqlx.DB) ([]*Package, error) {
	var pkgs []*Package
	err := db.Select(&pkgs, getActivePackagesV1Sql)
	if err != nil {
		return nil, err
	}
	for _, pkg := range pkgs {
		err = json.Unmarshal(pkg.SubscriptionsJson, &pkg.Subscriptions)
		if err != nil {
			return nil, err
		}
	}
	return pkgs, err
}

func PackageDBInsert(db *sqlx.DB, instanceID, hash, name, version string, subscriptions map[string]bool) error {
	jsonSubscriptions, err := json.Marshal(subscriptions)
	if err != nil {
		return err
	}
	activeTTL := time.Now().UTC().Add(TTLInterval)
	_, err = db.Exec(insertPackageV1Sql, instanceID, hash, name, version, jsonSubscriptions, activeTTL)
	return err
}

func PackageDBUpdateTTL(db *sqlx.DB, instanceID string) error {
	activeTTL := time.Now().UTC().Add(TTLInterval)
	_, err := db.Exec(updatePackageV1Sql, activeTTL, instanceID)
	return err
}
