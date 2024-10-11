package init_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"server/src/models"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var dbInstance *gorm.DB
var dbOnce sync.Once

func SetUpTestDatabase(log *logr.Logger) (*gorm.DB, func()) {

	dbOnce.Do(func() {
		log.Info("Setting up local DB")

		wd, err := os.Getwd() // Get the current working directory
		if err != nil {
			log.Error(err, "os.Getwd() failed")
		}

		composeFile := filepath.Join(wd, "../../../docker-compose.yaml") // Concatenate the working directory with the file

		// Check if the services are already up
		cmd := exec.Command("docker-compose", "-f", composeFile, "ps")
		out, err := cmd.Output()
		if err != nil {
			log.Error(err, "Failed while checking DB status.")
		}

		// If the output is empty, the services are not up
		if len(out) <= len("Name   Command   State   Ports\n------------------------------\n") {
			cmd := exec.Command("docker-compose", "-f", composeFile, "up", "-d", "postgres-db")
			_, err = cmd.Output()
			if err != nil {
				log.Error(err, "Failed while setting up DB.")
			}
		}

		// Check if the services are up
		cmd = exec.Command("docker-compose", "-f", composeFile, "ps")
		out, err = cmd.Output()
		if err != nil {
			log.Error(err, "Failed while checking DB status.")
		}

		if len(out) == 0 {
			log.Error(err, "no DB instance found to run tests with")
		}

		// Wait a bit for the DB to be ready
		time.Sleep(5 * time.Second)

		dsn := "host=localhost port=5440 user=user password=pass dbname=db sslmode=disable"
		dbInstance, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Error(err, "failed to connect to database")
		}

		err = dbInstance.AutoMigrate(&models.ReportSchedule{})
		if err != nil {
			log.Error(err, "failed to migrate database")
		}
	})

	cleanup := func() {
		dbInstance.Unscoped().Where("1 = 1").Delete(&models.ReportSchedule{})
	}

	return dbInstance, cleanup
}
