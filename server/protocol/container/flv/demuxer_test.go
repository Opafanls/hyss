package flv

import (
	"github.com/Opafanls/hylan/server/core/av"
	"testing"
)

func TestDemuxer_DemuxH(t *testing.T) {
	t.Logf("start")
	type args struct {
		p *av.Packet
	}
	file := []byte{70, 76, 86, 1, 5, 0, 0, 0, 9, 0, 0, 0, 0, 18, 0, 2, 92, 0, 0, 0, 0, 0, 0, 0, 2, 0, 10, 111, 110, 77, 101, 116, 97, 68, 97, 116, 97, 8, 0, 0, 0, 21, 0, 8, 100, 117, 114, 97, 116, 105, 111, 110, 0, 64, 131, 213, 86, 4, 24, 147, 117, 0, 5, 119, 105, 100, 116, 104, 0, 64, 132, 0, 0, 0, 0, 0, 0, 0, 6, 104, 101, 105, 103, 104, 116, 0, 64, 118, 128, 0, 0, 0, 0, 0, 0, 13, 118, 105, 100, 101, 111, 100, 97, 116, 97, 114, 97, 116, 101, 0, 64, 129, 150, 180, 0, 0, 0, 0, 0, 9}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "1",
			args: args{
				p: &av.Packet{
					Data: file,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Demuxer{}
			if err := d.DemuxH(tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("DemuxH() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				t.Logf("decoce packet: %+v header: %+v", tt.args.p, tt.args.p.Header)
			}
		})
	}
}
