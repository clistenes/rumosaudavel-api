package repositories

import (

	sql "database/sql"
)

type UserRepository struct {
	DB *sql.DB
}
