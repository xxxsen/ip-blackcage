package dbkit

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/didi/gendry/builder"
)

type IQueryer interface {
	QueryContext(ctx context.Context, sql string, args ...interface{}) (*sql.Rows, error)
}

const (
	defaultQueryLength = 50
)

// ColumnMap 返回列名与顺序的映射关系
func ColumnMap(rows *sql.Rows) (map[string]int, error) {
	cmn, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	rs := make(map[string]int)
	for idx, cn := range cmn {
		rs[cn] = idx
	}
	return rs, nil
}

func createScanRowsConfig(opts ...ScanRowOption) *scanRowConfig {
	c := &scanRowConfig{
		tagname: "json",
		limit:   20,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// readSingleInstanceType 从一个slice切片获取到元素的类型
func readSingleInstanceType(ptr interface{}) reflect.Type {
	ref := reflect.TypeOf(ptr)
	if ref.Kind() != reflect.Ptr {
		return reflect.TypeOf(ptr)
	}
	ref = ref.Elem()
	if ref.Kind() != reflect.Slice {
		return reflect.TypeOf(ptr)
	}
	itemType := ref.Elem()
	return itemType

}

func ensurePtrTypeSlice(ptr interface{}) error {
	typ := reflect.TypeOf(ptr)
	if ptr == nil || reflect.ValueOf(ptr).IsNil() || typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("invalid ptr type")
	}
	return nil
}

func ExtractLimitFromQuery(q map[string]interface{}) (int, bool) {
	lm, ok := q["_limit"]
	if !ok {
		return defaultQueryLength, false
	}
	realTyp, ok := lm.([]uint)
	if !ok {
		return defaultQueryLength, false
	}
	sz := len(realTyp)
	if sz > 2 || sz == 0 {
		return defaultQueryLength, false
	}
	return int(realTyp[sz-1]), true
}

// SimpleQuery 执行一次简单的db查询, 并将数据绑定到ptr对象中
// ptr类型必须为*[]*Type, 例如*[]*PostMetaTab(指向slice的指针)
func SimpleQuery(ctx context.Context, client IQueryer, table string,
	q map[string]interface{}, ptr interface{}, opts ...ScanRowOption) error {
	if err := ensurePtrTypeSlice(ptr); err != nil {
		return err
	}
	c := createScanRowsConfig(opts...)
	if limit, ok := ExtractLimitFromQuery(q); ok {
		c.limit = limit
	}
	fields, _, err := extractFieldNamesFromType(readSingleInstanceType(ptr), c.tagname)
	if err != nil {
		return err
	}
	sql, args, err := builder.BuildSelect(table, q, fields)
	if err != nil {
		return fmt.Errorf("build select failed, err:%w", err)
	}
	rows, err := client.QueryContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("do select failed, err:%w", err)
	}
	defer rows.Close()
	return innerScan(rows, ptr, c)
}

func innerScan(rows *sql.Rows, ptr interface{}, c *scanRowConfig) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("scan panic:%v, stack:%s", e, string(debug.Stack()))
		}
	}()
	ptrObj := reflect.ValueOf(ptr)
	if !ptrObj.Elem().CanSet() {
		return fmt.Errorf("not settable object")
	}
	if ptrObj.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("ptr should be *[]*STRUCT type")
	}
	columnMap, err := ColumnMap(rows)
	if err != nil {
		return fmt.Errorf("read columns failed, err:%w", err)
	}
	arr := reflect.MakeSlice(ptrObj.Elem().Type(), 0, c.limit)
	arrItemObj := arr.Type().Elem()
	for rows.Next() {
		inst, fields := newInstance(arrItemObj.Elem(), columnMap, c.tagname)
		if len(fields) < len(columnMap) {
			return fmt.Errorf("column count:%d in db > column count:%d in struct", len(columnMap), len(fields))
		}
		if err := rows.Scan(fields...); err != nil {
			return fmt.Errorf("scan fields fail, err:%w", err)
		}
		arr = reflect.Append(arr, reflect.ValueOf(inst))
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("scan rows err:%w", err)
	}
	ptrObj.Elem().Set(arr)
	return nil
}

// ScanRows 将rows数据映射到ptr对象中, ptr类型必须为*[]*Type, 例如*[]*PostMetaTab(指向slice的指针)
// 对于ptr中的单一元素(*PostMetaTab), 必须保证通过tagname取出来的字段数大于等于rows中select的字段数
func ScanRows(rows *sql.Rows, ptr interface{}, opts ...ScanRowOption) error {
	if err := ensurePtrTypeSlice(ptr); err != nil {
		return err
	}
	c := createScanRowsConfig(opts...)
	return innerScan(rows, ptr, c)
}

// newInstance 创建新的实例, 并将指定的columns初始化, 之后返回初始化后的column
func newInstance(ptr reflect.Type, columns map[string]int, tagname string) (interface{}, []interface{}) {
	refType := ptr
	refVal := reflect.New(refType)
	fields := make([]interface{}, refType.NumField())
	setLoc := 0
	for i := 0; i < refType.NumField(); i++ {
		tag, ok := readFieldTag(refType.Field(i), tagname)
		if !ok {
			continue
		}
		loc, ok := columns[tag]
		if !ok {
			continue
		}
		f := refVal.Elem().Field(i)
		kind := f.Kind()
		if kind == reflect.Ptr {
			//
			fieldType := refType.Field(i).Type.Elem()
			//
			f.Set(reflect.New(fieldType))
			f = f.Elem()
		}
		fields[loc] = f.Addr().Interface()
		setLoc++
	}
	return refVal.Interface(), fields[:setLoc]
}

func extractFieldNamesFromType(typ reflect.Type, tagname string) ([]string, []int, error) {
	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("ptr should STRUCT PTR type")
	}
	typ = typ.Elem()
	fields := make([]string, 0, typ.NumField())
	idx := make([]int, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tagvalue, ok := readFieldTag(field, tagname)
		if !ok {
			continue
		}
		fields = append(fields, tagvalue)
		idx = append(idx, i)
	}
	return fields, idx, nil
}

func ExtractFieldNames(ptr interface{}, tagname string) ([]string, []int, error) {
	typ := reflect.TypeOf(ptr)
	return extractFieldNamesFromType(typ, tagname)
}

func readFieldTag(field reflect.StructField, tagname string) (string, bool) {
	tagvalue, ok := field.Tag.Lookup(tagname)
	if !ok {
		return "", false
	}
	if idx := strings.Index(tagvalue, ","); idx > 0 {
		tagvalue = tagvalue[:idx]
	}
	if len(tagvalue) == 0 || tagvalue == "-" {
		return "", false
	}
	return tagvalue, true
}
