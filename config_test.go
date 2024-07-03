package main

import "testing"

func TestConfigFromEnv(t *testing.T) {
	c := map[string]string{
		"SOURCE_PROVIDER":          "Minio",
		"SOURCE_REGION":            "us-east-1",
		"SOURCE_ENDPOINT":          "http://localhost:9000",
		"SOURCE_ACCESS_KEY_ID":     "test",
		"SOURCE_SECRET_ACCESS_KEY": "test",
		"DEST_PROVIDER":            "Minio",
		"DEST_REGION":              "us-east-1",
		"DEST_ENDPOINT":            "http://localhost:9000",
		"DEST_ACCESS_KEY_ID":       "test",
		"DEST_SECRET_ACCESS_KEY":   "test",
		"CRYPT_PASSWORD":           "test45367824",
		"CRYPT_PASSWORD2":          "test2435143632",
	}

	_, err := ConfigFromEnv(c)
	if err != nil {
		t.Error(err)
	}
}
