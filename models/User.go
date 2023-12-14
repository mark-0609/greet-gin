package models

import (
	"github.com/olivere/elastic/v7"
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

// bool query 条件
type EsSearch struct {
	MustQuery    []elastic.Query
	MustNotQuery []elastic.Query
	ShouldQuery  []elastic.Query
	Filters      []elastic.Query
	Sorters      []elastic.Sorter
	From         int //分页
	Size         int
}

type SearchRequest struct {
	Nickname  string `json:"nickname"  form:"nickname"`
	Phone     string `json:"phone"  form:"phone"`
	Identity  string `json:"identity"  form:"identity"`
	Ancestral string `json:"ancestral"  form:"ancestral"`
	Num       int    `json:"num"  form:"num"`
	Size      int    `json:"size"  form:"size"`
}

func (r *SearchRequest) ToFilter() *EsSearch {
	var search EsSearch
	if len(r.Nickname) != 0 {
		search.ShouldQuery = append(search.ShouldQuery, elastic.NewMatchQuery("Nickname", r.Nickname))
	}
	if len(r.Phone) != 0 {
		search.ShouldQuery = append(search.ShouldQuery, elastic.NewMatchQuery("Phone", r.Phone))
	}
	if len(r.Ancestral) != 0 {
		search.ShouldQuery = append(search.ShouldQuery, elastic.NewMatchQuery("Ancestral", r.Ancestral))
	}
	if len(r.Identity) != 0 {
		search.ShouldQuery = append(search.ShouldQuery, elastic.NewMatchQuery("Identity", r.Identity))
	}

	if search.Sorters == nil {
		search.Sorters = append(search.Sorters, elastic.NewFieldSort("create_time").Desc())
	}

	search.From = (r.Num - 1) * r.Size
	search.Size = r.Size
	return &search
}
