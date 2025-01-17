package webquery

import (
	"errors"
	"gorm.io/gorm"
)

func NewModel(db *gorm.DB, value interface{}) *Model {
	return &Model{
		DB:    db,
		Value: value,
	}
}

type Model struct {
	DB    *gorm.DB
	Value interface{}
}

func (m *Model) List(query *Query) (rets []interface{}, err error) {
	err = query.Bind(m.DB).Model(m.Value).Find(&rets).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	return rets, nil
}

func (m *Model) ListAndCount(query *Query) (rets []interface{}, count int64, err error) {
	db := query.BindFilter(m.DB).Model(m.Value)
	if err = db.Count(&count).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	db = query.BindPage(db)
	db = query.BindSort(db)
	err = db.Find(&rets).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	return rets, count, nil
}
