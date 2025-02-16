package scope

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	DB  *gorm.DB
	dsn = "gorm:gorm@tcp(localhost:3306)/gorm?charset=utf8&parseTime=True&loc=Local"
)

type User struct {
	gorm.Model
	Name     string     `gorm:"column:name"`
	Age      uint       `gorm:"column:age"`
	Sex      string     `gorm:"column:sex"`
	Birthday *time.Time `gorm:"column:birthday"`
	Address  *string    `gorm:"column:address"`
}

func init() {
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Println("failed to connect database, got error", err)
		os.Exit(1)
	}

	runMigrations()
}

func runMigrations() {
	var err error
	models := []any{&User{}}

	if err = DB.Migrator().DropTable(models...); err != nil {
		log.Printf("Didn't drop table, got error %v\n", err)
		os.Exit(1)
	}

	if err = DB.AutoMigrate(models...); err != nil {
		log.Printf("Failed to auto migrate, but got error %v\n", err)
		os.Exit(1)
	}

	for _, m := range models {
		if !DB.Migrator().HasTable(m) {
			log.Printf("Didn't create table for %#v\n", m)
			os.Exit(1)
		}
	}
}

type GetUserOptions struct {
	Age      int
	Birthday *time.Time
	Address  *string
}

func GetUser(name string, opts GetUserOptions) *User {
	var (
		birthday = time.Now().Round(time.Second)
		user     = User{
			Name:     name,
			Age:      18,
			Birthday: &birthday,
		}
	)

	if opts.Age > 0 {
		user.Age = uint(opts.Age)
	}

	if opts.Birthday != nil {
		user.Birthday = opts.Birthday
	}

	if opts.Address != nil {
		user.Address = opts.Address
	}

	return &user
}

func CleanUsers() {
	DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&User{})
}
