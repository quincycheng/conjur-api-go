package conjurapi

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TempFileForTesting(prefix string, fileContents string, t *testing.T) (string, error) {
	tmpfile, err := ioutil.TempFile(t.TempDir(), prefix)
	if err != nil {
		return "", err
	}

	if _, err := tmpfile.Write([]byte(fileContents)); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	return tmpfile.Name(), err
}

func TestConfig_IsValid(t *testing.T) {
	t.Run("Return without error for valid configuration", func(t *testing.T) {
		config := Config{
			Account:      "account",
			ApplianceURL: "appliance-url",
		}

		err := config.validate()
		assert.NoError(t, err)
	})

	t.Run("Return error for invalid configuration", func(t *testing.T) {
		config := Config{
			Account: "account",
		}

		err := config.validate()
		assert.Error(t, err)

		errString := err.Error()
		assert.Contains(t, errString, "Must specify an ApplianceURL")
	})
}

func TestConfig_IsHttps(t *testing.T) {
	t.Run("Return true for configuration with SSLCert", func(t *testing.T) {
		config := Config{
			SSLCert: "cert",
		}

		isHttps := config.IsHttps()
		assert.True(t, isHttps)
	})

	t.Run("Return true for configuration with SSLCertPath", func(t *testing.T) {
		config := Config{
			SSLCertPath: "path/to/cert",
		}

		isHttps := config.IsHttps()
		assert.True(t, isHttps)
	})

	t.Run("Return false for configuration without SSLCert or SSLCertPath", func(t *testing.T) {
		config := Config{}

		isHttps := config.IsHttps()
		assert.False(t, isHttps)
	})

}

func TestConfig_LoadFromEnv(t *testing.T) {
	t.Run("Given configuration and authentication credentials in env", func(t *testing.T) {
		e := ClearEnv()
		defer e.RestoreEnv()

		os.Setenv("CONJUR_ACCOUNT", "account")
		os.Setenv("CONJUR_APPLIANCE_URL", "appliance-url")

		t.Run("Returns Config loaded with values from env", func(t *testing.T) {
			config := &Config{}
			config.mergeEnv()

			assert.EqualValues(t, *config, Config{
				Account:      "account",
				ApplianceURL: "appliance-url",
			})
		})
	})
}

var versiontests = []struct {
	in    string
	label string
	out   bool
}{
	{"version: 4", "version 4", true},
	{"version: 5", "version 5", false},
	{"", "empty version", false},
}

func TestConfig_mergeYAML(t *testing.T) {
	t.Run("No other netrc specified", func(t *testing.T) {
		home := os.Getenv("HOME")
		assert.NotEmpty(t, home)

		e := ClearEnv()
		defer e.RestoreEnv()

		os.Setenv("HOME", home)
		os.Setenv("CONJUR_ACCOUNT", "account")
		os.Setenv("CONJUR_APPLIANCE_URL", "appliance-url")

		t.Run("Uses $HOME/.netrc by deafult", func(t *testing.T) {
			config, err := LoadConfig()
			assert.NoError(t, err)

			assert.EqualValues(t, config, Config{
				Account:      "account",
				ApplianceURL: "appliance-url",
				NetRCPath:    path.Join(home, ".netrc"),
			})
		})
	})

	for index, versiontest := range versiontests {
		t.Run(fmt.Sprintf("Given a filled conjurrc file with %s", versiontest.label), func(t *testing.T) {
			conjurrcFileContents := fmt.Sprintf(`
---
appliance_url: http://path/to/appliance%v
account: some account%v
cert_file: "/path/to/cert/file/pem%v"
netrc_path: "/path/to/netrc/file%v"
%s
`, index, index, index, index, versiontest.in)

			tmpFileName, err := TempFileForTesting("TestConfigVersion", conjurrcFileContents, t)
			defer os.Remove(tmpFileName) // clean up
			assert.NoError(t, err)

			t.Run(fmt.Sprintf("Returns Config loaded with values from file and V4: %t", versiontest.out), func(t *testing.T) {
				config := &Config{}
				config.mergeYAML(tmpFileName)

				assert.EqualValues(t, *config, Config{
					Account:      fmt.Sprintf("some account%v", index),
					ApplianceURL: fmt.Sprintf("http://path/to/appliance%v", index),
					NetRCPath:    fmt.Sprintf("/path/to/netrc/file%v", index),
					SSLCertPath:  fmt.Sprintf("/path/to/cert/file/pem%v", index),
					V4:           versiontest.out,
				})
			})
		})
	}

	t.Run("Throws errors when conjurrc is present but unparsable", func(t *testing.T) {
		badConjurrc := `
---
appliance_url: http://path/to/appliance
account: some account
cert_file: "C:\badly\escaped\path"
`

		tmpFileName, err := TempFileForTesting("TestConfigParsingErroHandling", badConjurrc, t)
		defer os.Remove(tmpFileName) // clean up
		assert.NoError(t, err)

		config := &Config{}
		err = config.mergeYAML(tmpFileName)
		assert.Error(t, err)
	})
}
