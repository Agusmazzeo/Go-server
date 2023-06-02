package controllers

import "server/src/data"

func (c Controller) GetAllCustomers() []data.Customers {
	var customers []data.Customers
	db := c.DbHandler.GetDBClient()
	result := db.Find(&customers)
	if result.Error != nil {
		panic(result.Error)
	}
	return customers
}
