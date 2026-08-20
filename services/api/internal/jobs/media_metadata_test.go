package jobs

import "testing"

func TestParseProbe(t *testing.T) {
	probe, err := parseProbe([]byte(`{"streams":[{"codec_type":"video","codec_name":"h264","width":1080,"height":1920}],"format":{"duration":"12.345","bit_rate":"2400000"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if probe.Width == nil || *probe.Width != 1080 || probe.Height == nil || *probe.Height != 1920 || probe.DurationMS == nil || *probe.DurationMS != 12345 || probe.BitrateBPS == nil || *probe.BitrateBPS != 2400000 || probe.Codec != "h264" {
		t.Fatalf("unexpected probe: %+v", probe)
	}
}
