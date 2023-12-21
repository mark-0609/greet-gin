package user

import (
	"context"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"greet_gin/database"
	"greet_gin/models"
	"reflect"
	"strconv"
)

type ServiceUserEs struct {
	Client  *elastic.Client
	Index   string
	Mapping string
	Ctx     context.Context
}

const (
	EsRetryLimit     = 3
	ServiceUserIndex = "service_user"
	Mapping          = `{
	"mappings":{
		"properties":{
			"id": 				{ "type": "long" },
			"username": 		{ "type": "keyword" },
			"nickname":			{ "type": "text" },
			"phone":			{ "type": "keyword" },
			"age":				{ "type": "long" },
			"ancestral":		{ "type": "text" },
			"identity":         { "type": "text" },
			"update_time":		{ "type": "long" },
			"create_time":		{ "type": "long" }
			}
		}
	}`
)

func NewUserService(ctx context.Context) (*ServiceUserEs, error) {
	es := &ServiceUserEs{
		Client:  database.InitES(),
		Index:   ServiceUserIndex,
		Mapping: Mapping,
		Ctx:     ctx,
	}
	if err := es.createIndex(); err != nil {
		return &ServiceUserEs{}, err
	}
	return es, nil
}

// createIndex 创建索引
func (es *ServiceUserEs) createIndex() error {
	exists, err := es.Client.IndexExists(es.Index).Do(es.Ctx)
	if err != nil {
		logrus.Errorf("ServiceUserEs init exist failed err is %s\n", err)
		return err
	}
	if !exists {
		_, err := es.Client.CreateIndex(es.Index).Body(es.Mapping).Do(es.Ctx)
		if err != nil {
			logrus.Errorf("ServiceUserEs init failed err is %s\n", err)
			return err
		}
	}
	return nil
}

func (es *ServiceUserEs) BatchAdd(user []models.User) error {
	var err error
	for i := 0; i < EsRetryLimit; i++ {
		if err = es.batchAdd(user); err != nil {
			logrus.Errorf("batch add user failed:%v", err)
			continue
		}
		return err
	}
	return err
}

func (es *ServiceUserEs) batchAdd(user []models.User) error {
	req := es.Client.Bulk().Index(es.Index)
	for _, u := range user {
		doc := elastic.NewBulkIndexRequest().Id(strconv.Itoa(u.Id)).Doc(u)
		req.Add(doc)
	}
	res, err := req.Do(es.Ctx)
	if err != nil {
		logrus.Errorf("batchAdd do failed:%v", err)
		return err
	}
	// 任何子请求失败，该 `errors` 标志被设置为 `true` ，并且在相应的请求报告出错误明细
	// 所以如果没有出错，说明全部成功了，直接返回即可
	if !res.Errors {
		return nil
	}
	for _, it := range res.Failed() {
		if it.Error == nil {
			continue
		}
		return &elastic.Error{
			Status:  it.Status,
			Details: it.Error,
		}
	}
	return nil
}

func (es *ServiceUserEs) BatchUpdate(user []models.User) error {
	var err error
	for i := 0; i < EsRetryLimit; i++ {
		if err = es.batchUpdate(user); err != nil {
			logrus.Errorf("batch update failed:%v", err)
			continue
		}
		return err
	}
	return err
}

func (es *ServiceUserEs) batchUpdate(user []models.User) error {
	req := es.Client.Bulk().Index(es.Index)
	for _, u := range user {
		//u.UpdateTime = uint64(time.Now().UnixNano()) / uint64(time.Millisecond)
		doc := elastic.NewBulkUpdateRequest().Id(strconv.Itoa(u.Id)).Doc(u)
		req.Add(doc)
	}

	if req.NumberOfActions() < 0 {
		return nil
	}
	res, err := req.Do(es.Ctx)
	if err != nil {
		return err
	}
	// 任何子请求失败，该 `errors` 标志被设置为 `true` ，并且在相应的请求报告出错误明细
	// 所以如果没有出错，说明全部成功了，直接返回即可
	if !res.Errors {
		return nil
	}
	for _, it := range res.Failed() {
		if it.Error == nil {
			continue
		}
		return &elastic.Error{
			Status:  it.Status,
			Details: it.Error,
		}
	}

	return nil
}

