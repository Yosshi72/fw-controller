package fwconfig

import (
	"reflect"
	"testing"
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
		want2   []string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, err := ConfigReader(tt.args.configFile)
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
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("ConfigReader() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func TestConfigWriter(t *testing.T) {
	type args struct {
		containername string
		configFile    string
		newUntrustIf  string
		newTrustIf    []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ConfigWriter(tt.args.containername, tt.args.configFile, tt.args.newUntrustIf, tt.args.newTrustIf); (err != nil) != tt.wantErr {
				t.Errorf("ConfigWriter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_updateZone(t *testing.T) {
	type args struct {
		zoneMap     map[string]interface{}
		trustZone   []string
		untrustZone string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := updateZone(tt.args.zoneMap, tt.args.trustZone, tt.args.untrustZone); (err != nil) != tt.wantErr {
				t.Errorf("updateZone() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMatchElements(t *testing.T) {
	type args struct {
		slice1 []string
		slice2 []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"case1: Unordered match",
			args{[]string{"eth-a","eth-b","eth-c"},[]string{"eth-c","eth-a","eth-b"}},
			true,
		},
		{
			"case2: Different number of elements",
			args{[]string{"eth-a","eth-b","eth-c"},[]string{"eth-a","eth-b"}},
			false,
		},
		{
			"case3: Elements have different values",
			args{[]string{"eth-a","eth-b","eth-c"},[]string{"eth-c","eth-A","eth-b"}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchElements(tt.args.slice1, tt.args.slice2); got != tt.want {
				t.Errorf("MatchElements() = %v, want %v", got, tt.want)
			}
		})
	}
}