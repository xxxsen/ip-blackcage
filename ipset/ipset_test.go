package ipset

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPSet(t *testing.T) {
	set, err := New()
	assert.NoError(t, err)
	ctx := context.Background()
	v1, v2, err := set.Version(ctx)
	assert.NoError(t, err)
	t.Logf("v1:%s, v2:%s", v1, v2)
}

func TestAddAndTest(t *testing.T) {
	set, err := New()
	assert.NoError(t, err)
	ctx := context.Background()
	setname := "test_set"
	{
		err = set.Create(ctx, setname, SetTypeHashNet, WithExist())
		assert.NoError(t, err)
	}
	{
		err = set.Add(ctx, setname, "1.2.3.4")
		assert.NoError(t, err)
		err = set.Add(ctx, setname, "2.3.4.5")
		assert.NoError(t, err)
	}
	{
		ok, err := set.Test(ctx, setname, "1.2.3.4")
		assert.NoError(t, err)
		assert.True(t, ok)
		ok, err = set.Test(ctx, setname, "2.3.4.5")
		assert.NoError(t, err)
		assert.True(t, ok)
		ok, err = set.Test(ctx, setname, "3.3.3.3")
		assert.NoError(t, err)
		assert.False(t, ok)
	}
	{
		err = set.Del(ctx, setname, "1.2.3.4")
		assert.NoError(t, err)
		ok, err := set.Test(ctx, setname, "1.2.3.4")
		assert.NoError(t, err)
		assert.False(t, ok)
		ok, err = set.Test(ctx, setname, "2.3.4.5")
		assert.NoError(t, err)
		assert.True(t, ok)
	}
	{
		err = set.Destroy(ctx, setname)
		assert.NoError(t, err)
		_, err := set.Test(ctx, setname, "1.2.3.4")
		assert.Error(t, err)
		_, err = set.Test(ctx, setname, "2.3.4.5")
		assert.Error(t, err)
	}
	{
		err = set.Destroy(ctx, "aaaa", WithExist())
		assert.NoError(t, err)
		err = set.Destroy(ctx, "aaaa")
		assert.Error(t, err)
	}

}

func TestListRaw(t *testing.T) {
	set := MustNew()
	ctx := context.Background()
	setname := "test_set"
	err := set.Create(ctx, setname, SetTypeHashNet, WithExist())
	assert.NoError(t, err)
	data, err := set.ListRaw(ctx, setname, WithOutput(OutputTypeXml))
	assert.NoError(t, err)
	t.Logf("data:%s", string(data))
}

func TestList(t *testing.T) {
	set := MustNew()
	ctx := context.Background()
	setname := "test_set"
	err := set.Create(ctx, setname, SetTypeHashNet, WithExist())
	assert.NoError(t, err)
	defer set.Destroy(ctx, setname)
	set.Add(ctx, setname, "1.2.3.4", WithExist())
	header, ips, err := set.List(ctx, setname)
	assert.NoError(t, err)
	t.Logf("header:%+v", *header)
	t.Logf("ips:%+v", ips)
}

func TestCidr(t *testing.T) {
	set := MustNew()
	setname := "test_set"
	ctx := context.Background()
	err := set.Create(ctx, setname, SetTypeHashNet, WithExist())
	assert.NoError(t, err)
	defer set.Destroy(ctx, setname, WithExist())
	err = set.Add(ctx, setname, "62.133.47.0/24")
	assert.NoError(t, err)
	err = set.Add(ctx, setname, "62.133.47.0/24", WithExist())
	assert.NoError(t, err)
	err = set.Add(ctx, setname, "62.133.47.0/24")
	assert.Error(t, err)
}

func TestRestore(t *testing.T) {
	set := MustNew()
	setname := "test_restore"
	ctx := context.Background()
	err := set.Create(ctx, setname, SetTypeHashNet, WithExist())
	assert.NoError(t, err)
	defer set.Destroy(ctx, setname)
	err = set.Restore(ctx, setname, []string{"5.5.5.5", "5.5.5.6", "5.5.5.7", "6.6.6.0/10"})
	assert.NoError(t, err)
}

func TestCreateWithMaxElem(t *testing.T) {
	set := MustNew()
	setname := "test_set"
	ctx := context.Background()
	err := set.Create(ctx, setname, SetTypeHashNet, WithExist(), WithMaxElement(10))
	assert.NoError(t, err)
	defer set.Destroy(ctx, setname)
}
