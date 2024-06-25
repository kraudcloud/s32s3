package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"gitlab.com/zephyrtronium/valid"
)

type (
	S3Config struct {
		Provider        string `env:"PROVIDER"`
		Region          string `env:"REGION"`
		Endpoint        string `env:"ENDPOINT"`
		AccessKeyID     string `env:"ACCESS_KEY_ID"`
		SecretAccessKey string `env:"SECRET_ACCESS_KEY"`
	}

	Source struct {
		S3Config
		Name string
	}

	Dest struct {
		S3Config
		Name string
	}

	Crypt struct {
		Name      string `env:"NAME"`
		Password  string `env:"PASSWORD"`
		Password2 string `env:"PASSWORD2"`
	}

	BackupConfig struct {
		Dest   Dest   `env:"DEST"`
		Source Source `env:"SOURCE"`
		Crypt  Crypt  `env:"CRYPT"`

		Prefix string `env:"BUCKET_PREFIX"`
	}
)

var supportedProviders = []string{"Minio"}

func (s S3Config) Validate() error {
	return valid.Check(nil, []valid.Condition{
		{Name: "provider", Missing: s.Provider == "", Invalid: !slices.Contains(supportedProviders, s.Provider)},
		{Name: "region", Missing: s.Region == ""},
		{Name: "endpoint", Missing: s.Endpoint == ""},
		{Name: "access key id", Missing: s.AccessKeyID == ""},
		{Name: "secret access key", Missing: s.SecretAccessKey == ""},
	})
}

func (s Crypt) Validate() error {
	return valid.Check(nil, []valid.Condition{
		{Name: "password", Missing: s.Password == "", Invalid: len(s.Password) < 8},
		{Name: "password2", Missing: s.Password2 == "", Invalid: s.Password == s.Password2},
	})
}

func (b BackupConfig) Validate() error {
	var outer error
	if err := b.Source.Validate(); err != nil {
		outer = errors.Join(outer, fmt.Errorf("source: %w", err))
	}

	if err := b.Dest.Validate(); err != nil {
		outer = errors.Join(outer, fmt.Errorf("dest: %w", err))
	}

	if err := b.Crypt.Validate(); err != nil {
		outer = errors.Join(outer, fmt.Errorf("crypt: %w", err))
	}

	return outer
}

func Config() (BackupConfig, error) {
	return ConfigFromEnv(envSliceToMap(os.Environ()))
}

const (
	sourceName    = "source"
	destName      = "dest"
	cryptName     = "crypt"
	prefixDefault = "backup-"
)

func ConfigFromEnv(c map[string]string) (BackupConfig, error) {
	out := BackupConfig{
		Dest: Dest{
			Name: destName,
		},
		Source: Source{
			Name: sourceName,
		},
		Crypt: Crypt{
			Name: cryptName,
		},
		Prefix: prefixDefault,
	}

	err := fromEnvStruct(c, "", &out)
	if err != nil {
		return BackupConfig{}, err
	}

	if err := out.Validate(); err != nil {
		return BackupConfig{}, err
	}

	return out, nil
}

func envSliceToMap(c []string) map[string]string {
	out := make(map[string]string)
	for _, v := range c {
		key, value, ok := strings.Cut(v, "=")
		if !ok {
			continue
		}

		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	return out
}

func fromEnvStruct(c map[string]string, prefix string, out any) error {
	fvs := reflect.ValueOf(out).Elem()
	fields := reflect.TypeOf(out).Elem()
	for _, field := range reflect.VisibleFields(fields) {
		tag := field.Tag.Get("env")

		if prefix != "" {
			if tag != "" {
				tag = prefix + "_" + tag
			} else {
				tag = prefix
			}
		}

		fv := fvs.FieldByName(field.Name)
		if field.Type.Kind() == reflect.Struct {
			err := fromEnvStruct(c, tag, fv.Addr().Interface())
			if err != nil {
				return err
			}

			continue
		}

		if tag == "" {
			continue
		}

		value, ok := c[tag]
		if !ok {
			continue
		}
		unmarshalValue(value, field.Type, fv)
	}

	return nil
}

func unmarshalValue(value string, typ reflect.Type, out reflect.Value) error {
	switch typ.Kind() {
	case reflect.String:
		out.SetString(value)
	case reflect.Int:
		vv, err := strconv.ParseInt(value, 0, 64)
		if err != nil {
			return err
		}
		out.SetInt(vv)
	case reflect.Bool:
		out.SetBool(value == "true")
	case reflect.Float64:
		vv, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		out.SetFloat(vv)
	default:
		return fmt.Errorf("unsupported type: %s", typ.Kind())
	}

	return nil
}
