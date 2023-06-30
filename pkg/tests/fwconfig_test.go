package test

import (
	"reflect"
	"testing"
	"github.com/Yosshi72/fw-controller/pkg/fwconfig"
)

func TestConfigReader(t *testing.T) {
	type args struct {
		configFile string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		want1   string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := fwconfig.ConfigReader(tt.args.configFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConfigReader() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ConfigReader() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
