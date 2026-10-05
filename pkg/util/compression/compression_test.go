package compression

import (
	"flag"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/topfreegames/pitaya/v3/pkg/helpers"
)

var update = flag.Bool("update", false, "update .golden files")

var ins = []struct {
	name string
	data string
}{
	{"compression_deflate_test_1", "test"},
	{"compression_deflate_test_2", "{a:1,b:2}"},
	{"compression_deflate_test_3", "Neque porro quisquam est qui dolorem ipsum quia dolor sit amet, consectetur, adipisci velit"},
}

func TestCompressionDeflate(t *testing.T) {
	for _, in := range ins {
		t.Run(in.name, func(t *testing.T) {
			b, err := DeflateData([]byte(in.data))
			require.NoError(t, err)
			gp := filepath.Join("fixtures", in.name+".golden")
			if *update {
				t.Log("updating golden file")
				helpers.WriteFile(t, gp, b)
			}
			expected := helpers.ReadFile(t, gp)

			assert.Equal(t, expected, b)
		})
	}
}

func TestCompressionInflate(t *testing.T) {
	for _, in := range ins {
		t.Run(in.name, func(t *testing.T) {
			inputFile := filepath.Join("fixtures", in.name+".golden")
			input := helpers.ReadFile(t, inputFile)

			result, err := InflateData(input)
			require.NoError(t, err)

			assert.Equal(t, string(result), in.data)
		})
	}
}

func TestCompressionInflateIncorrectData(t *testing.T) {
	t.Run("compression_deflate_incorrect_data", func(t *testing.T) {
		input := "arbitrary data"

		result, err := InflateData([]byte(input))
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCompressionDeflateDoesNotAllocateWriterPerCall(t *testing.T) {
	data := []byte(ins[2].data)
	_, err := DeflateData(data)
	require.NoError(t, err)

	const runs = 100
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < runs; i++ {
		_, err = DeflateData(data)
		require.NoError(t, err)
	}
	runtime.ReadMemStats(&after)

	assert.Less(t, (after.TotalAlloc-before.TotalAlloc)/runs, uint64(400*1024))
}

func TestCompressionDeflateConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := []byte(fmt.Sprintf("%s %d", ins[2].data, i))
			b, err := DeflateData(data)
			assert.NoError(t, err)
			result, err := InflateData(b)
			assert.NoError(t, err)
			assert.Equal(t, data, result)
		}(i)
	}
	wg.Wait()
}
