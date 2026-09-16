package scope

import (
	"context"
	"crypto/rand"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const defaultTestDSN = "gorm:gorm@tcp(localhost:3306)/gorm?charset=utf8&parseTime=True&loc=Local"

type User struct {
	gorm.Model
	Name     string     `gorm:"column:name"`
	Age      uint       `gorm:"column:age"`
	Sex      string     `gorm:"column:sex"`
	Birthday *time.Time `gorm:"column:birthday"`
	Address  *string    `gorm:"column:address"`
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("MySQL integration test")
	}

	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = defaultTestDSN
	}
	config, err := mysqldriver.ParseDSN(dsn)
	require.NoError(t, err)
	config.ParseTime = true
	if config.Timeout == 0 {
		config.Timeout = 3 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 3 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 3 * time.Second
	}
	connector, err := mysqldriver.NewConnector(config)
	require.NoError(t, err)
	conn := sql.OpenDB(connector)
	t.Cleanup(func() { assert.NoError(t, conn.Close()) })

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	require.NoError(t, conn.PingContext(ctx), "MySQL is unavailable")

	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn}), &gorm.Config{
		DisableAutomaticPing: true,
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: "fries_scope_" + strings.ToLower(rand.Text()) + "_",
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 10*time.Second)
		defer cancel()
		assert.NoError(t, db.WithContext(ctx).Migrator().DropTable(&User{}))
	})
	require.NoError(t, db.WithContext(ctx).Migrator().CreateTable(&User{}))
	return db.WithContext(t.Context())
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
