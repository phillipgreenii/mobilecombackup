package mobilecombackup

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseFlagsCorrect(t *testing.T) {
	var tests = []struct {
		args []string
		conf config
	}{
		{[]string{},
			config{repoPath: ".", pathsToProcess: []string{}}},

		{[]string{"-repo", "r/path", "myPath1", "myPath2"},
			config{repoPath: "r/path", pathsToProcess: []string{"myPath1", "myPath2"}}},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			conf, output, err := parseFlags("prog", tt.args)
			if err != nil {
				t.Errorf("err got %v, want nil", err)
			}
			if output != "" {
				t.Errorf("output got %q, want empty", output)
			}
			if !reflect.DeepEqual(*conf, tt.conf) {
				t.Errorf("conf got %+v, want %+v", *conf, tt.conf)
			}
		})
	}
}

func TestParseFlagsError(t *testing.T) {
	var tests = []struct {
		args   []string
		errstr string
	}{
		{[]string{"-repo"}, "flag needs an argument: -repo"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			conf, output, err := parseFlags("prog", tt.args)
			if conf != nil {
				t.Errorf("conf got %v, want nil", conf)
			}
			if !strings.Contains(err.Error(), tt.errstr) {
				t.Errorf("err got %q, want to find %q", err.Error(), tt.errstr)
			}
			if !strings.Contains(output, "Usage of prog") {
				t.Errorf("output got %q", output)
			}
		})
	}
}

func TestValidateConfigCorrect(t *testing.T) {
	var tests = []struct {
		desc string
		conf config
	}{
		{"specified repo path and single pathsToProcess",
			config{repoPath: "other/path", pathsToProcess: []string{"myPath"}}},
		{"default repo path and multiple pathsToProcess",
			config{repoPath: ".", pathsToProcess: []string{"myPath1", "myPath2"}}},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateConfig(&tt.conf)
			if err != nil {
				t.Errorf("err got %v, want nil", err)
			}
		})
	}
}

func TestValidateConfigError(t *testing.T) {
	var tests = []struct {
		desc   string
		conf   config
		errstr string
	}{
		{"specified repo path and no pathsToProcess",
			config{repoPath: "other/path", pathsToProcess: []string{}},
			"Atleast one path to process must be specified"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateConfig(&tt.conf)
			if !strings.Contains(err.Error(), tt.errstr) {
				t.Errorf("err got %q, want to find %q", err.Error(), tt.errstr)
			}
		})
	}
}
