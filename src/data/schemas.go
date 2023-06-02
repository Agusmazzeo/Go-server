package data

type Customers struct {
	Number           int    `gorm:"primaryKey;column:customerNumber;type:int(11)"`
	Name             string `gorm:"column:customerName;type:varchar(50)"`
	ContactLastName  string `gorm:"column:contactLastName;type:varchar(50)"`
	ContactFirstName string `gorm:"column:contactFirstName;type:varchar(50)"`
	Phone            string `gorm:"type:varchar(50)"`
}

type Employees struct {
	Number           int    `gorm:"primaryKey;column:employeeNumber;type:int(11)"`
	FirstName        string `gorm:"column:firstName;type:varchar(50)"`
	LastName         string `gorm:"column:lastName;type:varchar(50)"`
	Email            string `gorm:"column:email;type:varchar(50)"`
	ContactFirstName string `gorm:"column:contactFirstName;type:varchar(50)"`
	JobTitle         string `gorm:"column:jobTitle;type:varchar(50)"`
}
