package controller

import (
	"database/sql"
	"fmt"
	"regexp"

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

			stmt, err := db.Prepare("SELECT user FROM user where user = ?")
			if err != nil {
				err := fmt.Errorf("error prepare query user: %v", err)
				return err
			}
			defer stmt.Close()
			res, err := stmt.Query(su.Name)
			if err != nil {
				err := fmt.Errorf("error execute query user: %v", err)
				return err
			}

			if !res.Next() {
				c.logger.Debug("create sql user ", su.Name)
				// stmt, err := db.Prepare("CREATE USER IF NOT EXISTS `?`@`?` IDENTIFIED BY `?`")
				// if err != nil {
				// 	err := fmt.Errorf("error prepare creating user: %v", err)
				// 	return err
				// }
				// defer stmt.Close()
				// _, err = stmt.Exec(su.Name, su.Host, su.Password)
				_, err := db.Query("CREATE USER IF NOT EXISTS '" + su.Name + "'@'" + su.Host + "' IDENTIFIED BY '" + su.Password + "'")
				if err != nil {
					err := fmt.Errorf("error execute creating user: %v", err)
					return err
				}

				for _, pr := range su.Privileges {
					c.logger.Debug("create privileges user ", su.Name)
					validGrant, err := regexp.Compile(`^[a-zA-Z0-9._%*'\s]$`)
					if err != nil {
						err := fmt.Errorf("error compile regex: %v", err)
						return err
					}
					if validGrant.MatchString(pr) {
						_, err := db.Query(pr)
						if err != nil {
							c.logger.Error("error creating privileges: %v\n", err)
						}
						_, err = db.Query("FLUSH PRIVILEGES")
						if err != nil {
							c.logger.Error("error flush privileges: %v\n", err)
						}
					} else {
						c.logger.Error("error validating grant: %s\n", pr)
					}

				}
			}
		}

		for _, ds := range newRds.Spec.Databases {
			c.logger.Debug("query existing database ", ds)
			stmt, err := db.Prepare("SELECT schema_name FROM information_schema.schemata WHERE schema_name= ?")
			if err != nil {
				err := fmt.Errorf("error prepare query schema: %v", err)
				return err
			}
			defer stmt.Close()
			res, err := stmt.Query(ds)
			if err != nil {
				err := fmt.Errorf("error execute query schema: %v", err)
				return err
			}

			if !res.Next() {
				c.logger.Debug("create database ", ds)
				validSchema, err := regexp.Compile(`^[a-zA-Z0-9_%*]$`)
				if err != nil {
					c.logger.Error("error compile regex for schema: %v", err)
					return err
				}
				if validSchema.MatchString(ds) {
					_, err := db.Query("CREATE DATABASE IF NOT EXISTS " + ds)
					if err != nil {
						c.logger.Error("error creating database: %v\n", err)
					}
				} else {
					c.logger.Error("error schema name validation: %s\n", ds)
				}
			}
		}

	} else {
		return fmt.Errorf("unsupported database type for user management")
	}
	return nil
}
