package controllers

import "server/src/data"

func (c Controller) GetAllEmployees() []data.Employees {
	var employees []data.Employees
	db := c.DbHandler.GetDBClient()
	result := db.Find(&employees)
	if result.Error != nil {
		panic(result.Error)
	}
	return employees
}
