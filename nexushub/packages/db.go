package packages

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jmoiron/sqlx"
)

type Package struct {
	InstanceID        string   `db:"instance_id"`
	PackageHash       string   `db:"package_hash"`
	Name              string   `db:"name"`
	Version           string   `db:"version"`
	SubscriptionsJson []byte   `db:"subscriptions"`
	Subscriptions     []string `db:"-"`
}

const packageSchema = `
CREATE TABLE IF NOT EXISTS package_v1 (
	instance_id STRING PRIMARY KEY NOT NULL,
	package_hash STRING NOT NULL,
	name STRING NOT NULL,
	version STRING NOT NULL,
	subscriptions JSONB NOT NULL
);
`

const getPackageByInstanceIDV1Sql = `
SELECT instance_id, package_hash, name, version, subscriptions FROM package_v1 WHERE instance_id = $1;
`

const getPackageByHashV1Sql = `
SELECT instance_id, package_hash, name, version, subscriptions FROM package_v1 WHERE package_hash = $1;
`

const insertPackageV1Sql = `
INSERT INTO package_v1 (instance_id, package_hash, name, version, subscriptions)
VALUES ($1, $2, $3, $4, $5);
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

func PackageDBInsert(db *sqlx.DB, instanceID, hash, name, version string, subscriptions []string) error {
	jsonSubscriptions, err := json.Marshal(subscriptions)
	if err != nil {
		return err
	}
	_, err = db.Exec(insertPackageV1Sql, instanceID, hash, name, version, jsonSubscriptions)
	return err
}
