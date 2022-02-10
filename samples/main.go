package main

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
)

type TagHandler struct {
	HandlerFn func(value reflect.Value, field reflect.StructField) error
}

func (th TagHandler) Handle(v interface{}) error {
	return th.handleValue(reflect.ValueOf(v))
}

func (th TagHandler) handleValue(val reflect.Value) error {
	kind := val.Kind()
	switch kind {
	case reflect.Struct:
		return th.handleStruct(val)
	case reflect.Array, reflect.Slice:
		return th.handleArray(val)
	case reflect.Map:
		return th.handleMap(val)
	case reflect.Ptr:
		return th.handleValue(val.Elem())
	}
	return nil
}

func (th TagHandler) handleStruct(val reflect.Value) error {
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		if err := th.HandlerFn(val.Field(i), typ.Field(i)); err != nil {
			return err
		}
		if err := th.handleValue(val.Field(i)); err != nil {
			return err
		}
	}
	return nil
}

func (th TagHandler) handleArray(val reflect.Value) error {
	for i := 0; i < val.Len(); i++ {
		if err := th.handleValue(val.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

func (th TagHandler) handleMap(val reflect.Value) error {
	for _, key := range val.MapKeys() {
		if err := th.handleValue(val.MapIndex(key)); err != nil {
			return err
		}
	}
	return nil
}

func handleValidateTag(value reflect.Value, field reflect.StructField) error {
	tag, ok := field.Tag.Lookup("validate")
	if !ok {
		return nil
	}
	match, err := regexp.Compile(tag)
	if err != nil {
		return fmt.Errorf("validation regexp syntax error: %v", err)
	}

	str := valueToString(value)
	if !match.MatchString(str) {
		return fmt.Errorf("invalid field (%v::%v) %v != %v", field.Type, field.Name, str, tag)
	}
	return nil
}

func valueToString(value reflect.Value) string {
	return fmt.Sprintf("%v", value.Interface())
}

type Person struct {
	BirthYear int       `json:"birth_year" validate:"^(19|20)\\d\\d$"`
	Name      Name      `json:"name"`
	Email     string    `json:"email" validate:"^[\\w-\\.]+@([\\w-]+\\.)+[\\w-]{2,4}$"`
	Friends   []*Person `json:"friends"`
}

type Name struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type Config struct {
	HttpMaxRetries    int    `conf:"HTTP_MAX_RETRIES"`
	ElasticsearchHost string `conf:"ELASTICSEARCH_HOST"`
}

func handleConfigTag(value reflect.Value, field reflect.StructField) error {
	tag, ok := field.Tag.Lookup("conf")
	if !ok {
		return nil
	}
	envvar, ok := os.LookupEnv(tag)
	if !ok {
		return nil
	}
	return setValue(value, envvar)
}

func setValue(value reflect.Value, envvar string) error {
	switch value.Kind() {
	case reflect.String:
		value.SetString(envvar)
	case reflect.Int:
		n, err := strconv.Atoi(envvar)
		if err != nil {
			return err
		}
		value.SetInt(int64(n))
	}
	return nil
}

//segments := strings.Split(tag, ";")
//if len(segments) != 2 {
//return fmt.Errorf("invalid configuration tag specified: %s", tag)
//}

func main() {
	th := TagHandler{
		HandlerFn: handleValidateTag,
	}

	err := th.Handle(Person{
		Name: Name{
			FirstName: "Lasse Martin",
			LastName:  "Jakobsen",
		},
		BirthYear: 1990,
		Email:     "lasse@tengen.dk",
		Friends: []*Person{
			{
				Name:      Name{FirstName: "Iaf", LastName: "Nofrens"},
				BirthYear: 1992,
				Email:     "l33tboi95@hotmail.com",
			},
		},
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	cfgHandler := TagHandler{
		HandlerFn: handleConfigTag,
	}

	var cfg Config
	err = cfgHandler.Handle(&cfg)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf(`ElasticsearchHost: %s, HttpMaxRetries: %d\n`,
		cfg.ElasticsearchHost, cfg.HttpMaxRetries)
}

//Email string `json:"email" validate:"^[\\w-\\.]+@([\\w-]+\\.)+[\\w-]{2,4}$"`
