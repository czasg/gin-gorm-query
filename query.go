package webquery

import (
	"gorm.io/gorm"
	"strconv"
	"strings"
)

var PageParam = "page"         // 翻页参数，默认为 page
var PageSizeParam = "pageSize" // 翻页面数，默认 pageSize
var DefaultPageSize = 10       // 默认每一页的条目数量
var MaxPageSize = 100          // 最大允许查询的数量
var SortParam = "sort"         // 排序字段，默认 sort

type Config struct {
	PageParam       string
	PageSizeParam   string
	DefaultPageSize int
	MaxPageSize     int
	SortParam       string
}

func (c *Config) Default() *Config {
	if c.PageParam == "" {
		c.PageParam = PageParam
	}
	if c.PageSizeParam == "" {
		c.PageSizeParam = PageSizeParam
	}
	if c.DefaultPageSize == 0 {
		c.DefaultPageSize = DefaultPageSize
	}
	if c.MaxPageSize == 0 {
		c.MaxPageSize = MaxPageSize
	}
	if c.SortParam == "" {
		c.SortParam = SortParam
	}
	return c
}

type Query struct {
	Filters  []Filter
	Page     int
	PageSize int
	Sorts    []Sort
	sort     string
	Config   *Config
}

func (q *Query) Parse(c IQuery) error {
	if q.Config == nil {
		q.Config = &Config{}
	}
	q.Config = q.Config.Default()
	q.parsePage(c)
	q.parseSort(c)
	return q.parseFilter(c)
}

func (q *Query) parsePage(c IQuery) {
	q.Page, _ = strconv.Atoi(c.Query(q.Config.PageParam))
	if q.Page < 1 {
		q.Page = 1
	}
	q.PageSize, _ = strconv.Atoi(c.Query(q.Config.PageSizeParam))
	if q.PageSize < 1 {
		q.PageSize = 1
	}
	if q.Config.MaxPageSize > 0 && q.PageSize > q.Config.MaxPageSize {
		q.PageSize = q.Config.MaxPageSize
	}
}

func (q *Query) parseSort(c IQuery) {
	q.sort = strings.TrimSpace(c.Query(q.Config.SortParam))
}

func (q *Query) parseFilter(c IQuery) error {
	for _, filter := range q.Filters {
		err := filter.Parse(c)
		if err != nil {
			return err
		}
	}
	return nil
}

func (q *Query) Bind(db *gorm.DB) *gorm.DB {
	db = q.BindFilter(db)
	db = q.BindPage(db)
	db = q.BindSort(db)
	return db
}

func (q *Query) BindPage(db *gorm.DB) *gorm.DB {
	offset := (q.Page - 1) * q.PageSize
	return db.Offset(offset).Limit(q.PageSize)
}

func (q *Query) BindSort(db *gorm.DB) *gorm.DB {
	if q.sort == "" {
		return db
	}
	for _, sort := range strings.Split(q.sort, ",") {
		sort = strings.TrimSpace(sort)
		if sort == "" {
			continue
		}
		sortKey := sort
		sortMode := "ASC"
		if strings.HasPrefix(sortKey, "-") {
			sortMode = "DESC"
		}
		// sql 防注入
		sortKey = sqlAntiInject(sortKey)

		for _, s := range q.Sorts {
			if s.Key != sortKey {
				continue
			}
			if s.Field != "" {
				sortKey = s.Field
			}
			db = db.Order(sortKey + " " + sortMode)
			break
		}
	}
	return db
}

func (q *Query) BindFilter(db *gorm.DB) *gorm.DB {
	for _, filter := range q.Filters {
		db = filter.Bind(db)
	}
	return db
}

type Sort struct {
	Key   string
	Field string
}
