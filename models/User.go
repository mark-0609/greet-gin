package models

import (
	"time"
)

type User struct {
	Id        int        `gorm:"primaryKey"`
	UserName  string     `gorm:"not null;size:64"`
	Nickname  string     `gorm:"not null;size:64"`
	Phone     string     `gorm:"not null;size:11"`
	Age       int        `gorm:"not null"`
	PassWord  string     `gorm:"not null;size:225"`
	Ancestral string     `gorm:"not null;size:225"`
	Identity  string     `gorm:"not null;size:225"`
	CreatedAt *time.Time `gorm:"not null;comment:'创建时间';type:datetime;"  `
	UpdatedAt time.Time  `gorm:"not null;autoUpdateTime:milli;comment:'更新时间';type:datetime;" json:"updated_at"`
	DeletedAt time.Time  `gorm:"comment:'删除时间';type:datetime;" sql:"index" json:"deleted_at"`
}

func (a *User) BeforeCreate() error {
	// d := 24 * time.Hour
	// jwtRes := new(tools.JWT)
	// stdClaims := jwt.StandardClaims{
	// 	ExpiresAt: time.Now().Add(d).Unix(),
	// 	IssuedAt:  time.Now().Unix(),
	// }
	// claims := tools.CustomClaims{
	// 	Time:           time.Now().Unix(),
	// 	StandardClaims: stdClaims,
	// }
	// token, err := jwtRes.CreateToken(claims)
	// if err != nil {
	// 	logrus.WithError(err).Fatal("config is wrong, can not generate jwt")
	// }
	// a.Token = token
	return nil
}
