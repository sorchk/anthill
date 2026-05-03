package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID           int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string         `gorm:"uniqueIndex;size:255;not null" json:"username"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	Role         string         `gorm:"size:50;default:viewer" json:"role"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

func GetUserByID(db *gorm.DB, id int64) (*User, error) {
	var user User
	err := db.First(&user, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, err
	}
	return &user, err
}

func GetUserByUsername(db *gorm.DB, username string) (*User, error) {
	var user User
	err := db.Where("username = ?", username).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, err
	}
	return &user, err
}

func ListUsers(db *gorm.DB) ([]User, error) {
	var users []User
	err := db.Order("id").Find(&users).Error
	return users, err
}

func CreateUser(db *gorm.DB, username, password, role string) (*User, error) {
	user := &User{
		Username: username,
		Role:     role,
	}
	if err := user.SetPassword(password); err != nil {
		return nil, err
	}
	if err := db.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func UpdateUser(db *gorm.DB, id int64, username, role string) error {
	return db.Model(&User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"username":  username,
		"role":      role,
		"updated_at": time.Now(),
	}).Error
}

func UpdateUserPassword(db *gorm.DB, id int64, password string) error {
	user := &User{}
	if err := db.First(user, id).Error; err != nil {
		return err
	}
	if err := user.SetPassword(password); err != nil {
		return err
	}
	return db.Model(user).Update("password_hash", user.PasswordHash).Error
}

func DeleteUser(db *gorm.DB, id int64) error {
	result := db.Delete(&User{}, id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}