package query

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"strings"
	"strconv"
	"time"
)

type IQuery interface {
	Query(string) string
}

var sqlAntiInjectRules = []string{`%`, `#`, `-`, `'`, `"`, "/", "*"}

type ParseFunc func(f Filter, c IQuery) error
type BindFunc func(f Filter, db *gorm.DB) *gorm.DB

func sqlAntiInject(sql string) string {
	for _, rule := range sqlAntiInjectRules {
		sql = strings.ReplaceAll(sql, rule, "")
	}
	return strings.TrimSpace(sql)
}

func parseValue(f Filter, c IQuery) (string, error) {
	if f.GetKey() == "" {
		return "", errors.New("filter key is empty")
	}
	value := c.Query(f.GetKey())
	if value == "" {
		if f.IsRequired() {
			return "", fmt.Errorf("filter key [%s] is required", f.GetKey())
		}
		return "", nil
	}
	return strings.TrimSpace(value), nil
}

type Filter interface {
	Parse(c IQuery) error
	Bind(db *gorm.DB) *gorm.DB
	GetKey() string
	GetFields() []string
	GetSymbol() string
	GetValue() interface{}
	SetValue(value interface{})
	IsRequired() bool
}

// StringFilter
type StringFilter struct {
	Key        string    // 前端传入参数名
	Field      string    // 数据库单列
	Fields     []string  // 数据库多列
	Symbol     string    // 查询条件
	ParseValue string    // 解析值
	Value      string    // 用户指定的过滤值
	parsed     bool      // 是否解析过
	Required   bool      // 是否必选
	ParseFunc  ParseFunc // 自定义转换函数
	BindFunc   BindFunc  // 自定义绑定到查询条件
}

func (f *StringFilter) GetKey() string {
	return f.Key
}

func (f *StringFilter) GetFields() []string {
	if len(f.Fields) > 0 {
		return f.Fields
	}
	if f.Field == "" {
		f.Field = f.Key
	}
	f.Fields = []string{f.Field}
	return f.Fields
}

func (f *StringFilter) GetSymbol() string {
	if f.Symbol == "" {
		return "="
	}
	return strings.ToUpper(f.Symbol)
}

func (f *StringFilter) IsRequired() bool {
	return f.Required
}

func (f *StringFilter) GetValue() interface{} {
	value := f.Value
	if value == "" {
		value = f.ParseValue
	}
	if value == "" {
		return ""
	}
	if f.GetSymbol() == "LIKE" {
		value = "%" + value + "%"
	} else if strings.HasPrefix(f.GetSymbol(), "LIKER") {
		value = value + "%"
	}
	return value
}

func (f *StringFilter) SetValue(value interface{}) {
	f.ParseValue, _ = value.(string)
	f.parsed = true
}

func (f *StringFilter) Parse(c IQuery) error {
	if f.ParseFunc != nil {
		return f.ParseFunc(f, c)
	}
	value, err := parseValue(f, c)
	if err != nil || value == "" {
		return err
	}
	value = sqlAntiInject(value)
	f.SetValue(value)
	return nil
}

func (f *StringFilter) Bind(db *gorm.DB) *gorm.DB {
	if !f.parsed {
		return db
	}
	if f.BindFunc != nil {
		return f.BindFunc(f, db)
	}
	fields := f.GetFields()
	if len(fields) == 1 {
		return db.Where(fields[0]+" "+f.GetSymbol()+" ?", f.GetValue())
	}
	query := make([]string, len(fields))
	values := make([]interface{}, len(fields))
	for i, field := range f.GetFields() {
		query[i] = field + " " + f.GetSymbol() + " ?"
		values[i] = f.GetValue()
	}
	return db.Where(strings.Join(query, " OR "), values...)
}

// StringArrayFilter
type StringArrayFilter struct {
	Key        string    // 前端传入参数名
	Field      string    // 数据库单列
	Fields     []string  // 数据库多列
	Symbol     string    // 查询条件
	Sep        string    // 分隔符
	ParseValue []string  // 解析值
	Value      []string  // 用户指定的过滤值
	parsed     bool      // 是否解析过
	Required   bool      // 是否必选
	ParseFunc  ParseFunc // 自定义转换函数
	BindFunc   BindFunc  // 自定义绑定到查询条件
}

