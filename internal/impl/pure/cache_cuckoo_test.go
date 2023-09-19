package pure

import (
	"context"
	"fmt"
	"testing"

	"github.com/benthosdev/benthos/v4/public/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCuckooCacheStandard(t *testing.T) {
	t.Parallel()

	defConf, err := cuckooCacheConfig().ParseYAML(``, nil)
	require.NoError(t, err)

	logger := service.MockResources().Logger()

	c, err := cuckooMemCacheFromConfig(defConf, logger)
	require.NoError(t, err)

	ctx := context.Background()
	key := "foo"

	_, err = c.Get(ctx, key)
	assert.EqualError(t, err, "key does not exist")

	err = c.Add(ctx, key, nil, nil)
	assert.NoError(t, err)

	err = c.Add(ctx, key, nil, nil)
	assert.EqualError(t, err, "key already exists")

	err = c.Set(ctx, key, nil, nil)
	assert.NoError(t, err)

	value, err := c.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, string(value), "t")
}

func TestCuckooCacheScalable(t *testing.T) {
	t.Parallel()

	defConf, err := cuckooCacheConfig().ParseYAML(`scalable: true`, nil)
	require.NoError(t, err)

	logger := service.MockResources().Logger()

	c, err := cuckooMemCacheFromConfig(defConf, logger)
	require.NoError(t, err)

	ctx := context.Background()
	key := "foo"

	_, err = c.Get(ctx, key)
	assert.EqualError(t, err, "key does not exist")

	err = c.Add(ctx, key, nil, nil)
	assert.NoError(t, err)

	err = c.Add(ctx, key, nil, nil)
	assert.EqualError(t, err, "key already exists")

	err = c.Set(ctx, key, nil, nil)
	assert.NoError(t, err)

	value, err := c.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, string(value), "t")
}

func TestCuckooCacheDelete(t *testing.T) {
	t.Parallel()

	defConf, err := cuckooCacheConfig().ParseYAML(``, nil)
	require.NoError(t, err)

	logger := service.MockResources().Logger()

	c, err := cuckooMemCacheFromConfig(defConf, logger)
	require.NoError(t, err)

	ctx := context.Background()
	key := "foo"

	err = c.Set(ctx, key, nil, nil)
	assert.NoError(t, err)

	err = c.Delete(ctx, key)
	assert.NoError(t, err)

	_, err = c.Get(ctx, key)
	assert.EqualError(t, err, "key does not exist")
}

func TestCuckooCacheInitValues(t *testing.T) {
	t.Parallel()

	defConf, err := cuckooCacheConfig().ParseYAML(`
cap: 1024
init_values:
  foo: t
  bar: t
`, nil)
	require.NoError(t, err)

	logger := service.MockResources().Logger()

	c, err := cuckooMemCacheFromConfig(defConf, logger)
	require.NoError(t, err)

	ctx := context.Background()

	exp := "t"
	if act, err := c.Get(ctx, "foo"); err != nil {
		t.Error(err)
	} else if string(act) != exp {
		t.Errorf("Wrong result: %v != %v", string(act), exp)
	}

	if act, err := c.Get(ctx, "bar"); err != nil {
		t.Error(err)
	} else if string(act) != exp {
		t.Errorf("Wrong result: %v != %v", string(act), exp)
	}
}

func TestCuckooCacheBatchedSet(t *testing.T) {
	t.Parallel()

	defConf, err := cuckooCacheConfig().ParseYAML(``, nil)
	require.NoError(t, err)

	logger := service.MockResources().Logger()

	var c service.Cache
	c, err = cuckooMemCacheFromConfig(defConf, logger)
	require.NoError(t, err)

	batchedSet, ok := c.(batchedCache)
	require.True(t, ok)

	items := []service.CacheItem{
		{Key: "foo"},
		{Key: "bar"},
	}

	err = batchedSet.SetMulti(context.Background(), items...)
	require.NoError(t, err)

	{
		v, err := c.Get(context.Background(), "foo")
		require.NoError(t, err)
		assert.EqualValues(t, "t", v)
	}
	{
		v, err := c.Get(context.Background(), "bar")
		require.NoError(t, err)
		assert.EqualValues(t, "t", v)
	}
	{
		v, err := c.Get(context.Background(), "baz")
		assert.EqualError(t, err, "key does not exist")
		assert.Empty(t, v)
	}
}

func BenchmarkCuckoo(b *testing.B) {
	defConf, err := cuckooCacheConfig().ParseYAML(``, nil)
	require.NoError(b, err)

	logger := service.MockResources().Logger()

	c, err := cuckooMemCacheFromConfig(defConf, logger)
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		j := i % cuckooCacheFieldCapDefaultValue
		key, value := fmt.Sprintf("key%v", j), []byte("t")

		err = c.Set(ctx, key, value, nil)
	}

	_ = err
}

func BenchmarkCuckooParallel(b *testing.B) {
	defConf, err := cuckooCacheConfig().ParseYAML(``, nil)
	require.NoError(b, err)

	logger := service.MockResources().Logger()

	c, err := cuckooMemCacheFromConfig(defConf, logger)
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()

	b.RunParallel(func(p *testing.PB) {
		var err error

		for i := 0; p.Next(); i++ {
			j := i % cuckooCacheFieldCapDefaultValue
			key, value := fmt.Sprintf("key%v", j), []byte("t")

			err = c.Set(ctx, key, value, nil)
		}

		_ = err
	})
}
