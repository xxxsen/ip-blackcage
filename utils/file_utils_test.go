package utils

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestReadFile(t *testing.T) {
	ips := "1.2.3.4\n2.3.4.5\r\n4.5.6.7\r\n\n\n\n1.1.1.1/24\r\r\r\n"
	f := "/tmp/test_file_" + uuid.NewString()
	err := os.WriteFile(f, []byte(ips), 0644)
	assert.NoError(t, err)
	readIps, err := ReadIPListFromFile(f)
	assert.NoError(t, err)
	for _, ip := range readIps {
		t.Logf("ips:%s", ip)
	}
}