func (f *StringArrayFilter) GetKey() string {
	return f.Key
}

func (f *StringArrayFilter) GetFields() []string {
	if len(f.Fields) > 0 {
		return f.Fields
	}
	if f.Field == "" {
		f.Field = f.Key
	}
	f.Fields = []string{f.Field}
	return f.Fields
}

func (f *StringArrayFilter) GetSymbol() string {
	if f.Symbol == "" {
		return "IN"
	}
	return strings.ToUpper(f.Symbol)
}

func (f *StringArrayFilter) GetSep() string {
	if f.Sep == "" {
		f.Sep = ","
	}
	return f.Sep
}

func (f *StringArrayFilter) IsRequired() bool {
	return f.Required
}

func (f *StringArrayFilter) GetValue() interface{} {
	if f.Value != nil {
		return f.Value
	}
	return f.ParseValue
}

func (f *StringArrayFilter) SetValue(value interface{}) {
	f.ParseValue, _ = value.([]string)
	f.parsed = true
}

func (f *StringArrayFilter) Parse(c IQuery) error {
	if f.ParseFunc != nil {
		return f.ParseFunc(f, c)
	}
	value, err := parseValue(f, c)
	if err != nil || value == "" {
		return err
	}
	value = sqlAntiInject(value)
	f.SetValue(strings.Split(value, f.GetSep()))
	return nil
}

func (f *StringArrayFilter) Bind(db *gorm.DB) *gorm.DB {
	if !f.parsed {
		return db
	}
	if f.BindFunc != nil {
		return f.BindFunc(f, db)
	}
	fields := f.GetFields()
	if len(fields) == 1 {
		return db.Where(fields[0]+" "+f.GetSymbol()+" ?", f.GetValue())
	}
	query := make([]string, len(fields))
	values := make([]interface{}, len(fields))
	for i, field := range f.GetFields() {
		query[i] = field + " " + f.GetSymbol() + " ?"
		values[i] = f.GetValue()
	}
	return db.Where(strings.Join(query, " OR "), values...)
}

// IntFilter
type IntFilter struct {
	Key        string      // 前端传入参数名
	Field      string      // 数据库单列
	Fields     []string    // 数据库多列
	Symbol     string      // 查询条件
	ParseValue int         // 解析值
	Value      interface{} // 用户指定的过滤值
	parsed     bool        // 是否解析过
	Required   bool        // 是否必选
	ParseFunc  ParseFunc   // 自定义转换函数
	BindFunc   BindFunc    // 自定义绑定到查询条件
}

func (f *IntFilter) GetKey() string {
	return f.Key
}

func (f *IntFilter) GetFields() []string {
	if len(f.Fields) > 0 {
		return f.Fields
	}
	if f.Field == "" {
		f.Field = f.Key
	}
	f.Fields = []string{f.Field}
	return f.Fields
}

func (f *IntFilter) GetSymbol() string {
	if f.Symbol == "" {
		return "="
	}
	return strings.ToUpper(f.Symbol)
}

func (f *IntFilter) IsRequired() bool {
	return f.Required
}

func (f *IntFilter) GetValue() interface{} {
	if f.Value != nil {
		return f.Value
	}
	return f.ParseValue
}

func (f *IntFilter) SetValue(value interface{}) {
	f.ParseValue, _ = value.(int)
	f.parsed = true
}

