package launchdarkly

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(v int64) *int64 {
	return &v
}

func TestFetchAllOffsetPages_UsesTotalCount(t *testing.T) {
	t.Parallel()

	offsets := make([]int64, 0)
	items, err := fetchAllOffsetPages[int](2, 0, func(offset, limit int64) (offsetPage[int], error) {
		offsets = append(offsets, offset)
		switch offset {
		case 0:
			return offsetPage[int]{items: []int{1, 2}, totalCount: int64Ptr(5)}, nil
		case 2:
			return offsetPage[int]{items: []int{3, 4}, totalCount: int64Ptr(5)}, nil
		case 4:
			return offsetPage[int]{items: []int{5}, totalCount: int64Ptr(5)}, nil
		default:
			return offsetPage[int]{items: []int{}, totalCount: int64Ptr(5)}, nil
		}
	})
	require.NoError(t, err)

	assert.Equal(t, []int{1, 2, 3, 4, 5}, items)
	assert.Equal(t, []int64{0, 2, 4}, offsets)
}

func TestFetchAllOffsetPages_StopsWithoutTotalCount(t *testing.T) {
	t.Parallel()

	offsets := make([]int64, 0)
	items, err := fetchAllOffsetPages[int](2, 0, func(offset, limit int64) (offsetPage[int], error) {
		offsets = append(offsets, offset)
		switch offset {
		case 0:
			return offsetPage[int]{items: []int{1, 2}}, nil
		case 2:
			return offsetPage[int]{items: []int{3}}, nil
		default:
			return offsetPage[int]{items: []int{}}, nil
		}
	})
	require.NoError(t, err)

	assert.Equal(t, []int{1, 2, 3}, items)
	assert.Equal(t, []int64{0, 2}, offsets)
}

func TestFetchAllOffsetPages_RespectsInitialOffset(t *testing.T) {
	t.Parallel()

	offsets := make([]int64, 0)
	items, err := fetchAllOffsetPages[int](2, 4, func(offset, limit int64) (offsetPage[int], error) {
		offsets = append(offsets, offset)
		switch offset {
		case 4:
			return offsetPage[int]{items: []int{5, 6}, totalCount: int64Ptr(7)}, nil
		case 6:
			return offsetPage[int]{items: []int{7}, totalCount: int64Ptr(7)}, nil
		default:
			return offsetPage[int]{items: []int{}, totalCount: int64Ptr(7)}, nil
		}
	})
	require.NoError(t, err)

	assert.Equal(t, []int{5, 6, 7}, items)
	assert.Equal(t, []int64{4, 6}, offsets)
}

func TestFetchAllOffsetPages_StopsOnEmptyPage(t *testing.T) {
	t.Parallel()

	callCount := 0
	items, err := fetchAllOffsetPages[int](2, 0, func(offset, limit int64) (offsetPage[int], error) {
		callCount++
		return offsetPage[int]{items: []int{}, totalCount: int64Ptr(10)}, nil
	})
	require.NoError(t, err)

	assert.Empty(t, items)
	assert.Equal(t, 1, callCount)
}

func TestFetchAllOffsetPages_PropagatesError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("boom")
	_, err := fetchAllOffsetPages[int](2, 0, func(offset, limit int64) (offsetPage[int], error) {
		return offsetPage[int]{}, expectedErr
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestFetchAllOffsetPages_ReturnsPartialItemsOnError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("boom")
	items, err := fetchAllOffsetPages[int](2, 0, func(offset, limit int64) (offsetPage[int], error) {
		if offset == 0 {
			return offsetPage[int]{items: []int{1, 2}, totalCount: int64Ptr(5)}, nil
		}
		return offsetPage[int]{}, expectedErr
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, []int{1, 2}, items)
}

func TestFetchAllOffsetPages_RejectsInvalidLimit(t *testing.T) {
	t.Parallel()

	_, err := fetchAllOffsetPages[int](0, 0, func(offset, limit int64) (offsetPage[int], error) {
		return offsetPage[int]{}, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pagination limit must be greater than zero")
}

func TestFetchAllOffsetPagesWithOptionalInt32Total(t *testing.T) {
	t.Parallel()

	offsets := make([]int64, 0)
	items, err := fetchAllOffsetPagesWithOptionalInt32Total[int](2, 0, func(offset, limit int64) ([]int, *int32, error) {
		offsets = append(offsets, offset)
		switch offset {
		case 0:
			total := int32(3)
			return []int{1, 2}, &total, nil
		case 2:
			total := int32(3)
			return []int{3}, &total, nil
		default:
			return []int{}, nil, nil
		}
	})
	require.NoError(t, err)

	assert.Equal(t, []int{1, 2, 3}, items)
	assert.Equal(t, []int64{0, 2}, offsets)
}

func TestFetchAllOffsetPagesWithInt32Total(t *testing.T) {
	t.Parallel()

	offsets := make([]int64, 0)
	items, err := fetchAllOffsetPagesWithInt32Total[int](2, 0, func(offset, limit int64) ([]int, int32, error) {
		offsets = append(offsets, offset)
		switch offset {
		case 0:
			return []int{1, 2}, 3, nil
		case 2:
			return []int{3}, 3, nil
		default:
			return []int{}, 3, nil
		}
	})
	require.NoError(t, err)

	assert.Equal(t, []int{1, 2, 3}, items)
	assert.Equal(t, []int64{0, 2}, offsets)
}
