package controller

import (
	"database/sql"
	"fmt"
	"strings"

	rdsv1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	_ "github.com/go-sql-driver/mysql"
)

func (c *Controller) CreateSqlUser(newRds *rdsv1alpha1.Rds) error {

	if newRds.Spec.Datastoretype == "MySQL" {

		c.logger.Debug("connecting database")
		db, err := sql.Open("mysql", "root:"+newRds.Spec.Password+"@tcp("+newRds.Status.Ip+":"+newRds.Spec.Port+")/mysql")

		if err != nil {
			err := fmt.Errorf("error connecting database: %v", err)
			return err
		}

		err = db.Ping()
		if err != nil {
			err := fmt.Errorf("error connecting database as root: %v", err)
			return err
		}

		defer db.Close()

		for _, su := range *newRds.Spec.Users {

			res, err := db.Query("SELECT user FROM user where user = '" + su.Name + "'")
			if err != nil {
				err := fmt.Errorf("error query user: %v", err)
				return err
			}

			if !res.Next() {
				c.logger.Debug("create sql user ", su.Name)
				_, err := db.Query("CREATE USER '" + su.Name + "'@'" + su.Host + "' IDENTIFIED BY '" + su.Password + "'")
				if err != nil {
					err := fmt.Errorf("error creating user: %v", err)
					return err
				}

				for _, pr := range su.Privileges {
					c.logger.Debug("create privileges user ", su.Name)
					// this query must be validated against sql injection
					if strings.Contains(pr, "GRANT") {
						_, err := db.Query(pr)
						if err != nil {
							err = fmt.Errorf("error creating privileges: %v\n", err)
						}
						_, err = db.Query("FLUSH PRIVILEGES")
						if err != nil {
							err = fmt.Errorf("error flush privileges: %v\n", err)
						}
					} else {
						err = fmt.Errorf("privileges contains no GRANT: %s\n", pr)
					}

				}
			}
		}

		for _, ds := range newRds.Spec.Databases {
			c.logger.Debug("query existing database ", ds)
			res, err := db.Query("SELECT schema_name FROM information_schema.schemata WHERE schema_name='" + ds + "'")
			if err != nil {
				err := fmt.Errorf("error query user: %v", err)
				return err
			}

			if !res.Next() {
				c.logger.Debug("create database ", ds)
				_, err := db.Query("CREATE DATABASE " + ds)
				if err != nil {
					err = fmt.Errorf("error creating database: %v\n", err)
				}
			}
		}

	} else {
		return fmt.Errorf("unsupported database type for user management")
	}
	return nil
}
