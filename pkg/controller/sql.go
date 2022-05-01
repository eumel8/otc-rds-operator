package controller

import (
	"database/sql"
	"fmt"

	// "reflect"

	rdsv1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	_ "github.com/go-sql-driver/mysql"
)

// type UserList map[string]rdsv1alpha1.Users

// func createSqlUser(newRds *rdsv1alpha1.Rds) error {

func createSqlUser(newRds *rdsv1alpha1.Rds) error {

	if newRds.Spec.Datastoretype == "MySQL" {

		db, err := sql.Open("mysql", "root:"+newRds.Spec.Password+"@tcp("+newRds.Status.Ip+":"+newRds.Spec.Port+")/mysql")
		defer db.Close()

		if err != nil {
			err := fmt.Errorf("error connecting database: %v", err)
			return err
		}

		for _, su := range *newRds.Spec.Users {
			res, err := db.Query("SELECT user FROM user where user = '" + su.Name + "'")

			if err != nil {
				err := fmt.Errorf("error query user: %v", err)
				return err
			}

			if res != nil {
				fmt.Println("grant access here and create user")
				_, err := db.Query("CREATE USER '" + su.Name + "'@'" + su.Host + "' IDENTIFIED BY '" + su.Password + "'")
				fmt.Printf("error creating user: %v\n", err)

				for _, pr := range su.Privileges {
					fmt.Println("PRIV")
					// this query must be validated against sql injection
					_, err := db.Query(pr)
					if err != nil {
						fmt.Printf("error creating privileges: %v\n", err)
					}
					_, err = db.Query("FLUSH PRIVILEGES")

					if err != nil {
						fmt.Printf("error flush privileges: %v\n", err)
					}

				}
			}
		}

		for _, ds := range *&newRds.Spec.Databases {
			res, err := db.Query("SELECT schema_name FROM information_schema.schemata WHERE schema_name='" + ds + "'")
			if err != nil {
				err := fmt.Errorf("error query user: %v\n", err)
				return err
			}

			if res != nil {
				fmt.Println("create database")
				_, err := db.Query("CREATE DATABASE '" + ds + "")
				if err != nil {
					fmt.Printf("error creating database: %v\n", err)
				}
			}
		}

	} else {
		return fmt.Errorf("unsupported database type for user management")
	}
	return nil
}