func (es *ServiceUserEs) BatchDel(user []models.User) error {
	var err error
	for i := 0; i < EsRetryLimit; i++ {
		if err = es.batchDel(user); err != nil {
			logrus.Errorf("batch del user failed:%v", err)
			continue
		}
		return err
	}
	return err
}

func (es *ServiceUserEs) batchDel(user []models.User) error {
	req := es.Client.Bulk().Index(es.Index)
	for _, u := range user {
		doc := elastic.NewBulkDeleteRequest().Id(strconv.Itoa(u.Id))
		req.Add(doc)
	}

	if req.NumberOfActions() < 0 {
		return nil
	}

	res, err := req.Do(es.Ctx)
	if err != nil {
		return err
	}
	// 任何子请求失败，该 `errors` 标志被设置为 `true` ，并且在相应的请求报告出错误明细
	// 所以如果没有出错，说明全部成功了，直接返回即可
	if !res.Errors {
		return nil
	}
	for _, it := range res.Failed() {
		if it.Error == nil {
			continue
		}
		return &elastic.Error{
			Status:  it.Status,
			Details: it.Error,
		}
	}

	return nil
}

// 根据id 批量获取
func (es *ServiceUserEs) GetUser(ids []int) ([]models.User, error) {
	userES := make([]models.User, 0, len(ids))
	idStr := make([]string, 0, len(ids))
	for _, id := range ids {
		idStr = append(idStr, strconv.Itoa(id))
	}
	resp, err := es.Client.Search(es.Index).Query(
		elastic.NewIdsQuery().Ids(idStr...)).Size(len(ids)).Do(es.Ctx)

	if err != nil {
		return nil, err
	}

	if resp.TotalHits() == 0 {
		return nil, nil
	}
	for _, e := range resp.Each(reflect.TypeOf(&models.User{})) {
		us := e.(models.User)
		userES = append(userES, us)
	}
	return userES, nil
}

type UserSearchReq struct {
	Nickname  string `json:"nickname"`
	Phone     string `json:"phone"`
	Identity  string `json:"identity"`
	Ancestral string `json:"ancestral"`
	Num       int    `json:"num"`
	Size      int    `json:"size"`
}

type SearchResult struct {
	List  []models.User `json:"list"`
	Count int64         `json:"count"`
}

// ToFilter 请求参数过滤
func (r *UserSearchReq) ToFilter() *database.ElasticSearch {
	var search database.ElasticSearch
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

func (es *ServiceUserEs) Search(filter *database.ElasticSearch) (SearchResult, error) {
	boolQuery := elastic.NewBoolQuery()
	boolQuery.Must(filter.MustQuery...)
	boolQuery.MustNot(filter.MustNotQuery...)
	boolQuery.Should(filter.ShouldQuery...)
	boolQuery.Filter(filter.Filters...)

	// 当should不为空时，保证至少匹配should中的一项
	if len(filter.MustQuery) == 0 && len(filter.MustNotQuery) == 0 && len(filter.ShouldQuery) > 0 {
		boolQuery.MinimumShouldMatch("1")
	}

	service := es.Client.Search().Index(es.Index).Query(boolQuery).SortBy(filter.Sorters...).From(filter.From).Size(filter.Size)
	resp, err := service.Do(es.Ctx)
	if err != nil {
		return SearchResult{}, err
	}

	if resp.TotalHits() == 0 {
		return SearchResult{}, nil
	}
	userES := make([]models.User, 0)
	for _, e := range resp.Each(reflect.TypeOf(&models.User{})) {
		us := e.(*models.User)
		userES = append(userES, *us)
	}
	return SearchResult{
		userES,
		resp.TotalHits(),
	}, nil
}
