package controller

import (
	"database/sql"
	"fmt"

	// "reflect"

	rdsv1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	_ "github.com/go-sql-driver/mysql"
)

// type UserList map[string]rdsv1alpha1.Users

func createSqlUser(newRds *rdsv1alpha1.Rds) error {

	if newRds.Spec.Datastoretype == "MySQL" {

		db, err := sql.Open("mysql", "root:"+newRds.Spec.Password+"@tcp("+newRds.Status.Ip+":"+newRds.Spec.Port+")/mysql")
		defer db.Close()

		if err != nil {
			err := fmt.Errorf("error connecting database: %v", err)
			return err
		}

		for _, su := range *newRds.Spec.Users {

			fmt.Println("SQL")
			fmt.Println(su.Name)

			res, err := db.Query("SELECT user FROM users where user == " + su.Name)

			if err != nil {
				err := fmt.Errorf("error query user: %v", err)
				return err
			}

			for res.Next() {
				err := res.Scan(&su.Name)
				if err != nil {
					fmt.Println("grant access here and create user")
					continue
				}
				fmt.Println("next")
			}
		}

	} else {
		return fmt.Errorf("unsupported database type for user management")
	}
	return nil
}
