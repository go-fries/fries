package builder

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestBuilder_CreateAndCount(t *testing.T) {
	users := []*User{
		GetUser("CreateUser1", GetUserOptions{Age: 18}),
		GetUser("CreateUser2", GetUserOptions{Age: 20}),
		GetUser("CreateUser3", GetUserOptions{Age: 22}),
	}

	CleanUsers()

	builder := New[*User](DB)
	for _, user := range users {
		assert.Empty(t, user.ID)

		created := builder.Create(user)
		assert.NoError(t, created.Error)
		assert.Equal(t, int64(1), created.RowsAffected)
		assert.NotEmpty(t, user.ID)
	}

	var count int64
	require.NoError(t, builder.Count(&count).Error)
	assert.Equal(t, int64(len(users)), count)

	var dbUsers []User
	require.NoError(t, DB.Find(&dbUsers).Error)
	assert.Equal(t, len(users), len(dbUsers))
	for i, user := range dbUsers {
		assert.Equal(t, users[i].Name, user.Name)
		assert.Equal(t, users[i].Age, user.Age)
		assert.Equal(t, users[i].Birthday, user.Birthday)
		assert.Nil(t, user.Address)
	}
}

func TestBuilder_Page(t *testing.T) {
	users := []*User{
		GetUser("PageUser1", GetUserOptions{Age: 18}),
		GetUser("PageUser2", GetUserOptions{Age: 20}),
		GetUser("PageUser3", GetUserOptions{Age: 22}),
	}

	CleanUsers()

	builder := New[*User](DB)
	builder.Create(users)

	var count int64
	require.NoError(t, builder.Count(&count).Error)
	assert.Equal(t, int64(len(users)), count)

	// Page
	var dbUsers []User
	require.NoError(t, builder.Page(1, 2).Find(&dbUsers).Error)
	assert.Len(t, dbUsers, 2)
	assert.Equal(t, users[0].Name, dbUsers[0].Name)

	require.NoError(t, builder.Page(2, 2).Find(&dbUsers).Error)
	assert.Len(t, dbUsers, 1)
	assert.Equal(t, users[2].Name, dbUsers[0].Name)

	// Paginate
	paginatedUsers, err := builder.Debug().Paginate(1, 2)
	require.NoError(t, err)
	assert.Len(t, paginatedUsers, 2)
	assert.Equal(t, users[0].Name, paginatedUsers[0].Name)

	paginatedUsers, err = builder.Paginate(2, 2)
	require.NoError(t, err)
	assert.Len(t, paginatedUsers, 1)
	assert.Equal(t, users[2].Name, paginatedUsers[0].Name)
}
