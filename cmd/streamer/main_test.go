package main

import "testing"

func TestResolveShardConfigDefaultsToSingleShard(t *testing.T) {
	t.Setenv("STREAMER_SHARD_COUNT", "")
	t.Setenv("STREAMER_SHARD_INDEX", "")
	t.Setenv("POD_NAME", "")

	index, count, err := resolveShardConfig()
	if err != nil {
		t.Fatalf("resolveShardConfig() error = %v", err)
	}
	if index != 0 || count != 1 {
		t.Fatalf("resolveShardConfig() = (%d, %d), want (0, 1)", index, count)
	}
}

func TestResolveShardConfigUsesPodOrdinal(t *testing.T) {
	t.Setenv("STREAMER_SHARD_COUNT", "3")
	t.Setenv("STREAMER_SHARD_INDEX", "")
	t.Setenv("POD_NAME", "streamer-streamer-2")

	index, count, err := resolveShardConfig()
	if err != nil {
		t.Fatalf("resolveShardConfig() error = %v", err)
	}
	if index != 2 || count != 3 {
		t.Fatalf("resolveShardConfig() = (%d, %d), want (2, 3)", index, count)
	}
}

func TestResolveShardConfigRejectsInvalidExplicitIndex(t *testing.T) {
	t.Setenv("STREAMER_SHARD_COUNT", "2")
	t.Setenv("STREAMER_SHARD_INDEX", "2")
	t.Setenv("POD_NAME", "streamer-streamer-0")

	_, _, err := resolveShardConfig()
	if err == nil || err.Error() != "STREAMER_SHARD_INDEX must be between 0 and STREAMER_SHARD_COUNT - 1" {
		t.Fatalf("expected invalid shard index error, got %v", err)
	}
}

func TestParseStatefulSetOrdinal(t *testing.T) {
	tests := []struct {
		name    string
		podName string
		want    int
		wantErr string
	}{
		{name: "valid", podName: "streamer-streamer-7", want: 7},
		{name: "missing suffix", podName: "streamer-streamer", wantErr: `POD_NAME "streamer-streamer" does not contain a valid StatefulSet ordinal`},
		{name: "missing hyphen", podName: "streamer", wantErr: `POD_NAME "streamer" does not contain a StatefulSet ordinal`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseStatefulSetOrdinal(test.podName)
			if test.wantErr != "" {
				if err == nil || err.Error() != test.wantErr {
					t.Fatalf("expected error %q, got %v", test.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseStatefulSetOrdinal() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("parseStatefulSetOrdinal() = %d, want %d", got, test.want)
			}
		})
	}
}
