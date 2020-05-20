package diff

import (
	"reflect"
	"testing"
)

func TestBytes(t *testing.T) {
	type args struct {
		oldbs []byte
		newbs []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Bytes(tt.args.oldbs, tt.args.newbs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Bytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkBytes(b *testing.B) {
	oldfile := []byte("ABCDEF")
	newfile := []byte("ABCDEFG")
	for i := 0; i < b.N; i++ {
		Bytes(oldfile, newfile)
	}
}