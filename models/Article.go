package models

import "time"

type Article struct {
	Id          int        `gorm:"primaryKey"`
	ArticleName string     `gorm:"not null;size:64"`
	Content     string     `gorm:"not null;size:255"`
	UserId      int        `gorm:"not null" sql:"index"`
	Desc        string     `gorm:"not null;size:20"`
	Status      int        `gorm:"comment:'状态';not null;size:1" sql:"index"`
	CreatedAt   *time.Time `gorm:"not null;comment:'创建时间';type:datetime;"`
	UpdatedAt   time.Time  `gorm:"not null;autoUpdateTime:milli;comment:'更新时间';type:datetime;" json:"updated_at"`
	DeletedAt   time.Time  `gorm:"comment:'删除时间';type:datetime;" sql:"index" json:"deleted_at"`
}