func (f *IntFilter) Parse(c IQuery) error {
	if f.ParseFunc != nil {
		return f.ParseFunc(f, c)
	}
	value, err := parseValue(f, c)
	if err != nil || value == "" {
		return err
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	f.SetValue(intValue)
	return nil
}

func (f *IntFilter) Bind(db *gorm.DB) *gorm.DB {
	if !f.parsed {
		return db
	}
	if f.BindFunc != nil {
		return f.BindFunc(f, db)
	}
	fields := f.GetFields()
	if len(fields) == 1 {
		return db.Where(fields[0]+" "+f.GetSymbol()+" ?", f.GetValue())
	}
	query := make([]string, len(fields))
	values := make([]interface{}, len(fields))
	for i, field := range f.GetFields() {
		query[i] = field + " " + f.GetSymbol() + " ?"
		values[i] = f.GetValue()
	}
	return db.Where(strings.Join(query, " OR "), values...)
}

// IntArrayFilter
type IntArrayFilter struct {
	Key        string      // 前端传入参数名
	Field      string      // 数据库单列
	Fields     []string    // 数据库多列
	Symbol     string      // 查询条件
	ParseValue []int       // 解析值
	Sep        string      // 分隔符
	Value      interface{} // 用户指定的过滤值
	parsed     bool        // 是否解析过
	Required   bool        // 是否必选
	ParseFunc  ParseFunc   // 自定义转换函数
	BindFunc   BindFunc    // 自定义绑定到查询条件
}

func (f *IntArrayFilter) GetKey() string {
	return f.Key
}

func (f *IntArrayFilter) GetFields() []string {
	if len(f.Fields) > 0 {
		return f.Fields
	}
	if f.Field == "" {
		f.Field = f.Key
	}
	f.Fields = []string{f.Field}
	return f.Fields
}

func (f *IntArrayFilter) GetSymbol() string {
	if f.Symbol == "" {
		return "IN"
	}
	return strings.ToUpper(f.Symbol)
}

func (f *IntArrayFilter) GetSep() string {
	if f.Sep == "" {
		f.Sep = ","
	}
	return f.Sep
}

func (f *IntArrayFilter) IsRequired() bool {
	return f.Required
}

func (f *IntArrayFilter) GetValue() interface{} {
	if f.Value != nil {
		return f.Value
	}
	return f.ParseValue
}

func (f *IntArrayFilter) SetValue(value interface{}) {
	switch v := value.(type) {
	case int:
		f.ParseValue = append(f.ParseValue, v)
	case []int:
		f.ParseValue = v
	}
	f.parsed = true
}

func (f *IntArrayFilter) Parse(c IQuery) error {
	if f.ParseFunc != nil {
		return f.ParseFunc(f, c)
	}
	value, err := parseValue(f, c)
	if err != nil || value == "" {
		return err
	}
	for _, val := range strings.Split(value, f.GetSep()) {
		intValue, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		f.SetValue(intValue)
	}
	return nil
}

func (f *IntArrayFilter) Bind(db *gorm.DB) *gorm.DB {
	if !f.parsed {
		return db
	}
	if f.BindFunc != nil {
		return f.BindFunc(f, db)
	}
	fields := f.GetFields()
	if len(fields) == 1 {
		return db.Where(fields[0]+" "+f.GetSymbol()+" ?", f.GetValue())
	}
	query := make([]string, len(fields))
	values := make([]interface{}, len(fields))
	for i, field := range f.GetFields() {
		query[i] = field + " " + f.GetSymbol() + " ?"
		values[i] = f.GetValue()
	}
	return db.Where(strings.Join(query, " OR "), values...)
}

// BoolFilter
type BoolFilter struct {
	Key        string      // 前端传入参数名
	Field      string      // 数据库单列
	Fields     []string    // 数据库多列
	Symbol     string      // 查询条件
	ParseValue bool        // 解析值
	Value      interface{} // 用户指定的过滤值
	parsed     bool        // 是否解析过
	Required   bool        // 是否必选
	ParseFunc  ParseFunc   // 自定义转换函数
	BindFunc   BindFunc    // 自定义绑定到查询条件
}

func (f *BoolFilter) GetKey() string {
	return f.Key
}

func (f *BoolFilter) GetFields() []string {
	if len(f.Fields) > 0 {
		return f.Fields
	}
	if f.Field == "" {
		f.Field = f.Key
	}
	f.Fields = []string{f.Field}
	return f.Fields
}

func (f *BoolFilter) GetSymbol() string {
	if f.Symbol == "" {
		f.Symbol = "="
	}
	return strings.ToUpper(f.Symbol)
}

func (f *BoolFilter) IsRequired() bool {
	return f.Required
}

func (f *BoolFilter) GetValue() interface{} {
	if f.Value != nil {
		return f.Value
	}
	return f.ParseValue
}

func (f *BoolFilter) SetValue(value interface{}) {
	f.ParseValue, _ = value.(bool)
	f.parsed = true
}

func (f *BoolFilter) Parse(c IQuery) error {
	if f.ParseFunc != nil {
		return f.ParseFunc(f, c)
	}
	value, err := parseValue(f, c)
	if err != nil || value == "" {
		return err
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	f.SetValue(boolValue)
	return nil
}

func (f *BoolFilter) Bind(db *gorm.DB) *gorm.DB {
	if !f.parsed {
		return db
	}
	if f.BindFunc != nil {
		return f.BindFunc(f, db)
	}
	fields := f.GetFields()
	if len(fields) == 1 {
		return db.Where(fields[0]+" "+f.GetSymbol()+" ?", f.GetValue())
	}
	query := make([]string, len(fields))
	values := make([]interface{}, len(fields))
	for i, field := range f.GetFields() {
		query[i] = field + " " + f.GetSymbol() + " ?"
		values[i] = f.GetValue()
	}
	return db.Where(strings.Join(query, " OR "), values...)
}

// TimeFilter
type TimeFilter struct {
	Key        string    // 前端传入参数名
	Field      string    // 数据库单列
	Fields     []string  // 数据库多列
	Symbol     string    // 查询条件
	Layout     string    // 时间格式
	ParseValue time.Time // 解析值
	Value      time.Time // 用户指定的过滤值
	parsed     bool      // 是否解析过
	Required   bool      // 是否必选
	ParseFunc  ParseFunc // 自定义转换函数
	BindFunc   BindFunc  // 自定义绑定到查询条件
}

func (f *TimeFilter) GetKey() string {
	return f.Key
}

func (f *TimeFilter) GetFields() []string {
	if len(f.Fields) > 0 {
		return f.Fields
	}
	if f.Field == "" {
		f.Field = f.Key
	}
	f.Fields = []string{f.Field}
	return f.Fields
}

func (f *TimeFilter) GetSymbol() string {
	if f.Symbol == "" {
		f.Symbol = "="
	}
	return strings.ToUpper(f.Symbol)
}

func (f *TimeFilter) GetLayout() string {
	if f.Layout == "" {
		f.Layout = "2006-01-02 15:04:05"
	}
	return f.Layout
}

func (f *TimeFilter) IsRequired() bool {
	return f.Required
}

func (f *TimeFilter) GetValue() interface{} {
	if !f.Value.IsZero() {
		return f.Value
	}
	return f.ParseValue
}

func (f *TimeFilter) SetValue(value interface{}) {
	f.ParseValue, _ = value.(time.Time)
	f.parsed = true
}

func (f *TimeFilter) Parse(c IQuery) error {
	if f.ParseFunc != nil {
		return f.ParseFunc(f, c)
	}
	value, err := parseValue(f, c)
	if err != nil || value == "" {
		return err
	}
	timeValue, err := time.ParseInLocation(f.GetLayout(), value, time.Local)
	if err != nil {
		return err
	}
	f.SetValue(timeValue)
	return nil
}

func (f *TimeFilter) Bind(db *gorm.DB) *gorm.DB {
	if !f.parsed {
		return db
	}
	if f.BindFunc != nil {
		return f.BindFunc(f, db)
	}
	fields := f.GetFields()
	if len(fields) == 1 {
		return db.Where(fields[0]+" "+f.GetSymbol()+" ?", f.GetValue())
	}
	query := make([]string, len(fields))
	values := make([]interface{}, len(fields))
	for i, field := range f.GetFields() {
		query[i] = field + " " + f.GetSymbol() + " ?"
		values[i] = f.GetValue()
	}
	return db.Where(strings.Join(query, " OR "), values...)
}
