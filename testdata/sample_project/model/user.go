package model

import (
	"errors"
	"time"
)

// User 用户实体
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Age       int       `json:"age"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate 验证用户数据
func (u *User) Validate() error {
	if u.Name == "" {
		return errors.New("name is required")
	}
	if u.Age < 0 || u.Age > 150 {
		return errors.New("invalid age")
	}
	return nil
}

// IsAdult 判断是否成年
func (u *User) IsAdult() bool {
	return u.Age >= 18
}

// UpdateTimestamp 更新时间戳
func (u *User) UpdateTimestamp() {
	u.UpdatedAt = time.Now()
}

// Profile 用户档案
type Profile struct {
	UserID  int64  `json:"user_id"`
	Avatar  string `json:"avatar"`
	Bio     string `json:"bio"`
	Address Address
}

// Address 地址信息
type Address struct {
	Country  string `json:"country"`
	Province string `json:"province"`
	City     string `json:"city"`
	Street   string `json:"street"`
}

// FullAddress 获取完整地址
func (a *Address) FullAddress() string {
	return a.Country + " " + a.Province + " " + a.City + " " + a.Street
}
