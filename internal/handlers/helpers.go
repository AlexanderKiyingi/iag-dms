package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/store"
	"github.com/alvor-technologies/iag-platform-go/apierr"
)

// bindJSONCoerced reads the raw request body, rewrites string-encoded
// numeric/bool JSON values to match dst's struct field types, then runs the
// normal gin binding (so binding:"required" and other validation tags still
// apply). DMS talks directly to its frontend (no gateway normalization), so
// browser forms that submit numbers as strings (e.g. {"lat":"0.34"}) would
// otherwise 400 with "cannot unmarshal string into Go struct field".
func bindJSONCoerced(c *gin.Context, dst any) error {
	raw, err := c.GetRawData()
	if err != nil {
		return err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(coerceJSONScalars(raw, dst)))
	return c.ShouldBindJSON(dst)
}

// coerceScalarStrings rewrites string values in a decoded JSON object to the
// scalar Go type of the matching struct field (keyed by json tag), recursing
// into nested struct and slice-of-struct fields (line-item arrays). Only
// numeric/bool target fields are coerced; genuine string fields, empty strings,
// and unparseable values are left as-is so real bad input still errors.
func coerceScalarStrings(t reflect.Type, m map[string]any) {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		val, present := m[name]
		if !present {
			continue
		}
		ft := f.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if s, ok := val.(string); ok && s == "" {
			switch ft.Kind() {
			case reflect.Float32, reflect.Float64,
				reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Bool:
				// Blank numeric/bool form value clears the field: JSON null decodes
				// to zero (value) or nil (pointer); "" would fail the unmarshal.
				m[name] = nil
				continue
			}
		}
		switch ft.Kind() {
		case reflect.Struct:
			if nested, ok := val.(map[string]any); ok {
				coerceScalarStrings(ft, nested)
			}
		case reflect.Slice, reflect.Array:
			et := ft.Elem()
			for et.Kind() == reflect.Ptr {
				et = et.Elem()
			}
			if et.Kind() == reflect.Struct {
				if arr, ok := val.([]any); ok {
					for _, el := range arr {
						if nested, ok := el.(map[string]any); ok {
							coerceScalarStrings(et, nested)
						}
					}
				}
			}
		case reflect.Float32, reflect.Float64:
			if s, ok := val.(string); ok && s != "" {
				if v, err := strconv.ParseFloat(s, 64); err == nil {
					m[name] = v
				}
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if s, ok := val.(string); ok && s != "" {
				if v, err := strconv.ParseInt(s, 10, 64); err == nil {
					m[name] = v
				}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if s, ok := val.(string); ok && s != "" {
				if v, err := strconv.ParseUint(s, 10, 64); err == nil {
					m[name] = v
				}
			}
		case reflect.Bool:
			if s, ok := val.(string); ok && s != "" {
				if v, err := strconv.ParseBool(s); err == nil {
					m[name] = v
				}
			}
		}
	}
}

// coerceJSONScalars returns raw JSON with string-encoded numeric/bool values
// rewritten to match dst's struct field types. Handles a single object or an
// array of objects. On any problem returns raw unchanged.
func coerceJSONScalars(raw []byte, dst any) []byte {
	t := reflect.TypeOf(dst)
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil {
		return raw
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		var rows []map[string]any
		if json.Unmarshal(raw, &rows) != nil {
			return raw
		}
		et := t.Elem()
		for _, m := range rows {
			coerceScalarStrings(et, m)
		}
		if out, err := json.Marshal(rows); err == nil {
			return out
		}
	case reflect.Struct:
		var m map[string]any
		if json.Unmarshal(raw, &m) != nil {
			return raw
		}
		coerceScalarStrings(t, m)
		if out, err := json.Marshal(m); err == nil {
			return out
		}
	}
	return raw
}

func listOpts(c *gin.Context) store.ListOpts {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	return store.ListOpts{
		Limit: limit, Offset: offset, Q: c.Query("q"),
		Status: c.Query("status"), Channel: c.Query("channel"),
		DistributorID: c.Query("distributorId"), RepID: c.Query("repId"),
		BeatID: c.Query("beatId"),
	}
}

func paginated(c *gin.Context, items any, total int) {
	opts := listOpts(c)
	meta := gin.H{"total": total, "limit": opts.Limit, "offset": opts.Offset}
	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"data":  items,
		"meta":  meta,
	})
}

func notFound(c *gin.Context) {
	apierr.JSONStatus(c, http.StatusNotFound, "not found")
}

func badRequest(c *gin.Context, msg string) {
	apierr.JSONStatus(c, http.StatusBadRequest, msg)
}
