package storage

import "github.com/ortuman/jackal/model"

// userStorage defines storage operations for users
type userStorage interface {
	InsertOrUpdateUser(user *model.User) error
	DeleteUser(username string) error
	FetchUser(username string) (*model.User, error)
	UserExists(username string) (bool, error)
}

// InsertOrUpdateUser inserts a new user entity into storage,
// or updates it in case it's been previously inserted.
func InsertOrUpdateUser(user *model.User) error {
	return instance().InsertOrUpdateUser(user)
}

// DeleteUser deletes a user entity from storage.
func DeleteUser(username string) error {
	return instance().DeleteUser(username)
}

// FetchUser retrieves from storage a user entity.
func FetchUser(username string) (*model.User, error) {
	return instance().FetchUser(username)
}

// UserExists returns whether or not a user exists within storage.
func UserExists(username string) (bool, error) {
	return instance().UserExists(username)
}
