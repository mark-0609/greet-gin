package service

import (
	"context"
	"fmt"
	"github.com/olivere/elastic/v7"
	"greet_gin/models"
	"reflect"
	"strconv"
)

type UserService struct {
	Es *UserES
}

const EsRetryLimit = 3

var mappingTpl = `{
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

type UserES struct {
	Client  *elastic.Client
	Index   string
	Mapping string
}

func NewUserService(es *UserES) *UserService {
	return &UserService{
		Es: es,
	}
}

func NewUserES(client *elastic.Client) *UserES {
	index := fmt.Sprintf("%s_%s", "service", "user")
	userEs := &UserES{
		Client:  client,
		Index:   index,
		Mapping: mappingTpl,
	}

	userEs.init()

	return userEs
}

func (es *UserES) init() {
	ctx := context.Background()

	exists, err := es.Client.IndexExists(es.Index).Do(ctx)
	if err != nil {
		fmt.Printf("userEs init exist failed err is %s\n", err)
		return
	}

	if !exists {
		_, err := es.Client.CreateIndex(es.Index).Body(es.Mapping).Do(ctx)
		if err != nil {
			fmt.Printf("userEs init failed err is %s\n", err)
			return
		}
	}
}

func (es *UserES) BatchAdd(ctx context.Context, user []models.User) error {
	var err error
	for i := 0; i < EsRetryLimit; i++ {
		if err = es.batchAdd(ctx, user); err != nil {
			fmt.Println("batch add failed ", err)
			continue
		}
		return err
	}
	return err
}

func (es *UserES) batchAdd(ctx context.Context, user []models.User) error {
	req := es.Client.Bulk().Index(es.Index)
	for _, u := range user {
		//u.UpdateTime = uint64(time.Now().UnixNano()) / uint64(time.Millisecond)
		//u.CreateTime = uint64(time.Now().UnixNano()) / uint64(time.Millisecond)
		doc := elastic.NewBulkIndexRequest().Id(strconv.Itoa(u.Id)).Doc(u)
		req.Add(doc)
	}
	if req.NumberOfActions() < 0 {
		return nil
	}
	res, err := req.Do(ctx)
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

func (es *UserES) BatchUpdate(ctx context.Context, user []models.User) error {
	var err error
	for i := 0; i < EsRetryLimit; i++ {
		if err = es.batchUpdate(ctx, user); err != nil {
			continue
		}
		return err
	}
	return err
}

func (es *UserES) batchUpdate(ctx context.Context, user []models.User) error {
	req := es.Client.Bulk().Index(es.Index)
	for _, u := range user {
		//u.UpdateTime = uint64(time.Now().UnixNano()) / uint64(time.Millisecond)
		doc := elastic.NewBulkUpdateRequest().Id(strconv.Itoa(u.Id)).Doc(u)
		req.Add(doc)
	}

	if req.NumberOfActions() < 0 {
		return nil
	}
	res, err := req.Do(ctx)
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

func (es *UserES) BatchDel(ctx context.Context, user []models.User) error {
	var err error
	for i := 0; i < EsRetryLimit; i++ {
		if err = es.batchDel(ctx, user); err != nil {
			continue
		}
		return err
	}
	return err
}

func (es *UserES) batchDel(ctx context.Context, user []models.User) error {
	req := es.Client.Bulk().Index(es.Index)
	for _, u := range user {
		doc := elastic.NewBulkDeleteRequest().Id(strconv.Itoa(u.Id))
		req.Add(doc)
	}

	if req.NumberOfActions() < 0 {
		return nil
	}

	res, err := req.Do(ctx)
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
func (es *UserES) MGet(ctx context.Context, IDS []uint64) ([]models.User, error) {
	userES := make([]models.User, 0, len(IDS))
	idStr := make([]string, 0, len(IDS))
	for _, id := range IDS {
		idStr = append(idStr, strconv.FormatUint(id, 10))
	}
	resp, err := es.Client.Search(es.Index).Query(
		elastic.NewIdsQuery().Ids(idStr...)).Size(len(IDS)).Do(ctx)

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

type SearchResult struct {
	List  []models.User `json:"list"`
	Count int64         `json:"count"`
}

func (es *UserES) Search(ctx context.Context, filter *models.EsSearch) (SearchResult, error) {
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
	resp, err := service.Do(ctx)
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
