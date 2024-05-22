package main

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/rclone/rclone/backend/crypt"
	"github.com/rclone/rclone/backend/s3"
	"github.com/rclone/rclone/fs/config/obscure"
)

type (
	Wrapped[T any] struct {
		Name  string
		Type  string
		Value T
	}

	BackupConfig struct {
		Dest   Wrapped[s3.Options]    `config:"DEST"`
		Source Wrapped[s3.Options]    `config:"SOURCE"`
		Crypt  Wrapped[crypt.Options] `config:"CRYPT"`

		BackupBucket   string `config:"BACKUP_BUCKET"`
		ExpirationDays int    `config:"EXPIRATION_DAYS"`
	}
)

func (n Wrapped[T]) EncodeIni(w io.Writer) error {
	v := reflect.ValueOf(n.Value)
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("invalid type: %s", v.Kind())
	}

	// https://rclone.org/crypt/#configuration
	// https://rclone.org/s3/#configuration
	fmt.Fprintf(w, "[%s]\n", n.Name)
	fmt.Fprintf(w, "  type = %s\n", n.Type)
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		t := v.Type().Field(i)
		if f.IsZero() {
			continue
		}

		fmt.Fprintf(w, "  %s = %v\n", t.Tag.Get("config"), f.Interface())
	}

	fmt.Fprintln(w)
	return nil
}

func Config() (BackupConfig, error) {
	return ConfigFromEnv(envSliceToMap(os.Environ()))
}

const (
	sourceName = "source"
	destName   = "dest"
	cryptName  = "crypt"
	bucketName = "backups"
)

func (c BackupConfig) Validate() error {
	if c.Source.Value.Endpoint == "" {
		return fmt.Errorf("source endpoint is required")
	}

	if c.Dest.Value.Endpoint == "" {
		return fmt.Errorf("dest endpoint is required")
	}

	if c.Crypt.Value.Password == "" {
		return fmt.Errorf("crypt password is required")
	}

	if c.Crypt.Value.Password2 == "" {
		return fmt.Errorf("crypt password2 is required")
	}

	return nil
}

func ConfigFromEnv(c map[string]string) (BackupConfig, error) {
	out := BackupConfig{
		Dest: Wrapped[s3.Options]{
			Name: destName,
			Type: "s3",
		},
		Source: Wrapped[s3.Options]{
			Name: sourceName,
			Type: "s3",
		},
		Crypt: Wrapped[crypt.Options]{
			Name: cryptName,
			Type: "crypt",
			Value: crypt.Options{
				Remote:                  fmt.Sprintf("%s:%s", destName, bucketName),
				FilenameEncryption:      "standard",
				FilenameEncoding:        "base32",
				DirectoryNameEncryption: true,
				Suffix:                  ".bin",
			},
		},
		BackupBucket: bucketName,
	}

	err := fromEnvStruct(c, "", &out)
	if err != nil {
		return BackupConfig{}, err
	}

	pas1, err := obscure.Obscure(out.Crypt.Value.Password)
	if err != nil {
		return BackupConfig{}, fmt.Errorf("obscure password: %w", err)
	}

	pas2, err := obscure.Obscure(out.Crypt.Value.Password2)
	if err != nil {
		return BackupConfig{}, fmt.Errorf("obscure password2: %w", err)
	}

	out.Crypt.Value.Password = pas1
	out.Crypt.Value.Password2 = pas2
	out.Crypt.Value.Remote = fmt.Sprintf("%s:%s", destName, out.BackupBucket)

	if err := out.Validate(); err != nil {
		return BackupConfig{}, err
	}

	if os.Getenv("DEBUG_CONFIG") != "" {
		EncodeConfig(os.Stderr, out)
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
		tag, tagExists := field.Tag.Lookup("config")
		if prefix != "" {
			if tagExists {
				tag = prefix + "_" + tag
			} else {
				tag = prefix
			}
		}

		tag = strings.ToUpper(tag)
		fv := fvs.FieldByName(field.Name)
		if field.Type.Kind() == reflect.Struct {
			err := fromEnvStruct(c, tag, fv.Addr().Interface())
			if err != nil {
				return err
			}

			continue
		}

		if !tagExists {
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
